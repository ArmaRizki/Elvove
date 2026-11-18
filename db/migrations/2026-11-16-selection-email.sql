-- The photo-selection notice, which used to travel in the settlement e-mail.
--
-- The client now gets the selection link in its own message, sent before the
-- settlement one: the photos are ready, so the choice comes first and the
-- settlement notice is left to ask for the remaining balance only.
--
-- The two tracking columns mirror the other customer e-mails, so the panel can
-- show when this one went out and how often.
--
-- Additive only, so an older backend keeps working.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-16-selection-email.sql

BEGIN;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS selection_email_sent_at timestamp(6);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS selection_email_count integer DEFAULT 0;

COMMIT;

-- Rollback:
-- ALTER TABLE bookings DROP COLUMN IF EXISTS selection_email_count;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS selection_email_sent_at;
