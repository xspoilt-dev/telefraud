package exporter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"telefraud/internal/models"
)

// ExportCSV formats verified scammer entities and identifiers as CSV bytes.
func ExportCSV(records []models.ScammerWithIdentifiers) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header
	header := []string{"scammer_id", "status", "threat_level", "category", "reason", "report_count", "user_ids", "usernames", "phones", "created_at"}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, r := range records {
		var userIDs, usernames, phones []string
		for _, id := range r.Identifiers {
			switch id.Kind {
			case models.KindUserID:
				userIDs = append(userIDs, id.Value)
			case models.KindUsername:
				usernames = append(usernames, "@"+id.Value)
			case models.KindPhone:
				phones = append(phones, id.Value)
			}
		}

		row := []string{
			strconv.FormatInt(r.Scammer.ID, 10),
			r.Scammer.Status,
			r.Scammer.ThreatLevel,
			r.Scammer.Category,
			r.Scammer.Reason,
			strconv.Itoa(r.Scammer.ReportCount),
			strings.Join(userIDs, ";"),
			strings.Join(usernames, ";"),
			strings.Join(phones, ";"),
			r.Scammer.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportJSON formats verified scammers as pretty-printed JSON bytes.
func ExportJSON(records []models.ScammerWithIdentifiers) ([]byte, error) {
	return json.MarshalIndent(records, "", "  ")
}

// ImportCSV parses CSV content into ScammerImportItem slice.
func ImportCSV(r io.Reader) ([]models.ScammerImportItem, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv is empty or missing data rows")
	}

	header := records[0]
	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[strings.ToLower(strings.TrimSpace(col))] = i
	}

	var items []models.ScammerImportItem
	for lineNum, row := range records[1:] {
		var item models.ScammerImportItem

		getCol := func(names ...string) string {
			for _, name := range names {
				if idx, ok := colIdx[name]; ok && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
			}
			return ""
		}

		if uidStr := getCol("user_id", "user_ids", "id"); uidStr != "" {
			// handle semicolon separated
			parts := strings.Split(uidStr, ";")
			if id, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64); err == nil {
				item.UserID = id
			}
		}

		if uname := getCol("username", "usernames", "handle"); uname != "" {
			parts := strings.Split(uname, ";")
			item.Username = strings.TrimPrefix(strings.TrimSpace(parts[0]), "@")
		}

		if phone := getCol("phone", "phones", "phone_number"); phone != "" {
			parts := strings.Split(phone, ";")
			item.Phone = strings.TrimSpace(parts[0])
		}

		item.ThreatLevel = getCol("threat_level", "threat")
		if item.ThreatLevel == "" {
			item.ThreatLevel = models.ThreatMedium
		}

		item.Category = getCol("category")
		if item.Category == "" {
			item.Category = models.CategoryOther
		}

		item.Reason = getCol("reason", "description")
		if item.Reason == "" {
			item.Reason = "Imported from blacklist file"
		}

		item.Status = getCol("status")
		if item.Status == "" {
			item.Status = models.StatusVerified
		}

		if item.UserID == 0 && item.Username == "" && item.Phone == "" {
			continue // skip empty rows
		}

		_ = lineNum
		items = append(items, item)
	}

	return items, nil
}

// ImportJSON parses JSON content into ScammerImportItem slice.
func ImportJSON(r io.Reader) ([]models.ScammerImportItem, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read json: %w", err)
	}

	// Try parsing as array of ScammerWithIdentifiers
	var complexRecords []models.ScammerWithIdentifiers
	if err := json.Unmarshal(data, &complexRecords); err == nil && len(complexRecords) > 0 {
		var items []models.ScammerImportItem
		for _, cr := range complexRecords {
			item := models.ScammerImportItem{
				ThreatLevel: cr.Scammer.ThreatLevel,
				Category:    cr.Scammer.Category,
				Reason:      cr.Scammer.Reason,
				Status:      cr.Scammer.Status,
			}
			for _, id := range cr.Identifiers {
				switch id.Kind {
				case models.KindUserID:
					if uid, err := strconv.ParseInt(id.Value, 10, 64); err == nil {
						item.UserID = uid
					}
				case models.KindUsername:
					item.Username = id.Value
				case models.KindPhone:
					item.Phone = id.Value
				}
			}
			if item.UserID != 0 || item.Username != "" || item.Phone != "" {
				items = append(items, item)
			}
		}
		return items, nil
	}

	// Fallback to simple array of ScammerImportItem
	var simpleItems []models.ScammerImportItem
	if err := json.Unmarshal(data, &simpleItems); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}
	return simpleItems, nil
}
