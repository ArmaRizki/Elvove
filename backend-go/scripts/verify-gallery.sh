# The client photo-selection endpoints, checked against the throwaway database
# and a stubbed Google Drive.
#
# Run from the repository root, after the schema has been loaded into the same
# throwaway database the main script uses:
#   ./scripts/verify-gallery.sh
#
# It runs the backend from its Docker image, because the WebP conversion needs
# cwebp, which only the image carries. GOOGLE_OAUTH_* points at a local stub, so
# no real Drive account is touched.
set -euo pipefail

DB_CONTAINER=elvove-go-verify-db
API_CONTAINER=elvove-go-verify-gallery
API_IMAGE=elvove-verify-gallery:local
API_PORT=58098
DRIVE_PORT=58097
WORK_DIR=$(mktemp -d)
PASS=0
FAIL=0

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
API_PID=

log() { printf '\n\033[1m%s\033[0m\n' "$*"; }
ok()  { printf '  \033[32mok\033[0m   %s\n' "$*"; PASS=$((PASS + 1)); }
bad() { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }

cleanup() {
  # The database container is shared by name with verify-staging.sh, so it has
  # to be removed here too; otherwise the next run conflicts on the name.
  docker rm -f "$API_CONTAINER" "$DB_CONTAINER" >/dev/null 2>&1 || true
  [[ -n ${STUB_PID:-} ]] && kill "$STUB_PID" 2>/dev/null || true
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

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

log '0. Throwaway database'
docker rm -f "$DB_CONTAINER" >/dev/null 2>&1 || true
docker run -d --name "$DB_CONTAINER" \
  -e POSTGRES_PASSWORD=verifypass -e POSTGRES_DB=elvove_verify \
  -p "127.0.0.1:55432:5432" postgres:16-alpine >/dev/null
for _ in $(seq 1 30); do
  docker exec "$DB_CONTAINER" pg_isready -U postgres -d elvove_verify >/dev/null 2>&1 && break
  sleep 1
done
docker exec photography-local-db pg_dump -U postgres --schema-only --no-owner --no-privileges \
  photographer_portfolio > "$WORK_DIR/schema.sql"
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q < "$WORK_DIR/schema.sql" >/dev/null

ADMIN_HASH='$2a$10$uTjM/UcGJgO1aiPmfF.ZDuWvQ9h/PsjHYtUxBKcJ5pJeqQgQ.Udpy'
docker exec -i "$DB_CONTAINER" psql -U postgres -d elvove_verify -q <<SQL >/dev/null
INSERT INTO admins (username, email, password, created_at, updated_at)
VALUES ('verifyadmin', 'verify@example.test', '$ADMIN_HASH', NOW(), NOW());
INSERT INTO packages (title, price, display_order, highlighted, photo_selection_limit, created_at, updated_at)
VALUES ('Origin', 1500000.00, 1, false, 10, NOW(), NOW());
INSERT INTO site_terms (title, introduction, content, version, created_at, updated_at)
VALUES ('Terms & Conditions', NULL, 'Isi syarat dan ketentuan.', '1', NOW(), NOW());
INSERT INTO bookings (name, email, phone, event_date, event_time, package_title, package_price,
                      event_location, message, terms_accepted, terms_accepted_at, terms_version,
                      status, confirmation_email_count, selection_token, selection_limit, raw_folder_id,
                      created_at, updated_at)
VALUES ('Klien Galeri', 'klien@example.test', '081234567890', CURRENT_DATE, '09:00:00',
        'Origin', 1500000.00, 'Bandung', 'Halo', TRUE, NOW(), '1', 'COMPLETED', 0,
        'aabbccddeeff00112233445566778899', 10, 'folderRAW123456', NOW(), NOW()),
       ('Klien Tanpa Folder', 'belum@example.test', '081234567893', CURRENT_DATE, '13:00:00',
        'Origin', 1500000.00, 'Bandung', 'Halo', TRUE, NOW(), '1', 'PENDING', 0,
        'bbccddeeff00112233445566778899aa', 10, NULL, NOW(), NOW());
SQL
ok 'fixture inserted (package with a 10-photo limit, one finished booking with a folder, one without)'

log '1. Stub Google Drive'
cat > "$WORK_DIR/stub.py" <<'PY'
import json, re, sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import unquote_plus

FOLDER = "folderRAW123456"
FILES = [
    {"id": "fileAAA111111", "name": "IMG_0001.jpg", "mimeType": "image/jpeg", "imageMediaMetadata": {"width": 6000, "height": 4000}},
    {"id": "fileBBB222222", "name": "IMG_0002.jpg", "mimeType": "image/jpeg", "imageMediaMetadata": {"width": 6000, "height": 4000}},
]
# folders maps a (parent id, name) pair to the id the stub handed out, so a
# second lookup answers with the same folder — that is what makes the "create
# only when missing" logic testable.
state = {
    "folders": {},
    "folders_created": [],
    "uploads": [],
    "copies": [],
    "codes": [],
    "renames": [],
    "trashed": [],
    "shared": [],
}
FIXED_IDS = {"webp": "folderWEBP12345", "foto terpilih": "folderEDIT12345",
             "foto edit": "folderFOTOEDIT12"}


def folder_name(query):
    """The name a Drive folder lookup is searching for."""
    match = re.search(r"name = '(.*?)'", query)
    return match.group(1) if match else ""

def png_64():
    """A valid 64x64 RGB PNG, so cwebp has real pixels to convert."""
    import struct, zlib

    def chunk(tag, payload):
        return (struct.pack(">I", len(payload)) + tag + payload +
                struct.pack(">I", zlib.crc32(tag + payload) & 0xffffffff))

    width = height = 64
    rows = []
    for y in range(height):
        row = bytearray([0])
        for x in range(width):
            row += bytes([(x * 4) % 256, (y * 4) % 256, 128])
        rows.append(bytes(row))
    return (b"\x89PNG\r\n\x1a\n" +
            chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)) +
            chunk(b"IDAT", zlib.compress(b"".join(rows))) +
            chunk(b"IEND", b""))

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def _json(self, payload, status=200):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_PATCH(self):
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length else b""
        body = json.loads(raw.decode() or "{}")
        file_id = unquote_plus(self.path).split("/")[4].split("?")[0]
        if body.get("trashed"):
            state["trashed"].append(file_id)
            return self._json({"id": file_id})
        for key, folder_id in list(state["folders"].items()):
            if folder_id == file_id:
                parent, old_name = key.split("/", 1)
                del state["folders"][key]
                state["folders"][parent + "/" + body.get("name", old_name)] = folder_id
                state["renames"].append(old_name + " -> " + body.get("name", ""))
                return self._json({"id": file_id})
        self._json({"error": "unknown file " + file_id}, 404)

    def do_POST(self):
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length else b""
        if self.path == "/token":
            form = raw.decode()
            if "grant_type=authorization_code" in form:
                state["codes"].append(form)
                return self._json({"access_token": "stub-token", "refresh_token": "stub-refresh-token", "expires_in": 3600})
            return self._json({"access_token": "stub-token", "expires_in": 3600})
        if "/copy" in self.path:
            # /drive/v3/files/<id>/copy?fields=id
            state["copies"].append(self.path.split("/")[4])
            return self._json({"id": "copied" + str(len(state["copies"]))})
        if self.path.startswith("/upload"):
            state["uploads"].append(len(raw))
            return self._json({"id": "webp" + str(len(state["uploads"]))})
        if self.path.startswith("/drive/v3/files"):
            query = unquote_plus(self.path)
            body = json.loads(raw.decode() or "{}")
            name = body.get("name", "")
            parents = body.get("parents") or []
            parent = parents[0] if parents else "root"
            # A permission share ends in /permissions. Recording the id is what
            # lets a test see the folder became readable by link.
            if "/permissions" in query:
                state["shared"].append(self.path.split("/")[4].split("?")[0])
                return self._json({"id": "permission"})
            key = parent + "/" + name
            if key in state["folders"]:
                return self._json({"id": state["folders"][key]})
            folder_id = FIXED_IDS.get(name, "folder" + str(len(state["folders"]) + 100))
            state["folders"][key] = folder_id
            state["folders_created"].append(key)
            return self._json({"id": folder_id})
        self._json({"error": "unexpected POST " + self.path}, 404)

    def do_GET(self):
        if self.path.startswith("/token"):
            return self._json({"access_token": "stub-token", "expires_in": 3600})
        if self.path.startswith("/state"):
            return self._json(state)
        if self.path.startswith("/seed/"):
            # /seed/<parent>/<id>/<name> puts a folder in the stub as if it had
            # been made before, which is how an older "mustedit" folder is
            # simulated.
            _, _, parent, folder_id, name = self.path.split("/", 4)
            state["folders"][unquote_plus(parent) + "/" + unquote_plus(name)] = folder_id
            return self._json({"ok": True})
        if "alt=media" in self.path:
            # A 1x1 PNG, enough for cwebp to convert.
            data = png_64()
            self.send_response(200)
            self.send_header("Content-Type", "image/png")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)
            return
        if "/permissions" in self.path:
            # The permission list: an already shared folder answers with its
            # anyone permission, so a second share is skipped.
            file_id = unquote_plus(self.path).split("/")[4].split("?")[0]
            if file_id in state["shared"]:
                return self._json({"permissions": [{"type": "anyone"}]})
            return self._json({"permissions": []})
        if self.path.startswith("/drive/v3/about"):
            return self._json({"user": {"emailAddress": "fotografer@elvove.test", "displayName": "Elvove"}})
        if self.path.startswith("/drive/v3/files/"):
            # One file or folder by id: what Folder(id) reads, and the bytes of
            # a photo, which a separate branch above already answered.
            file_id = unquote_plus(self.path).split("/")[4].split("?")[0]
            for key, folder_id in state["folders"].items():
                if folder_id == file_id:
                    parent, name = key.split("/", 1)
                    return self._json({"id": folder_id, "name": name, "parents": [parent]})
            if file_id == FOLDER:
                return self._json({"id": FOLDER, "name": "Foto Klien", "parents": ["root"]})
            self.send_response(404)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"error": {"message": "File not found."}}')
            return
        if self.path.startswith("/drive/v3/files"):
            # The Drive query arrives URL-encoded; decoding it keeps the folder
            # lookups from matching on a stray "%27".
            query = unquote_plus(self.path)
            # Creating a folder is a GET-then-POST: an existing name answers with
            # a hit, a missing one with an empty list.
            if "mimeType = 'application/vnd.google-apps.folder'" in query:
                name = folder_name(query)
                parent = FOLDER if FOLDER in query else "root"
                folder_id = state["folders"].get(parent + "/" + name)
                if folder_id:
                    return self._json({"files": [{"id": folder_id, "name": name}]})
                return self._json({"files": []})
            if FOLDER in query:
                return self._json({"files": FILES})
            return self._json({"files": []})
        self._json({"error": "unexpected GET " + self.path}, 404)

HTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
PY
python3 "$WORK_DIR/stub.py" "$DRIVE_PORT" &
STUB_PID=$!
sleep 1
ok "Drive stub listening on 127.0.0.1:$DRIVE_PORT"

log '2. Backend image with Drive pointed at the stub'
(cd "$REPO_ROOT" && docker build -q -t "$API_IMAGE" . >/dev/null)
ok "built $API_IMAGE (it carries cwebp)"
docker rm -f "$API_CONTAINER" >/dev/null 2>&1 || true
# --network host lets the container reach the Drive stub and the throwaway
# database on the host's loopback.
docker run -d --name "$API_CONTAINER" --network host \
  -e DB_URL="postgres://postgres:verifypass@127.0.0.1:55432/elvove_verify?sslmode=disable" \
  -e JWT_SECRET='verify-secret-at-least-32-bytes-long!!' \
  -e SMTP_ENCRYPTION_KEY='verify-smtp-key-at-least-32-bytes-long!' \
  -e GOOGLE_OAUTH_CLIENT_ID='stub-client' \
  -e GOOGLE_OAUTH_CLIENT_SECRET='stub-secret' \
  -e GOOGLE_OAUTH_REFRESH_TOKEN='stub-refresh' \
  -e DRIVE_TOKEN_URL="http://127.0.0.1:$DRIVE_PORT/token" \
  -e DRIVE_API_BASE="http://127.0.0.1:$DRIVE_PORT/drive/v3" \
  -e DRIVE_UPLOAD_BASE="http://127.0.0.1:$DRIVE_PORT/upload/drive/v3" \
  -e PORT="$API_PORT" -e UPLOAD_DIR=/tmp/uploads \
  -e MAIL_SITE_URL="http://localhost:$API_PORT" \
  "$API_IMAGE" >/dev/null
