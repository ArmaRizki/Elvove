-- One column per folder: drop the duplicate folder ids.
--
-- client_folder_id always held the same folder as raw_folder_id (the client
-- folder IS the raw folder), and selection_folder_id the same as
-- folder_mustedit_id (the "foto terpilih" folder). Two names for one folder
-- meant two sources of truth; every reader now uses raw_folder_id and
-- folder_mustedit_id only.
--
-- Copy first, drop second: a booking with only client_folder_id filled keeps
-- its folder. All steps are idempotent, so the migration can run twice.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-17-drop-duplicate-folder-columns.sql

BEGIN;

-- Bookings whose folder was recorded only under the old name.
UPDATE bookings SET raw_folder_id = client_folder_id
    WHERE (raw_folder_id IS NULL OR raw_folder_id = '')
      AND client_folder_id IS NOT NULL AND client_folder_id != '';

UPDATE bookings SET folder_mustedit_id = selection_folder_id
    WHERE (folder_mustedit_id IS NULL OR folder_mustedit_id = '')
      AND selection_folder_id IS NOT NULL AND selection_folder_id != '';

ALTER TABLE bookings DROP COLUMN IF EXISTS client_folder_id;
ALTER TABLE bookings DROP COLUMN IF EXISTS selection_folder_id;

COMMIT;

-- Rollback (data cannot come back; the columns start empty):
-- ALTER TABLE bookings ADD COLUMN IF NOT EXISTS client_folder_id character varying(100);
-- ALTER TABLE bookings ADD COLUMN IF NOT EXISTS selection_folder_id character varying(100);
