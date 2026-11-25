-- Events written straight into the photographer's Google Calendar.
--
-- The feed address could only be subscribed to, never updated in place, and a
-- calendar app fetched it hours late. Writing through the Calendar API instead
-- needs to remember which Google event belongs to which booking, so a
-- reschedule moves that event rather than adding a second one, and deleting the
-- booking can take the event with it.
--
-- Nullable: a booking without an event (not accepted yet, or Google was down
-- when it was accepted) is normal. The panel's sync button fills it in later.
--
-- Additive, so an older backend keeps working.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-26-google-calendar.sql

BEGIN;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS google_event_id character varying(1024);

COMMIT;

-- Rollback:
-- ALTER TABLE bookings DROP COLUMN IF EXISTS google_event_id;
