--+ migration:000009_tax_masters
--+ description: Rollback tax code masters and components

-- Drop tax-related tables
DROP TABLE IF EXISTS "tax_code_components";
DROP TABLE IF EXISTS "tax_code_versions";
DROP TABLE IF EXISTS "tax_codes";