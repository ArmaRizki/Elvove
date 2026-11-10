# Go backend migration

This directory is the Go replacement for the Spring Boot backend. It uses the
existing PostgreSQL schema and frontend API contract, and it is what serves the
local stack (`photography-local-backend`). Production still runs the Java
backend until it is deployed separately, and the two were compared endpoint by
endpoint: responses match, apart from the cases listed at the end of this file.

## Implemented API surface

- PostgreSQL connection pool and health endpoint.
- Admin login using existing bcrypt hashes and HS256 JWT tokens.
- Public About, Contact, Terms, Social Links, Categories, Packages, FAQs and
  Photos read endpoints, including existing filters and response field names.
- Authentication middleware, rate limits for login and booking creation.
- JSON and error response conventions compatible with the frontend.
- Admin CRUD for categories, packages, FAQs, photos, and singleton content.
- Booking status transitions, acceptance, rescheduling, and deletion.
- Multipart image uploads with signature/size checks, WebP variants, and
  private original storage.
- Booking notification recipient settings and SMTP settings, AES-GCM password
  encryption compatible with the Java implementation, booking confirmation,
  and best-effort new-booking notifications.
- Booking confirmation, reschedule and administrator notification e-mails
  rendered from `internal/httpapi/email_template.go`, a direct port of the Java
  `BookingEmailTemplate`: table-based HTML with inline CSS, hidden inbox preview
  line, escaped booking values, plain-text alternative part,
  `multipart/alternative` message, RFC 2047 subject encoding and `Reply-To` set
  to the configured sender address.
- First-admin bootstrap from `ADMIN_USERNAME`/`ADMIN_EMAIL`/`ADMIN_PASSWORD`,
  run only while the `admins` table is empty, with Java's password rules.

## E-mail configuration

- `MAIL_SITE_URL` (default `https://elvove.site`) supplies the footer link and
  header logo URL, mirroring Java's `app.mail.site-url`.
- `MAIL_ADMIN_URL` (default `https://elvove.site`) is the admin panel base the
  administrator notification links to as
  `{MAIL_ADMIN_URL}/admin/bookings?booking=ID`, mirroring Java's
  `app.mail.admin-url`. It must be an absolute http(s) URL without credentials,
  or the notification is skipped and logged.
- The sender display name and address come from the `smtp_settings` table, as
  Java's `SmtpMailService.getSenderName()`/`getSenderEmail()` do; the brand name
  falls back to `Elvove` when SMTP is not configured yet.
- Missing booking values (`event_time`, `package_title`, `package_price`,
  `event_location`, unset dates) render as "Akan dikonfirmasi" instead of
  failing the query, matching the Java template.

## Timestamps and time zone

Java calls `TimeZone.setDefault(Asia/Jakarta)` at startup and the columns are
`timestamp without time zone`, so every stored value is a WIB wall-clock and
every response carries it without an offset (`2026-11-10T14:02:37`), while
`date` columns render as `2026-12-10`. Go reproduces this through two rules:

- `store.WallClock(t)` takes the WIB fields of a moment and re-tags them as UTC.
  pgx renders a `time.Time` in the session timezone (UTC here), so this makes it
  write the WIB wall-clock verbatim. `store.Now()` is `WallClock(time.Now())`.
- The pool also opens every connection with `timezone=Asia/Jakarta`, so a column
  filled by SQL `NOW()` (the `updated_at` and `created_at` of every admin write)
  holds the same WIB wall clock Java's `LocalDateTime.now()` writes. Without it
  those rows landed seven hours early.
- Values read back **must not** be converted again. pgx returns the stored
  wall-clock tagged UTC, so `normalize` and `wibTimestamp` format them as-is;
  calling `.In(jakartaZone)` on either side adds a second +7 and renders 13:54
  as 20:54. `internal/store/store_test.go` pins both halves.

Rows written by Java, and every row written since, are WIB. Rows an earlier
revision of this Go backend wrote through `NOW()` (photos, categories, packages,
FAQs, and every `updated_at`) were UTC and read 7 hours early. A booking that
Go created and then updated shows it directly: its `updated_at` is earlier than
its `created_at`.

```sql
select id, created_at, updated_at from bookings where updated_at < created_at;
```

## Verifying against a real PostgreSQL and SMTP server

`scripts/verify-staging.sh` runs the whole SQL and mail path end to end. It
dumps the schema read-only from the local Java database (`photography-local-db`)
into a throwaway container, adds synthetic fixtures, starts a Mailpit sink, runs
the built server, and checks 25 assertions across login, list, status
transitions, WIB timestamps, public booking validation, and the transactional
reschedule. Both containers are removed on exit; nothing outside the
`elvove-go-verify-` prefix is touched.

```sh
./scripts/verify-staging.sh          # 25 passed, 0 failed
./scripts/verify-staging.sh --keep   # leave containers up for debugging
```

Two real bugs were found only by running it, and both are now covered by unit
tests: `event_date` rendering as a full timestamp instead of a bare date, and
timestamps shifting by seven hours twice (once on write, once on read).

## Remaining verification

- SMTP delivery to a real provider (Gmail) is untested; it needs live
  credentials the maintainer must supply. `SMTP_ENCRYPTION_KEY` must match
  Java's value (at least 32 bytes) to decrypt the stored password. The local
  Mailpit run proves the send path, the transaction boundary, and the rendered
  message, not provider-specific TLS or authentication behaviour.
- ARM64 image builds are supported, but run-time testing should be performed on
  native ARM64 or a staging host.

## Deliberate differences from Java

Both backends were probed side by side and now answer identically for every
request the UI can make. These remain, all reachable only by a hand-made
request:

- A body sent with an unsupported `Content-Type`, and `/api/admin/upload` without
  a `file` part: Java's catch-all turns both into `500`, Go answers `400`.
- Numeric coercion Jackson applies silently (`{"status":1}`, `"port":"587"`):
  Java accepts them, Go rejects the body.
- `PUT /api/admin/settings/smtp` with an unknown `encryption`: Java throws
  `IllegalArgumentException` and answers `500`, Go answers `400` with the same
  message.
- A path id that is not a positive number (`/api/admin/bookings/abc`): Java
  answers the unreadable-body `400`, or `404` for `0`, while Go answers
  `400 Invalid resource id.`
- SMTP with `encryption=NONE` and `authentication=true`: Java sends the
  credentials over the plain connection, Go's SMTP client refuses to send a
  password unencrypted, so the send fails with the usual mail error.

## Local run

Set `DB_URL`, `JWT_SECRET` (at least 32 bytes), and `FRONTEND_URL`; set
`SMTP_ENCRYPTION_KEY` for SMTP support, optionally `MAIL_SITE_URL` and
`MAIL_ADMIN_URL` for e-mail links, and `ADMIN_USERNAME`/`ADMIN_EMAIL`/
`ADMIN_PASSWORD` when the admins table is still empty, then run
`go run ./cmd/api`.
