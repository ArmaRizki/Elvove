-- Google Drive connected from the admin panel.
--
-- Until now the credentials were environment variables and the raw folder was
-- pasted per booking. Now the connection lives in the database and is edited at
-- /admin/drive, so the admin can connect an account and have the per-booking
-- folders created automatically.
--
-- One row (id = 1), like smtp_settings: there is one Drive account.
--   client_id / client_secret  OAuth client of type "Desktop app"
--   encrypted_refresh_token    AES-GCM with SMTP_ENCRYPTION_KEY, never returned
--   root_folder_id             the parent every client folder is created in
--   root_folder_name           its display name, so the panel can show it
--   connected_email            the account, read from Drive after connecting
--   connected_at               when the consent was completed
--
-- bookings gains folder_webp_id and folder_mustedit_id: the two subfolders are
-- created with the client folder, so the client page and the submit step reuse
-- them instead of looking them up again on every request.
--
-- All columns are additive, so an older backend keeps working.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-16-drive-settings.sql

BEGIN;

CREATE TABLE IF NOT EXISTS drive_settings (
    id integer PRIMARY KEY,
    client_id character varying(255),
    encrypted_client_secret text,
    encrypted_refresh_token text,
    root_folder_id character varying(100),
    root_folder_name character varying(255),
    connected_email character varying(255),
    connected_at timestamp(6),
    created_at timestamp(6) DEFAULT NOW(),
    updated_at timestamp(6) DEFAULT NOW()
);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS client_folder_id character varying(100);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS folder_webp_id character varying(100);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS folder_mustedit_id character varying(100);

COMMIT;

-- Rollback:
-- ALTER TABLE bookings DROP COLUMN IF EXISTS folder_mustedit_id;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS folder_webp_id;
-- ALTER TABLE bookings DROP COLUMN IF EXISTS client_folder_id;
-- DROP TABLE IF EXISTS drive_settings;
