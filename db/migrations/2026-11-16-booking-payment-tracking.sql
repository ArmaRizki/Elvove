-- Records the payment points, the delivery link and the invoice / settlement
-- e-mails of a booking.
--
-- The two payment points are timestamps rather than booleans: the admin panel
-- shows when a payment was recorded, and the API sets them so that "lunas"
-- always implies a down-payment date. Until this migration the only payment
-- information the system held was the package price.
--
-- delivery_url is the Google Drive folder holding the edited photos. The
-- invoice e-mail is the last message of a booking and carries that link, so the
-- admin pastes it once here instead of writing it into every mail by hand.
--
-- The four e-mail columns follow the confirmation tracking added on 2026-11-08:
-- they are written only after SMTP accepts the message, so a failed delivery
-- leaves them unchanged and the admin can retry without inventing a history.
--
-- All seven columns are additive and nullable, so the migration is backward
-- compatible: an older backend keeps working against the new schema, and the
-- application stays usable while the migration runs.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-16-booking-payment-tracking.sql

BEGIN;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS down_payment_paid_at timestamp(6);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS settlement_paid_at timestamp(6);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS delivery_url character varying(500);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS invoice_email_sent_at timestamp(6);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS invoice_email_count integer;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS settlement_email_sent_at timestamp(6);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS settlement_email_count integer;

-- Existing rows predate the two e-mails, so they count as "never sent".
UPDATE bookings SET invoice_email_count = 0 WHERE invoice_email_count IS NULL;
UPDATE bookings SET settlement_email_count = 0 WHERE settlement_email_count IS NULL;

ALTER TABLE bookings
    ALTER COLUMN invoice_email_count SET DEFAULT 0;

ALTER TABLE bookings
    ALTER COLUMN settlement_email_count SET DEFAULT 0;

COMMIT;

-- Rollback (drops the delivery link, the payment history and both delivery logs):
-- ALTER TABLE bookings DROP COLUMN IF EXISTS settlement_email_count;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS settlement_email_sent_at;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS invoice_email_count;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS invoice_email_sent_at;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS delivery_url;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS settlement_paid_at;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS down_payment_paid_at;