for _ in $(seq 1 30); do
  curl -sf "http://127.0.0.1:$API_PORT/health" >/dev/null 2>&1 && break
  sleep 1
done
BASE="http://127.0.0.1:$API_PORT"
if curl -sf "$BASE/health" >/dev/null; then
  ok "API listening on $BASE"
else
  bad 'API did not start'
  docker logs "$API_CONTAINER" 2>&1 | sed -n '1,40p'
  exit 1
fi
TOKEN=$(curl -s -X POST "$BASE/api/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"verifyadmin","password":"verify-password-123"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
[[ -n $TOKEN ]] && ok 'login returned a token' || bad 'login returned no token'

log '3. The page is closed until the raw folder is saved'
assert_json 'an unknown token answers 404' \
  "$(curl -s "$BASE/api/gallery/ffffffffffffffffffffffffffffffff")" \
  "d['status']==404"
assert_json 'a malformed token answers 404 without touching the database' \
  "$(curl -s "$BASE/api/gallery/nope")" \
  "d['status']==404"
assert_json 'a booking without a raw folder says the photos are not ready' \
  "$(curl -s "$BASE/api/gallery/bbccddeeff00112233445566778899aa")" \
  "d['status']==409 and 'belum siap' in d['message']"

log '4. The old raw-folder endpoint is gone'
# Folders are made automatically now; the paste-a-link endpoint was removed so
# nothing can reset a booking's folder tree and delivered link.
assert_json 'PATCH /raw-folder answers 404' \
  "$(curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"rawFolderUrl":"https://drive.google.com/drive/folders/folderRAW123456"}' \
     "$BASE/api/admin/bookings/1/raw-folder")" \
  "d['status']==404"
