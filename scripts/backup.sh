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
