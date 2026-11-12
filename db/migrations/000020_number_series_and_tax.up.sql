--+ migration:000020_number_series_and_tax
--+ description: Document number series and company tax registrations

-- document_number_series: Document numbering configuration
CREATE TABLE IF NOT EXISTS "document_number_series" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_type" TEXT NOT NULL,
  "financial_year_id" UUID NOT NULL,
  "branch_id" UUID,
  "tax_registration_id" UUID,
  "prefix" VARCHAR(40) NOT NULL,
  "suffix" VARCHAR(40) NOT NULL,
  "padding_width" SMALLINT NOT NULL,
  "next_number" BIGINT NOT NULL,
  "is_default" BOOLEAN NOT NULL DEFAULT false,
  "is_active" BOOLEAN NOT NULL DEFAULT true,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for document_number_series
ALTER TABLE "document_number_series"
  ADD CONSTRAINT "document_number_series_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_number_series"
  ADD CONSTRAINT "document_number_series_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_number_series"
  ADD CONSTRAINT "document_number_series_financial_year_id_ref"
  FOREIGN KEY ("financial_year_id") REFERENCES "financial_years" ("id");

-- company_tax_registrations: GSTIN registrations
CREATE TABLE IF NOT EXISTS "company_tax_registrations" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "gstin" VARCHAR(15) NOT NULL,
  "state_code" VARCHAR(2) NOT NULL,
  "registration_type" TEXT NOT NULL,
  "legal_name" VARCHAR(220),
  "trade_name" VARCHAR(180),
  "address_line1" VARCHAR(180),
  "address_line2" VARCHAR(180),
  "city" VARCHAR(100),
  "district" VARCHAR(100),
  "postal_code" VARCHAR(12),
  "country_code" VARCHAR(2) NOT NULL DEFAULT 'IN',
  "valid_from" DATE NOT NULL,
  "valid_to" DATE,
  "is_default" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for company_tax_registrations
ALTER TABLE "company_tax_registrations"
  ADD CONSTRAINT "company_tax_registrations_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "company_tax_registrations"
  ADD CONSTRAINT "company_tax_registrations_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- Add unique constraint for active GSTIN
CREATE UNIQUE INDEX IF NOT EXISTS "company_tax_registrations_tenant_company_gstin_idx"
  ON "company_tax_registrations" ("tenant_id", "company_id", "gstin")
  WHERE "gstin" IS NOT NULL AND "status" = 'active';

