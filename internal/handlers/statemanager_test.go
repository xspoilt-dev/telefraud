package handlers

import (
	"sync"
	"testing"
)

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()

	userID := int64(999888777)

	// Default empty session
	sess := sm.Get(userID)
	if sess.State != StateNone {
		t.Errorf("Expected StateNone, got %v", sess.State)
	}

	// Update state
	sm.SetState(userID, StateReportWaitTarget)
	sess = sm.Get(userID)
	if sess.State != StateReportWaitTarget {
		t.Errorf("Expected StateReportWaitTarget, got %v", sess.State)
	}

	// Update session fields
	sm.Update(userID, func(s *UserSession) {
		s.ReportTargetRaw = "@scammer_user"
		s.ReportCategory = "Financial Fraud"
	})

	sess = sm.Get(userID)
	if sess.ReportTargetRaw != "@scammer_user" || sess.ReportCategory != "Financial Fraud" {
		t.Errorf("Session update failed: %+v", sess)
	}

	// Clear session
	sm.Clear(userID)
	sess = sm.Get(userID)
	if sess.State != StateNone {
		t.Errorf("Expected session to be cleared, got %+v", sess)
	}
}

func TestSessionManagerConcurrent(t *testing.T) {
	sm := NewSessionManager()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			sm.SetState(uid, StateReportWaitTarget)
			sm.Update(uid, func(s *UserSession) {
				s.ReportTargetRaw = "target"
			})
			_ = sm.Get(uid)
			sm.Clear(uid)
		}(int64(i))
	}

	wg.Wait()
}
