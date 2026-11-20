#!/bin/sh
# Back up the site's uploaded images, keeping the last few days.
#
# Run from the stack's folder on the server (the one with compose.yml):
#
#   ./backup-uploads.sh                 # writes local-data/backups/uploads/
#   DEST=/mnt/other/disk ./backup-uploads.sh   # onto a different disk
#
# The images are what the database points at: rows are cheap to restore, the
# originals are not — they cannot be re-created once lost.
#
# WHERE THE COPY GOES, AND WHY IT MATTERS
#
# On this server / is a 5.8 GB SD card and the stack's own disk holds Docker. A
# copy written to the same card protects against a deleted file or a bad upload,
# not against the card failing — the very thing that loses everything at once.
# Point DEST at a different physical disk (an external drive, another machine
# over NFS) for a copy that survives. The script refuses to pretend: if DEST is
# on the same filesystem as the uploads, it says so and still runs, because a
# same-disk copy is better than none.
#
# The archive is written to a temporary name, verified with tar -tzf, and only
# then renamed: a half-written file never sits there looking like a backup.
set -eu

STACK_DIR=${STACK_DIR:-$(cd "$(dirname "$0")" && pwd)}
SRC=${SRC:-$STACK_DIR/local-data/uploads}
DEST=${DEST:-$STACK_DIR/local-data/backups/uploads}
KEEP=${KEEP:-7}

[ -d "$SRC" ] || { echo "uploads backup: $SRC does not exist" >&2; exit 1; }

mkdir -p "$DEST"

# Say it plainly when the copy lands on the same disk as the source.
src_fs=$(df -P "$SRC" | awk 'NR==2 {print $1}')
dst_fs=$(df -P "$DEST" | awk 'NR==2 {print $1}')
if [ "$src_fs" = "$dst_fs" ]; then
	echo "uploads backup: WARNING $DEST is on the same filesystem ($dst_fs) as $SRC." >&2
	echo "uploads backup: this copy does not survive the disk failing. Set DEST to another disk." >&2
fi

stamp=$(date +%Y%m%d-%H%M%S)
final="$DEST/uploads-$stamp.tar.gz"
partial="$DEST/.uploads-$stamp.partial"

# The uploads folder holds images and nothing else; a symlink in it would make
# tar read whatever it points at (a `/etc/passwd` link in a folder that was once
# writable ended up in this very archive during testing). The check refuses to
# run at all, rather than archiving a stranger's file or skipping it quietly.
if find "$SRC" -type l | grep -q .; then
	echo "uploads backup: symlinks found under $SRC, refusing to archive:" >&2
	find "$SRC" -type l >&2
	exit 1
fi

if ! tar -C "$SRC" -czf "$partial" .; then
	echo "uploads backup: tar did not finish" >&2
	rm -f "$partial"
	exit 1
fi

# A file that does not open is not a backup; check before keeping it.
if ! tar -tzf "$partial" >/dev/null 2>&1; then
	echo "uploads backup: $partial does not read back" >&2
	rm -f "$partial"
	exit 1
fi
files=$(tar -tzf "$partial" | grep -cv '/$' || true)
if [ "$files" -lt 1 ]; then
	echo "uploads backup: no files in the archive" >&2
	rm -f "$partial"
	exit 1
fi

mv "$partial" "$final"
echo "$(date -Iseconds) wrote $final ($files files, $(du -h "$final" | cut -f1))"

# Prune: keep the newest $KEEP archives, delete the rest. A run repeated on the
# same day (a manual retry beside the cron one) replaces that day's copy instead
# of being kept as a second one, so "keep 3" means three days, not three runs.
today=$(date +%Y%m%d)
for old in $(ls -1t "$DEST"/uploads-*.tar.gz 2>/dev/null | tail -n +$((KEEP + 1))); do
	case "$(basename "$old")" in
	"uploads-$today-"*) continue ;;
	esac
	rm -f "$old"
	echo "$(date -Iseconds) removed old backup $old"
done
