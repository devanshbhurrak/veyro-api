--+ migration:000020_number_series_and_tax
--+ description: Rollback number series and tax registrations

-- Drop number series and tax registrations
DROP TABLE IF EXISTS "company_tax_registrations";
DROP TABLE IF EXISTS "document_number_series";