--+ migration:000018_manual_journal_lines
--+ description: Rollback manual journal lines

-- Drop manual journal lines
DROP TABLE IF EXISTS "manual_journal_lines";