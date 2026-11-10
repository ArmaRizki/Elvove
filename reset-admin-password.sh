
#!/usr/bin/env bash
set -euo pipefail

CONTAINER="photography-local-db"
DB_NAME="photographer_portfolio"
DB_USER="postgres"
ADMIN_USERNAME="superadmin"

echo "================================"
echo "     ELVOVE SUPERADMIN PASSWORD"
echo "================================"

if ! docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null | grep -q true; then
    echo "ERROR: Container database tidak berjalan."
    exit 1
fi

if ! python3 -c "import bcrypt" 2>/dev/null; then
    echo "ERROR: Python bcrypt belum terpasang."
    exit 1
fi

read -rsp "Masukkan password baru: " NEW_PASSWORD
echo

if [[ -z "$NEW_PASSWORD" ]]; then
    echo "ERROR: Password tidak boleh kosong."
    exit 1
fi

HASH=$(NEW_PASSWORD="$NEW_PASSWORD" python3 - <<'PY'
import os
import bcrypt

password = os.environ["NEW_PASSWORD"].encode("utf-8")

if len(password) > 72:
    raise SystemExit("Password maksimal 72 byte.")

print(bcrypt.hashpw(password, bcrypt.gensalt()).decode())
PY
)

unset NEW_PASSWORD

# Update menggunakan SQL melalui stdin
RESULT=$(printf '%s\n' "$HASH" | docker exec -i "$CONTAINER" \
    psql -X -U "$DB_USER" -d "$DB_NAME" -t -A \
    -c "UPDATE admins SET password = trim(current_setting('app.new_password', true)) WHERE username = '$ADMIN_USERNAME';" 2>&1 || true)

# Gunakan parameter SQL melalui file sementara di container
RESULT=$(printf '%s' "$HASH" | docker exec -i "$CONTAINER" \
    sh -c 'read HASH; export HASH; psql -X -U postgres -d photographer_portfolio -c "UPDATE admins SET password = '\''$HASH'\'' WHERE username = '\''superadmin'\'';"' 2>&1)

unset HASH

if ! echo "$RESULT" | grep -q "UPDATE 1"; then
    echo "ERROR: Password gagal diperbarui."
    echo "$RESULT"
    exit 1
fi

echo "Password admin berhasil diperbarui."
echo "Username: $ADMIN_USERNAME"

