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
INSERT INTO bookings (name, email, phone, event_date, event_time, package_title, package_price,
                      event_location, message, terms_accepted, terms_accepted_at, terms_version,
                      status, confirmation_email_count, created_at, updated_at)
VALUES
  ('Klien Satu', 'klien1@example.test', '081234567890', CURRENT_DATE + 30, '09:30:00',
   'Paket Verifikasi', 2500000.00, 'Bandung', 'Halo', TRUE, NOW(), '1', 'PENDING', 0, NOW(), NOW()),
  ('Klien Dua', 'klien2@example.test', '081234567891', CURRENT_DATE + 31, '10:00:00',
   'Paket Verifikasi', 2500000.00, 'Jakarta', 'Halo', TRUE, NOW(), '1', 'COMPLETED', 0, NOW(), NOW());
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
# rescheduleBooking accepts only PENDING or RESCHEDULE_REQUESTED: an ACCEPTED
# booking has an agreed schedule, and a COMPLETED one is closed. Both must be
# refused for their status, before any mail is attempted.
COMPLETED_BODY=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"eventDate":"2030-02-01","eventTime":"09:00"}' \
  "$BASE/api/admin/bookings/2/reschedule")
assert_json 'COMPLETED cannot be rescheduled (transition guard, not mail)' "$COMPLETED_BODY" \
  "'tidak dapat diubah' in d['message'] and 'Selesai' in d['message']"
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c "update bookings set status='ACCEPTED' where id=1;" >/dev/null
ACCEPTED_BODY=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"eventDate":"2030-02-01","eventTime":"09:00"}' \
  "$BASE/api/admin/bookings/1/reschedule")
assert_json 'ACCEPTED cannot be rescheduled either' "$ACCEPTED_BODY" \
  "'tidak dapat diubah' in d['message'] and 'Diterima' in d['message']"
assert_json 'ACCEPTED cannot be set back to RESCHEDULE_REQUESTED' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"status":"RESCHEDULE_REQUESTED"}' "$BASE/api/admin/bookings/1/status")" \
  "d['status']==400 and 'tidak dapat diubah' in d['message']"
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c "update bookings set status='PENDING' where id=1;" >/dev/null
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

# New flow: a reschedule request must pass through ACCEPTED again; completing
# straight from RESCHEDULE_REQUESTED is refused.
assert_json 'RESCHEDULE_REQUESTED cannot jump straight to COMPLETED' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"status":"COMPLETED"}' "$BASE/api/admin/bookings/1/status")" \
  "d['status']==400 and 'tidak dapat diubah' in d['message'] and 'Selesai' in d['message']"
assert_json 'RESCHEDULE_REQUESTED moves to ACCEPTED' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"status":"ACCEPTED"}' "$BASE/api/admin/bookings/1/status")" \
  "d['status']=='ACCEPTED'"
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c "update bookings set status='PENDING' where id=1;" >/dev/null

log '9. Payment status and the invoice / settlement e-mails'
# The invoice is the last message of a booking, so nothing can be billed yet.
assert_json 'invoice refused before the booking is accepted' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
     "$BASE/api/admin/bookings/1/invoice-email")" \
  "d['status']==400 and 'setelah pemesanan selesai' in d['message']"
ACCEPTED_ONE=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"status":"ACCEPTED"}' "$BASE/api/admin/bookings/1/status")
assert_json 'a fresh booking reports UNPAID with no payment dates' "$ACCEPTED_ONE" \
  "d['status']=='ACCEPTED' and d['paymentStatus']=='UNPAID' and d['downPaymentPaidAt'] is None and d['settlementPaidAt'] is None"
assert_json 'invoice refused while the booking is only accepted' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
     "$BASE/api/admin/bookings/1/invoice-email")" \
  "d['status']==400 and 'setelah pemesanan selesai' in d['message']"
assert_json 'settlement e-mail is refused before the DP is recorded' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
     "$BASE/api/admin/bookings/1/settlement-email")" \
  "d['status']==400 and 'Tandai DP' in d['message']"
DOWN_PAID=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"paymentStatus":"DOWN_PAYMENT_PAID"}' "$BASE/api/admin/bookings/1/payment")
assert_json 'PATCH /payment records the down payment' "$DOWN_PAID" \
  "d['paymentStatus']=='DOWN_PAYMENT_PAID' and d['downPaymentPaidAt'] is not None and d['settlementPaidAt'] is None"

# The selection link has its own notice now, sent before the settlement one, so
# the choice comes first and the settlement e-mail only asks for money.
assert_json 'the selection e-mail is refused while the photos are not uploaded' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
     "$BASE/api/admin/bookings/1/selection-email")" \
  "d['status']==400 and 'Folder foto klien belum ada' in d['message']"
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c "update bookings set raw_folder_id='folderRAW123456' where id=1;" >/dev/null
PICK_BEFORE=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/messages" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["total"])')
PICK=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  "$BASE/api/admin/bookings/1/selection-email")
assert_json 'the selection e-mail answers and is tracked' "$PICK" \
  "d['selectionEmailCount']==1 and d['sentTo']=='klien1@example.test' and 'Email pilihan foto berhasil dikirim' in d['message']"
PICK_MAILED=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/messages")
assert_json 'the selection notice reached the SMTP server' "$PICK_MAILED" \
  "d['total']==$PICK_BEFORE+1 and d['messages'][0]['Subject']=='Pilih Foto untuk Diedit — Elvove'"
