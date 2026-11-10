#!/usr/bin/env bash
# Verifies the Go backend against a throwaway PostgreSQL + SMTP server.
#
# Everything runs in containers named with the "elvove-go-verify-" prefix and on
# non-default ports, so a running Java stack (photography-local-*) is never
# touched. Both containers are removed when the script exits.
#
# Usage:  ./scripts/verify-staging.sh [--keep]
#   --keep   leave the containers running for debugging
#
# Exit status: 0 when every check passes, 1 otherwise.
set -euo pipefail

DB_CONTAINER=elvove-go-verify-db
MAIL_CONTAINER=elvove-go-verify-mail
DB_PORT=55432
API_PORT=58099
MAIL_HTTP_PORT=58081
SCOPE_DB=photography-local-db
SCOPE_DB_NAME=photographer_portfolio
WORK_DIR=$(mktemp -d)
KEEP=0
[[ ${1:-} == --keep ]] && KEEP=1

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
BACKEND_DIR="$REPO_ROOT"
API_BIN="$WORK_DIR/elvove-api"
PASS=0
FAIL=0

log()  { printf '\n\033[1m%s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mok\033[0m   %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }

cleanup() {
  [[ -n ${API_PID:-} ]] && kill "$API_PID" 2>/dev/null || true
  if [[ $KEEP -eq 1 ]]; then
    printf '\n--keep given: containers kept, database on 127.0.0.1:%s\n' "$DB_PORT"
    return
  fi
  docker rm -f "$DB_CONTAINER" "$MAIL_CONTAINER" >/dev/null 2>&1 || true
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

# assert_json <label> <json> <python expression over variable d>
assert_json() {
  local label=$1 json=$2 expr=$3
  if printf '%s' "$json" | python3 -c "
import json,sys
d=json.load(sys.stdin)
sys.exit(0 if ($expr) else 1)
" 2>/dev/null; then
    ok "$label"
  else
    bad "$label (got: ${json:0:300})"
  fi
}

log '1. Schema from the local Java database (read-only pg_dump)'
docker run -d --name "$DB_CONTAINER" \
  -e POSTGRES_PASSWORD=verifypass -e POSTGRES_DB=elvove_verify \
  -p "127.0.0.1:$DB_PORT:5432" postgres:16-alpine >/dev/null
for _ in $(seq 1 30); do
  docker exec "$DB_CONTAINER" pg_isready -U postgres -d elvove_verify >/dev/null 2>&1 && break
  sleep 1
done
if docker exec "$SCOPE_DB" pg_dump -U postgres --schema-only --no-owner --no-privileges \
     "$SCOPE_DB_NAME" > "$WORK_DIR/schema.sql" 2>/dev/null; then
  ok "dumped schema from $SCOPE_DB (read-only)"
else
  bad "could not dump schema from $SCOPE_DB"
  exit 1
fi
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q < "$WORK_DIR/schema.sql" >/dev/null
ok 'schema loaded into the throwaway database'

log '2. SMTP sink (Mailpit) and fixture data'
docker run -d --name "$MAIL_CONTAINER" \
  -p "127.0.0.1:$MAIL_HTTP_PORT:8025" axllent/mailpit:v1.33.1 >/dev/null
sleep 2
MAIL_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$MAIL_CONTAINER")

# bcrypt hash of "verify-password-123"
ADMIN_HASH='$2a$10$uTjM/UcGJgO1aiPmfF.ZDuWvQ9h/PsjHYtUxBKcJ5pJeqQgQ.Udpy'
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q <<SQL >/dev/null
INSERT INTO admins (username, email, password, created_at, updated_at)
VALUES ('verifyadmin', 'verify@example.test', '$ADMIN_HASH', NOW(), NOW());
INSERT INTO packages (title, price, display_order, highlighted, created_at, updated_at)
VALUES ('Paket Verifikasi', 2500000.00, 1, true, NOW(), NOW());
INSERT INTO site_terms (title, introduction, content, version, created_at, updated_at)
-- introduction is deliberately NULL: production has it NULL, and a bare scan
-- into string rejected every booking with a 500 until this was fixed.
VALUES ('Terms & Conditions', NULL, 'Isi syarat dan ketentuan.', '1', NOW(), NOW());
INSERT INTO smtp_settings (id, host, port, username, encrypted_password, encryption, authentication, sender_name, sender_email)
VALUES (1, '$MAIL_IP', 1025, NULL, NULL, 'NONE', false, 'Elvove', 'no-reply@elvove.test');
INSERT INTO bookings (name, email, phone, event_type, event_date, event_time, package_title, package_price,
                      event_location, budget, message, terms_accepted, terms_accepted_at, terms_version,
                      status, confirmation_email_count, created_at, updated_at)
VALUES
  ('Klien Satu', 'klien1@example.test', '081234567890', 'Wisuda', CURRENT_DATE + 30, '09:30:00',
   'Paket Verifikasi', 2500000.00, 'Bandung', '3000000', 'Halo', TRUE, NOW(), '1', 'PENDING', 0, NOW(), NOW()),
  ('Klien Dua', 'klien2@example.test', '081234567891', 'Wisuda', CURRENT_DATE + 31, '10:00:00',
   'Paket Verifikasi', 2500000.00, 'Jakarta', '3500000', 'Halo', TRUE, NOW(), '1', 'COMPLETED', 0, NOW(), NOW());
SQL
ok 'fixture inserted (admin, package, terms, smtp, 2 bookings)'

log '3. Build and start the Go backend'
(cd "$BACKEND_DIR" && go build -o "$API_BIN" ./cmd/api)
mkdir -p "$WORK_DIR/uploads"
DB_URL="postgres://postgres:verifypass@127.0.0.1:$DB_PORT/elvove_verify?sslmode=disable" \
JWT_SECRET='verify-secret-at-least-32-bytes-long!!' \
SMTP_ENCRYPTION_KEY='verify-smtp-key-at-least-32-bytes-long!' \
PORT=$API_PORT UPLOAD_DIR="$WORK_DIR/uploads" MAIL_SITE_URL="http://localhost:$API_PORT" \
  "$API_BIN" > "$WORK_DIR/api.log" 2>&1 &
API_PID=$!
for _ in $(seq 1 30); do
  curl -sf "http://127.0.0.1:$API_PORT/health" >/dev/null 2>&1 && break
  sleep 1
done
if curl -sf "http://127.0.0.1:$API_PORT/health" >/dev/null; then
  ok "API listening on 127.0.0.1:$API_PORT"
else
  bad 'API did not start'; sed -n '1,40p' "$WORK_DIR/api.log"; exit 1
fi
BASE="http://127.0.0.1:$API_PORT"

log '4. Auth and bookingColumns (GET /api/admin/bookings)'
TOKEN=$(curl -s -X POST "$BASE/api/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"verifyadmin","password":"verify-password-123"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
[[ -n $TOKEN ]] && ok 'login returned a token' || bad 'login returned no token'
LIST=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/bookings")
assert_json 'eventDate is a bare date like Java LocalDate' "$LIST" "isinstance(d,list) and len(d)==2 and d[0]['eventDate'].count('-')==2 and 'T' not in d[0]['eventDate']"
assert_json 'eventTime is an ISO string like Java LocalTime' "$LIST" "'T' not in d[1]['eventTime'] and d[1]['eventTime'].count(':')==2"
assert_json 'packagePrice stays a JSON number' "$LIST" "isinstance(d[0]['packagePrice'],float)"
assert_json 'createdAt has no zone designator (Java LocalDateTime)' "$LIST" "isinstance(d[0]['createdAt'],str) and not d[0]['createdAt'].endswith('Z')"

log '5. fetchBooking and the transition rules (PATCH /status)'
PATCHED=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"status":"ACCEPTED"}' "$BASE/api/admin/bookings/1/status")
assert_json 'PATCH /status returns the full BookingResponse' "$PATCHED" "d['status']=='ACCEPTED' and {'id','name','email','packageTitle','createdAt'} <= set(d)"
# Java's JVM runs in Asia/Jakarta, so LocalDateTime.now() lands as WIB wall
# clock; a column filled by SQL NOW() must hold the same fields, not UTC.
WIB_HOUR=$(date -u -d '+7 hours' +%Y-%m-%dT%H 2>/dev/null || date -u -v+7H +%Y-%m-%dT%H)
UPDATED_HOUR=$(printf '%s' "$PATCHED" | python3 -c 'import sys,json;print(json.load(sys.stdin)["updatedAt"][:13])')
[[ $UPDATED_HOUR == "$WIB_HOUR" ]] \
  && ok 'updatedAt is WIB wall clock like Java LocalDateTime.now()' \
  || bad "updatedAt '$UPDATED_HOUR' is not the WIB hour '$WIB_HOUR'"
SAME=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"status":"ACCEPTED"}' "$BASE/api/admin/bookings/1/status")
assert_json 'same-status message matches Java' "$SAME" "'sudah berstatus' in d['message'] and 'Diterima' in d['message']"
BADMOVE=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"status":"PENDING"}' "$BASE/api/admin/bookings/2/status")
assert_json 'disallowed transition uses Indonesian labels' "$BADMOVE" "'tidak dapat diubah' in d['message'] and 'Selesai' in d['message']"
EMPTY=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{}' "$BASE/api/admin/bookings/1/status")
assert_json 'missing status gives Validation Failed + fieldErrors' "$EMPTY" "d['error']=='Validation Failed' and 'status' in (d['fieldErrors'] or {})"

