--+ migration:000019_company_settings
--+ description: Rollback company settings and financial years

-- Drop company settings and financial years
DROP TABLE IF EXISTS "accounting_periods";
DROP TABLE IF EXISTS "financial_years";
DROP TABLE IF EXISTS "company_settings";