GALLERY_TOKEN=aabbccddeeff00112233445566778899

log '5. GET /api/gallery/{token}'
GALLERY=$(curl -s "$BASE/api/gallery/$GALLERY_TOKEN")
assert_json 'the page lists the photos and the package limit' "$GALLERY" \
  "d['limit']==10 and d['packageTitle']=='Origin' and len(d['photos'])==2 and d['submittedAt'] is None"
assert_json 'nothing is selected yet' "$GALLERY" \
  "all(p['selected'] is False for p in d['photos'])"

log '6. WebP preview is converted once and stored in Drive'
CODE=$(curl -s -o "$WORK_DIR/one.webp" -w '%{http_code}' "$BASE/api/gallery/$GALLERY_TOKEN/photos/fileAAA111111")
if [[ $CODE == 200 ]] && [[ -s $WORK_DIR/one.webp ]]; then
  ok 'the preview answers with bytes'
else
  bad "the preview answered $CODE"
  docker logs "$API_CONTAINER" 2>&1 | tail -10
fi
if [[ $(head -c 4 "$WORK_DIR/one.webp") == RIFF ]]; then ok 'the preview is a WebP file'; else bad 'the preview is not WebP'; fi
sleep 1
STATE=$(curl -s "http://127.0.0.1:$DRIVE_PORT/state")
assert_json 'the WebP was uploaded to the webp subfolder' "$STATE" \
  "len(d['uploads'])==1 and d['folders']['folderRAW123456/webp']=='folderWEBP12345'"
curl -s -o /dev/null "$BASE/api/gallery/$GALLERY_TOKEN/photos/fileAAA111111"
STATE=$(curl -s "http://127.0.0.1:$DRIVE_PORT/state")
assert_json 'a second request does not upload again' "$STATE" "len(d['uploads'])==1"
assert_json 'a photo id from another folder answers 404' \
  "$(curl -s "$BASE/api/gallery/$GALLERY_TOKEN/photos/fileZZZ999999")" \
  "d['status']==404"

log '7. PUT /selection enforces the package limit'
assert_json 'saving one photo answers with the count' \
  "$(curl -s -X PUT -H 'Content-Type: application/json' -d '{"fileIds":["fileAAA111111"]}' \
     "$BASE/api/gallery/$GALLERY_TOKEN/selection")" \
  "d['selectedCount']==1 and d['limit']==10"
assert_json 'an id that is not in the folder is refused' \
  "$(curl -s -X PUT -H 'Content-Type: application/json' -d '{"fileIds":["fileZZZ999999"]}' \
     "$BASE/api/gallery/$GALLERY_TOKEN/selection")" \
  "d['error']=='Validation Failed' and 'fileIds' in (d['fieldErrors'] or {})"
