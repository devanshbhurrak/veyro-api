--+ migration:000009_tax_masters
--+ description: Tax code masters and components

-- tax_codes: Company tax-rule identifiers
CREATE TABLE IF NOT EXISTS "tax_codes" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "code" VARCHAR(40) NOT NULL,
  "name" VARCHAR(100) NOT NULL,
  "tax_family" TEXT NOT NULL DEFAULT 'gst',
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add composite foreign keys for tax_codes
ALTER TABLE "tax_codes"
  ADD CONSTRAINT "tax_codes_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "tax_codes"
  ADD CONSTRAINT "tax_codes_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- Add unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS "tax_codes_tenant_company_code_unique_idx"
  ON "tax_codes" ("tenant_id", "company_id", "code")
  WHERE "code" IS NOT NULL;

-- tax_code_versions: Effective-dated versions of tax codes
CREATE TABLE IF NOT EXISTS "tax_code_versions" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "tax_code_id" UUID NOT NULL,
  "version_number" INTEGER NOT NULL,
  "tax_treatment" TEXT NOT NULL,
  "valid_from" DATE NOT NULL,
  "valid_to" DATE,
  "description" TEXT,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add composite foreign keys for tax_code_versions
ALTER TABLE "tax_code_versions"
  ADD CONSTRAINT "tax_code_versions_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "tax_code_versions"
  ADD CONSTRAINT "tax_code_versions_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "tax_code_versions"
  ADD CONSTRAINT "tax_code_versions_tax_code_id_ref"
  FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id");

-- Add unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS "tax_code_versions_tenant_company_version_number_idx"
  ON "tax_code_versions" ("tenant_id", "company_id", "tax_code_id", "version_number")
  WHERE "tax_code_id" IS NOT NULL AND "version_number" IS NOT NULL;

-- tax_code_components: Component rates for each tax-code version
CREATE TABLE IF NOT EXISTS "tax_code_components" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "tax_code_version_id" UUID NOT NULL,
  "supply_scope" TEXT NOT NULL,
  "component_code" VARCHAR(20) NOT NULL,
  "component_name" VARCHAR(60) NOT NULL,
  "calculation_method" TEXT NOT NULL DEFAULT 'percentage',
  "rate_percentage" NUMERIC(9,6),
  "amount_per_unit" NUMERIC(20,6),
  "unit_id" UUID,
  "calculation_order" SMALLINT NOT NULL
);

-- Add composite foreign keys for tax_code_components
ALTER TABLE "tax_code_components"
  ADD CONSTRAINT "tax_code_components_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "tax_code_components"
  ADD CONSTRAINT "tax_code_components_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "tax_code_components"
  ADD CONSTRAINT "tax_code_components_tax_code_version_id_ref"
  FOREIGN KEY ("tax_code_version_id") REFERENCES "tax_code_versions" ("id");

ALTER TABLE "tax_code_components"
  ADD CONSTRAINT "tax_code_components_unit_id_ref"
  FOREIGN KEY ("unit_id") REFERENCES "units" ("id");

-- Add unique constraint
CREATE UNIQUE INDEX IF NOT EXISTS "tax_code_components_tenant_company_scope_code_idx"
  ON "tax_code_components" ("tenant_id", "company_id", "tax_code_version_id", "supply_scope", "component_code")
  WHERE "tax_code_version_id" IS NOT NULL AND "supply_scope" IS NOT NULL AND "component_code" IS NOT NULL;

