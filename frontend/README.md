# Photographer Portfolio – Frontend

React 19 + Vite + Tailwind CSS v4 + React Router + Axios.

## Design
Editorial photography aesthetic: ink-black background, brass/gold accent,
Fraunces (display serif) + Inter (body). Signature motif is a film
contact-sheet / sprocket-hole divider used throughout as section separators.

## Setup

```bash
cd frontend
npm install --legacy-peer-deps
cp .env.example .env
# edit .env if your backend isn't on localhost:8080
npm run dev
```

Note: `--legacy-peer-deps` is required because `lucide-react@0.383.0` (pinned
for its social brand icons — newer major versions dropped Instagram/Facebook/
YouTube/LinkedIn icons) lists React 18 as a peer, while this project uses
React 19. This is a known-safe combination in practice.

Runs on **http://localhost:5173** by default, talking to the backend on
**http://localhost:8080**.

## Structure

```
src/
  components/    Navbar, Footer, AdminLayout, AdminResourceManager (generic CRUD UI), ...
  pages/         Public pages (Home, About, Portfolio, ...)
  pages/admin/   Admin dashboard + management screens
  services/      api.js (axios instance + JWT interceptor), publicApi.js, adminApi.js
  context/       AuthContext (admin login state)
```

## Admin panel

Visit `/admin/login`. On first start with an empty `admins` table, the backend
creates the admin from its `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD`
env vars (password at least 12 characters, not a known default) and refuses to
start if they are missing or weak. It never changes an existing admin on later
starts; there are no built-in default credentials.

## Build for production

```bash
npm run build
```

Outputs static files to `dist/` — deploy to Netlify, Vercel, or serve via
Nginx alongside the Spring Boot backend.

## Notes / things to double check before going live
- The video gallery expects **embeddable** video URLs (e.g. a YouTube
  `https://www.youtube.com/embed/VIDEO_ID` link, not a normal watch link).
- The contact page's Google Map field expects an **embed** URL
  (Google Maps -> Share -> Embed a map -> copy the `src` from the iframe).
- Update `.env`'s `VITE_API_BASE_URL` to your real backend URL when deploying.