assert_json 'a malformed id is refused' \
  "$(curl -s -X PUT -H 'Content-Type: application/json' -d '{"fileIds":["../etc/passwd"]}' \
     "$BASE/api/gallery/$GALLERY_TOKEN/selection")" \
  "d['error']=='Validation Failed'"
ELEVEN=$(python3 -c 'import json;print(json.dumps({"fileIds":["fileAAA111111","fileBBB222222"]+["x%09d" % i for i in range(9)]}))')
assert_json 'more photos than the package allows is refused' \
  "$(curl -s -X PUT -H 'Content-Type: application/json' -d "$ELEVEN" \
     "$BASE/api/gallery/$GALLERY_TOKEN/selection")" \
  "d['error']=='Validation Failed' and 'Maksimal 10' in (d['fieldErrors'] or {}).get('fileIds','')"
assert_json 'the saved choice is reflected on the page' \
  "$(curl -s "$BASE/api/gallery/$GALLERY_TOKEN")" \
  "sum(1 for p in d['photos'] if p['selected'])==1"
ADMIN_BOOKING=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/bookings/1" \
  | python3 -c 'import sys,json;print(json.dumps(json.load(sys.stdin)))')
assert_json 'the admin detail counts the photos and the picks' "$ADMIN_BOOKING" \
  "d['photoCount']==2 and d['selectedCount']==1 and d['selectionLimit']==10"
ADMIN_LIST=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/bookings" \
  | python3 -c 'import sys,json;d=json.load(sys.stdin);rows=d["content"] if isinstance(d,dict) else d;print(json.dumps([r for r in rows if r["id"]==1][0]))')
assert_json 'the admin list carries no photo counts (they are detail-only)' "$ADMIN_LIST" \
  "'photoCount' not in d and 'selectedCount' not in d"

log '8. POST /submit copies the originals into foto terpilih'
assert_json 'submitting answers with the chosen count' \
  "$(curl -s -X POST "$BASE/api/gallery/$GALLERY_TOKEN/submit")" \
  "d['selectedCount']==1 and 'sudah kami terima' in d['message']"
STATE=$(curl -s "http://127.0.0.1:$DRIVE_PORT/state")
assert_json 'one original was copied into the foto terpilih folder' "$STATE" \
  "d['folders']['folderRAW123456/foto terpilih']=='folderEDIT12345' and d['copies']==['fileAAA111111']"
assert_json 'the page is locked afterwards' \
  "$(curl -s "$BASE/api/gallery/$GALLERY_TOKEN")" \
  "d['submittedAt'] is not None"
assert_json 'a second submit is refused' \
  "$(curl -s -X POST "$BASE/api/gallery/$GALLERY_TOKEN/submit")" \
  "d['status']==409 and 'sudah dikirim' in d['message']"
assert_json 'the choice can no longer be changed' \
  "$(curl -s -X PUT -H 'Content-Type: application/json' -d '{"fileIds":["fileBBB222222"]}' \
     "$BASE/api/gallery/$GALLERY_TOKEN/selection")" \
  "d['status']==409 and 'tidak dapat diubah' in d['message']"

log '9. The package quota is saved and snapshotted into a booking'
assert_json 'a package keeps the photo quota it is given' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"title":"Ascend","price":2500000,"photoSelectionLimit":30,"features":[]}' \
     "$BASE/api/admin/packages")" \
  "d['photoSelectionLimit']==30"
assert_json 'a quota above 1000 is refused' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"title":"Terlalu Banyak","price":1000,"photoSelectionLimit":1001,"features":[]}' \
     "$BASE/api/admin/packages")" \
  "d['error']=='Validation Failed' and 'photoSelectionLimit' in (d['fieldErrors'] or {})"
ASCEND_ID=$(curl -s "$BASE/api/public/packages" \
  | python3 -c 'import sys,json;print([p["id"] for p in json.load(sys.stdin) if p["title"]=="Ascend"][0])')
BOOKING=$(curl -s -X POST -H 'Content-Type: application/json' \
  -d "{\"name\":\"Klien Kuota\",\"email\":\"kuota@example.test\",\"phone\":\"081234567891\",\"eventDate\":\"2026-12-01\",\"eventTime\":\"10:00\",\"packageId\":$ASCEND_ID,\"termsAccepted\":true,\"termsVersion\":\"1\"}" \
  "$BASE/api/bookings")
