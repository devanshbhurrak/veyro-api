--+ migration:000010_documents
--+ description: Rollback business document tables

-- Drop document-related tables
DROP TABLE IF EXISTS "document_lines";
DROP TABLE IF EXISTS "documents";