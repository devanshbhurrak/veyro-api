--+ migration:000007_master_data
--+ description: Rollback master data tables

-- Drop units and categories
DROP TABLE IF EXISTS "units";
DROP TABLE IF EXISTS "item_categories";