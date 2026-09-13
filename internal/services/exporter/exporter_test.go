package exporter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"telefraud/internal/models"
)

func TestExportAndImportCSV(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	records := []models.ScammerWithIdentifiers{
		{
			Scammer: models.Scammer{
				ID:          1001,
				Status:      models.StatusVerified,
				ThreatLevel: models.ThreatHigh,
				Category:    models.CategoryFinancial,
				Reason:      "Fake investment scam",
				ReportCount: 3,
				CreatedAt:   now,
			},
			Identifiers: []models.ScammerIdentifier{
				{Kind: models.KindUserID, Value: "123456789"},
				{Kind: models.KindUsername, Value: "fake_scammer"},
				{Kind: models.KindPhone, Value: "+1234567890"},
			},
		},
	}

	csvBytes, err := ExportCSV(records)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	if !strings.Contains(string(csvBytes), "fake_scammer") {
		t.Errorf("Expected CSV to contain 'fake_scammer', got: %s", string(csvBytes))
	}

	items, err := ImportCSV(bytes.NewReader(csvBytes))
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 imported item, got %d", len(items))
	}

	item := items[0]
	if item.UserID != 123456789 {
		t.Errorf("Expected UserID 123456789, got %d", item.UserID)
	}
	if item.Username != "fake_scammer" {
		t.Errorf("Expected Username 'fake_scammer', got '%s'", item.Username)
	}
	if item.Phone != "+1234567890" {
		t.Errorf("Expected Phone '+1234567890', got '%s'", item.Phone)
	}
	if item.ThreatLevel != models.ThreatHigh {
		t.Errorf("Expected ThreatLevel HIGH, got '%s'", item.ThreatLevel)
	}
}

func TestExportAndImportJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	records := []models.ScammerWithIdentifiers{
		{
			Scammer: models.Scammer{
				ID:          2002,
				Status:      models.StatusVerified,
				ThreatLevel: models.ThreatCritical,
				Category:    models.CategoryCrypto,
				Reason:      "Phishing drainer bot",
				ReportCount: 5,
				CreatedAt:   now,
			},
			Identifiers: []models.ScammerIdentifier{
				{Kind: models.KindUserID, Value: "987654321"},
				{Kind: models.KindUsername, Value: "crypto_drainer"},
			},
		},
	}

	jsonBytes, err := ExportJSON(records)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	items, err := ImportJSON(bytes.NewReader(jsonBytes))
	if err != nil {
		t.Fatalf("ImportJSON failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	if items[0].Username != "crypto_drainer" {
		t.Errorf("Expected crypto_drainer, got %s", items[0].Username)
	}
	if items[0].UserID != 987654321 {
		t.Errorf("Expected 987654321, got %d", items[0].UserID)
	}
}
