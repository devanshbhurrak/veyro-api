--+ migration:000003_parties
--+ description: Rollback party and group tables

-- Drop party roles
DROP TABLE IF EXISTS "party_roles";
DROP TABLE IF EXISTS "party_groups";
DROP TABLE IF EXISTS "parties";

-- Note: companies table will be dropped by next migration
-- Note: tenants table will be dropped by next migration