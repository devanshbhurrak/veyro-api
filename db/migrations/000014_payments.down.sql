--+ migration:000014_payments
--+ description: Rollback payment and allocation tables

-- Drop payment-related tables
DROP TABLE IF EXISTS "cash_bank_accounts";
DROP TABLE IF EXISTS "document_applications";
DROP TABLE IF EXISTS "payment_allocations";