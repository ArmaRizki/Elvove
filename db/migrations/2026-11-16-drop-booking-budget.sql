-- Drops bookings.budget. The booking form no longer asks for it and the API no
-- longer reads or writes it, so the column was the last thing left.
--
-- Safe by construction: it aborts when any booking still holds a budget value,
-- so no client-entered data can disappear silently. A database restored from a
-- dump taken after the drop simply reports that there is nothing to do.
--
-- Rollback: ALTER TABLE bookings ADD COLUMN budget varchar(255);
-- (the values themselves are not recoverable, hence the guard above)

BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'bookings' AND column_name = 'budget'
  ) THEN
    RAISE NOTICE 'bookings.budget is already absent; nothing to drop';
    RETURN;
  END IF;
  IF EXISTS (SELECT 1 FROM bookings WHERE budget IS NOT NULL AND btrim(budget) <> '') THEN
    RAISE EXCEPTION 'some bookings still hold a budget value; aborting drop';
  END IF;
END $$;

ALTER TABLE bookings DROP COLUMN IF EXISTS budget;

COMMIT;