log '6. Public booking: fieldErrors, 404 package, WIB timestamps'
INVALID=$(curl -s -X POST "$BASE/api/bookings" -H 'Content-Type: application/json' \
  -d '{"name":"","email":"bad","phone":"1","eventDate":"2020-01-01","eventTime":"","termsAccepted":false,"termsVersion":""}')
assert_json 'invalid booking reports fieldErrors per field' "$INVALID" "len(d['fieldErrors'])==8 and d['fieldErrors']['email']=='Enter a valid email address' and d['fieldErrors']['packageId']=='Package is required'"
# packageId is @NotNull only, so a present 0 passes validation and the lookup 404s.
ZERO=$(curl -s -X POST "$BASE/api/bookings" -H 'Content-Type: application/json' \
  -d '{"name":"Klien Tiga","email":"klien3@example.test","phone":"081234567892","eventDate":"2030-01-02","eventTime":"11:00","packageId":0,"termsAccepted":true,"termsVersion":"1"}')
assert_json 'packageId 0 passes @NotNull and 404s like Java' "$ZERO" "d['status']==404 and d['message']=='PackageItem not found with id: 0'"
NOTFOUND=$(curl -s -X POST "$BASE/api/bookings" -H 'Content-Type: application/json' \
  -d '{"name":"Klien Tiga","email":"klien3@example.test","phone":"081234567892","eventDate":"2030-01-02","eventTime":"11:00","packageId":999,"termsAccepted":true,"termsVersion":"1"}')
