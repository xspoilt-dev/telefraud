package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/config"
)

// BackupResult holds information on a completed database dump.
type BackupResult struct {
	FilePath  string
	FileName  string
	SizeBytes int64
	Duration  time.Duration
}

// BackupManager handles automated database snapshots and retention rotation.
type BackupManager struct {
	cfg *config.Config
	bot *telego.Bot
}

// NewBackupManager creates a BackupManager.
func NewBackupManager(cfg *config.Config, bot *telego.Bot) *BackupManager {
	return &BackupManager{
		cfg: cfg,
		bot: bot,
	}
}

// RunBackup creates a fresh compressed PostgreSQL backup and prunes old archives.
func (bm *BackupManager) RunBackup(ctx context.Context) (*BackupResult, error) {
	start := time.Now()

	backupDir := bm.cfg.BackupDir
	if backupDir == "" {
		backupDir = "./backups"
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("telefraud_db_%s.sql.gz", timestamp)
	fullPath := filepath.Join(backupDir, filename)

	// Check if backup.sh exists
	scriptPath := "./scripts/backup.sh"
	if _, err := os.Stat(scriptPath); err == nil {
		cmd := exec.CommandContext(ctx, "bash", scriptPath)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("DATABASE_URL=%s", bm.cfg.DatabaseURL),
			fmt.Sprintf("BACKUP_DIR=%s", backupDir),
			fmt.Sprintf("BACKUP_RETENTION_DAYS=%d", bm.cfg.RetentionDays),
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("[Backup] script output: %s", string(out))
			return nil, fmt.Errorf("backup script failed: %w (output: %s)", err, string(out))
		}
	} else {
		// Fallback: run pg_dump pipeline directly
		cmd := exec.CommandContext(ctx, "bash", "-c", fmt.Sprintf("pg_dump '%s' --clean --if-exists --no-owner | gzip -9 > '%s'", bm.cfg.DatabaseURL, fullPath))
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("pg_dump failed: %w (%s)", err, string(out))
		}
	}

	// Find newest backup file in directory
	files, err := filepath.Glob(filepath.Join(backupDir, "telefraud_db_*.sql.gz"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("backup archive not found after dump")
	}
	latestFile := files[len(files)-1]

	stat, err := os.Stat(latestFile)
	if err != nil {
		return nil, fmt.Errorf("stat backup file: %w", err)
	}

	// Prune archives older than RetentionDays
	retentionCutoff := time.Now().AddDate(0, 0, -bm.cfg.RetentionDays)
	for _, f := range files {
		fStat, sErr := os.Stat(f)
		if sErr == nil && fStat.ModTime().Before(retentionCutoff) {
			_ = os.Remove(f)
		}
	}

	return &BackupResult{
		FilePath:  latestFile,
		FileName:  filepath.Base(latestFile),
		SizeBytes: stat.Size(),
		Duration:  time.Since(start),
	}, nil
}

// SendBackupFile sends the backup document to a given chat ID.
func (bm *BackupManager) SendBackupFile(ctx context.Context, chatID int64, res *BackupResult) error {
	_ = ctx
	file, err := os.Open(res.FilePath)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer file.Close()

	caption := fmt.Sprintf(
		"<b>💾 PostgreSQL Database Backup</b>\n\n"+
			"<b>File:</b> <code>%s</code>\n"+
			"<b>Size:</b> <code>%.2f MB</code>\n"+
			"<b>Timestamp:</b> %s\n"+
			"<b>Retention:</b> %d days",
		res.FileName,
		float64(res.SizeBytes)/(1024*1024),
		time.Now().UTC().Format("Jan 02, 2006 15:04:05 UTC"),
		bm.cfg.RetentionDays,
	)

	docMsg := tu.Document(
		tu.ID(chatID),
		tu.File(file),
	).WithCaption(caption).WithParseMode(telego.ModeHTML)

	_, err = bm.bot.SendDocument(docMsg)
	return err
}

// StartDailyBackupWorker starts the automated midnight background backup worker.
func (bm *BackupManager) StartDailyBackupWorker(ctx context.Context) {
	log.Println("[Backup] Automated Daily Backup worker started (scheduled daily at 00:00 UTC)")
	go func() {
		for {
			now := time.Now().UTC()
			// Calculate next 00:00:00 UTC
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
			timer := time.NewTimer(time.Until(next))

			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				log.Println("[Backup] Executing scheduled daily database backup...")
				res, err := bm.RunBackup(ctx)
				if err != nil {
					log.Printf("[Backup] Daily backup error: %v", err)
					continue
				}
				log.Printf("[Backup] Daily backup complete: %s (%.2f MB in %v)", res.FileName, float64(res.SizeBytes)/(1024*1024), res.Duration)

				// Deliver to Admin Log Chat if configured
				if bm.cfg.AdminLogChatID != 0 {
					if err := bm.SendBackupFile(ctx, bm.cfg.AdminLogChatID, res); err != nil {
						log.Printf("[Backup] Failed to send backup to Admin Log Chat: %v", err)
					}
				}
			}
		}
	}()
}
