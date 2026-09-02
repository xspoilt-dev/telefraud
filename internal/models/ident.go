package models

import (
	"regexp"
	"strconv"
	"strings"
)

// Telegram usernames: 5-32 chars of [a-z0-9_], case-insensitive.
var usernameRe = regexp.MustCompile(`^[a-z0-9_]{5,32}$`)

// NormalizeUsername canonicalizes a raw Telegram username: strips a leading
// '@', lowercases, and validates the Telegram username format. It returns the
// canonical form (no '@') and whether the input was valid.
func NormalizeUsername(raw string) (string, bool) {
	u := strings.TrimSpace(raw)
	u = strings.TrimPrefix(u, "@")
	u = strings.ToLower(u)
	if !usernameRe.MatchString(u) {
		return "", false
	}
	return u, true
}

// NormalizePhone canonicalizes a raw phone number to E.164-ish form ("+<digits>").
// It strips every non-digit character and validates a plausible length.
func NormalizePhone(raw string) (string, bool) {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	if len(d) < 8 || len(d) > 15 {
		return "", false
	}
	return "+" + d, true
}

// NormalizeUserID validates a raw user id string as a positive int64.
func NormalizeUserID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// UserIDValue formats an int64 user id as its canonical identifier value.
func UserIDValue(id int64) string {
	return strconv.FormatInt(id, 10)
}
