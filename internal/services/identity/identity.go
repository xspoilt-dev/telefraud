// Package identity resolves Telegram accounts into canonical scammer entities.
//
// A scammer changes usernames and phone numbers, but the numeric user id (when
// known) is immutable. The resolver links every observed identifier — user id,
// username, phone — onto one canonical entity, so a lookup by any identifier
// finds the same scammer, and newly discovered identifiers are folded in.
package identity

import (
	"context"
	"errors"
	"sort"
	"time"

	"telefraud/internal/models"
)

var (
	// ErrNoMatch reports that no scammer entity matches the given identifiers.
	ErrNoMatch = errors.New("identity: no matching scammer")
	// ErrNoIdentifiers reports that no identifiers were supplied.
	ErrNoIdentifiers = errors.New("identity: no identifiers provided")
	// errNotFound reports a missing scammer entity (programmer error).
	errNotFound = errors.New("identity: scammer not found")
)

// Store is the persistence boundary for the resolver. Implementations must be
// safe for concurrent use. Merge is the only multi-step mutation and must be
// atomic.
type Store interface {
	// FindByValue returns identifiers matching (kind, value); empty if none.
	FindByValue(ctx context.Context, kind models.IdentifierKind, value string) ([]models.ScammerIdentifier, error)
	// Get returns a scammer entity and its identifiers by id.
	Get(ctx context.Context, id int64) (*models.Scammer, []models.ScammerIdentifier, error)
	// Create inserts a scammer entity plus identifiers, returning the new id.
	Create(ctx context.Context, s *models.Scammer, idents []models.ScammerIdentifier) (int64, error)
	// Update persists entity attribute changes.
	Update(ctx context.Context, s *models.Scammer) error
	// AddIdentifier links a new identifier to a scammer; if (kind, value)
	// already exists it only refreshes last_seen_at.
	AddIdentifier(ctx context.Context, ident models.ScammerIdentifier) error
	// TouchIdentifier refreshes last_seen_at on an existing identifier.
	TouchIdentifier(ctx context.Context, identifierID int64) error
	// Merge folds dropID into keepID atomically: reassigns moveIDs, deletes
	// deleteIDs, applies attrs to the keeper, and removes the dropID entity.
	Merge(ctx context.Context, keepID, dropID int64, moveIDs, deleteIDs []int64, attrs models.Scammer) error
}

// Candidates is a set of normalized identifiers observed for a target. Zero
// values mean "unknown". Username and Phone must already be normalized via
// models.NormalizeUsername / models.NormalizePhone; UserID must be positive.
type Candidates struct {
	UserID   int64
	Username string
	Phone    string
}

// Empty reports whether no identifier is present.
func (c Candidates) Empty() bool {
	return c.UserID == 0 && c.Username == "" && c.Phone == ""
}

type pair struct {
	kind  models.IdentifierKind
	value string
}

func (c Candidates) pairs() []pair {
	out := make([]pair, 0, 3)
	if c.UserID != 0 {
		out = append(out, pair{models.KindUserID, models.UserIDValue(c.UserID)})
	}
	if c.Username != "" {
		out = append(out, pair{models.KindUsername, c.Username})
	}
	if c.Phone != "" {
		out = append(out, pair{models.KindPhone, c.Phone})
	}
	return out
}

// Resolver links identifiers to canonical scammer entities.
type Resolver struct {
	store Store
}

// NewResolver builds a Resolver over the given store.
func NewResolver(s Store) *Resolver {
	return &Resolver{store: s}
}

// Resolve finds the canonical scammer matching any of the candidates'
// identifiers, merging distinct entities when a candidate set links them.
// It returns ErrNoMatch when nothing matches.
func (r *Resolver) Resolve(ctx context.Context, c Candidates) (*models.Scammer, error) {
	if c.Empty() {
		return nil, ErrNoIdentifiers
	}
	ids, err := r.matchIDs(ctx, c)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNoMatch
	}
	keep, err := r.mergeIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return r.get(ctx, keep)
}

// Get returns a scammer entity with all its linked identifiers.
func (r *Resolver) Get(ctx context.Context, id int64) (*models.Scammer, []models.ScammerIdentifier, error) {
	return r.store.Get(ctx, id)
}

// Link attaches a newly observed identifier to a scammer. If the identifier
// already belongs to a different entity, the two entities are merged. Returns
// the canonical scammer.
func (r *Resolver) Link(ctx context.Context, scammerID int64, kind models.IdentifierKind, value, source string) (*models.Scammer, error) {
	found, err := r.store.FindByValue(ctx, kind, value)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		now := time.Now()
		if err := r.store.AddIdentifier(ctx, models.ScammerIdentifier{
			ScammerID:   scammerID,
			Kind:        kind,
			Value:       value,
			Source:      source,
			FirstSeenAt: now,
			LastSeenAt:  now,
		}); err != nil {
			return nil, err
		}
		return r.get(ctx, scammerID)
	}

	ident := found[0]
	if ident.ScammerID == scammerID {
		if err := r.store.TouchIdentifier(ctx, ident.ID); err != nil {
			return nil, err
		}
		return r.get(ctx, scammerID)
	}

	// Same identifier under a different entity → fold the two together.
	keep, err := r.mergeTwo(ctx, scammerID, ident.ScammerID)
	if err != nil {
		return nil, err
	}
	return r.get(ctx, keep)
}

