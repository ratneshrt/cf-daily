-- One-time correction for the assigned_date off-by-one.
--
-- Before this change the IST midnight timestamp was cast to a date in the
-- database session timezone (UTC), so every row landed on the previous
-- calendar day: the problem for 2026-10-04 IST was stored as 2026-10-03.
-- Reads made the same mistake, so the bug was invisible - but now that dates
-- are passed as YYYY-MM-DD strings, the existing rows are one day behind.
--
-- Run this ONCE, and only if the rows predate the date-string change. Check
-- first:
--
--   SELECT id, assigned_date, name FROM daily_problems ORDER BY assigned_date DESC LIMIT 5;
--
-- If the newest row's assigned_date is yesterday in IST while that problem was
-- in fact today's, this migration is the fix. If the dates already look right,
-- skip it.
--
-- The unique index is dropped first because Postgres checks uniqueness per row
-- during an UPDATE, so shifting every row at once would otherwise collide.

BEGIN;

ALTER TABLE daily_problems
    DROP CONSTRAINT IF EXISTS daily_problems_assigned_date_key;

DROP INDEX IF EXISTS daily_problems_assigned_date_key;

UPDATE daily_problems
SET assigned_date = assigned_date + 1;

CREATE UNIQUE INDEX daily_problems_assigned_date_key
    ON daily_problems (assigned_date);

COMMIT;