# The token is a secret only the e-mail carries, so the public booking
# response must not include it; it is read from the row instead.
read -r SNAPSHOT BOOKING_TOKEN <<<"$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tA -F' ' -c \
  "SELECT selection_limit, selection_token FROM bookings WHERE email='kuota@example.test'")"
if [[ $SNAPSHOT == 30 ]]; then
  ok 'the booking snapshots the package quota (30)'
else
  bad "the booking snapshotted selection_limit=$SNAPSHOT, want 30"
fi
if [[ ${#BOOKING_TOKEN} -ge 32 ]] && [[ $BOOKING_TOKEN =~ ^[a-f0-9]+$ ]]; then
  ok 'a new booking gets its own selection token'
else
  bad "a new booking has no usable selection token (got: ${BOOKING_TOKEN:0:40})"
fi
if printf '%s' "$BOOKING" | grep -q "$BOOKING_TOKEN"; then
  bad 'the public booking response must not leak the selection token'
else
  ok 'the public booking response does not leak the token'
fi
# A booking taken while packages still had a quota of 0 keeps that 0 in its
# snapshot. Its page must not stay closed for good, so 0 falls back to the
# package's quota.
docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c "UPDATE bookings SET selection_limit=0 WHERE id=1" >/dev/null
assert_json 'a zero snapshot falls back to the package quota' \
  "$(curl -s "$BASE/api/gallery/$GALLERY_TOKEN")" \
  "d['limit']==10"
docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -q \
  -c "UPDATE bookings SET selection_limit=10 WHERE id=1" >/dev/null

log '10. Drive is connected from the admin panel'
assert_json 'the panel starts disconnected' \
  "$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/drive")" \
  "d['connected'] is False and d['tokenConfigured'] is False and d['encryptionReady'] is True"
assert_json 'connecting without a client id is refused' \
  "$(curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"clientId":"","clientSecret":"x"}' "$BASE/api/admin/drive")" \
  "d['error']=='Validation Failed' and 'clientId' in (d['fieldErrors'] or {})"
assert_json 'the OAuth client is stored and never returned' \
  "$(curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"clientId":"stub-client-id.apps.googleusercontent.com","clientSecret":"stub-client-secret"}' \
     "$BASE/api/admin/drive")" \
  "d['clientId'].startswith('stub-client-id') and d['secretConfigured'] is True and 'stub-client-secret' not in json.dumps(d)"
assert_json 'the consent address is built for the stored client' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/drive/connect")" \
  "'authUrl' in d and 'redirect_uri=' in d['authUrl'] and 'state=' in d['authUrl'] and 'drive%2Fcallback' in d['authUrl']"
# A state this server did not sign would let an attacker attach their own Drive
# account, so the callback has to refuse it.
FORGED=$(curl -s -o /dev/null -w '%{redirect_url}' "$BASE/api/admin/drive/callback?code=abc&state=forged")
if [[ $FORGED == *"drive=error"*"reason=state"* ]]; then
  ok 'a callback with a forged state is refused'
else
  bad "a forged state reached the callback (redirect: $FORGED)"
fi

AUTH_URL=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/drive/connect" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["authUrl"])')
STATE=$(python3 -c "import urllib.parse,sys;print(urllib.parse.parse_qs(urllib.parse.urlparse(sys.argv[1]).query)['state'][0])" "$AUTH_URL")
CONNECTED=$(curl -s -o /dev/null -w '%{redirect_url}' "$BASE/api/admin/drive/callback?code=stub-code&state=$STATE")
if [[ $CONNECTED == *"drive=connected"* ]]; then
  ok 'the callback stores the refresh token and returns to the panel'
else
  bad "the callback redirected to $CONNECTED"
fi
assert_json 'the panel now reports the connected account' \
  "$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/drive")" \
  "d['connected'] is True and d['accountEmail']=='fotografer@elvove.test' and d['tokenConfigured'] is True"
STORED_TOKEN=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "SELECT encrypted_refresh_token FROM drive_settings WHERE id=1")
if [[ -n $STORED_TOKEN && $STORED_TOKEN != *"stub-refresh-token"* ]]; then
  ok 'the refresh token is stored encrypted, not in clear'
