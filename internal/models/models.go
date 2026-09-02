// Package models defines the persistent domain structures for @telefraudbot.
//
// The central design constraint is that a Telegram scammer is not a single
// identifier: the numeric User ID is immutable, but the @username and the phone
// number are freely changeable. A scammer is therefore modelled as a canonical
// entity (Scammer) that owns a set of linked identifiers (ScammerIdentifier).
// Reports, lookups and moderation always resolve through identifiers, so a
// username change never severs the trail to the account.
package models

import "time"

// IdentifierKind discriminates the ways a scammer account can be pinned.
type IdentifierKind string

const (
	// KindUserID is the immutable numeric Telegram user id (strongest identity).
	KindUserID IdentifierKind = "user_id"
	// KindUsername is the @username, case-insensitive and changeable.
	KindUsername IdentifierKind = "username"
	// KindPhone is the E.164-style phone number, changeable.
	KindPhone IdentifierKind = "phone"
)

// Valid reports whether k is a known identifier kind.
func (k IdentifierKind) Valid() bool {
	switch k {
	case KindUserID, KindUsername, KindPhone:
		return true
	default:
		return false
	}
}

// Identifier sources.
const (
	SourceReport   = "REPORT"
	SourceJoinScan = "JOIN_SCAN"
	SourceAdmin    = "ADMIN"
	SourceImport   = "IMPORT"
)

// Scammer lifecycle status.
const (
	StatusPending    = "PENDING"
	StatusVerified   = "VERIFIED"
	StatusSuspicious = "SUSPICIOUS"
	StatusCleared    = "CLEARED"
)

// Threat levels, in ascending severity.
const (
	ThreatLow      = "LOW"
	ThreatMedium   = "MEDIUM"
	ThreatHigh     = "HIGH"
	ThreatCritical = "CRITICAL"
)

// Scammer is the canonical fraudster entity. All identifiers hang off it, so
// the record survives username and phone changes.
type Scammer struct {
	ID          int64     `json:"id" db:"id"`
	Status      string    `json:"status" db:"status"`
	ThreatLevel string    `json:"threat_level" db:"threat_level"`
	ReportCount int       `json:"report_count" db:"report_count"`
	Category    string    `json:"category" db:"category"`
	Reason      string    `json:"reason" db:"reason"`
	AddedBy     *int64    `json:"added_by,omitempty" db:"added_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ScammerIdentifier links one handle/number to a Scammer. (kind, value) is
// globally unique: an identifier can belong to exactly one canonical scammer.
type ScammerIdentifier struct {
	ID          int64          `json:"id" db:"id"`
	ScammerID   int64          `json:"scammer_id" db:"scammer_id"`
	Kind        IdentifierKind `json:"kind" db:"kind"`
	Value       string         `json:"value" db:"value"`
	IsPrimary   bool           `json:"is_primary" db:"is_primary"`
	Source      string         `json:"source" db:"source"`
	FirstSeenAt time.Time      `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt  time.Time      `json:"last_seen_at" db:"last_seen_at"`
}

// User is a registered bot user or system admin.
type User struct {
	UserID    int64     `json:"user_id" db:"user_id"`
	Username  string    `json:"username,omitempty" db:"username"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name,omitempty" db:"last_name"`
	Role      string    `json:"role" db:"role"`
	IsBanned  bool      `json:"is_banned" db:"is_banned"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Report is a user submission awaiting admin review. Raw target fields are
// preserved as submitted; ScammerID links to the resolved canonical entity.
type Report struct {
	ID             int64      `json:"id" db:"id"`
	ReporterID     int64      `json:"reporter_id" db:"reporter_id"`
	ScammerID      *int64     `json:"scammer_id,omitempty" db:"scammer_id"`
	TargetUserID   *int64     `json:"target_user_id,omitempty" db:"target_user_id"`
	TargetUsername string     `json:"target_username,omitempty" db:"target_username"`
	TargetPhone    string     `json:"target_phone,omitempty" db:"target_phone"`
	Category       string     `json:"category" db:"category"`
	Description    string     `json:"description" db:"description"`
	Status         string     `json:"status" db:"status"`
	ReviewerID     *int64     `json:"reviewer_id,omitempty" db:"reviewer_id"`
	ReviewerNotes  string     `json:"reviewer_notes,omitempty" db:"reviewer_notes"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
}

// ReportProof is a media attachment for a report.
type ReportProof struct {
	ID           int64     `json:"id" db:"id"`
	ReportID     int64     `json:"report_id" db:"report_id"`
	FileID       string    `json:"file_id" db:"file_id"`
	FileUniqueID string    `json:"file_unique_id,omitempty" db:"file_unique_id"`
	FileType     string    `json:"file_type" db:"file_type"`
	Caption      string    `json:"caption,omitempty" db:"caption"`
	UploadedAt   time.Time `json:"uploaded_at" db:"uploaded_at"`
}

// Group is a Telegram group protected by the bot.
type Group struct {
	GroupID           int64      `json:"group_id" db:"group_id"`
	Title             string     `json:"title" db:"title"`
	Username          string     `json:"username,omitempty" db:"username"`
	AutoBanEnabled    bool       `json:"auto_ban_enabled" db:"auto_ban_enabled"`
	AutoDeleteEnabled bool       `json:"auto_delete_enabled" db:"auto_delete_enabled"`
	ScanOnJoinEnabled bool       `json:"scan_on_join_enabled" db:"scan_on_join_enabled"`
	DailyScanEnabled  bool       `json:"daily_scan_enabled" db:"daily_scan_enabled"`
	WarnOnDetected    bool       `json:"warn_on_detected" db:"warn_on_detected"`
	LastScannedAt     *time.Time `json:"last_scanned_at,omitempty" db:"last_scanned_at"`
	JoinedAt          time.Time  `json:"joined_at" db:"joined_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// ModerationLog is an audit trail entry for group enforcement.
type ModerationLog struct {
	ID           int64     `json:"id" db:"id"`
	GroupID      *int64    `json:"group_id,omitempty" db:"group_id"`
	ScammerID    *int64    `json:"scammer_id,omitempty" db:"scammer_id"`
	TargetUserID int64     `json:"target_user_id" db:"target_user_id"`
	Action       string    `json:"action" db:"action"`
	Reason       string    `json:"reason" db:"reason"`
	ExecutedBy   int64     `json:"executed_by" db:"executed_by"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// SystemSetting is a key/value configuration parameter.
type SystemSetting struct {
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// BroadcastLog records an admin announcement.
type BroadcastLog struct {
	ID             int64     `json:"id" db:"id"`
	InitiatedBy    int64     `json:"initiated_by" db:"initiated_by"`
	MessageText    string    `json:"message_text" db:"message_text"`
	RecipientCount int       `json:"recipient_count" db:"recipient_count"`
	SuccessCount   int       `json:"success_count" db:"success_count"`
	FailedCount    int       `json:"failed_count" db:"failed_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
