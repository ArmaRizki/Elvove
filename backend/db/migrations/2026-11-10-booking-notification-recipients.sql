-- Adds administrator booking e-mail notifications.
--
-- booking_notification_settings holds one row (id = 1) with the enabled flag.
-- booking_notification_recipients holds the list of administrator addresses.
-- Both tables are new, so the migration is additive and backward compatible.
-- The recipient list starts empty: no address is ever inserted here.
--
-- Apply with:
--   psql "$DATABASE_URL" -f backend/db/migrations/2026-11-10-booking-notification-recipients.sql

CREATE TABLE IF NOT EXISTS booking_notification_settings (
    id          bigint PRIMARY KEY,
    enabled     boolean     NOT NULL DEFAULT false,
    updated_at  timestamp(6) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS booking_notification_recipients (
    id          bigserial PRIMARY KEY,
    email       varchar(255) NOT NULL,
    created_at  timestamp(6) NOT NULL DEFAULT now()
);

-- Case-insensitive uniqueness: the same address cannot be stored twice.
CREATE UNIQUE INDEX IF NOT EXISTS ux_booking_notification_recipients_email_lower
    ON booking_notification_recipients (lower(email));

-- Rollback (drops the configuration; no booking data is touched):
-- DROP TABLE IF EXISTS booking_notification_recipients;
-- DROP TABLE IF EXISTS booking_notification_settings;
