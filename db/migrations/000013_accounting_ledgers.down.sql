--+ migration:000013_accounting_ledgers
--+ description: Rollback accounting and settlement ledger tables

-- Drop accounting ledger tables
DROP TABLE IF EXISTS "party_open_items";
DROP TABLE IF EXISTS "journal_lines";
DROP TABLE IF EXISTS "journal_entries";
DROP TABLE IF EXISTS "posting_batches";