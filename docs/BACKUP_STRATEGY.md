# 💾 Daily Backup & Disaster Recovery Strategy

This document outlines the backup policy, automated execution scripts, retention schedule, and recovery procedures for the **@telefraudbot** PostgreSQL database.

---

## 🎯 Backup Objectives & Specifications

- **Backup Frequency**: Daily at 00:00 UTC (configurable).
- **Format**: Compressed SQL plain-text format (`pg_dump` piped through `gzip` -> `.sql.gz`).
- **Naming Convention**: `telefraud_db_YYYYMMDD_HHMMSS.sql.gz`
- **Retention Period**: 30 days (older backups automatically pruned).
- **Delivery Targets**: Local filesystem storage (`./backups`) and optional automated dispatch to Telegram Admin Log Channel.

---

## 🛠️ Backup Methods

### Method 1: Embedded Go Backup Worker (`internal/backup`)
The bot includes a built-in Go worker module that runs continuously inside the main application process.

- Executes `pg_dump` using standard OS command execution.
- Compresses output using standard `gzip`.
- Checks for expired backup files beyond the `BACKUP_RETENTION_DAYS` window and purges them.
- Sends an HTML alert notification to `ADMIN_LOG_CHAT_ID` upon successful backup completion.

---

### Method 2: Standalone Shell Script (`scripts/backup.sh`)
For system administrators who prefer standard systemd or cron scheduling outside the Go process:

```bash
#!/usr/bin/env bash
# scripts/backup.sh - Automated PostgreSQL Backup for telefraudbot

set -euo pipefail

DB_URL="${DATABASE_URL:-postgres://telefraud:secret@localhost:5432/telefraud_db?sslmode=disable}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/telefraud_db_${TIMESTAMP}.sql.gz"

mkdir -p "${BACKUP_DIR}"

echo "[$(date)] Starting PostgreSQL Backup..."

# Execute pg_dump and compress
pg_dump "${DB_URL}" --clean --if-exists --no-owner | gzip -9 > "${BACKUP_FILE}"

SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "[$(date)] Backup completed successfully: ${BACKUP_FILE} (${SIZE})"

# Prune old backups older than RETENTION_DAYS
echo "[$(date)] Pruning backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -type f -name "telefraud_db_*.sql.gz" -mtime +"${RETENTION_DAYS}" -delete

echo "[$(date)] Backup procedure completed."
```

---

## ⏰ Crontab Setup Guide

To run the standalone backup script every night at 00:00:

```bash
# Open crontab editor
crontab -e

# Add daily execution entry
0 0 * * * /home/xspoilt/Documents/telefraud/scripts/backup.sh >> /var/log/telefraud_backup.log 2>&1
```

---

## 🔄 Disaster Recovery Procedure

To restore the PostgreSQL database from a `.sql.gz` backup file:

### Step 1: Locate the target backup file
```bash
ls -lh ./backups/
# Example: telefraud_db_20260902_000000.sql.gz
```

### Step 2: Test & Verify Backup Integrity
```bash
gunzip -t ./backups/telefraud_db_20260902_000000.sql.gz
```

### Step 3: Restore Database
```bash
# Decompress and import directly into PostgreSQL
gunzip -c ./backups/telefraud_db_20260902_000000.sql.gz | psql -U telefraud -d telefraud_db
```

---

## 🔔 Telegram Admin Backup Alert Example

Upon backup completion, the bot sends an HTML notification to the Admin Log Channel:

```html
<b>💾 Daily Backup Completed</b>

<b>File:</b> <code>telefraud_db_20260902_000000.sql.gz</code>
<b>Size:</b> <code>4.2 MB</code>
<b>Status:</b> <b>SUCCESS</b>
<b>Retention:</b> 30 Days

<i>Database integrity verified.</i>
```
