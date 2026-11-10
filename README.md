# ELVOVE — Photography Website

ELVOVE is a full-stack photography website for presenting photographic work, publishing services and packages, receiving booking inquiries, and managing site content through a secure admin dashboard.

**Live demo:** _add your deployed URL here once live_

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React 19 · Vite 8 · Tailwind CSS v4 · React Router 7 · Axios |
| Backend | Go 1.23+ · `net/http` routing · pgx v5 |
| Legacy backend | Spring Boot 3.3.4 · Java 17 (kept in `backend/`, still what production runs) |
| Database | PostgreSQL |
| Auth | JWT HS256 (golang-jwt v5) + BCrypt password hashing |
| Fonts | Fraunces (display serif) + Inter (body), self-hosted via Fontsource |
| Icons | lucide-react |

No microservices, no Redis, no Kafka, no WebSockets, no external CMS — deliberately kept simple and easy for one developer to maintain. One `compose.yml` runs the whole local stack (Postgres, Go API, Vite frontend, nginx), and `backend-go/Dockerfile` is what a container host deploys.

---

## Design

Editorial, film-photography aesthetic rather than a generic template:

- **Palette:** ink black `#0e0d0c` · warm ivory `#f3efe7` · brass/gold accent `#b08d57` · deep burgundy `#5c2333`
- **Signature motif:** a film contact-sheet / sprocket-hole divider (`.sprocket-rule`) and frame-numbered section labels (`FRAME 001`, `FRAME 002`...) used throughout instead of generic numbering
- Full-bleed imagery, generous negative space, minimal chrome

---

## Features

### Public site (9 pages)
Home · About · Portfolio (category filter + lightbox) · Packages · Latest Gallery · Booking (validated inquiry form) · Contact · Privacy · Terms

### Admin dashboard (JWT-protected, `/admin`)
- Dashboard with live stats (photos, bookings, packages)
- CRUD: Photos (with category + image upload), Categories, Packages (dynamic feature lists), FAQs
- Bookings inbox with status workflow and a calendar view
- Single-record editors: About content, Contact info, Social links, Terms, SMTP settings
- Ordering via a `displayOrder` field on the ordered content types

### Backend safeguards
- JWT authentication, BCrypt-hashed passwords, protected `/api/admin/**` routes
- No default credentials: an admin row must exist in the database before anyone can log in; there is no first-run bootstrap
- Per-IP rate limiting on login and on the public booking form to block spam
- Field-level JSON validation errors, matching the shapes the frontend already reads
- Image upload validation (magic-byte type check + configurable size limit, 50 MB by default) with path-traversal protection, WebP thumbnails, and originals served only to authenticated admins

---

## Project Structure
web-photography/
├── backend-go/                    # active backend (Go)
│   ├── Dockerfile                 # multi-stage build, non-root runtime image
│   ├── cmd/api/                   # entrypoint
│   ├── internal/config/           # env parsing
│   ├── internal/httpapi/          # routes, handlers, auth, validation, e-mail
│   ├── internal/store/            # pgx pool + query helpers
│   └── scripts/                   # run-web.sh, set-dev-admin.sh, verify-staging.sh
├── backend/                       # legacy Java/Spring Boot backend (production)
│   └── src/main/java/com/portfolio/photographer/
├── frontend/
│   ├── src/pages/                 # public pages
│   ├── src/pages/admin/           # admin dashboard screens
│   ├── src/components/            # Navbar, Footer, AdminResourceManager, ...
│   ├── src/services/              # api.js (axios + JWT interceptor), publicApi.js, adminApi.js
│   └── src/utils/imageUrl.js      # Resolves uploaded-image URLs against the backend origin
├── nginx/default.conf             # proxies /api and /uploads to the backend
├── compose.yml                    # local stack: db, backend, frontend, nginx
└── .env.example

---

## Database Schema

14 tables: `admins`, `categories`, `photos`, `packages` (+ `package_features`), `bookings`, `contact_info`, `about_info`, `faqs`, `social_links`, `site_terms`, `smtp_settings`, `booking_notification_settings`, `booking_notification_recipients`.

The Go backend uses plain SQL through pgx and does **not** create or alter tables — it expects this schema to exist. Incremental changes are handwritten SQL files in `backend/db/migrations/`, applied by hand.

---

## API Overview

