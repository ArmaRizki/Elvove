-- How long a session lasts, per package ("Origin: 2 hours").
--
-- The calendar feed needs an end time: an event with only a start is either
-- rejected or shown as a point in time. Rather than hard-code a duration in the
-- calendar code, the photographer states it once per package and the booking
-- inherits it.
--
-- Zero means "not set yet". The eight packages that existed before this
-- migration are left at zero on purpose: the real durations are the
-- photographer's to state, and guessing them would put wrong hours in her
-- calendar. The calendar falls back to two hours until a package is filled in.
--
-- Additive and nullable-with-default, so an older backend keeps working.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-26-package-session-minutes.sql

BEGIN;

ALTER TABLE packages
    ADD COLUMN IF NOT EXISTS session_minutes integer NOT NULL DEFAULT 0;

COMMIT;

-- Rollback:
-- ALTER TABLE packages DROP COLUMN IF EXISTS session_minutes;
