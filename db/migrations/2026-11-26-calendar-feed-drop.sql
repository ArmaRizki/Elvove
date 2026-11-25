-- The subscription link is gone; the calendar is written through the API.
--
-- Batch 1 added a `calendar_feed` row holding the token for the public
-- `/api/calendar/<token>.ics` subscription. Batch 2 replaced it with the
-- Calendar API, which needs no link at all. Keeping the table would leave an
-- unused token lying around that still opens the old (now removed) route, so
-- the row and the table go.
--
-- Reversible: re-run db/migrations/2026-11-26-calendar-feed.sql to recreate it.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-26-calendar-feed-drop.sql

BEGIN;

DROP TABLE IF EXISTS calendar_feed;

COMMIT;

-- Rollback:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-26-calendar-feed.sql