// Register records a target (typically from a report): it resolves any existing
// entity, links new identifiers, bumps the report count, or creates a new
// entity when nothing matches. Returns the canonical scammer.
func (r *Resolver) Register(ctx context.Context, c Candidates, reason, category string, addedBy *int64) (*models.Scammer, error) {
	if c.Empty() {
		return nil, ErrNoIdentifiers
	}

	s, err := r.Resolve(ctx, c)
	if err == nil {
		for _, p := range c.pairs() {
			if _, err := r.Link(ctx, s.ID, p.kind, p.value, models.SourceReport); err != nil {
				return nil, err
			}
		}
		s.ReportCount++
		if s.Category == "" {
			s.Category = category
		}
		if err := r.store.Update(ctx, s); err != nil {
			return nil, err
		}
		return r.get(ctx, s.ID)
	}
	if !errors.Is(err, ErrNoMatch) {
		return nil, err
	}

	now := time.Now()
	idents := make([]models.ScammerIdentifier, 0, len(c.pairs()))
	for i, p := range c.pairs() {
		// The user id is the strongest, most stable identity; prefer it as
		// primary. Fall back to the first identifier when no user id is known.
		primary := p.kind == models.KindUserID
		if i == 0 && c.UserID == 0 {
			primary = true
		}
		idents = append(idents, models.ScammerIdentifier{
			Kind:        p.kind,
			Value:       p.value,
			IsPrimary:   primary,
			Source:      models.SourceReport,
			FirstSeenAt: now,
			LastSeenAt:  now,
		})
	}

	id, err := r.store.Create(ctx, &models.Scammer{
		Status:      models.StatusPending,
		ThreatLevel: models.ThreatMedium,
		ReportCount: 1,
		Category:    category,
		Reason:      reason,
		AddedBy:     addedBy,
	}, idents)
	if err != nil {
		return nil, err
	}
	return r.get(ctx, id)
}

// matchIDs returns the distinct scammer ids referenced by any candidate.
func (r *Resolver) matchIDs(ctx context.Context, c Candidates) ([]int64, error) {
	seen := map[int64]struct{}{}
	for _, p := range c.pairs() {
		found, err := r.store.FindByValue(ctx, p.kind, p.value)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			seen[f.ScammerID] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// mergeIDs folds a set of distinct entity ids into the oldest one (lowest id).
func (r *Resolver) mergeIDs(ctx context.Context, ids []int64) (int64, error) {
	keep := ids[0]
	for _, drop := range ids[1:] {
		if drop == keep {
			continue
		}
		var err error
		keep, err = r.mergeTwo(ctx, keep, drop)
		if err != nil {
			return 0, err
		}
	}
	return keep, nil
}

// mergeTwo folds dropID into keepID and returns the keeper id.
func (r *Resolver) mergeTwo(ctx context.Context, keepID, dropID int64) (int64, error) {
	if keepID == dropID {
		return keepID, nil
	}

	keep, keepIdents, err := r.store.Get(ctx, keepID)
	if err != nil {
		return 0, err
	}
	drop, dropIdents, err := r.store.Get(ctx, dropID)
	if err != nil {
		return 0, err
	}

	move, del := mergeIdentifierPlan(keepIdents, dropIdents)
	attrs := mergeAttributes(keep, drop)
	if err := r.store.Merge(ctx, keepID, dropID, move, del, attrs); err != nil {
		return 0, err
	}
	return keepID, nil
}

func (r *Resolver) get(ctx context.Context, id int64) (*models.Scammer, error) {
	s, _, err := r.store.Get(ctx, id)
	return s, err
}

// mergeIdentifierPlan partitions the source's identifiers into those to
// reassign onto the keeper (move) and duplicates to drop (delete).
func mergeIdentifierPlan(keepIdents, dropIdents []models.ScammerIdentifier) (move, delete []int64) {
	existing := make(map[[2]string]struct{}, len(keepIdents))
	for _, i := range keepIdents {
		existing[[2]string{string(i.Kind), i.Value}] = struct{}{}
	}
	for _, d := range dropIdents {
		key := [2]string{string(d.Kind), d.Value}
		if _, dup := existing[key]; dup {
			delete = append(delete, d.ID)
			continue
		}
		move = append(move, d.ID)
		existing[key] = struct{}{}
	}
	return move, delete
}

// mergeAttributes combines two entity records: counts sum, the stronger threat
// level wins, and empty keeper fields are backfilled from the source.
func mergeAttributes(keep, drop *models.Scammer) models.Scammer {
	out := *keep
	out.ReportCount = keep.ReportCount + drop.ReportCount
	if threatRank(drop.ThreatLevel) > threatRank(keep.ThreatLevel) {
		out.ThreatLevel = drop.ThreatLevel
	}
	if out.Category == "" {
		out.Category = drop.Category
	}
	if out.Reason == "" {
		out.Reason = drop.Reason
	}
	return out
}

func threatRank(level string) int {
	switch level {
	case models.ThreatCritical:
		return 4
	case models.ThreatHigh:
		return 3
	case models.ThreatMedium:
		return 2
	case models.ThreatLow:
		return 1
	default:
		return 0
	}
}
