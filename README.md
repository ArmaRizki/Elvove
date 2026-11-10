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
| Public pages | 14 | 10 |
| Admin sections | 13 | 12 (new: calendar, Terms, SMTP; 4 modules gone) |
| Booking flow | `PENDING → CONTACTED → COMPLETED` | `PENDING → ACCEPTED → RESCHEDULE_REQUESTED → COMPLETED` |
| E-mail | none | SMTP stored in DB, confirmation, reschedule and admin notifications |
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
- All timestamps use `Asia/Jakarta` wall clock, matching the old JVM's
  `LocalDateTime.now()`; the pool opens every connection with
  `timezone=Asia/Jakarta`.
- E-mail templates are a direct port: table-based HTML, hidden preheader,
  plain-text alternative, RFC 2047 subject, `Reply-To` set to the sender.

`backend/` still holds the Java implementation for reference. Documented,
intentional differences live in `backend-go/README.md`.

---

## 2. Removed

Four content modules were dropped from both the API and the UI, along with their
tables, controllers, services, repositories and admin screens:

- **Galleries** — curated photo sets
- **Services** — service list and pricing copy
- **Testimonials**
- **Videos / Films** — dropped with `backend/db/migrations/2026-11-09-drop-videos.sql`

The public **FAQ** page is also gone (the FAQ API and admin editor remain).
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
- Accepting a booking only changes the status; the confirmation e-mail is a
  separate, explicit action so it can be retried.
- Confirmation and reschedule e-mails are tracked
  (`confirmation_email_sent_at`, `confirmation_email_count`) and written only
  after SMTP accepts the message, so a failure leaves the row retryable.
- Rescheduling stores the new date/time and sends a notification to the client.
- New bookings notify the configured administrators, after the transaction
  commits, without delaying the customer or rolling back the booking.
- Admin calendar view and a custom time picker for schedule changes.

### Terms & Conditions

- `site_terms` holds the document and an incrementing `version`.
- A booking records `terms_accepted`, `terms_accepted_at` and `terms_version`.
- Saving different content bumps the version; if the version the client accepted
  is stale, the API answers `409` and asks them to review the new text.
- Admin editor with Markdown rendering, plus a public `/terms-and-conditions`
  page.

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
| Booking form | 5/10 min | unchanged |
| Uploads | extension in a 5-item list | magic-byte signature, `.gif` dropped, path-traversal guard |
| Originals | publicly served | admin-only |
| Reverse proxy | none | nginx with `server_tokens off`, security headers and a login rate-limit zone |

Passwords stay bcrypt-hashed and the JWT is still stateless with no server-side
session.

---

## 5. Database

14 tables:

`admins`, `categories`, `photos`, `packages`, `package_features`, `bookings`,
`contact_info`, `about_info`, `faqs`, `social_links`, `site_terms`,
`smtp_settings`, `booking_notification_settings`,
`booking_notification_recipients`.

New since Lens & Light: `site_terms`, `smtp_settings`, the two notification
tables, plus `booking` columns for terms acceptance and confirmation-email
tracking. `galleries`, `services`, `testimonials` and `videos` no longer exist.

Schema changes are handwritten SQL in `backend/db/migrations/`, applied by hand.
The Go backend never creates or alters tables.

---

## 6. Frontend

- Rebranded from Lens & Light to ELVOVE: logo, canonical URL, and a light theme
  with a crimson accent (`#E11D48`) replacing the dark ink/brass palette.
- `lucide-react` upgraded from `0.383.0` to `^1.70.0`; `react-icons` added.
- New admin screens: calendar, Terms editor, SMTP settings.
- Shared `AdminResourceManager` for the simple CRUD lists, plus `AdminLayout`,
  `EmptyState`, `LoadingState`, `CustomTimePicker` and `WhatsAppButton`.
- `utils/bookingTime.js`, `utils/termsMarkdown.jsx` and `utils/uploadLimits.js`
  keep time handling, Markdown rendering and the upload-size hints in one place.

---

## 7. Deployment

`compose.yml` runs the whole stack locally — Postgres, the Go API, Vite, nginx —
and a production variant in `/opt/compose` runs Postgres, the Go image, a static
nginx frontend and the proxy, with the database and uploads on named/bind mounts.

- `backend-go/Dockerfile`: multi-stage build, static binary, non-root uid 10001,
  `libwebp-tools` for the thumbnail conversion.
- `nginx/default.conf`: proxies `/api` and `/uploads`, serves the SPA, and
  applies rate limiting plus security headers.
- `backend-go/scripts/`: `run-web.sh` (the stack behind nginx),
  `set-dev-admin.sh` (admin password on the throwaway copy database) and
  `verify-staging.sh` (end-to-end checks).

Backend environment: `DB_*` or `DB_URL`, `JWT_SECRET`, `SMTP_ENCRYPTION_KEY`,
`FRONTEND_URL`, `MAIL_SITE_URL`, `MAIL_ADMIN_URL`, `ADMIN_*`,
`UPLOAD_MAX_FILE_SIZE`, `UPLOAD_MAX_REQUEST_SIZE`, `PORT`, `UPLOAD_DIR`.

---

## 8. Structure

```
backend-go/          Go API (the running backend)
  cmd/api/           entrypoint
  internal/config/   environment parsing
  internal/httpapi/  routes, handlers, auth, validation, e-mail
  internal/store/    pgx pool and query helpers
  scripts/           run-web.sh, set-dev-admin.sh, verify-staging.sh
backend/             legacy Java/Spring Boot implementation (reference)
frontend/            React 19 + Vite + Tailwind
  src/pages/         public pages
  src/pages/admin/   admin screens
  src/services/      axios clients and API maps
nginx/               reverse proxy configuration
compose.yml          local stack
```

---

## 9. Tests

- 22 Java test files (the original implementation), including booking workflow,
  e-mail templates, rate limiter, upload limits and time zone.
- 9 Go test files covering auth, the limiter, booking validation and dates,
  e-mail rendering, SMTP encryption and the store.
- `backend-go/scripts/verify-staging.sh` drives a real stack end to end with 25
  assertions (status workflow, WIB timestamps, SMTP failure rollback, upload
  limits, CORS, photo variants).

```bash
cd backend-go && go test ./...
cd backend && ./mvnw test     # requires the Java 17 toolchain
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

---

## Notes

- `backend-go/README.md` lists the deliberate differences from the Java
  implementation (unreadable-body handling, Jackson coercions, unknown SMTP
  encryption mode, non-positive path ids).
- Uploaded images are stored on local disk; a container host needs a persistent
  volume for `UPLOAD_DIR`, or they are lost on redeploy.
