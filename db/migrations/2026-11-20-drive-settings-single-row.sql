-- The Google Drive connection is one row, and the database says so.
--
-- drive_settings was created with `id integer PRIMARY KEY` only, and the comments
-- around it read "one row (id = 1), like smtp_settings". Every statement in the
-- backend already names that row (`WHERE id=1`, `ON CONFLICT (id)`), so nothing
-- is wrong with the code; what was missing is the constraint that makes the rule
-- true for anything else that writes to this table — a manual psql insert, a
-- migration, a future admin panel.
--
-- Without it a second row could exist, and then:
--   * the read (`SELECT ... FROM drive_settings WHERE id=1`) would quietly keep
--     using row 1 while an UPDATE that forgot the WHERE clause could touch both;
--   * a save built as `INSERT ... RETURNING` or an upsert on another key would
--     create row 2, and the panel would show a connection the server never reads.
--
-- A single-row table is expressed as a CHECK on the key, which is portable and
-- needs no trigger:
--
--   CHECK (id = 1)
--
-- Apply with:
--   psql "$DATABASE_URL" -f db/migrations/2026-11-20-drive-settings-single-row.sql

BEGIN;

-- Refuse to guess: if the table already holds more than one row, someone has to
-- decide which connection is real before the constraint can be added. The DO
-- block fails loudly instead of dropping the extra rows.
DO $$
DECLARE
    extra integer;
BEGIN
    SELECT count(*) INTO extra FROM drive_settings WHERE id <> 1;
    IF extra > 0 THEN
        RAISE EXCEPTION 'drive_settings holds % row(s) besides id=1; delete them before adding the single-row constraint', extra;
    END IF;
END
$$;

ALTER TABLE drive_settings
    DROP CONSTRAINT IF EXISTS drive_settings_single_row;

ALTER TABLE drive_settings
    ADD CONSTRAINT drive_settings_single_row CHECK (id = 1);

COMMIT;

-- Rollback:
-- ALTER TABLE drive_settings DROP CONSTRAINT IF EXISTS drive_settings_single_row;
