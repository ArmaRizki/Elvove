-- The Terms page used to hard-code a two-column grid that split at every
-- Heading 1. The admin now picks the column count, the column widths, and
-- whether a new column starts at a Heading 1 or at an explicit "Pisah kolom"
-- marker, so those three values live next to the content.
-- Rollback: ALTER TABLE site_terms DROP COLUMN layout_columns, DROP COLUMN
-- layout_width, DROP COLUMN layout_split;

BEGIN;

ALTER TABLE site_terms
  ADD COLUMN layout_columns smallint NOT NULL DEFAULT 2,
  ADD COLUMN layout_width varchar(12) NOT NULL DEFAULT 'equal',
  ADD COLUMN layout_split varchar(12) NOT NULL DEFAULT 'heading';

COMMIT;
