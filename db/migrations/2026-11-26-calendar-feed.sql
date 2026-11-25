-- The calendar feed: one address the photographer subscribes to once in Google
-- Calendar, Apple Calendar or Outlook, and every booking appears there.
--
-- Subscribing means the calendar app fetches the address on its own, with no
-- login, so the URL itself has to be the key. It holds a random token, and the
-- token is a secret: whoever knows the address sees the client names, times and
-- places in the feed. Rotating the row (the panel offers it) makes the old
-- address stop working.
--
-- Only the token lives here. The events are built from `bookings` on every
-- fetch, so the feed is never stale and no second copy of client data exists.
--
-- Additive; nothing else changes.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-26-calendar-feed.sql

BEGIN;

CREATE TABLE IF NOT EXISTS calendar_feed (
    id integer PRIMARY KEY,
    token character varying(64) NOT NULL,
    created_at timestamp(6) DEFAULT NOW(),
    CONSTRAINT calendar_feed_single_row CHECK (id = 1)
);

COMMIT;

-- Rollback:
-- DROP TABLE IF EXISTS calendar_feed;
