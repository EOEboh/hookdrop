#!/bin/bash
# Back up the hookdrop SQLite database to Cloudflare R2.
#
# Run by the hookdrop-backup systemd timer installed by provision.sh, which
# supplies the environment from /opt/hookdrop/.env. Run it by hand to test:
#   sudo systemctl start hookdrop-backup.service
#   journalctl -u hookdrop-backup.service -n 50
#
# Uses `sqlite3 .backup`, not `cp`. The database runs in WAL mode with a live
# writer attached, so copying the file gives you a torn snapshot missing
# whatever sits in the WAL. `.backup` takes a consistent one online, without
# stopping the container.

set -euo pipefail

APP_DIR="${APP_DIR:-/opt/hookdrop}"
DB="${DB:-$APP_DIR/data/hookdrop.db}"
BACKUP_DIR="${BACKUP_DIR:-$APP_DIR/backups}"
KEEP_LOCAL_DAYS="${KEEP_LOCAL_DAYS:-7}"

: "${R2_ACCOUNT_ID:?set in $APP_DIR/.env}"
: "${R2_ACCESS_KEY_ID:?set in $APP_DIR/.env}"
: "${R2_SECRET_ACCESS_KEY:?set in $APP_DIR/.env}"
: "${R2_BUCKET:?set in $APP_DIR/.env}"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
NAME="hookdrop-$STAMP.db"
OUT="$BACKUP_DIR/$NAME"

mkdir -p "$BACKUP_DIR"

echo "→ snapshotting $DB"
# .timeout matches the app's own _busy_timeout. Without it a snapshot taken
# while a webhook is mid-write can lose the lock race and return SQLITE_BUSY,
# and the CLI does not retry on its own.
sqlite3 "$DB" ".timeout 10000" ".backup '$OUT'"

# sqlite3 can exit 0 having written nothing. An empty or absent snapshot must
# fail the run loudly rather than be gzipped and uploaded as a real backup.
[ -s "$OUT" ] || { echo "✗ snapshot produced no file"; exit 1; }

# A snapshot that will not open is not a backup. Catch it here, while the
# source database is still sitting right there, rather than during a restore.
echo "→ verifying"
result="$(sqlite3 -readonly "$OUT" 'PRAGMA integrity_check;')"
[ "$result" = "ok" ] || { echo "✗ integrity check failed: $result"; rm -f "$OUT"; exit 1; }
users="$(sqlite3 -readonly "$OUT" 'SELECT count(*) FROM users;')"
echo "  ok — $users users, $(du -h "$OUT" | cut -f1)"

# The snapshot inherits WAL journal mode from the source, so the two reads
# above created sidecar files next to it. They are empty and useless once the
# snapshot is closed, but nothing else removes them and they would pile up in
# this directory forever.
rm -f "$OUT-wal" "$OUT-shm"

gzip -f "$OUT"
OUT="$OUT.gz"
NAME="$NAME.gz"

echo "→ uploading to r2://$R2_BUCKET/$NAME"
AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID" \
AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY" \
AWS_DEFAULT_REGION=auto \
	aws s3 cp "$OUT" "s3://$R2_BUCKET/$NAME" \
	--endpoint-url "https://$R2_ACCOUNT_ID.r2.cloudflarestorage.com" \
	--only-show-errors

echo "→ pruning local copies older than ${KEEP_LOCAL_DAYS}d"
find "$BACKUP_DIR" \( -name 'hookdrop-*.db.gz' -o -name 'hookdrop-*.db-wal' \
	-o -name 'hookdrop-*.db-shm' \) -mtime "+$KEEP_LOCAL_DAYS" -delete

echo "✓ $NAME"
