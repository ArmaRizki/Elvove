-- The About story moved from a Markdown textarea to a rich-text editor, so the
-- stored value is now HTML. Markup pushes the same prose past the old
-- varchar(3000) limit (1944 characters became 2002), and the admin should be
-- able to write more than that, so the column becomes unlimited text.
-- Rollback: ALTER TABLE about_info ALTER COLUMN story TYPE varchar(3000);
-- (fails if any row is longer than 3000 characters).

BEGIN;

ALTER TABLE about_info ALTER COLUMN story TYPE text;

COMMIT;
