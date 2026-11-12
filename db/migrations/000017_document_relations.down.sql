--+ migration:000017_document_relations
--+ description: Rollback document relations and attachments

-- Drop document relation tables
DROP TABLE IF EXISTS "document_attachments";
DROP TABLE IF EXISTS "document_line_relations";
DROP TABLE IF EXISTS "document_relations";