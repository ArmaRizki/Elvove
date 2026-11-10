#!/usr/bin/env bash
# Sets the admin password on the throwaway database used by --copy-db and
# remembers it, so the same password keeps working after every restart.
#
# The hash is computed with pgcrypto inside the container instead of being
# pasted in, so the stored value is a real bcrypt hash of the current password
# rather than a fixed string that could drift away from this file.
#
# The password lives in local-data/go-web/admin-password (gitignored). The Java
# database is never touched: only the copy is updated.
#
# Usage:  ./scripts/set-dev-admin.sh [--user NAME] [--show]
#   --user NAME   admin username to update (default: superadmin)
#   --show        print the current dev password and exit
set -euo pipefail

CONTAINER=${COPY_DB_CONTAINER:-elvove-go-copy-db}
DB_NAME=${DB_NAME:-photographer_portfolio}
USERNAME=superadmin
SHOW=0
NEW=0

while [[ $# -gt 0 ]]; do
  case $1 in
    --show) SHOW=1; shift ;;
    --new) NEW=1; shift ;;
    --user)
      [[ $# -ge 2 ]] || { echo "error: --user needs a name" >&2; exit 1; }
      USERNAME=$2; shift 2 ;;
    -h|--help)
      sed -n '2,17p' "$0" | sed 's/^# \{0,1\}//'
      exit 0 ;;
    *)
      echo "error: unknown option $1" >&2; exit 1 ;;
  esac
done

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
CRED_DIR="$REPO_ROOT/local-data/go-web"
CRED_FILE="$CRED_DIR/admin-password"

if [[ $SHOW -eq 1 ]]; then
  [[ -f $CRED_FILE ]] || { echo "no dev password stored yet; run $0 first" >&2; exit 1; }
  printf 'user: %s\npassword: %s\n' "$(head -1 "$CRED_FILE")" "$(sed -n 2p "$CRED_FILE")"
  exit 0
fi

docker inspect "$CONTAINER" >/dev/null 2>&1 || {
  echo "error: $CONTAINER is not running; run ./scripts/run-web.sh --copy-db first" >&2
  exit 1
}

mkdir -p "$CRED_DIR"

# Keep the existing password so restarts do not invalidate it, unless the caller
# explicitly asks for a new one with --new.
PASSWORD=""
if [[ $NEW -eq 1 || ! -f $CRED_FILE ]]; then
  # Readable but not guessable, and avoids quoting trouble in shell and SQL.
  PASSWORD="elvove-$(head -c 9 /dev/urandom | base64 | tr -dc 'a-z0-9' | head -c 10)"
  printf '%s\n%s\n' "$USERNAME" "$PASSWORD" > "$CRED_FILE"
  chmod 600 "$CRED_FILE"
elif [[ -f $CRED_FILE ]]; then
  USERNAME=$(head -1 "$CRED_FILE")
  PASSWORD=$(sed -n 2p "$CRED_FILE")
fi

# crypt()/gen_salt() come from pgcrypto; the copy is a disposable container, so
# creating the extension there is safe.
HASH=$(docker exec "$CONTAINER" psql -U postgres -d "$DB_NAME" -tAc \
  "CREATE EXTENSION IF NOT EXISTS pgcrypto; SELECT crypt('$PASSWORD', gen_salt('bf', 12));" \
  | tail -1 | tr -d '[:space:]')

[[ -n $HASH ]] || { echo "error: could not compute a bcrypt hash" >&2; exit 1; }

UPDATED=$(docker exec "$CONTAINER" psql -U postgres -d "$DB_NAME" -tAc \
  "UPDATE admins SET password='$HASH' WHERE username='$USERNAME'; SELECT count(*) FROM admins WHERE username='$USERNAME';" \
  | tail -1 | tr -d '[:space:]')

[[ $UPDATED == 1 ]] || {
  echo "error: no admin named '$USERNAME' in $CONTAINER" >&2
  exit 1
}

printf 'Admin password set on %s\n  user:     %s\n  password: %s\n  stored:   %s\n' \
  "$CONTAINER" "$USERNAME" "$PASSWORD" "$CRED_FILE"
