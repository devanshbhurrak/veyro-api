--+ migration:000006_accounts
--+ description: Rollback account management tables

-- Drop accounts
DROP TABLE IF EXISTS "accounts";

-- Note: companies, branches, tenants and other parent tables will be dropped by later migrations