else
  bad "the refresh token looks stored in clear: $STORED_TOKEN"
fi

log '11. The root folder and the folder per booking'
assert_json 'a link that is not a Drive folder is refused as the root' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"rootFolderUrl":"bukan link"}' "$BASE/api/admin/drive/root")" \
  "d['error']=='Validation Failed' and 'rootFolderUrl' in (d['fieldErrors'] or {})"
assert_json 'the root folder is read from Drive and stored' \
  "$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"rootFolderUrl":"https://drive.google.com/drive/folders/folderRAW123456"}' \
     "$BASE/api/admin/drive/root")" \
  "d['rootFolderId']=='folderRAW123456' and d['rootFolderName']=='Foto Klien'"
BACKFILL=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/drive/backfill")
assert_json 'bookings made before Drive was connected get a folder' "$BACKFILL" \
  "d['created']>=2 and d['failed']==0"
BOOKING_FOLDERS=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tA -F' ' -c \
  "SELECT COALESCE(raw_folder_id,'-'), COALESCE(folder_webp_id,'-'), COALESCE(folder_mustedit_id,'-'), COALESCE(folder_edit_id,'-') FROM bookings WHERE id=1")
read -r FOLDER_ID WEBP_FOLDER MUSTEDIT_FOLDER EDIT_FOLDER <<<"$BOOKING_FOLDERS"
if [[ $FOLDER_ID != "-" && $WEBP_FOLDER == folderWEBP12345 && $MUSTEDIT_FOLDER == folderEDIT12345 && $EDIT_FOLDER == folderFOTOEDIT12 ]]; then
  ok 'the booking folder holds the webp, foto terpilih and foto edit subfolders'
else
  bad "booking 1 folders = $BOOKING_FOLDERS"
fi
# Both the booking folder and the foto edit folder have to be readable by link:
# the admin opens the first from their own account, the client the second.
assert_json 'the booking folder and the foto edit folder are shared as viewer' \
  "$(curl -s "http://127.0.0.1:$DRIVE_PORT/state")" \
  "d['shared'].count('$FOLDER_ID')==1 and d['shared'].count('folderFOTOEDIT12')==1"
# A folder the admin pasted by hand before this feature existed is the booking's
# folder, so the backfill must not leave a second, empty one beside it.
if [[ $FOLDER_ID == folderRAW123456 ]]; then
  ok 'a booking with a pasted folder keeps it instead of getting a new one'
else
  bad "booking 1 client folder = $FOLDER_ID, expected folderRAW123456"
fi

log '12. A new booking gets its folder without pasting a link'
NEW=$(curl -s -X POST -H 'Content-Type: application/json' \
  -d "{\"name\":\"Klien Otomatis\",\"email\":\"otomatis@example.test\",\"phone\":\"081234567892\",\"eventDate\":\"2026-12-02\",\"eventTime\":\"11:00\",\"packageId\":$ASCEND_ID,\"termsAccepted\":true,\"termsVersion\":\"1\"}" \
  "$BASE/api/bookings")
AUTO_ID=$(printf '%s' "$NEW" | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
for _ in $(seq 1 20); do
  AUTO_ROW=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tA -F' ' -c \
    "SELECT COALESCE(raw_folder_id,''), COALESCE(folder_webp_id,''), COALESCE(folder_mustedit_id,''), COALESCE(folder_edit_id,'') FROM bookings WHERE id=$AUTO_ID")
  [[ -n ${AUTO_ROW%% *} ]] && break
  sleep 0.5
done
read -r AUTO_FOLDER AUTO_WEBP AUTO_MUSTEDIT AUTO_EDIT <<<"$AUTO_ROW"
if [[ -n $AUTO_FOLDER && -n $AUTO_WEBP && -n $AUTO_MUSTEDIT && -n $AUTO_EDIT ]]; then
  ok 'the new booking has its client, webp, foto terpilih and foto edit folders'
else
  bad "automatic folders for booking $AUTO_ID = $AUTO_ROW"
fi
AUTO_TOKEN=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "SELECT selection_token FROM bookings WHERE id=$AUTO_ID")
# A booking made before the rename still has a "mustedit" folder; touching it
# again must rename that folder instead of creating a second one, and the
# invoice link must point at it without anybody pasting anything.
curl -s "http://127.0.0.1:$DRIVE_PORT/seed/$FOLDER_ID/folderOLD1234567/mustedit" >/dev/null
docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -q -c \
  "UPDATE bookings SET folder_mustedit_id='folderOLD1234567', delivery_url='https://gallery.example.test/album-lama' WHERE id=1" >/dev/null
