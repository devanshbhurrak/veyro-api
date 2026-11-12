--+ migration:000005_media
--+ description: Rollback media assets

-- Drop media assets
DROP TABLE IF EXISTS "media_assets";

-- Note: companies, branches, tenants and other parent tables will be dropped by later migrations