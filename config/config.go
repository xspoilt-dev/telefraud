package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration settings.
type Config struct {
	BotToken            string
	DatabaseURL         string
	AdminIDs            []int64
	AdminLogChatID      int64
	ParseMode           string
	BackupDir           string
	BackupCronExpr      string
	RetentionDays       int
	DeveloperName       string
	DeveloperUsername   string
	DailyGroupScanCron  string
	EnableNewMemberScan bool
	Debug               bool
	RequiredChannelID   int64
	RequiredChannelLink string
}

// LoadConfig reads configuration settings from environment variables.
//
// It first loads a `.env` file from the working directory if present (real
// environment variables take precedence — godotenv never overrides an already
// set variable). Missing `.env` is not an error.
func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		token = "YOUR_TELEGRAM_BOT_TOKEN"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://telefraud:secret@localhost:5432/telefraud_db?sslmode=disable"
	}

	adminIDsStr := os.Getenv("ADMIN_IDS")
	var adminIDs []int64
	if adminIDsStr != "" {
		for _, part := range strings.Split(adminIDsStr, ",") {
			part = strings.TrimSpace(part)
			if id, err := strconv.ParseInt(part, 10, 64); err == nil {
				adminIDs = append(adminIDs, id)
			}
		}
	}

	adminLogChatID, _ := strconv.ParseInt(os.Getenv("ADMIN_LOG_CHAT_ID"), 10, 64)

	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "./backups"
	}

	retentionDays := 30
	if rStr := os.Getenv("BACKUP_RETENTION_DAYS"); rStr != "" {
		if r, err := strconv.Atoi(rStr); err == nil && r > 0 {
			retentionDays = r
		}
	}

	devName := os.Getenv("DEVELOPER_NAME")
	if devName == "" {
		devName = "xspoilt"
	}

	devUsername := os.Getenv("DEVELOPER_USERNAME")
	if devUsername == "" {
		devUsername = "@xspoilt"
	}

	reqChannelID, _ := strconv.ParseInt(os.Getenv("REQUIRED_CHANNEL_ID"), 10, 64)
	if reqChannelID == 0 {
		reqChannelID = -1004458146162
	}

	reqChannelLink := os.Getenv("REQUIRED_CHANNEL_LINK")
	if reqChannelLink == "" {
		reqChannelLink = "https://t.me/telefraud_info"
	}

	return &Config{
		BotToken:            token,
		DatabaseURL:         dbURL,
		AdminIDs:            adminIDs,
		AdminLogChatID:      adminLogChatID,
		ParseMode:           "HTML",
		BackupDir:           backupDir,
		BackupCronExpr:      "0 0 * * *", // Daily at midnight
		RetentionDays:       retentionDays,
		DeveloperName:       devName,
		DeveloperUsername:   devUsername,
		DailyGroupScanCron:  "0 3 * * *", // Daily group scan at 03:00 AM
		EnableNewMemberScan: true,
		Debug:               debug,
		RequiredChannelID:   reqChannelID,
		RequiredChannelLink: reqChannelLink,
	}, nil
}

// IsAdmin checks whether a Telegram user ID is listed as an admin.
func (c *Config) IsAdmin(userID int64) bool {
	for _, adminID := range c.AdminIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}

// String returns formatted debug configuration details.
func (c *Config) String() string {
	return fmt.Sprintf("Config[ParseMode=%s, Admins=%d, Dev=%s]", c.ParseMode, len(c.AdminIDs), c.DeveloperName)
}
