# Photographer Portfolio – Backend (Phase 1: Foundation)

## What's in this phase
- Maven project (Spring Boot 3.3.4, Java 17 — matches your working I. Anwar & Associates setup)
- `application.yml` — PostgreSQL + JWT config via environment variables
- 13 JPA entities covering every table from the spec:
  Admin, Category, Photo, Gallery, ServiceItem, PackageItem, Testimonial,
  Video, Booking (with status enum), Contact, About, Faq, SocialLinks
- Auditing (`createdAt`/`updatedAt`) built into every entity via `BaseEntity`

## Before running locally
1. Install PostgreSQL, create a database:
   ```sql
   CREATE DATABASE photographer_portfolio;
   ```
2. Set environment variables (PowerShell — remember `$env:`, not `export`):
   ```powershell
   $env:DB_USERNAME="postgres"
   $env:DB_PASSWORD="your_password"
   $env:JWT_SECRET="a-long-random-string-at-least-32-chars"
   $env:SMTP_ENCRYPTION_KEY="another-long-random-string-at-least-32-chars"
   $env:ADMIN_USERNAME="your_admin"
   $env:ADMIN_EMAIL="you@example.com"
   $env:ADMIN_PASSWORD="a-strong-password-12-plus-chars"
   ```
   `DB_PASSWORD`, `JWT_SECRET` (≥ 32 bytes) and `SMTP_ENCRYPTION_KEY` (≥ 32 bytes) are required; the backend fails to start without them.
   On first start with an empty `admins` table, the backend creates the admin from `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` (password ≥ 12 chars, not a known default) and refuses to start if they are missing or weak. It never changes an existing admin on later starts.
3. From `/backend`: `mvn spring-boot:run`

Note: I couldn't run `mvn compile` in this sandbox (Maven Central isn't reachable from here), so please run a build locally before we move on — catch any issues now while the codebase is still small.

## Booking e-mail

The booking confirmation and schedule-change e-mails are rendered server-side by
`BookingEmailTemplate` as table-based HTML with inline styles (600 px, no
scripts, no external stylesheet) and are sent by `SmtpMailService.sendHtml` as a
`multipart/alternative` message, so a client that cannot render HTML still gets
a readable plain-text version. The header branding is text-based, so the e-mail
loads no external asset.

One optional variable controls the address shown in the footer:

| Variable | Default | Purpose |
| --- | --- | --- |
| `MAIL_SITE_URL` | `https://elvove.site` | Address shown in the footer. |

Preview the rendered e-mail locally, without SMTP and without emailing anyone:

```bash
cd backend
mvn -o -q test-compile
mvn -o -q dependency:build-classpath -Dmdep.outputFile=cp.txt -DincludeScope=test
java -cp "target/classes:target/test-classes:$(cat cp.txt)" \
     com.portfolio.photographer.service.BookingEmailPreview booking-confirmation-preview.html
```

The HTML file can be opened in any browser; the plain-text part is written next
to it with a `.txt` suffix.

## Next phases
2. Repositories, Services, DTOs
3. REST Controllers + JWT Security (admin login, protected routes)
4. Image upload handling
5–7. React frontend (public pages, booking/testimonials/contact, admin dashboard)
8. Integration testing + deployment README