assert_json 'unknown package gives Java not-found message' "$NOTFOUND" "d['status']==404 and d['message']=='PackageItem not found with id: 999'"
CREATED=$(curl -s -X POST "$BASE/api/bookings" -H 'Content-Type: application/json' \
  -d "{\"name\":\"Klien Tiga\",\"email\":\"klien3@example.test\",\"phone\":\"081234567892\",\"eventDate\":\"$(date -d '+40 days' +%Y-%m-%d 2>/dev/null || date -v+40d +%Y-%m-%d)\",\"eventTime\":\"11:00\",\"packageId\":1,\"termsAccepted\":true,\"termsVersion\":\"1\"}")
assert_json 'created booking echoes WIB timestamps without a zone' "$CREATED" "isinstance(d.get('id'),int) and not d['createdAt'].endswith('Z') and 'T' in d['createdAt']"

log '7. Reschedule rolls back when SMTP fails'
# rescheduleBooking only accepts PENDING or ACCEPTED (Java allows exactly those),
# so booking 2 (COMPLETED) must be refused for its status before any mail is
# attempted. Asserting it here keeps the next phase honest about which row it uses.
COMPLETED_BODY=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"eventDate":"2030-02-01","eventTime":"09:00"}' \
  "$BASE/api/admin/bookings/2/reschedule")
assert_json 'COMPLETED cannot be rescheduled (transition guard, not mail)' "$COMPLETED_BODY" \
  "'tidak dapat diubah' in d['message'] and 'Selesai' in d['message']"
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c "update bookings set status='ACCEPTED' where id=1;" >/dev/null
BEFORE=$(docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "select status||'|'||event_date||'|'||event_time from bookings where id=1;")
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c 'update smtp_settings set port=1599 where id=1;' >/dev/null
DOWN_BODY=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"eventDate":"2030-02-01","eventTime":"09:00"}' \
  "$BASE/api/admin/bookings/1/reschedule")
AFTER=$(docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "select status||'|'||event_date||'|'||event_time from bookings where id=1;")
# Assert the mail message, not just the code: a status refusal is also a 400.
assert_json 'failed mail answers 400 with the mail message' "$DOWN_BODY" \
  "d['status']==400 and 'email pemberitahuan jadwal gagal dikirim' in d['message']"
[[ $BEFORE == "$AFTER" ]] && ok 'row rewound: status and schedule unchanged' || bad "row changed: '$BEFORE' -> '$AFTER'"
grep -q 'reschedule mail failed' "$WORK_DIR/api.log" \
  && ok 'the SMTP error was logged, not swallowed' \
  || bad 'no SMTP error in the log'

log '8. Reschedule commits when SMTP succeeds'
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c 'update smtp_settings set port=1025 where id=1;' >/dev/null
MAIL_BEFORE=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/messages" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["total"])')
UP_BODY=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"eventDate":"2030-02-01","eventTime":"09:00"}' \
  "$BASE/api/admin/bookings/1/reschedule")
# A success body carries the new status as a string; an error body carries the
# HTTP code as an int, so the type tells the two apart.
assert_json 'successful mail answers 200 with the Java message' "$UP_BODY" \
  "d.get('status')=='RESCHEDULE_REQUESTED' and d.get('message')=='Jadwal diperbarui dan email pemberitahuan jadwal telah dikirim.'"
assert_json 'the row was committed' \
  "$(docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
     "select json_build_object('s',status,'d',event_date,'t',event_time) from bookings where id=1;")" \
  "d['s']=='RESCHEDULE_REQUESTED' and str(d['d'])=='2030-02-01' and str(d['t']).startswith('09:00')"
MAILED=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/messages")
assert_json 'exactly one schedule e-mail reached the SMTP server' "$MAILED" \
  "d['total']==$MAIL_BEFORE+1 and 'Jadwal Sesi Foto Diperbarui' in d['messages'][0]['Subject']"

printf '\n\033[1mResult: %d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
