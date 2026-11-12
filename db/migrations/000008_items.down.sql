--+ migration:000008_items
--+ description: Rollback items and related tables

-- Drop items and related tables
DROP TABLE IF EXISTS "item_barcodes";
DROP TABLE IF EXISTS "item_unit_conversions";
DROP TABLE IF EXISTS "items";