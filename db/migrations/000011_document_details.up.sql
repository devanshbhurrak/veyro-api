--+ migration:000011_document_details
--+ description: Document detail tables and snapshots

-- document_party_snapshots: Party information snapshots
CREATE TABLE IF NOT EXISTS "document_party_snapshots" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID NOT NULL,
  "snapshot_role" TEXT NOT NULL,
  "party_id" UUID,
  "company_tax_registration_id" UUID,
  "party_tax_registration_id" UUID,
  "display_name" VARCHAR(180) NOT NULL,
  "legal_name" VARCHAR(220),
  "gstin" VARCHAR(15),
  "pan" VARCHAR(10),
  "phone" VARCHAR(30),
  "email" VARCHAR(254),
  "address_line1" VARCHAR(180),
  "address_line2" VARCHAR(180),
  "city" VARCHAR(100),
  "district" VARCHAR(100),
  "state_code" VARCHAR(2),
  "postal_code" VARCHAR(12),
  "country_code" VARCHAR(2) NOT NULL DEFAULT 'IN',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add composite foreign keys for document_party_snapshots
ALTER TABLE "document_party_snapshots"
  ADD CONSTRAINT "document_party_snapshots_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_party_snapshots"
  ADD CONSTRAINT "document_party_snapshots_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_party_snapshots"
  ADD CONSTRAINT "document_party_snapshots_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

-- document_line_tax_components: Tax components for document lines
CREATE TABLE IF NOT EXISTS "document_line_tax_components" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID NOT NULL,
  "document_line_id" UUID NOT NULL,
  "tax_code_id" UUID,
  "tax_code_version_id" UUID,
  "tax_treatment_snapshot" TEXT NOT NULL,
  "component_code" VARCHAR(20) NOT NULL,
  "component_name_snapshot" VARCHAR(60) NOT NULL,
  "calculation_method_snapshot" TEXT NOT NULL,
  "taxable_base_amount" NUMERIC(20,2),
  "rate_percentage" NUMERIC(9,6),
  "amount_per_unit" NUMERIC(20,6),
  "calculation_quantity" NUMERIC(20,6),
  "calculation_unit_name" VARCHAR(40),
  "tax_amount" NUMERIC(20,2) NOT NULL,
  "input_tax_credit_status" TEXT NOT NULL DEFAULT 'not_applicable',
  "eligible_input_tax_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add composite foreign keys for document_line_tax_components
ALTER TABLE "document_line_tax_components"
  ADD CONSTRAINT "document_line_tax_components_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_line_tax_components"
  ADD CONSTRAINT "document_line_tax_components_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_line_tax_components"
  ADD CONSTRAINT "document_line_tax_components_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

ALTER TABLE "document_line_tax_components"
  ADD CONSTRAINT "document_line_tax_components_document_line_id_ref"
  FOREIGN KEY ("document_line_id") REFERENCES "document_lines" ("id");

-- sales_invoice_details: Sales invoice-specific data
CREATE TABLE IF NOT EXISTS "sales_invoice_details" (
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID,
  "seller_tax_registration_id" UUID,
  "buyer_tax_registration_id" UUID,
  "place_of_supply_state_code" VARCHAR(2),
  "supply_type" TEXT NOT NULL,
  "invoice_category" TEXT NOT NULL,
  "reverse_charge" BOOLEAN NOT NULL DEFAULT false,
  "supply_classification" TEXT,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY ("tenant_id", "company_id", "document_id")
);

-- Add composite foreign keys for sales_invoice_details
ALTER TABLE "sales_invoice_details"
  ADD CONSTRAINT "sales_invoice_details_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "sales_invoice_details"
  ADD CONSTRAINT "sales_invoice_details_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "sales_invoice_details"
  ADD CONSTRAINT "sales_invoice_details_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

-- purchase_bill_details: Purchase bill-specific data
CREATE TABLE IF NOT EXISTS "purchase_bill_details" (
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID,
  "company_tax_registration_id" UUID,
  "supplier_tax_registration_id" UUID,
  "supplier_invoice_number" VARCHAR(100),
  "supplier_invoice_date" DATE,
  "place_of_supply_state_code" VARCHAR(2),
  "reverse_charge" BOOLEAN NOT NULL DEFAULT false,
  "itc_review_status" TEXT NOT NULL DEFAULT 'pending',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY ("tenant_id", "company_id", "document_id")
);

-- Add composite foreign keys for purchase_bill_details
ALTER TABLE "purchase_bill_details"
  ADD CONSTRAINT "purchase_bill_details_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "purchase_bill_details"
  ADD CONSTRAINT "purchase_bill_details_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "purchase_bill_details"
  ADD CONSTRAINT "purchase_bill_details_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

