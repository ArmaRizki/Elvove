-- The site has exactly one terms row: the admin page edits "the" terms and the
-- API reads with ORDER BY id LIMIT 1.
--
-- The public GET /api/public/terms seeds that row when the table is empty. Two
-- requests arriving together could both see an empty table and both insert, and
-- from then on the API reads whichever row sorts first. The unique index on a
-- constant expression allows only one row, so the seed's ON CONFLICT DO NOTHING
-- makes the second writer a no-op instead of a duplicate.
--
-- Rollback: DROP INDEX site_terms_singleton;

BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS site_terms_singleton
  ON site_terms ((id IS NOT NULL));

COMMIT;
