--+ migration:000010_documents
--+ description: Business document tables

-- documents: Document headers
CREATE TABLE IF NOT EXISTS "documents" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "branch_id" UUID NOT NULL,
  "document_type" TEXT NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'draft',
  "financial_year_id" UUID,
  "number_series_id" UUID,
  "document_number" VARCHAR(80),
  "document_template_version_id" UUID,
  "document_date" DATE NOT NULL,
  "document_time" TIME,
  "due_date" DATE,
  "party_id" UUID,
  "currency_code" VARCHAR(3) NOT NULL DEFAULT 'INR',
  "items_subtotal" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "line_discount_total" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "document_discount_total" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "charges_subtotal" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "taxable_total" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "non_taxable_total" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "tax_total" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "round_off_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "grand_total" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "reference_number" VARCHAR(100),
  "notes" TEXT,
  "terms" TEXT,
  "created_by" UUID NOT NULL,
  "updated_by" UUID,
  "posted_by" UUID,
  "posted_at" TIMESTAMPTZ,
  "voided_by" UUID,
  "voided_at" TIMESTAMPTZ,
  "void_reason" TEXT,
  "row_version" BIGINT NOT NULL DEFAULT 1,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY ("created_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("updated_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("posted_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("voided_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for documents
ALTER TABLE "documents"
  ADD CONSTRAINT "documents_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "documents"
  ADD CONSTRAINT "documents_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "documents"
  ADD CONSTRAINT "documents_branch_id_ref"
  FOREIGN KEY ("branch_id") REFERENCES "branches" ("id");

-- Add unique constraint for document number
CREATE UNIQUE INDEX IF NOT EXISTS "documents_tenant_company_document_number_idx"
  ON "documents" ("tenant_id", "company_id", "document_number")
  WHERE "document_number" IS NOT NULL;

-- document_lines: Commercial line items
CREATE TABLE IF NOT EXISTS "document_lines" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID NOT NULL,
  "line_number" INTEGER NOT NULL,
  "line_type" TEXT NOT NULL,
  "item_id" UUID,
  "description_snapshot" VARCHAR(500) NOT NULL,
  "sku_snapshot" VARCHAR(80),
  "hsn_sac_snapshot" VARCHAR(12),
  "unit_id" UUID,
  "unit_name_snapshot" VARCHAR(40),
  "quantity" NUMERIC(20,6) NOT NULL DEFAULT 1,
  "free_quantity" NUMERIC(20,6) NOT NULL DEFAULT 0,
  "conversion_to_base_snapshot" NUMERIC(20,8),
  "base_quantity" NUMERIC(20,6),
  "warehouse_id" UUID,
  "unit_rate" NUMERIC(20,6) NOT NULL DEFAULT 0,
  "price_is_tax_inclusive" BOOLEAN NOT NULL DEFAULT false,
  "mrp_snapshot" NUMERIC(20,6),
  "discount_type" TEXT,
  "discount_rate" NUMERIC(9,6),
  "gross_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "discount_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "document_discount_allocation" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "taxable_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "non_taxable_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "tax_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "line_total" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "posting_account_id" UUID,
  "stock_adjustment_direction" TEXT,
  "stock_disposition" TEXT,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add composite foreign keys for document_lines
ALTER TABLE "document_lines"
  ADD CONSTRAINT "document_lines_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_lines"
  ADD CONSTRAINT "document_lines_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_lines"
  ADD CONSTRAINT "document_lines_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

-- Add unique constraint for line numbers
CREATE UNIQUE INDEX IF NOT EXISTS "document_lines_tenant_company_document_line_number_idx"
  ON "document_lines" ("tenant_id", "company_id", "document_id", "line_number")
  WHERE "document_id" IS NOT NULL AND "line_number" IS NOT NULL;

