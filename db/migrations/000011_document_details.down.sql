--+ migration:000011_document_details
--+ description: Rollback document detail tables and snapshots

-- Drop document detail tables
DROP TABLE IF EXISTS "purchase_bill_details";
DROP TABLE IF EXISTS "sales_invoice_details";
DROP TABLE IF EXISTS "document_line_tax_components";
DROP TABLE IF EXISTS "document_party_snapshots";