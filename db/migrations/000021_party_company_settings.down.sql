--+ migration:000021_party_company_settings
--+ description: Rollback party company settings and additional party tables

-- Drop party-related tables
DROP TABLE IF EXISTS "party_tax_registrations";
DROP TABLE IF EXISTS "party_addresses";
DROP TABLE IF EXISTS "party_company_settings";