| Prefix | Access | Examples |
|---|---|---|
| `/api/auth/login` | Public, rate-limited | `POST /api/auth/login` returns a JWT |
| `/api/public/**` | Public, read-only | `GET /api/public/photos`, `GET /api/public/packages` |
| `POST /api/bookings` | Public, rate-limited | Submit a booking inquiry |
| `/api/admin/**` | JWT required | CRUD for every resource, `/api/admin/upload`, `/api/admin/dashboard/summary`, settings |
| `/uploads/**` | Thumbnails and previews public; `originals/` admin-only | |

---

## Local Setup

### Prerequisites
Docker (with Compose) · Node.js only if you want to run the frontend outside the containers

### 1. Configure
```bash
cp .env.example .env
# fill in DB_PASSWORD, JWT_SECRET, SMTP_ENCRYPTION_KEY (32+ bytes each)
```

### 2. Start the stack
```bash
docker compose up -d --build
```
- Site: **http://localhost:8090** (nginx → frontend + `/api` + `/uploads`)
- API directly: **http://localhost:8080**
- Postgres: container `photography-local-db`, data in the `photography-local-db` volume, uploads in `./local-data/uploads`

Uploads are bind-mounted from `./local-data/uploads`; if that directory is owned by root (created by an earlier container), the local compose service runs as root (`user: "0:0"`). The image itself still defaults to a non-root user for deployment.

### 3. Admin login
The Go backend creates no default admin, and `production`/`local` share the same `admins` row. To set a password, use the helper at the repository root (it hashes with bcrypt via Python `bcrypt`, then updates the row):
```bash
./reset-admin-password.sh
```

### Frontend outside Docker (optional)
```bash
cd frontend
npm install --legacy-peer-deps
cp .env.example .env
npm run dev
```
Runs on **http://localhost:5173**, pointed at the API by `VITE_API_BASE_URL`.

> `--legacy-peer-deps` is required because `lucide-react` is pinned to `0.383.0` (for its social-media brand icons, dropped in later major versions) which lists React 18 as a peer while this project uses React 19 — a safe combination in practice.

---

## Deployment

Canonical production site: **https://elvove.site** (bare hostname — do not use or assume `www.elvove.site`). Official business/admin address: **elvo.vee@gmail.com**.

**Production still runs the Java backend in `backend/`.** The Go backend in `backend-go/` is API-compatible with it and can be deployed independently; nothing else has to change, because both use the same database and the same frontend contract.

Recommended free stack:
- **Database:** [Neon](https://neon.tech) — permanent free Postgres tier
- **Backend:** [Render](https://render.com) — free web service, built from `backend-go/Dockerfile`
- **Frontend:** [Vercel](https://vercel.com) — free static hosting for the Vite build

Environment variables on the backend host:

| Variable | Notes |
|---|---|
| `DB_URL` | `postgres://user:pass@host/db` — or set `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD` separately |
| `JWT_SECRET` | 32+ bytes, required; changing it logs every admin out |
| `SMTP_ENCRYPTION_KEY` | 32+ bytes, required to decrypt the SMTP password stored in the database. Must stay the same value the Java backend used |
| `FRONTEND_URL` | CORS origin, `https://elvove.site` in production |
| `MAIL_SITE_URL` | Site address used in the booking e-mail header and footer |
| `UPLOAD_MAX_FILE_SIZE` / `UPLOAD_MAX_REQUEST_SIZE` | Size syntax such as `50MB` / `100MB`; keep the request limit in step with nginx |
| `PORT` / `UPLOAD_DIR` | Default `8080` / `/app/uploads` |

SMTP host, port, username and password are not environment variables — they live in the `smtp_settings` table and are edited from the admin panel.

On the frontend host: `VITE_API_BASE_URL` pointing at the live backend URL. See `.env.example` at the repository root for the full list with placeholders.

**Known limitation:** uploaded images are stored on local disk (`uploads/`, `UPLOAD_DIR`). On platforms with an ephemeral filesystem (most free tiers), uploaded images are lost on redeploy/restart. Fine for a portfolio demo; for real production use, mount a persistent volume or move uploads to S3/Cloudinary.

---

## Security Checklist Before Going Live
- [ ] Change the admin password immediately after the first login (`./reset-admin-password.sh`)
- [ ] Set a strong, random `JWT_SECRET` (32+ bytes) via environment variable — never commit it
- [ ] Set `SMTP_ENCRYPTION_KEY` (32+ bytes) and keep it stable; losing it makes the stored SMTP password unreadable
- [ ] Set `FRONTEND_URL` to `https://elvove.site` (controls CORS; no `www`)
- [ ] Configure the sender address and SMTP credentials in the admin panel, and store only the encrypted password
- [ ] Set `VITE_API_BASE_URL` to your real deployed backend domain

---

## License
Personal/portfolio project — no license specified.
