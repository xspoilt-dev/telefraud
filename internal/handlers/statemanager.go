package handlers

import (
	"sync"
	"time"

	"telefraud/internal/models"
	"telefraud/internal/services/identity"
)

// WizardState defines the active stage of an interactive user interaction.
type WizardState string

const (
	StateNone WizardState = ""

	// Check wizard
	StateCheckWaitInput WizardState = "check_wait_input"

	// Report wizard
	StateReportWaitTarget      WizardState = "report_wait_target"
	StateReportWaitCategory    WizardState = "report_wait_category"
	StateReportWaitDescription WizardState = "report_wait_description"
	StateReportWaitProof       WizardState = "report_wait_proof"

	// Admin wizards
	StateBroadcastWaitText WizardState = "broadcast_wait_text"
	StateBlacklistAddWait  WizardState = "blacklist_add_wait"
)

// UserSession stores in-progress wizard drafts and state.
type UserSession struct {
	State     WizardState
	UpdatedAt time.Time

	// Report Draft
	ReportTargetRaw   string
	ReportCandidates  identity.Candidates
	ReportTargetUser  *int64
	ReportUsername    string
	ReportPhone       string
	ReportCategory    string
	ReportDescription string
	ReportProofs      []models.ReportProof

	// Broadcast Draft
	BroadcastText    string
	BroadcastPhotoID string
}

// SessionManager manages active user sessions in memory with concurrent access safety.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
}

// NewSessionManager creates an empty SessionManager.
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[int64]*UserSession),
	}
	go sm.cleanupWorker()
	return sm
}

// Get returns the session for a user.
func (sm *SessionManager) Get(userID int64) *UserSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sess, exists := sm.sessions[userID]
	if !exists {
		return &UserSession{State: StateNone, UpdatedAt: time.Now()}
	}
	return sess
}

// SetState updates only the state of the user.
func (sm *SessionManager) SetState(userID int64, state WizardState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sess, exists := sm.sessions[userID]
	if !exists {
		sess = &UserSession{}
		sm.sessions[userID] = sess
	}
	sess.State = state
	sess.UpdatedAt = time.Now()
}

// Update mutates the user session under lock.
func (sm *SessionManager) Update(userID int64, fn func(s *UserSession)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sess, exists := sm.sessions[userID]
	if !exists {
		sess = &UserSession{}
		sm.sessions[userID] = sess
	}
	fn(sess)
	sess.UpdatedAt = time.Now()
}

// Clear removes a user session.
func (sm *SessionManager) Clear(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, userID)
}

// Periodic cleanup of idle sessions (TTL: 1 hour).
func (sm *SessionManager) cleanupWorker() {
	ticker := time.NewTicker(15 * time.Minute)
	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for uid, sess := range sm.sessions {
			if now.Sub(sess.UpdatedAt) > 1*time.Hour {
				delete(sm.sessions, uid)
			}
		}
		sm.mu.Unlock()
	}
}
