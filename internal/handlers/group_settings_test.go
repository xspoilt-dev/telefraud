package handlers

import (
	"strings"
	"testing"
	"time"

	"telefraud/internal/models"
)

func TestToggleIcon(t *testing.T) {
	if got := toggleIcon(true); got != "🟢 ON" {
		t.Errorf("toggleIcon(true) = %q, want '🟢 ON'", got)
	}
	if got := toggleIcon(false); got != "🔴 OFF" {
		t.Errorf("toggleIcon(false) = %q, want '🔴 OFF'", got)
	}
}

func TestFormatGroupSettingsText(t *testing.T) {
	now := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	group := &models.Group{
		GroupID:        -1001234567890,
		Title:          "Crypto Discussion & Trading",
		LastScannedAt:  &now,
		AutoBanEnabled: true,
	}

	text := formatGroupSettingsText(group)
	if !strings.Contains(text, "Crypto Discussion &amp; Trading") && !strings.Contains(text, "Crypto Discussion & Trading") {
		t.Errorf("expected title in text, got %q", text)
	}
	if !strings.Contains(text, "-1001234567890") {
		t.Errorf("expected group ID in text, got %q", text)
	}
	if !strings.Contains(text, "Sep 13, 2026 10:00 UTC") {
		t.Errorf("expected last scanned time in text, got %q", text)
	}

	groupNever := &models.Group{
		GroupID: -100999,
		Title:   "New Group",
	}
	textNever := formatGroupSettingsText(groupNever)
	if !strings.Contains(textNever, "Never") {
		t.Errorf("expected 'Never' in text, got %q", textNever)
	}
}

func TestBuildGroupSettingsMarkup(t *testing.T) {
	group := &models.Group{
		GroupID:           -1001234567890,
		Language:          "en",
		AutoBanEnabled:    true,
		AutoDeleteEnabled: false,
		ScanOnJoinEnabled: true,
		DailyScanEnabled:  false,
		WarnOnDetected:    true,
	}

	markup := buildGroupSettingsMarkup(group)
	if markup == nil || len(markup.InlineKeyboard) != 7 {
		t.Fatalf("expected 7 rows in inline keyboard, got %v", markup)
	}

	// Verify row buttons and callback data
	row0 := markup.InlineKeyboard[0][0]
	if !strings.Contains(row0.Text, "🟢 ON") {
		t.Errorf("expected Auto-Ban button to be ON, got %q", row0.Text)
	}
	if row0.CallbackData != "grp_tgl:autoban:-1001234567890" {
		t.Errorf("unexpected callback data: %s", row0.CallbackData)
	}

	row1 := markup.InlineKeyboard[1][0]
	if !strings.Contains(row1.Text, "🔴 OFF") {
		t.Errorf("expected Auto-Delete button to be OFF, got %q", row1.Text)
	}

	// Verify language row
	row5 := markup.InlineKeyboard[5][0]
	if !strings.Contains(row5.Text, "Language: 🇺🇸 English") {
		t.Errorf("expected Language row with English, got %q", row5.Text)
	}
}
