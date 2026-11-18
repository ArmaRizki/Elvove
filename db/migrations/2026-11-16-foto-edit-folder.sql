-- The folder the finished edits go in, and the one the invoice points at.
--
-- Until now the invoice link pointed at "foto terpilih", the folder the client's
-- chosen originals are copied into. That folder is for the photographer to work
-- from, so it stays private; the client gets a "foto edit" folder instead, shared
-- as viewer with anyone holding the link.
--
-- Additive only, so an older backend keeps working.
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-16-foto-edit-folder.sql

BEGIN;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS folder_edit_id character varying(100);

COMMIT;

-- Rollback:
-- ALTER TABLE bookings DROP COLUMN IF EXISTS folder_edit_id;
