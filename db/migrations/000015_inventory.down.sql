--+ migration:000015_inventory
--+ description: Rollback stock ledger and balance projection tables

-- Drop inventory-related tables
DROP TABLE IF EXISTS "stock_balances";
DROP TABLE IF EXISTS "stock_movements";