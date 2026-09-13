package handlers

import (
	"testing"
	"time"
)

func TestParseCandidates(t *testing.T) {
	tests := []struct {
		input      string
		wantOK     bool
		wantUserID int64
		wantUser   string
		wantPhone  string
	}{
		{"@scam_handle", true, 0, "scam_handle", ""},
		{"@SCAM_Handle", true, 0, "scam_handle", ""},
		{"123456789", true, 123456789, "", ""},
		{"+1234567890", true, 0, "", "+1234567890"},
		{"some_username", true, 0, "some_username", ""},
		{"", false, 0, "", ""},
		{"   ", false, 0, "", ""},
	}

	for _, tt := range tests {
		cand, ok := parseCandidates(tt.input)
		if ok != tt.wantOK {
			t.Errorf("parseCandidates(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if cand.UserID != tt.wantUserID {
			t.Errorf("parseCandidates(%q) UserID = %d, want %d", tt.input, cand.UserID, tt.wantUserID)
		}
		if cand.Username != tt.wantUser {
			t.Errorf("parseCandidates(%q) Username = %q, want %q", tt.input, cand.Username, tt.wantUser)
		}
		if cand.Phone != tt.wantPhone {
			t.Errorf("parseCandidates(%q) Phone = %q, want %q", tt.input, cand.Phone, tt.wantPhone)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	d := 25*time.Hour + 30*time.Minute + 15*time.Second
	out := formatUptime(d)
	if out != "1d 1h 30m 15s" {
		t.Errorf("formatUptime(%v) = %q, want '1d 1h 30m 15s'", d, out)
	}

	d2 := 45*time.Minute + 10*time.Second
	out2 := formatUptime(d2)
	if out2 != "45m 10s" {
		t.Errorf("formatUptime(%v) = %q, want '45m 10s'", d2, out2)
	}
}
