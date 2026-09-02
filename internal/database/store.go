package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"telefraud/internal/models"
)

// IdentityStore is the PostgreSQL implementation of identity.Store.
type IdentityStore struct {
	pool *pgxpool.Pool
}

// NewIdentityStore builds an IdentityStore over a pool.
func NewIdentityStore(pool *pgxpool.Pool) *IdentityStore {
	return &IdentityStore{pool: pool}
}

const scammerColumns = "id, status, threat_level, report_count, COALESCE(category, ''), reason, added_by, created_at, updated_at"

const identifierColumns = "id, scammer_id, kind, value, is_primary, source, first_seen_at, last_seen_at"

func scanScammer(row pgx.Row) (*models.Scammer, error) {
	var s models.Scammer
	if err := row.Scan(
		&s.ID, &s.Status, &s.ThreatLevel, &s.ReportCount, &s.Category, &s.Reason,
		&s.AddedBy, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

func scanIdentifier(row pgx.Row) (models.ScammerIdentifier, error) {
	var i models.ScammerIdentifier
	var kind string
	if err := row.Scan(
		&i.ID, &i.ScammerID, &kind, &i.Value, &i.IsPrimary, &i.Source,
		&i.FirstSeenAt, &i.LastSeenAt,
	); err != nil {
		return i, err
	}
	i.Kind = models.IdentifierKind(kind)
	return i, nil
}

// FindByValue returns identifiers matching (kind, value); empty if none.
func (s *IdentityStore) FindByValue(ctx context.Context, kind models.IdentifierKind, value string) ([]models.ScammerIdentifier, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT "+identifierColumns+" FROM scammer_identifiers WHERE kind = $1 AND value = $2",
		string(kind), value,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ScammerIdentifier
	for rows.Next() {
		i, err := scanIdentifier(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// ListIdentifiers returns all identifiers for a scammer.
func (s *IdentityStore) ListIdentifiers(ctx context.Context, scammerID int64) ([]models.ScammerIdentifier, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT "+identifierColumns+" FROM scammer_identifiers WHERE scammer_id = $1 ORDER BY id",
		scammerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ScammerIdentifier
	for rows.Next() {
		i, err := scanIdentifier(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// Get returns a scammer entity and its identifiers by id.
func (s *IdentityStore) Get(ctx context.Context, id int64) (*models.Scammer, []models.ScammerIdentifier, error) {
	scam, err := scanScammer(s.pool.QueryRow(ctx, "SELECT "+scammerColumns+" FROM scammers WHERE id = $1", id))
	if err != nil {
		return nil, nil, err
	}
	idents, err := s.ListIdentifiers(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return scam, idents, nil
}

// Create inserts a scammer entity plus identifiers, returning the new id.
func (s *IdentityStore) Create(ctx context.Context, scam *models.Scammer, idents []models.ScammerIdentifier) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id int64
	err = tx.QueryRow(ctx,
		"INSERT INTO scammers (status, threat_level, report_count, category, reason, added_by) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		scam.Status, scam.ThreatLevel, scam.ReportCount, scam.Category, scam.Reason, scam.AddedBy,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	for _, ident := range idents {
		if _, err := tx.Exec(ctx,
			"INSERT INTO scammer_identifiers (scammer_id, kind, value, is_primary, source, first_seen_at, last_seen_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			id, string(ident.Kind), ident.Value, ident.IsPrimary, ident.Source, ident.FirstSeenAt, ident.LastSeenAt,
		); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

// Update persists entity attribute changes.
func (s *IdentityStore) Update(ctx context.Context, scam *models.Scammer) error {
	_, err := s.pool.Exec(ctx,
		"UPDATE scammers SET status = $1, threat_level = $2, report_count = $3, category = $4, reason = $5, added_by = $6, updated_at = NOW() WHERE id = $7",
		scam.Status, scam.ThreatLevel, scam.ReportCount, scam.Category, scam.Reason, scam.AddedBy, scam.ID,
	)
	return err
}

// AddIdentifier links a new identifier to a scammer; if (kind, value) already
// exists it only refreshes last_seen_at.
func (s *IdentityStore) AddIdentifier(ctx context.Context, ident models.ScammerIdentifier) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO scammer_identifiers (scammer_id, kind, value, is_primary, source, first_seen_at, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (kind, value) DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at`,
		ident.ScammerID, string(ident.Kind), ident.Value, ident.IsPrimary, ident.Source, ident.FirstSeenAt, ident.LastSeenAt,
	)
	return err
}

// TouchIdentifier refreshes last_seen_at on an existing identifier.
func (s *IdentityStore) TouchIdentifier(ctx context.Context, identifierID int64) error {
	_, err := s.pool.Exec(ctx, "UPDATE scammer_identifiers SET last_seen_at = NOW() WHERE id = $1", identifierID)
	return err
}

// Merge folds dropID into keepID atomically.
func (s *IdentityStore) Merge(ctx context.Context, keepID, dropID int64, moveIDs, deleteIDs []int64, attrs models.Scammer) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, identID := range moveIDs {
		if _, err := tx.Exec(ctx,
			"UPDATE scammer_identifiers SET scammer_id = $1 WHERE id = $2 AND scammer_id = $3",
			keepID, identID, dropID,
		); err != nil {
			return err
		}
	}
	for _, identID := range deleteIDs {
		if _, err := tx.Exec(ctx,
			"DELETE FROM scammer_identifiers WHERE id = $1 AND scammer_id = $2",
			identID, dropID,
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx,
		"UPDATE scammers SET report_count = $1, threat_level = $2, category = $3, reason = $4, updated_at = NOW() WHERE id = $5",
		attrs.ReportCount, attrs.ThreatLevel, attrs.Category, attrs.Reason, keepID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "DELETE FROM scammers WHERE id = $1", dropID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
