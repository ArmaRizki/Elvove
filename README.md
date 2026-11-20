# ELVOVE — Photography Website

Full-stack photography site: a public portfolio with packages and a validated
booking form, plus a JWT-protected admin dashboard for every piece of content.

This project started from
[Lens & Light Photography](https://github.com/Safdar7262/Lens-Light-Photography)
(React 19 + Spring Boot 3 + PostgreSQL) and has since been reworked into ELVOVE.
This README records what changed.

**Live:** https://elvove.site

---

## What changed at a glance

| Area | Lens & Light | ELVOVE |
|---|---|---|
| Backend | Spring Boot 3.3.4 · Java 17 · JPA/Hibernate | Go 1.23+ · `net/http` · pgx v5 — Java kept as reference |
| Content modules | Galleries, Services, Testimonials, Videos | Removed; replaced by Terms, SMTP and booking-notification settings |
| Public pages | 14 | 11 (new: photo selection) |
| Admin sections | 13 | 11 (new: calendar, Terms, SMTP; 5 modules gone) |
| Booking flow | `PENDING → CONTACTED → COMPLETED` | `PENDING → ACCEPTED → COMPLETED`, with `RESCHEDULE_REQUESTED` only while the date is not yet agreed |
| E-mail | none | SMTP stored in DB, confirmation, invoice, settlement, reschedule and admin notifications |
| Photo selection | none | `/pilih/<token>` per booking: WebP previews from the raw Drive folder, a per-package quota, the picks copied into `foto terpilih` |
| Uploads | 10 MB, extension check only | 50/100 MB, magic-byte check, WebP thumbnail + preview, original admin-only |
| First admin | hard-coded `admin` / `admin123` | env bootstrap, only while the table is empty, with Java's password rules |
| JWT secret | hard-coded in `application.yml` | environment variable, at least 32 bytes |
| Tests | none | 22 Java test files + 9 Go test files |
| Deployment | single `Dockerfile` for Render | full Compose stack: Postgres + Go API + static frontend + nginx |

---

## 1. Backend rewritten in Go

`backend-go/` is a complete replacement for the Spring Boot API, written against
the same database and the same JSON contract:

- `net/http` routing, pgx v5 connection pool, no ORM.
- Same endpoints, field names, status codes and error bodies (`timestamp`,
  `status`, `error`, `message`, `fieldErrors`).
- JWT HS256/384/512 verification, bcrypt hashes reused from the existing table.
- Rate limiting moved to a sliding window per key, as the Java service did.
- Gallery routes are limited per selection token (`30`/min and `300`/hour for the
  listing, `600`/min and `5000`/hour for previews, `60`/min for a save or a
  submit). The token is the only key to a booking's folder, and the caller's
  address cannot tell visitors apart behind the tunnel, where everyone shares
  one address.
- All timestamps use `Asia/Jakarta` wall clock, matching the old JVM's
  `LocalDateTime.now()`; the pool opens every connection with
  `timezone=Asia/Jakarta`.
- E-mail templates are a direct port: table-based HTML, hidden preheader,
  plain-text alternative, RFC 2047 subject, `Reply-To` set to the sender.

The Spring Boot implementation was deleted once the Go port reached feature
parity; `backend-go/README.md` lists the differences it had to preserve.

---

## 2. Removed

Five content modules were dropped from both the API and the UI, along with their
tables, controllers, services, repositories and admin screens:

- **Galleries** — curated photo sets
- **Services** — service list and pricing copy
- **Testimonials**
- **Videos / Films** — dropped with `db/migrations/2026-11-09-drop-videos.sql`
- **FAQs** — page, editor, API routes, table, sequence and every reference
  removed; the schema dump no longer contains `faqs`.

Two stray files from the original repo were removed: `Music - Shortcut.lnk` and
the unused `hero.png` / `vite.svg` assets.

Public routes went from `/`, about, portfolio, galleries, services, packages,
videos, testimonials, latest, booking, faq, contact, privacy, terms to:
`/`, about, portfolio, packages, latest, booking, contact, privacy, terms,
terms-and-conditions.

---

## 3. Added

### Booking workflow

- Statuses are `PENDING`, `ACCEPTED`, `RESCHEDULE_REQUESTED`, `COMPLETED`, with
  an explicit transition allow-list. `COMPLETED` is terminal.
- The schedule is locked once the customer has agreed to it. A reschedule can
  only start from `PENDING` or `RESCHEDULE_REQUESTED`; `ACCEPTED` moves on to
  `COMPLETED` only, and `COMPLETED` does not move at all.
- Accepting a booking only changes the status; the confirmation e-mail is a
  separate, explicit action so it can be retried. It carries a payment card:
  the 50% down payment worked out from the package price, the BCA account, and
  the static QRIS code at `/qris.png`.
- Payment state is tracked on the booking: `UNPAID`, `DOWN_PAYMENT_PAID`
  (`down_payment_paid_at`) or `PAID` (`down_payment_paid_at` +
  `settlement_paid_at`). The state is derived on read, and marking a booking
  `PAID` also records the down payment, so a booking can never be lunas without
  a down-payment date.
- Three further customer e-mails are manual, like the confirmation, and follow
  the order of the booking:
  - **Pilihan foto** (selection notice) — the link to the client page where the
    photos to edit are chosen, sent first, once the raw photos are uploaded. It
    is refused until the booking is accepted and has a raw folder; a package
    whose quota is still zero gets the **FOTO SESI ANDA** wording instead.
  - **Pelunasan** (settlement notice) — the remaining half, the BCA account, the
    QRIS code and the steps to settle it, due the day before the session. It
    carries no photo link: the choice has its own message, so this one only asks
    for money. It is refused until the down payment is recorded (the message
    states the DP has arrived) and for a booking already marked lunas.
  - **Invoice** — the last message of a booking: the transaction as it was paid
    (total, DP, pelunasan) plus the Google Drive link to the edited photos. It
    is numbered `INV-<year>-<id padded to four digits>` (e.g. `INV-2026-0012`),
    derived from the booking id and the current year, so it needs no counter
    table. It is refused until the booking is `COMPLETED`, marked lunas and has
    a `delivery_url`; it never asks for payment.
- The Drive link is stored on the booking (`delivery_url`, `PATCH
  /api/admin/bookings/{id}/delivery`) and validated as an http(s) URL, because
  the invoice renders it as a button.
- Confirmation, selection, invoice, settlement and reschedule e-mails are tracked
  (`confirmation_email_sent_at` / `_count`, `selection_email_sent_at` / `_count`,
  `invoice_email_sent_at` / `_count`, `settlement_email_sent_at` / `_count`) and
  written only after SMTP accepts the message, so a failure leaves the row
  retryable.
- Rescheduling stores the new date/time and sends a notification to the client.
- The customer e-mails end with a WhatsApp button pointing at the number in
  the site's contact settings (the same one the floating button uses), so a
  reply goes to one chat instead of an inbox nobody watches.
- New bookings notify the configured administrators, after the transaction
  commits, without delaying the customer or rolling back the booking. A failed
  send is retried three times, and the outcome (sent count, skip or failure) is
  logged with the booking id.
- Admin calendar view and a custom time picker for schedule changes.
- The booking form and the admin reschedule panel use their own date and time
  pickers (not the native controls), so a date always reads `dd/mm/yyyy`
  whatever the browser locale is. They still hand the API a `YYYY-MM-DD` date
  and an `HH:mm` time.

### Raw photo selection

- Every booking gets its own Google Drive folder, created in the background as
  soon as the booking is saved: `<event_date> - <client> (<package>) #<id>`,
  holding empty `webp`, `foto terpilih` and `foto edit` subfolders. Nobody pastes
  a Drive link: the folder ids are stored on
  the booking (`raw_folder_id`, `folder_webp_id`, `folder_mustedit_id` — which
  points at `foto terpilih` — and `folder_edit_id`), and the client page
  reads them from there. The booking folder is shared as **viewer with anyone
  holding the link**, and its subfolders inherit that: the admin opens it from
  their own browser, signed in to their own Google account rather than the one
  the photos belong to, so a private folder would answer "you need access".
  `foto edit` is shared on its own as well, because that is the link the client
  opens from the invoice. A folder link is therefore readable by whoever holds
  it, which is why the folder is never linked from the public site: it reaches
  the client only through the e-mail.
- Deleting a booking moves its Drive folder to the **trash**, contents included,
  rather than destroying it: Google keeps a trashed folder for 30 days, so a
  booking removed by mistake does not take the client's photos with it. The
  cached WebP previews for that booking are removed from disk. A Drive that is
  not connected, or a folder already gone, only leaves a line in the log.
- A booking that arrives while Drive is not connected is still saved; the folder
  is created later from `/admin/drive` (**Buat Folder yang Belum Ada**, which
  covers every booking without one) or from the booking detail panel. Creation
  is idempotent: an existing folder is reused, and one deleted in Drive is made
  again.
- Each booking carries a `selection_token` (32 random hex characters, created
  with the booking and unique per booking). The client page lives at
  `/pilih/<token>`; the token is the only credential, so there is nothing to
  guess and the page never asks for an e-mail or phone number.
- The selection link travels in its own **Pilihan Foto** notice, sent before the
  settlement one, so the client picks the photos while they are fresh and the
  settlement e-mail only asks for money. The address follows `MAIL_SITE_URL`, so
  a local run links to `http://localhost:8090/pilih/<token>` and production to
  its own host. The token is the only thing that can be missing: a booking taken
  before links existed has one minted when its folder is set up or when the
  notice is sent, so the link is never silently dropped. A package whose quota is
  zero gets a **FOTO SESI ANDA** panel that only shows the photos, so the message
  never promises a choice the page would refuse. Its payment deadline adapts: `paling lambat
  <H-1>` before the session, `secepatnya` once the session has passed, so it
  never states a past date.
- The page shows a grid of previews. Each preview is converted from the original
  to WebP (1600 px, q78) by `cwebp`, uploaded to a `webp` subfolder of the raw
  folder, and cached on disk under `UPLOAD_DIR/gallery/<token>/`, so Drive is
  read once per photo. The grid and the full-screen viewer show the thumbnail
  Drive already keeps for each photo (`drive.google.com/thumbnail?id=<id>`), so
  a folder of three hundred photos fills in at once and the server converts
  nothing; this endpoint is the fallback when a thumbnail fails, one request at
  a time. When the local cache is wiped, the smaller Drive `webp` copy refills
  it. Two conversions run at a time — a page asking for dozens at once used to
  exhaust memory and leave broken images.
- A tile is a button: tapping it picks or unpicks the photo. The expand icon on
  its corner opens the same preview full screen, uncropped, with the name, the
  position in the folder, arrow keys / buttons for the next photo and Escape to
  close. Viewing works even when picking is closed (no quota, or already
  submitted); the tile's own click is what picks.
- The quota per package is `packages.photo_selection_limit`, edited in
  `/admin/packages` (0–1000; e.g. Origin 10, Ascend 30). It is copied to
  `bookings.selection_limit` at booking time — a snapshot, like
  `package_title` and `package_price`, so later price/quota edits do not change
  an existing booking. A booking whose snapshot is `0` falls back to its
  package's quota, so a booking taken while every package still offered none
  does not stay closed for good.
- The invoice e-mail's link to the edited photos is not typed in: it is the
  booking's `foto edit` folder (`bookings.delivery_url`), filled in when the
  folders are ensured and kept afterwards, so "Periksa" and the backfill never
  overwrite a link that is already there. It is rewritten only when the old
  folder tree is gone from Drive and had to be rebuilt. The panel shows it
  read-only. The picked-photos subfolder used to be called `mustedit`; a folder
  still carrying that name is renamed the next time its booking is opened, so
  Drive never holds two.
- Picking saves as the client clicks (`PUT /api/gallery/{token}/selection`); the
  server enforces the quota and refuses ids that are not in the folder.
- `POST /api/gallery/{token}/submit` finishes the choice: it creates the
  `foto terpilih` folder next to the raw photos and **copies** the chosen originals
  into it at full resolution (not the WebP preview). The raw folder is never
  moved or deleted. A retry skips names already copied, so a partial failure
  does not duplicate files. The selection is then locked
  (`selection_submitted_at`), and further saves or submits answer `409`.
- The Drive connection is set up from `/admin/drive` (sidebar **Google Drive**),
  not from `.env`: an OAuth client of type *Web application* is pasted there, then
  **Hubungkan Akun Google** runs the consent in a new tab. The callback is
  `GET /api/admin/drive/callback`; it sits outside the admin JWT because Google
  redirects a browser to it, and it is protected by a signed `state`
  (timestamp + HMAC of `JWT_SECRET`, valid ten minutes) so nobody can attach a
  Drive account of their own.
- `drive_settings` holds one row: client id, client secret and refresh token
  (AES-GCM with `SMTP_ENCRYPTION_KEY`), the parent folder and the connected
  account. Secrets are never returned to the browser. All client folders are
  created inside that one parent — made in My Drive by **Buat Folder Induk**
  (default `ELVOVE Foto`) or set from a pasted link.
- `GOOGLE_OAUTH_*` in `.env` still works as a fallback for a deployment that was
  configured that way before the panel existed; the saved connection wins.
  `go run ./cmd/driveauth` remains an alternative way to obtain a refresh token.
- Without a connection the gallery endpoints answer `503` and the page says the
  feature is unavailable; the rest of the site is unaffected.
- Drive access uses OAuth on the account that owns the photos: a service
  account has no storage quota of its own, so it can read a shared folder but
  cannot upload into it. The OAuth consent screen must be published: while the
  app is in *Testing*, Google expires refresh tokens after seven days.

### Terms & Conditions

- `site_terms` holds the document and an incrementing `version`.
- A booking records `terms_accepted`, `terms_accepted_at` and `terms_version`.
- Saving different content bumps the version; if the version the client accepted
  is stale, the API answers `409` and asks them to review the new text.
- Admin rich-text editor for both the About story and the terms themselves
  (headings, bold/italic/underline, font size, alignment, lists, quotes, links).
  The text is stored as HTML and sanitized with DOMPurify on the public page;
  older Markdown rows are converted when the editor loads them.
- The column layout of `/terms-and-conditions` is set in `/admin/terms` and
  stored in `site_terms.layout_columns` (1 or 2), `layout_width`
  (`equal`, `wide-left`, `wide-right`) and `layout_split`: `heading` starts a
  new column at each Heading 1, `marker` only at a "Pisah kolom" marker the
  editor inserts as `<hr>` (the marker itself is not printed). Columns stack on
  phones. A layout change does not bump `version`, so accepted bookings are not
  invalidated by it.
- `about_info.story` became `text` in
  `db/migrations/2026-11-15-about-story-text.sql`: the HTML markup of the
  same prose already exceeded the old varchar(3000). The terms layout columns
  come from `db/migrations/2026-11-15-terms-layout.sql`.

### E-mail settings

- SMTP host, port, credentials, encryption mode, sender name and address live in
  `smtp_settings` and are edited from the admin panel — not environment
  variables.
- The stored password is AES-GCM encrypted with `SMTP_ENCRYPTION_KEY`; the
  envelope is compatible with the Java implementation, so an existing password
  keeps working after the migration.
- Notification recipients (`booking_notification_settings` +
  `booking_notification_recipients`): up to 20 addresses, case-insensitively
  de-duplicated, with an on/off switch.

### Images

- Uploads produce a WebP thumbnail (800 px, q55) and preview, and the original is
  served only to authenticated admins (`/uploads/originals/**` → `401`).
- Photos that predate the variants get them generated lazily on first read.
- File type is verified from the magic bytes, not just the extension.
- Deleting a photo removes its original and both variants.

---

## 4. Security changes

| Item | Before | After |
|---|---|---|
| JWT secret | literal in `application.yml` | `JWT_SECRET`, ≥ 32 bytes, startup fails otherwise |
| Database password | default `safdar` | required `DB_PASSWORD` |
| First admin | `admin` / `admin123` at startup | `ADMIN_*` env, only when `admins` is empty, min 12 chars, blocked common passwords, valid e-mail |
| SQL logging | `show-sql: true` | off by default (`JPA_SHOW_SQL` to enable) |
| CORS | one origin, default `localhost:5173` | comma-separated `FRONTEND_URL`, explicit allow-list |
| Login | no throttling | 20 attempts/15 min per IP, 5 failures/15 min per username |
| Booking form | 5/10 min | 3/min, 10/hour and 20/day per address; a refused request records nothing |
| Uploads | extension in a 5-item list | magic-byte signature, `.gif` dropped, path-traversal guard |
| Originals | publicly served | admin-only |
| Reverse proxy | none | nginx with `server_tokens off`, security headers and a login rate-limit zone |

Passwords stay bcrypt-hashed and the JWT is still stateless with no server-side
session.

---

## 5. Database

15 tables:

`admins`, `categories`, `photos`, `packages`, `package_features`, `bookings`,
`contact_info`, `about_info`, `social_links`, `site_terms`,
`smtp_settings`, `booking_notification_settings`,
`booking_notification_recipients`, `booking_photos`, `drive_settings`.

New since Lens & Light: `site_terms`, `smtp_settings`, the two notification
tables, plus `booking` columns for terms acceptance, confirmation-email
tracking, the two recorded payment points, the Drive link to the edited photos
and the invoice / settlement delivery logs
(`db/migrations/2026-11-16-booking-payment-tracking.sql`).
`booking_photos` (a cache of what a booking's Drive folder holds, with the WebP
file id and the selection), `packages.photo_selection_limit` and the booking
selection columns (`selection_token`, `selection_limit`, `raw_folder_id`,
`selection_folder_id`, `selection_submitted_at`) arrive in
`db/migrations/2026-11-16-photo-selection.sql`.
`selection_email_sent_at` / `_count`, the tracking of the selection notice, arrive in
`db/migrations/2026-11-16-selection-email.sql`.
`drive_settings` (the Google Drive connection, the parent folder and the
connected account) and three folder columns on `bookings`
(`raw_folder_id`, `folder_webp_id`, `folder_mustedit_id`) arrive in
`db/migrations/2026-11-16-drive-settings.sql`; apply it after the photo-selection
one, since both touch `bookings`. `folder_edit_id`, the folder the invoice links
to, arrives in `db/migrations/2026-11-16-foto-edit-folder.sql`.
`db/migrations/2026-11-17-drop-duplicate-folder-columns.sql` then copies the old
`client_folder_id` / `selection_folder_id` values into `raw_folder_id` /
`folder_mustedit_id` and drops those two columns. Apply it **last**: it reads
columns the earlier migrations create.
`galleries`, `services`, `testimonials`, `videos` and `faqs` no longer exist.

Schema changes are handwritten SQL in `db/migrations/`, applied by hand in
file-name order (a later date may depend on a column an earlier one adds).
The Go backend never creates or alters tables.

---

## 6. Frontend

- Rebranded from Lens & Light to ELVOVE: logo, canonical URL, and a light theme
  with a crimson accent (`#E11D48`) replacing the dark ink/brass palette.
- `lucide-react` upgraded from `0.383.0` to `^1.70.0`; `react-icons` added.
- New admin screens: calendar, Terms editor, SMTP settings, Google Drive
  (connect the account, pick the parent folder, backfill old bookings), plus a
  raw-photo panel per booking (its automatic Drive folder with a retry button,
  the client's selection link, progress) and the selection quota field in
  `/admin/packages`.
- `/pilih/:token` is the client page for picking the photos to edit. It is a
  public route outside the admin layout, and it reads nothing without the token.
- Shared `AdminResourceManager` for the simple CRUD lists, plus `AdminLayout`,
  `EmptyState`, `LoadingState`, `CustomTimePicker` and `WhatsAppButton`.
- `utils/bookingTime.js`, `utils/richText.js` and `utils/uploadLimits.js` keep
  time handling, rich-text HTML (Markdown upgrade + sanitizing) and the
  upload-size hints in one place.

---

## 7. Deployment

`compose.yml` runs the whole stack locally — Postgres, the Go API, Vite, nginx —
and a production variant in `/opt/compose` runs Postgres, the Go image, a static
nginx frontend and the proxy, with the database and uploads on named/bind mounts.

- `backend-go/Dockerfile`: multi-stage build, static binary, non-root uid 10001,
  `libwebp-tools` for the thumbnail conversion.
- `nginx/default.conf`: proxies `/api` and `/uploads`, serves the SPA, and
  applies rate limiting plus security headers. It streams gallery previews
  instead of buffering them, and `nginx/nginx.conf` (mounted by `compose.yml`)
  raises `worker_connections` from the image's 1024 to 16384, with the
  container's file descriptor limit raised to match: with the stock values, a
  folder page of photos exhausted the worker and answered 503.
- `frontend/nginx.prod.conf` (`frontend/Dockerfile.prod`): the frontend image.
  It lists the addresses the router knows and answers 404 for the rest, still
  serving `index.html` as the body so the "halaman tidak ada" screen appears —
  a wrong address used to be a 200, and its relative asset URLs resolved under
  the wrong folder. Keep the list in step with the routes in `App.jsx`.
- `backend-go/scripts/`: `run-web.sh` (the stack behind nginx),
  `set-dev-admin.sh` (admin password on the throwaway copy database),
  `verify-staging.sh` (end-to-end checks) and `verify-gallery.sh` (the photo
  selection endpoints against a stubbed Google Drive).
- `scripts/deploy.sh frontend|backend|nginx`: builds one part, puts it on the
  server, points `compose.yml` at it, then checks it — the frontend answers 200
  for the addresses the router knows and 404 for one it does not, the backend
  proves the gallery limiter refuses, nginx proves its container got the
  `worker_connections` and file descriptor limit. Migrations and `.env` stay
  manual on purpose.
- Backups, both on cron: `backend-go/scripts/backup-db.sh` (03:15, seven days)
  and `backend-go/scripts/backup-uploads.sh` (03:45, three days). The uploads
  script warns when its copy lands on the same filesystem as the images, which
  is the usual case: a copy on the failing disk survives a deleted file, not a
  dead card. Point `DEST` at another disk for a copy that does. Both verify the
  archive before keeping it, and both restores were run: the database row counts
  and the image checksums matched.

### HSTS

`Strict-Transport-Security` is set in Cloudflare, not here. The tunnel terminates
TLS at Cloudflare's edge, so a header added in `nginx/default.conf` would sit
behind the copy that actually reaches the browser, and the two could disagree.

Two switches in the Cloudflare dashboard, in this order:

1. **Always Use HTTPS** — SSL/TLS → Edge Certificates. Check it with
   `curl -o /dev/null -w '%{http_code} %{redirect_url}\n' http://elvove.site/`;
   it should answer `301 https://elvove.site/`. HSTS assumes the visitor already
   arrived over HTTPS once; this is what guarantees it. **On since 2026-11-20**
   (before that, `http://` answered 200 and the cleartext site was reachable).
2. **HTTP Strict Transport Security** — same page, *Enable HSTS*, max-age
   **15552000** (180 days), *include subdomains* and *preload* **off**.
   **On since 2026-11-20.** `include subdomains` stays off because `www` is not
   in DNS: with it on, any subdomain that does not serve HTTPS becomes
   unreachable, not merely unencrypted. `preload` stays off because the browser
   list is cached for months and a site that stops serving HTTPS becomes
   unreachable rather than merely insecure.

Because the header is now being sent, **the certificate must not lapse**: the
edge certificate is a Let's Encrypt one that Cloudflare renews, valid
2026-11-06 → 2027-02-04. If it ever stops renewing, visitors who already have
the header cannot reach the site at all — and they cannot click through a
warning, because HSTS removes that option. Check the date rather than trusting
renewal:

```sh
echo | openssl s_client -servername elvove.site -connect elvove.site:443 2>/dev/null \
  | openssl x509 -noout -dates
```

**Do not turn off the proxy or pause Cloudflare** while the header is live; both
make the site unreachable for existing visitors for up to 180 days.

`scripts/check-hsts.sh` reports which step is still missing, and what the header
actually says:

```sh
./scripts/check-hsts.sh              # defaults to https://elvove.site
```

It exits non-zero until both are done, and distinguishes a missing header from a
header that is present but too short to be useful.

Backend environment: `DB_*` or `DB_URL`, `JWT_SECRET`, `SMTP_ENCRYPTION_KEY`,
`FRONTEND_URL`, `MAIL_SITE_URL`, `MAIL_ADMIN_URL`, `ADMIN_*`,
`UPLOAD_MAX_FILE_SIZE`, `UPLOAD_MAX_REQUEST_SIZE`, `PORT`, `UPLOAD_DIR`.
`GOOGLE_OAUTH_CLIENT_ID` / `GOOGLE_OAUTH_CLIENT_SECRET` /
`GOOGLE_OAUTH_REFRESH_TOKEN` are only a fallback: the Drive connection is set up
in `/admin/drive` and stored in the database.

---

## 8. Structure

```
backend-go/          Go API (the running backend)
  cmd/api/           entrypoint
  internal/config/   environment parsing
  internal/httpapi/  routes, handlers, auth, validation, e-mail
  internal/store/    pgx pool and query helpers
  cmd/driveauth/     alternative one-off OAuth consent, prints a refresh token
  internal/drive/    Google Drive REST client
  scripts/           run-web.sh, set-dev-admin.sh, verify-staging.sh,
                     verify-gallery.sh
frontend/            React 19 + Vite + Tailwind
  src/pages/         public pages
  src/pages/admin/   admin screens
  src/services/      axios clients and API maps
nginx/               reverse proxy configuration
scripts/             deploy.sh (build + ship + verify), check-hsts.sh
compose.yml          local stack
```

---

## 9. Tests

- Go test files covering auth, the limiter, booking validation and dates,
  e-mail rendering (invoice, settlement, the adaptive payment deadline),
  SMTP encryption, the store and the Drive client.
- `backend-go/scripts/verify-staging.sh` drives a real stack end to end with 55
  assertions (status workflow, WIB timestamps, SMTP failure rollback, upload
  limits, CORS, photo variants, the four customer e-mails, the selection notice
  carrying the link and the settlement notice carrying none).
- `backend-go/scripts/verify-gallery.sh` covers the photo selection page and the
  Drive connection with 60 assertions against a stub Drive: token access, folder
  validation, the package quota, the WebP preview, the
  `foto terpilih` copy, the lock after submit, the `/admin/drive` setup (client
  storage, the signed OAuth state, the callback, the parent folder), the
  backfill, the automatic folder of a new booking, the rename of an older
  `mustedit` folder, the automatic invoice link to `foto edit`, the viewer share
  on that folder and the folder that is trashed when its booking is deleted. It runs the backend from
  its Docker image, because the WebP conversion needs `cwebp`, which only the
  image carries.

```bash
cd backend-go && go test ./...
cd backend-go && ./scripts/verify-staging.sh
cd backend-go && ./scripts/verify-gallery.sh   # needs Docker
```

---

## 10. Running locally

```bash
cp .env.example .env          # fill DB_PASSWORD, JWT_SECRET, SMTP_ENCRYPTION_KEY
docker compose up -d --build
```

- Site: http://localhost:8090 (nginx → frontend, `/api`, `/uploads`)
- API: http://localhost:8080
- Admin: http://localhost:8090/admin — the first admin is created from
  `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` in `.env`, only while
  `admins` is empty. To change it later, write a new bcrypt hash into the table:

  ```bash
  docker exec photography-local-db psql -U postgres -d photographer_portfolio -c \
    "CREATE EXTENSION IF NOT EXISTS pgcrypto; UPDATE admins SET password = crypt('NEW_PASSWORD', gen_salt('bf', 12)) WHERE username = 'YOUR_USERNAME';"
  ```

The Go backend creates no default admin. Uploads live in `./local-data/uploads`.

The photo selection page (`/pilih/<token>`) also needs Google Drive, which is
optional: without a connection the page answers that it is unavailable and every
other part of the site works as before. To switch it on:

1. Create a Google Cloud project (no billing account or credit card is needed:
   standard Drive API use is free, and the daily threshold before any charge is
   400,000,000 quota units, far above what a booking can spend).
2. Enable the **Google Drive API** for that project.
3. Create an OAuth client of type ***Web application*** — not *Desktop app*, which
   cannot be given an `https://` redirect URI — and register
   `{MAIL_SITE_URL}/api/admin/drive/callback` under *Authorized redirect URIs*.
4. On the OAuth consent screen, publish the app (**In production**). An app left
   in *Testing* is issued a refresh token that expires after seven days, which
   would silently break the folder creation every week. Publishing without
   verification is free: the consent screen shows an "unverified app" warning
   that the admin clicks through once, and the 100-user cap is irrelevant here.
5. Open **Google Drive** in the admin sidebar, paste the Client ID and Client
   Secret, press **Hubungkan Akun Google**, and finally **Buat Folder Induk** so
   every client folder has a parent.

The refresh token is stored in `drive_settings`, encrypted with
`SMTP_ENCRYPTION_KEY`, and belongs to the account that owns the photo folders;
it is never returned to the browser or committed.

---

## Notes

- `backend-go/README.md` lists the deliberate differences from the original
  implementation (unreadable-body handling, Jackson coercions, unknown SMTP
  encryption mode, non-positive path ids).
- Uploaded images are stored on local disk; a container host needs a persistent
  volume for `UPLOAD_DIR`, or they are lost on redeploy.