curl -s -o /dev/null -X POST -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/bookings/1/drive-folder"
assert_json 'an older mustedit folder is renamed, not duplicated' \
  "$(curl -s "http://127.0.0.1:$DRIVE_PORT/state")" \
  "'mustedit -> foto terpilih' in d['renames'] and d['folders']['$FOLDER_ID/foto terpilih']=='folderOLD1234567'"
# Checking the folders again must not replace a link the admin already gave out.
DELIVERY=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "SELECT COALESCE(delivery_url,'') FROM bookings WHERE id=1")
if [[ $DELIVERY == "https://gallery.example.test/album-lama" ]]; then
  ok 'an invoice link that is already set is kept'
else
  bad "delivery_url = $DELIVERY"
fi
docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -q -c \
  "UPDATE bookings SET delivery_url=NULL WHERE id=1" >/dev/null
curl -s -o /dev/null -X POST -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/bookings/1/drive-folder"
DELIVERY=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "SELECT COALESCE(delivery_url,'') FROM bookings WHERE id=1")
if [[ $DELIVERY == "https://drive.google.com/drive/folders/folderFOTOEDIT12" ]]; then
  ok 'an empty invoice link is filled with the foto edit folder'
else
  bad "delivery_url = $DELIVERY"
fi
# A foto edit folder deleted from Drive has to give a new link: keeping the old
# one would send the client to a folder that is gone.
docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -q -c \
  "UPDATE bookings SET folder_edit_id='folderGONE1234567', delivery_url='https://drive.google.com/drive/folders/folderGONE1234567' WHERE id=1" >/dev/null
curl -s -o /dev/null -X POST -H "Authorization: Bearer $TOKEN" "$BASE/api/admin/bookings/1/drive-folder"
DELIVERY=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "SELECT COALESCE(delivery_url,'') FROM bookings WHERE id=1")
if [[ $DELIVERY == "https://drive.google.com/drive/folders/folderFOTOEDIT12" ]]; then
  ok 'a recreated foto edit folder replaces the dead invoice link'
else
  bad "delivery_url = $DELIVERY"
fi

# The folder is named after the session, so Drive sorts by date.
FOLDER_NAME=$(curl -s "http://127.0.0.1:$DRIVE_PORT/state" \
  | python3 -c 'import sys,json;print([k for k in json.load(sys.stdin)["folders"] if "Klien Otomatis" in k][0].split("/",1)[1])')
if [[ $FOLDER_NAME == "2026-12-02 - Klien Otomatis (Ascend) #$AUTO_ID" ]]; then
  ok "the automatic folder is named \"$FOLDER_NAME\""
else
  bad "unexpected folder name: $FOLDER_NAME"
fi
# A brand-new booking has no photos yet; what matters is that its page is open
# without an admin having pasted anything.
assert_json 'its client page opens without anybody pasting a link' \
  "$(curl -s "$BASE/api/gallery/$AUTO_TOKEN")" \
  "d['limit']==30 and d['photos']==[] and d['clientName']=='Klien Otomatis'"

log '13. Deleting a booking moves its Drive folder to the trash'
AUTO_FOLDER_ID=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "SELECT raw_folder_id FROM bookings WHERE id=$AUTO_ID")
DELETED=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H "Authorization: Bearer $TOKEN" \
  "$BASE/api/admin/bookings/$AUTO_ID")
if [[ $DELETED == "204" ]]; then
  ok 'the booking is deleted'
else
  bad "DELETE answered $DELETED"
fi
assert_json 'its folder is trashed, not left behind' \
  "$(curl -s "http://127.0.0.1:$DRIVE_PORT/state")" \
  "d['trashed']==['$AUTO_FOLDER_ID']"
ROWS=$(docker exec "$DB_CONTAINER" psql -U postgres -d elvove_verify -tAc \
  "SELECT count(*) FROM bookings WHERE id=$AUTO_ID")
if [[ $ROWS == "0" ]]; then
  ok 'no row is left for the deleted booking'
else
  bad "the deleted booking still has $ROWS row(s)"
fi

printf '\n\033[1mResult: %d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
