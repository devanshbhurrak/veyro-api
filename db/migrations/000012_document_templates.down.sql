--+ migration:000012_document_templates
--+ description: Rollback document template tables

-- Drop template-related tables
DROP TABLE IF EXISTS "document_template_versions";
DROP TABLE IF EXISTS "document_templates";