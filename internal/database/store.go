package database

import (
	"context"
	"errors"
	"time"

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
const userColumns = "user_id, COALESCE(username, ''), first_name, COALESCE(last_name, ''), role, is_banned, COALESCE(language, 'en'), created_at, updated_at"
const groupColumns = "group_id, title, COALESCE(username, ''), COALESCE(language, 'en'), auto_ban_enabled, auto_delete_enabled, scan_on_join_enabled, daily_scan_enabled, warn_on_detected, last_scanned_at, joined_at, updated_at"
const reportColumns = "id, reporter_id, scammer_id, target_user_id, COALESCE(target_username, ''), COALESCE(target_phone, ''), category, description, status, reviewer_id, COALESCE(reviewer_notes, ''), created_at, reviewed_at"
const proofColumns = "id, report_id, file_id, COALESCE(file_unique_id, ''), file_type, COALESCE(caption, ''), uploaded_at"
const modLogColumns = "id, group_id, scammer_id, target_user_id, action, reason, executed_by, created_at"

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

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	if err := row.Scan(
		&u.UserID, &u.Username, &u.FirstName, &u.LastName, &u.Role,
		&u.IsBanned, &u.Language, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func scanGroup(row pgx.Row) (*models.Group, error) {
	var g models.Group
	if err := row.Scan(
		&g.GroupID, &g.Title, &g.Username, &g.Language, &g.AutoBanEnabled, &g.AutoDeleteEnabled,
		&g.ScanOnJoinEnabled, &g.DailyScanEnabled, &g.WarnOnDetected,
		&g.LastScannedAt, &g.JoinedAt, &g.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &g, nil
}

func scanReport(row pgx.Row) (*models.Report, error) {
	var r models.Report
	if err := row.Scan(
		&r.ID, &r.ReporterID, &r.ScammerID, &r.TargetUserID, &r.TargetUsername,
		&r.TargetPhone, &r.Category, &r.Description, &r.Status, &r.ReviewerID,
		&r.ReviewerNotes, &r.CreatedAt, &r.ReviewedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func scanProof(row pgx.Row) (models.ReportProof, error) {
	var p models.ReportProof
	if err := row.Scan(
		&p.ID, &p.ReportID, &p.FileID, &p.FileUniqueID, &p.FileType,
		&p.Caption, &p.UploadedAt,
	); err != nil {
		return p, err
	}
	return p, nil
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

// --- USER MANAGEMENT ---

// UpsertUser inserts or updates a registered user record.
func (s *IdentityStore) UpsertUser(ctx context.Context, u models.User) error {
	if u.Role == "" {
		u.Role = models.RoleUser
	}
	if u.Language == "" {
		u.Language = "en"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (user_id, username, first_name, last_name, role, is_banned, language, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   username = EXCLUDED.username,
		   first_name = EXCLUDED.first_name,
		   last_name = EXCLUDED.last_name,
		   updated_at = NOW()`,
		u.UserID, u.Username, u.FirstName, u.LastName, u.Role, u.IsBanned, u.Language,
	)
	return err
}

// GetUser fetches a user by user_id.
func (s *IdentityStore) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	return scanUser(s.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE user_id = $1", userID))
}

// SetUserRole modifies user's role (e.g. 'ADMIN', 'USER').
func (s *IdentityStore) SetUserRole(ctx context.Context, userID int64, role string) error {
	_, err := s.pool.Exec(ctx, "UPDATE users SET role = $1, updated_at = NOW() WHERE user_id = $2", role, userID)
	return err
}

// SetUserLanguage modifies user's preferred language.
func (s *IdentityStore) SetUserLanguage(ctx context.Context, userID int64, lang string) error {
	_, err := s.pool.Exec(ctx, "UPDATE users SET language = $1, updated_at = NOW() WHERE user_id = $2", lang, userID)
	return err
}

// SetUserBanned bans or unbans a user from bot interaction.
func (s *IdentityStore) SetUserBanned(ctx context.Context, userID int64, isBanned bool) error {
	_, err := s.pool.Exec(ctx, "UPDATE users SET is_banned = $1, updated_at = NOW() WHERE user_id = $2", isBanned, userID)
	return err
}

// ListAdmins returns all users with ADMIN or SUPERADMIN roles.
func (s *IdentityStore) ListAdmins(ctx context.Context) ([]models.User, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+userColumns+" FROM users WHERE role IN ('ADMIN', 'SUPERADMIN') ORDER BY user_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		admins = append(admins, *u)
	}
	return admins, rows.Err()
}

// GetAllUserIDs returns all registered user IDs.
func (s *IdentityStore) GetAllUserIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.pool.Query(ctx, "SELECT user_id FROM users WHERE is_banned = FALSE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// --- GROUP MANAGEMENT ---

// UpsertGroup inserts or updates a group record.
func (s *IdentityStore) UpsertGroup(ctx context.Context, g models.Group) error {
	if g.Language == "" {
		g.Language = "en"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO groups (group_id, title, username, language, auto_ban_enabled, auto_delete_enabled, scan_on_join_enabled, daily_scan_enabled, warn_on_detected, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		 ON CONFLICT (group_id) DO UPDATE SET
		   title = EXCLUDED.title,
		   username = EXCLUDED.username,
		   updated_at = NOW()`,
		g.GroupID, g.Title, g.Username, g.Language, g.AutoBanEnabled, g.AutoDeleteEnabled,
		g.ScanOnJoinEnabled, g.DailyScanEnabled, g.WarnOnDetected,
	)
	return err
}

// GetGroup retrieves a group by ID.
func (s *IdentityStore) GetGroup(ctx context.Context, groupID int64) (*models.Group, error) {
	return scanGroup(s.pool.QueryRow(ctx, "SELECT "+groupColumns+" FROM groups WHERE group_id = $1", groupID))
}

// SetGroupLanguage modifies a group's preferred language.
func (s *IdentityStore) SetGroupLanguage(ctx context.Context, groupID int64, lang string) error {
	_, err := s.pool.Exec(ctx, "UPDATE groups SET language = $1, updated_at = NOW() WHERE group_id = $2", lang, groupID)
	return err
}

// ListActiveGroups returns all groups registered in database.
func (s *IdentityStore) ListActiveGroups(ctx context.Context) ([]models.Group, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+groupColumns+" FROM groups ORDER BY group_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *g)
	}
	return list, rows.Err()
}

// UpdateGroupSettings updates group protection toggles.
func (s *IdentityStore) UpdateGroupSettings(ctx context.Context, groupID int64, autoBan, autoDelete, scanOnJoin, dailyScan, warnOnDetected bool) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE groups SET
		   auto_ban_enabled = $1,
		   auto_delete_enabled = $2,
		   scan_on_join_enabled = $3,
		   daily_scan_enabled = $4,
		   warn_on_detected = $5,
		   updated_at = NOW()
		 WHERE group_id = $6`,
		autoBan, autoDelete, scanOnJoin, dailyScan, warnOnDetected, groupID,
	)
	return err
}

// UpdateGroupLastScanned records the last time a group was audited.
func (s *IdentityStore) UpdateGroupLastScanned(ctx context.Context, groupID int64) error {
	_, err := s.pool.Exec(ctx, "UPDATE groups SET last_scanned_at = NOW(), updated_at = NOW() WHERE group_id = $1", groupID)
	return err
}

// GetAllGroupIDs returns all registered group IDs.
func (s *IdentityStore) GetAllGroupIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.pool.Query(ctx, "SELECT group_id FROM groups")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// RemoveGroup deletes a group record.
func (s *IdentityStore) RemoveGroup(ctx context.Context, groupID int64) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM groups WHERE group_id = $1", groupID)
	return err
}

// --- REPORTS & PROOFS MANAGEMENT ---

// CreateReportWithProofs inserts a report and associated proofs atomically.
func (s *IdentityStore) CreateReportWithProofs(ctx context.Context, r *models.Report, proofs []models.ReportProof) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ensure reporter exists
	_, _ = tx.Exec(ctx,
		"INSERT INTO users (user_id, first_name) VALUES ($1, 'Telegram User') ON CONFLICT (user_id) DO NOTHING",
		r.ReporterID,
	)

	var reportID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO reports (reporter_id, scammer_id, target_user_id, target_username, target_phone, category, description, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING') RETURNING id`,
		r.ReporterID, r.ScammerID, r.TargetUserID, r.TargetUsername, r.TargetPhone, r.Category, r.Description,
	).Scan(&reportID)
	if err != nil {
		return 0, err
	}

	for _, p := range proofs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO report_proofs (report_id, file_id, file_unique_id, file_type, caption)
			 VALUES ($1, $2, $3, $4, $5)`,
			reportID, p.FileID, p.FileUniqueID, p.FileType, p.Caption,
		); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return reportID, nil
}

// GetReport fetches a report by ID and all attached proofs.
func (s *IdentityStore) GetReport(ctx context.Context, reportID int64) (*models.Report, []models.ReportProof, error) {
	rep, err := scanReport(s.pool.QueryRow(ctx, "SELECT "+reportColumns+" FROM reports WHERE id = $1", reportID))
	if err != nil {
		return nil, nil, err
	}

	rows, err := s.pool.Query(ctx, "SELECT "+proofColumns+" FROM report_proofs WHERE report_id = $1 ORDER BY id", reportID)
	if err != nil {
		return rep, nil, nil
	}
	defer rows.Close()

	var proofs []models.ReportProof
	for rows.Next() {
		p, err := scanProof(rows)
		if err != nil {
			return rep, proofs, nil
		}
		proofs = append(proofs, p)
	}

	return rep, proofs, nil
}

// GetPendingReports returns paginated pending reports along with total pending count.
func (s *IdentityStore) GetPendingReports(ctx context.Context, limit, offset int) ([]models.Report, int64, error) {
	var total int64
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM reports WHERE status = 'PENDING'").Scan(&total)

	rows, err := s.pool.Query(ctx,
		"SELECT "+reportColumns+" FROM reports WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT $1 OFFSET $2",
		limit, offset,
	)
	if err != nil {
		return nil, total, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		r, err := scanReport(rows)
		if err != nil {
			return nil, total, err
		}
		reports = append(reports, *r)
	}
	return reports, total, rows.Err()
}

// UpdateReportStatus updates status and reviewer details for a report.
func (s *IdentityStore) UpdateReportStatus(ctx context.Context, reportID int64, status string, reviewerID *int64, reviewerNotes string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE reports SET
		   status = $1,
		   reviewer_id = $2,
		   reviewer_notes = $3,
		   reviewed_at = NOW()
		 WHERE id = $4`,
		status, reviewerID, reviewerNotes, reportID,
	)
	return err
}

// AddReportProof attaches a proof to an existing report.
func (s *IdentityStore) AddReportProof(ctx context.Context, p models.ReportProof) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO report_proofs (report_id, file_id, file_unique_id, file_type, caption)
		 VALUES ($1, $2, $3, $4, $5)`,
		p.ReportID, p.FileID, p.FileUniqueID, p.FileType, p.Caption,
	)
	return err
}

// --- DIRECT BLACKLIST & BULK EXPORT/IMPORT ---

// GetAllVerifiedScammers returns all verified fraudsters and their identifiers.
func (s *IdentityStore) GetAllVerifiedScammers(ctx context.Context) ([]models.ScammerWithIdentifiers, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+scammerColumns+" FROM scammers WHERE status = 'VERIFIED' ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ScammerWithIdentifiers
	for rows.Next() {
		scam, err := scanScammer(rows)
		if err != nil {
			return nil, err
		}
		idents, err := s.ListIdentifiers(ctx, scam.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, models.ScammerWithIdentifiers{
			Scammer:     *scam,
			Identifiers: idents,
		})
	}
	return result, rows.Err()
}

// ClearScammer marks matching scammer entities as CLEARED.
func (s *IdentityStore) ClearScammer(ctx context.Context, scammerID int64) error {
	_, err := s.pool.Exec(ctx, "UPDATE scammers SET status = 'CLEARED', updated_at = NOW() WHERE id = $1", scammerID)
	return err
}

// --- MODERATION LOGS ---

// LogModeration records a moderation event.
func (s *IdentityStore) LogModeration(ctx context.Context, log models.ModerationLog) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO moderation_logs (group_id, scammer_id, target_user_id, action, reason, executed_by)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		log.GroupID, log.ScammerID, log.TargetUserID, log.Action, log.Reason, log.ExecutedBy,
	)
	return err
}

// GetModerationLogs returns recent moderation logs for a group.
func (s *IdentityStore) GetModerationLogs(ctx context.Context, groupID int64, limit int) ([]models.ModerationLog, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT "+modLogColumns+" FROM moderation_logs WHERE group_id = $1 ORDER BY created_at DESC LIMIT $2",
		groupID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.ModerationLog
	for rows.Next() {
		var l models.ModerationLog
		if err := rows.Scan(&l.ID, &l.GroupID, &l.ScammerID, &l.TargetUserID, &l.Action, &l.Reason, &l.ExecutedBy, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// --- SYSTEM SETTINGS ---

// GetSetting reads a setting from system_settings.
func (s *IdentityStore) GetSetting(ctx context.Context, key, defaultValue string) (string, error) {
	var val string
	err := s.pool.QueryRow(ctx, "SELECT value FROM system_settings WHERE key = $1", key).Scan(&val)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultValue, nil
		}
		return defaultValue, err
	}
	return val, nil
}

// SetSetting writes a setting to system_settings.
func (s *IdentityStore) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO system_settings (key, value, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		key, value,
	)
	return err
}

// --- BROADCAST LOGS ---

// LogBroadcast records an admin broadcast.
func (s *IdentityStore) LogBroadcast(ctx context.Context, log models.BroadcastLog) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO broadcast_logs (initiated_by, message_text, recipient_count, success_count, failed_count)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		log.InitiatedBy, log.MessageText, log.RecipientCount, log.SuccessCount, log.FailedCount,
	).Scan(&id)
	return id, err
}

// --- GLOBAL STATS & USER REPORTS ---

type GlobalStats struct {
	VerifiedScammers int64
	TotalReports     int64
	PendingReports   int64
	ProtectedGroups  int64
	TotalUsers       int64
}

// GetGlobalStats returns aggregate counts for system metrics.
func (s *IdentityStore) GetGlobalStats(ctx context.Context) (GlobalStats, error) {
	var stats GlobalStats
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM scammers WHERE status = 'VERIFIED'").Scan(&stats.VerifiedScammers)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'PENDING') FROM reports").Scan(&stats.TotalReports, &stats.PendingReports)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM groups").Scan(&stats.ProtectedGroups)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	return stats, nil
}

type UserReportSummary struct {
	ID        int64
	Target    string
	Category  string
	Status    string
	CreatedAt time.Time
}

// GetUserReports returns recent reports submitted by a specific user.
func (s *IdentityStore) GetUserReports(ctx context.Context, reporterID int64) ([]UserReportSummary, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, COALESCE(NULLIF(target_username, ''), NULLIF(target_user_id::text, '0'), NULLIF(target_phone, ''), 'Unknown Target'), category, status, created_at
		 FROM reports WHERE reporter_id = $1 ORDER BY created_at DESC LIMIT 10`,
		reporterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserReportSummary
	for rows.Next() {
		var r UserReportSummary
		if err := rows.Scan(&r.ID, &r.Target, &r.Category, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// CreateUserReport records a new user report for admin review.
func (s *IdentityStore) CreateUserReport(ctx context.Context, reporterID int64, targetUser *int64, targetUsername, targetPhone, category, description string) (int64, error) {
	_, _ = s.pool.Exec(ctx,
		"INSERT INTO users (user_id, first_name) VALUES ($1, 'Telegram User') ON CONFLICT (user_id) DO NOTHING",
		reporterID,
	)

	var reportID int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO reports (reporter_id, target_user_id, target_username, target_phone, category, description, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'PENDING') RETURNING id`,
		reporterID, targetUser, targetUsername, targetPhone, category, description,
	).Scan(&reportID)
	return reportID, err
}
