-- Adds confirmation e-mail tracking to bookings.
--
-- The service writes these columns only after SMTP accepts a message, so a
-- failed delivery leaves them unchanged and the admin can retry.
--
-- Both columns are additive and nullable, so the migration is backward
-- compatible: an older backend keeps working against the new schema, and the
-- application stays usable while the migration runs.
--
-- Apply with:
--   psql "$DATABASE_URL" -f backend/db/migrations/2026-11-08-booking-confirmation-email-tracking.sql

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS confirmation_email_sent_at timestamp(6);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS confirmation_email_count integer;

-- Existing rows predate tracking, so they count as "no confirmation sent yet".
UPDATE bookings
SET confirmation_email_count = 0
WHERE confirmation_email_count IS NULL;

ALTER TABLE bookings
    ALTER COLUMN confirmation_email_count SET DEFAULT 0;

-- Rollback (drops the tracking history for every booking):
-- ALTER TABLE bookings DROP COLUMN IF EXISTS confirmation_email_count;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS confirmation_email_sent_at;