# Booking 1 was seeded without a token, like every booking taken before
# selection links existed. The notice has to carry a link the customer can open,
# so sending it mints one.
PICK_TOKEN=$(docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "select coalesce(selection_token,'') from bookings where id=1;")
if [[ ${#PICK_TOKEN} -ge 32 ]] && [[ $PICK_TOKEN =~ ^[a-f0-9]+$ ]]; then
  ok 'the selection notice mints a token for an older booking'
else
  bad "booking 1 still has no usable selection token (got: ${PICK_TOKEN:0:40})"
fi
# The link follows MAIL_SITE_URL, which is what the customer's browser opens,
# not the address the test talks to the API on.
SITE="http://localhost:$API_PORT"
PICK_ID=$(printf '%s' "$PICK_MAILED" | python3 -c 'import sys,json;print(json.load(sys.stdin)["messages"][0]["ID"])')
PICK_BODY=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/message/$PICK_ID")
if printf '%s' "$PICK_BODY" | grep -q "$SITE/pilih/$PICK_TOKEN"; then
  ok "the selection notice carries the link $SITE/pilih/<token>"
else
  bad 'the selection notice carries no selection link'
fi

SETTLE_BEFORE=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/messages" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["total"])')
SETTLEMENT=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  "$BASE/api/admin/bookings/1/settlement-email")
assert_json 'settlement e-mail answers and is tracked' "$SETTLEMENT" \
  "d['settlementEmailCount']==1 and d['sentTo']=='klien1@example.test' and 'Email pelunasan berhasil dikirim' in d['message']"
SETTLE_MAILED=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/messages")
assert_json 'the settlement notice reached the SMTP server' "$SETTLE_MAILED" \
  "d['total']==$SETTLE_BEFORE+1 and d['messages'][0]['Subject']=='Pelunasan Sisa Pembayaran — Elvove'"
SETTLE_ID=$(printf '%s' "$SETTLE_MAILED" | python3 -c 'import sys,json;print(json.load(sys.stdin)["messages"][0]["ID"])')
SETTLE_BODY=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/message/$SETTLE_ID")
if printf '%s' "$SETTLE_BODY" | grep -q "/pilih/"; then
  bad 'the settlement notice still carries the selection link'
else
  ok 'the settlement notice asks for the balance only, without the selection link'
fi

COMPLETED_ONE=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"status":"COMPLETED"}' "$BASE/api/admin/bookings/1/status")
assert_json 'the booking moves on to COMPLETED' "$COMPLETED_ONE" \
  "d['status']=='COMPLETED'"
# A finished booking with an open balance is not a closed order yet.
assert_json 'invoice refused while the balance is open' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
     "$BASE/api/admin/bookings/1/invoice-email")" \
  "d['status']==400 and 'lunas' in d['message']"

PAID_BODY=$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"paymentStatus":"PAID"}' "$BASE/api/admin/bookings/1/payment")
assert_json 'marking lunas records both payment points' "$PAID_BODY" \
  "d['paymentStatus']=='PAID' and d['downPaymentPaidAt'] is not None and d['settlementPaidAt'] is not None"
assert_json 'a settled booking refuses a settlement e-mail' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
     "$BASE/api/admin/bookings/1/settlement-email")" \
  "d['status']==400 and 'sudah ditandai lunas' in d['message']"

# The invoice carries the gallery link, so it is refused until one is saved.
assert_json 'invoice refused without a delivery link' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
     "$BASE/api/admin/bookings/1/invoice-email")" \
  "d['status']==400 and 'Link folder foto edit' in d['message']"
assert_json 'PATCH /delivery rejects a link that is not http(s)' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"deliveryUrl":"javascript:alert(1)"}' "$BASE/api/admin/bookings/1/delivery")" \
  "d['error']=='Validation Failed' and 'deliveryUrl' in (d['fieldErrors'] or {})"
assert_json 'PATCH /delivery stores the Drive folder' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"deliveryUrl":"https://drive.google.com/drive/folders/verify"}' "$BASE/api/admin/bookings/1/delivery")" \
  "d['deliveryUrl']=='https://drive.google.com/drive/folders/verify'"

INVOICE=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  "$BASE/api/admin/bookings/1/invoice-email")
assert_json 'invoice answers and is tracked' "$INVOICE" \
  "d['invoiceEmailCount']==1 and d['sentTo']=='klien1@example.test' and 'Email invoice berhasil dikirim' in d['message']"
# The number is derived from the booking id and the year, so it is stable.
INVOICE_MAILED=$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/messages")
assert_json 'the invoice carries the derived number in its subject' "$INVOICE_MAILED" \
  "d['messages'][0]['Subject'].startswith('Invoice INV-') and d['messages'][0]['Subject'].endswith('-0001 — Elvove')"
INVOICE_ID=$(printf '%s' "$INVOICE_MAILED" | python3 -c 'import sys,json;print(json.load(sys.stdin)["messages"][0]["ID"])')
assert_json 'the invoice carries the Drive link' \
  "$(curl -s "http://127.0.0.1:$MAIL_HTTP_PORT/api/v1/message/$INVOICE_ID")" \
  "'https://drive.google.com/drive/folders/verify' in (d.get('Text') or '')"
assert_json 'unknown payment status is an unreadable body' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"paymentStatus":"HALF"}' "$BASE/api/admin/bookings/1/payment")" \
  "d['status']==400 and d['message']=='Format permintaan tidak valid. Periksa kembali data yang dikirim.'"
assert_json 'missing payment status gives fieldErrors' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{}' "$BASE/api/admin/bookings/1/payment")" \
  "d['error']=='Validation Failed' and 'paymentStatus' in (d['fieldErrors'] or {})"
assert_json 'invoice for a missing booking answers 404' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/bookings/999/invoice-email")" \
  "d['status']==404 and d['message']=='Pemesanan tidak ditemukan.'"

printf '\n\033[1mResult: %d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
