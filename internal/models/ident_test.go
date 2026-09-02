package models

import "testing"

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"@Scammer", "scammer", true},
		{"Scammer", "scammer", true},
		{"SCAMMER", "scammer", true},
		{"  scammer  ", "scammer", true},
		{"@scam", "", false},  // too short
		{"@a b", "", false},   // space
		{"@scam!", "", false}, // invalid char
		{"@scam..mer", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeUsername(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizeUsername(%q) = (%q, %v); want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"+1 (234) 567-8901", "+12345678901", true},
		{"1234567890", "+1234567890", true},
		{"+91 98765 43210", "+919876543210", true},
		{"123", "", false},                   // too short
		{"+12345678901234567890", "", false}, // too long
		{"", "", false},
		{"+1-800-NO-SCAM", "", false}, // letters stripped -> 4 digits, too short
	}
	for _, tt := range tests {
		got, ok := NormalizePhone(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizePhone(%q) = (%q, %v); want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNormalizeUserID(t *testing.T) {
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"123456789", 123456789, true},
		{"  42  ", 42, true},
		{"0", 0, false},
		{"-5", 0, false},
		{"abc", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		got, ok := NormalizeUserID(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizeUserID(%q) = (%d, %v); want (%d, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestUserIDValue(t *testing.T) {
	if got := UserIDValue(123456789); got != "123456789" {
		t.Fatalf("UserIDValue(123456789) = %q; want \"123456789\"", got)
	}
}

func TestIdentifierKindValid(t *testing.T) {
	if !KindUserID.Valid() || !KindUsername.Valid() || !KindPhone.Valid() {
		t.Fatalf("known kinds must be valid")
	}
	if IdentifierKind("email").Valid() {
		t.Fatalf("unknown kind must be invalid")
	}
}
