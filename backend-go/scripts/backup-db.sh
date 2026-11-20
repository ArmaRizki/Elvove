#!/bin/sh
# Back up the site's Postgres database, keeping the last few days.
#
# Run from the stack's folder on the server (the one with compose.yml). It dumps
# through the running database container, so nothing needs PostgreSQL installed
# on the host and no password is written anywhere.
#
#   ./backup-db.sh              # writes local-data/backups/YYYYMMDD-HHMMSS.sql.gz
#   KEEP=14 ./backup-db.sh      # keep 14 days instead of 7
#   MIRROR=/mnt/sdcard/backups/db ./backup-db.sh   # second copy on another disk
#
# MIRROR is for the case this server has: the stack lives on one card and the
# Docker volumes (including Postgres' own data) on the other, so a copy of the
# dump on each disk means either one dying still leaves a file to restore from.
# The mirror is a copy, never a move: the dump in local-data/backups stays where
# it was, and a mirror that cannot be written fails the run instead of being
# skipped quietly.
#
# The dump is verified before the older ones are pruned: a backup that fails to
# gunzip is discarded and the script exits non-zero, so a cron mail says so
# instead of a silent empty file sitting there looking safe.
#
# Restore (destroys the current data):
#   gunzip -c local-data/backups/<file>.sql.gz \
#     | docker exec -i photography-db psql -U postgres -d photographer_portfolio
set -eu

STACK_DIR=${STACK_DIR:-$(cd "$(dirname "$0")" && pwd)}
CONTAINER=${CONTAINER:-photography-db}
DATABASE=${DATABASE:-photographer_portfolio}
DB_USER=${DB_USER:-postgres}
KEEP=${KEEP:-7}
MIRROR=${MIRROR:-}

BACKUP_DIR="$STACK_DIR/local-data/backups"
mkdir -p "$BACKUP_DIR"

stamp=$(date +%Y%m%d-%H%M%S)
final="$BACKUP_DIR/$stamp.sql.gz"
partial="$BACKUP_DIR/.$stamp.partial"

# --clean --if-exists makes the dump restorable over an existing database.
if ! docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DATABASE" --clean --if-exists \
	| gzip -9 >"$partial"; then
	echo "backup failed: pg_dump did not finish" >&2
	rm -f "$partial"
	exit 1
fi

# A dump that does not open is not a backup. Check it before keeping it.
if ! gzip -t "$partial" 2>/dev/null || [ ! -s "$partial" ]; then
	echo "backup failed: $partial is empty or not gzip" >&2
	rm -f "$partial"
	exit 1
fi
tables=$(gunzip -c "$partial" | grep -c '^CREATE TABLE' || true)
if [ "$tables" -lt 5 ]; then
	echo "backup failed: only $tables tables in the dump, expected the schema" >&2
	rm -f "$partial"
	exit 1
fi

mv "$partial" "$final"
echo "$(date -Iseconds) wrote $final ($tables tables, $(du -h "$final" | cut -f1))"

# Prune: keep the newest $KEEP dumps, delete the rest.
ls -1t "$BACKUP_DIR"/*.sql.gz 2>/dev/null | tail -n +$((KEEP + 1)) | while read -r old; do
	rm -f "$old"
	echo "$(date -Iseconds) removed old backup $old"
done

# The second copy, on another disk. Written after the first is safe, and its own
# retention is pruned the same way. A failure here is reported and fails the run:
# a mirror that silently stops being written is worse than none, because the
# other disk is then trusted for something it no longer holds.
if [ -n "$MIRROR" ]; then
	mkdir -p "$MIRROR"
	mirrored="$MIRROR/$(basename "$final")"
	if ! cp "$final" "$mirrored"; then
		echo "backup failed: could not write the mirror copy to $MIRROR" >&2
		exit 1
	fi
	echo "$(date -Iseconds) mirrored to $mirrored"
	ls -1t "$MIRROR"/*.sql.gz 2>/dev/null | tail -n +$((KEEP + 1)) | while read -r old; do
		rm -f "$old"
		echo "$(date -Iseconds) removed old mirror $old"
	done
fi
