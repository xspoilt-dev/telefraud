package i18n

import (
	"testing"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en", LangEN},
		{"EN", LangEN},
		{"bn", LangBN},
		{"bangla", LangBN},
		{"bengali", LangBN},
		{"hi", LangHI},
		{"hindi", LangHI},
		{"fr", LangEN},
		{"", LangEN},
	}

	for _, tt := range tests {
		got := NormalizeLanguage(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeLanguage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetAndFormat(t *testing.T) {
	// English
	enBtn := Get(LangEN, "btn_report")
	if enBtn != "🛡️ Report Fraudster" {
		t.Errorf("expected English button, got %q", enBtn)
	}

	// Bengali
	bnBtn := Get(LangBN, "btn_report")
	if bnBtn != "🛡️ প্রতারক রিপোর্ট করুন" {
		t.Errorf("expected Bengali button, got %q", bnBtn)
	}

	// Hindi
	hiBtn := Get(LangHI, "btn_report")
	if hiBtn != "🛡️ धोखेबाज़ रिपोर्ट करें" {
		t.Errorf("expected Hindi button, got %q", hiBtn)
	}

	// Fallback for missing key
	fallback := Get("invalid_lang", "btn_report")
	if fallback != "🛡️ Report Fraudster" {
		t.Errorf("expected English fallback, got %q", fallback)
	}

	// Format
	formatted := Format(LangEN, "lang_updated_user", "English")
	if formatted == "" {
		t.Errorf("expected formatted string")
	}
}
