package identity

import (
	"context"
	"sort"
	"testing"
	"time"

	"telefraud/internal/models"
)

// fakeStore is an in-memory Store for exercising resolver logic without
// PostgreSQL. It mirrors the postgres store's semantics.
type fakeStore struct {
	scammers  map[int64]*models.Scammer
	idents    map[int64]models.ScammerIdentifier
	nextScam  int64
	nextIdent int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		scammers:  map[int64]*models.Scammer{},
		idents:    map[int64]models.ScammerIdentifier{},
		nextScam:  1,
		nextIdent: 1,
	}
}

func (f *fakeStore) FindByValue(_ context.Context, kind models.IdentifierKind, value string) ([]models.ScammerIdentifier, error) {
	var out []models.ScammerIdentifier
	for _, i := range f.idents {
		if i.Kind == kind && i.Value == value {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, id int64) (*models.Scammer, []models.ScammerIdentifier, error) {
	s, ok := f.scammers[id]
	if !ok {
		return nil, nil, errNotFound
	}
	var idents []models.ScammerIdentifier
	for _, i := range f.idents {
		if i.ScammerID == id {
			idents = append(idents, i)
		}
	}
	sort.Slice(idents, func(a, b int) bool { return idents[a].ID < idents[b].ID })
	return s, idents, nil
}

func (f *fakeStore) Create(_ context.Context, s *models.Scammer, idents []models.ScammerIdentifier) (int64, error) {
	id := f.nextScam
	f.nextScam++
	cp := *s
	cp.ID = id
	f.scammers[id] = &cp
	for _, ident := range idents {
		iid := f.nextIdent
		f.nextIdent++
		ident.ID = iid
		ident.ScammerID = id
		f.idents[iid] = ident
	}
	return id, nil
}

func (f *fakeStore) Update(_ context.Context, s *models.Scammer) error {
	if _, ok := f.scammers[s.ID]; !ok {
		return errNotFound
	}
	cp := *s
	f.scammers[s.ID] = &cp
	return nil
}

func (f *fakeStore) AddIdentifier(_ context.Context, ident models.ScammerIdentifier) error {
	for id, i := range f.idents {
		if i.Kind == ident.Kind && i.Value == ident.Value {
			i.LastSeenAt = ident.LastSeenAt
			f.idents[id] = i
			return nil
		}
	}
	iid := f.nextIdent
	f.nextIdent++
	ident.ID = iid
	f.idents[iid] = ident
	return nil
}

func (f *fakeStore) TouchIdentifier(_ context.Context, identifierID int64) error {
	i, ok := f.idents[identifierID]
	if !ok {
		return errNotFound
	}
	i.LastSeenAt = time.Now()
	f.idents[identifierID] = i
	return nil
}

func (f *fakeStore) Merge(_ context.Context, keepID, dropID int64, moveIDs, deleteIDs []int64, attrs models.Scammer) error {
	for _, id := range moveIDs {
		i := f.idents[id]
		i.ScammerID = keepID
		f.idents[id] = i
	}
	for _, id := range deleteIDs {
		delete(f.idents, id)
	}
	attrs.ID = keepID
	f.scammers[keepID] = &attrs
	delete(f.scammers, dropID)
	return nil
}

func newResolver() *Resolver {
	return NewResolver(newFakeStore())
}

func TestResolveNoMatch(t *testing.T) {
	r := newResolver()
	_, err := r.Resolve(context.Background(), Candidates{UserID: 123456789})
	if err != ErrNoMatch {
		t.Fatalf("Resolve() error = %v; want ErrNoMatch", err)
	}
}

func TestResolveEmpty(t *testing.T) {
	r := newResolver()
	_, err := r.Resolve(context.Background(), Candidates{})
	if err != ErrNoIdentifiers {
		t.Fatalf("Resolve() error = %v; want ErrNoIdentifiers", err)
	}
}

func TestRegisterThenResolve(t *testing.T) {
	ctx := context.Background()
	r := newResolver()

	s, err := r.Register(ctx, Candidates{Username: "scammer"}, "fishing", "FINANCIAL_SCAM", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if s.ID == 0 {
		t.Fatalf("Register returned zero id")
	}

	got, err := r.Resolve(ctx, Candidates{Username: "scammer"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.ID != s.ID {
		t.Fatalf("Resolve id = %d; want %d", got.ID, s.ID)
	}
}

// The core requirement: a username maps to a user id, and once linked the two
// always resolve to the same canonical scammer.
func TestUsernameToUserIDLink(t *testing.T) {
	ctx := context.Background()
	r := newResolver()

	s, err := r.Register(ctx, Candidates{Username: "scammer"}, "spam", "", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Later, the scammer's user id is discovered.
	_, err = r.Link(ctx, s.ID, models.KindUserID, models.UserIDValue(123456789), models.SourceReport)
	if err != nil {
		t.Fatalf("Link: %v", err)
	}

	byUsername, err := r.Resolve(ctx, Candidates{Username: "scammer"})
	if err != nil {
		t.Fatalf("Resolve by username: %v", err)
	}
	byUserID, err := r.Resolve(ctx, Candidates{UserID: 123456789})
	if err != nil {
		t.Fatalf("Resolve by user id: %v", err)
	}
	if byUsername.ID != byUserID.ID || byUserID.ID != s.ID {
		t.Fatalf("username and user id resolve to different scammers: %d vs %d (want %d)",
			byUsername.ID, byUserID.ID, s.ID)
	}
}

// A scammer changes username; both the old and new handle still resolve to the
// same entity because the immutable user id ties them together.
func TestUsernameChangeStillResolves(t *testing.T) {
	ctx := context.Background()
	r := newResolver()

	s, err := r.Register(ctx, Candidates{UserID: 123456789, Username: "oldhandle"}, "spam", "", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// The scammer changes their username; the bot observes the new handle for
	// the same user id and links it.
	linked, err := r.Link(ctx, s.ID, models.KindUsername, "newhandle", models.SourceJoinScan)
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	if linked.ID != s.ID {
		t.Fatalf("Link returned id %d; want %d", linked.ID, s.ID)
	}

	for _, cand := range []Candidates{
		{UserID: 123456789},
		{Username: "oldhandle"},
		{Username: "newhandle"},
	} {
		got, err := r.Resolve(ctx, cand)
		if err != nil {
			t.Fatalf("Resolve(%+v): %v", cand, err)
		}
		if got.ID != s.ID {
			t.Fatalf("Resolve(%+v) = id %d; want %d", cand, got.ID, s.ID)
		}
	}

	_, idents, err := r.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(idents) != 3 {
		t.Fatalf("identifiers = %d; want 3 (user id + two usernames)", len(idents))
	}
}

// Two scammers reported under different identifiers are discovered to be the
// same person when a report links them; the entities merge.
func TestMergeOnLinkedIdentifiers(t *testing.T) {
	ctx := context.Background()
	r := newResolver()

	first, err := r.Register(ctx, Candidates{Username: "scammer"}, "spam", "", nil)
	if err != nil {
		t.Fatalf("Register 1: %v", err)
	}
	second, err := r.Register(ctx, Candidates{UserID: 555555}, "spam", "", nil)
	if err != nil {
		t.Fatalf("Register 2: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected distinct entities, got %d", first.ID)
	}

	// A report carries both identifiers, linking the two entities.
	merged, err := r.Register(ctx, Candidates{UserID: 555555, Username: "scammer"}, "spam", "", nil)
	if err != nil {
		t.Fatalf("Register 3: %v", err)
	}

	// The keeper is the older (lower id) entity; report counts sum to 3.
	if merged.ID != first.ID {
		t.Fatalf("merged id = %d; want keeper %d", merged.ID, first.ID)
	}
	if merged.ReportCount != 3 {
		t.Fatalf("report count = %d; want 3", merged.ReportCount)
	}

	// The folded entity is gone, and both identifiers resolve to the keeper.
	if _, _, err := r.Get(ctx, second.ID); err == nil {
		t.Fatalf("folded entity %d still present", second.ID)
	}
	for _, cand := range []Candidates{{Username: "scammer"}, {UserID: 555555}} {
		got, err := r.Resolve(ctx, cand)
		if err != nil {
			t.Fatalf("Resolve(%+v): %v", cand, err)
		}
		if got.ID != first.ID {
			t.Fatalf("Resolve(%+v) = %d; want %d", cand, got.ID, first.ID)
		}
	}
}

func TestMergeAttributes(t *testing.T) {
	keep := &models.Scammer{ID: 1, ReportCount: 2, ThreatLevel: models.ThreatHigh, Category: "A", Reason: "r1"}
	drop := &models.Scammer{ID: 2, ReportCount: 5, ThreatLevel: models.ThreatCritical, Category: "B", Reason: "r2"}

	got := mergeAttributes(keep, drop)
	if got.ReportCount != 7 {
		t.Fatalf("ReportCount = %d; want 7", got.ReportCount)
	}
	if got.ThreatLevel != models.ThreatCritical {
		t.Fatalf("ThreatLevel = %s; want CRITICAL (stronger wins)", got.ThreatLevel)
	}
	if got.Category != "A" {
		t.Fatalf("Category = %s; want keeper's value A", got.Category)
	}
	if got.Reason != "r1" {
		t.Fatalf("Reason = %s; want keeper's value r1", got.Reason)
	}
}

func TestMergeIdentifierPlan(t *testing.T) {
	keepIdents := []models.ScammerIdentifier{
		{ID: 1, Kind: models.KindUsername, Value: "scammer"},
		{ID: 2, Kind: models.KindPhone, Value: "+12345678901"},
	}
	dropIdents := []models.ScammerIdentifier{
		{ID: 3, Kind: models.KindUserID, Value: "555555"},      // unique -> move
		{ID: 4, Kind: models.KindUsername, Value: "scammer"},   // dup -> delete
		{ID: 5, Kind: models.KindUsername, Value: "oldhandle"}, // unique -> move
	}

	move, del := mergeIdentifierPlan(keepIdents, dropIdents)
	if len(move) != 2 || move[0] != 3 || move[1] != 5 {
		t.Fatalf("move = %v; want [3 5]", move)
	}
	if len(del) != 1 || del[0] != 4 {
		t.Fatalf("delete = %v; want [4]", del)
	}
}
