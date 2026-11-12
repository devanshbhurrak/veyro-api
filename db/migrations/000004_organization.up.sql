--+ migration:000004_organization
--+ description: Company and organization structure tables

-- companies: Company settings and accounting configuration
CREATE TABLE IF NOT EXISTS "companies" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "trade_name" VARCHAR(180) NOT NULL,
  "legal_name" VARCHAR(220),
  "pan" VARCHAR(10),
  "base_currency" VARCHAR(3) NOT NULL DEFAULT 'INR',
  "timezone" TEXT NOT NULL DEFAULT 'Asia/Kolkata',
  "business_type" TEXT,
  "gst_profile" TEXT NOT NULL DEFAULT 'not_registered',
  "email" CITEXT,
  "phone_e164" VARCHAR(16),
  "website" TEXT,
  "address_line1" VARCHAR(180),
  "address_line2" VARCHAR(180),
  "city" VARCHAR(100),
  "district" VARCHAR(100),
  "state_code" VARCHAR(2),
  "postal_code" VARCHAR(12),
  "country_code" VARCHAR(2) NOT NULL DEFAULT 'IN',
  "logo_asset_id" UUID,
  "is_primary" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for companies
ALTER TABLE "companies"
  ADD CONSTRAINT "companies_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

-- Add partial unique index for active primary company
CREATE UNIQUE INDEX IF NOT EXISTS "companies_tenant_active_primary_idx"
  ON "companies" ("tenant_id")
  WHERE "is_primary" = true AND "status" = 'active';

-- branches: Company operational locations
CREATE TABLE IF NOT EXISTS "branches" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "code" VARCHAR(40) NOT NULL,
  "name" VARCHAR(140) NOT NULL,
  "address_line1" VARCHAR(180),
  "address_line2" VARCHAR(180),
  "city" VARCHAR(100),
  "district" VARCHAR(100),
  "state_code" VARCHAR(2),
  "postal_code" VARCHAR(12),
  "country_code" VARCHAR(2) NOT NULL DEFAULT 'IN',
  "phone_e164" VARCHAR(16),
  "default_tax_registration_id" UUID,
  "is_default" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for branches
ALTER TABLE "branches"
  ADD CONSTRAINT "branches_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "branches"
  ADD CONSTRAINT "branches_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- Add composite tenant/company/branch unique keys
CREATE UNIQUE INDEX IF NOT EXISTS "branches_tenant_company_id_unique_idx"
  ON "branches" ("tenant_id", "company_id", "id")
  WHERE "id" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "branches_tenant_company_code_unique_idx"
  ON "branches" ("tenant_id", "company_id", "code")
  WHERE "code" IS NOT NULL;

-- warehouses: Stock storage locations
CREATE TABLE IF NOT EXISTS "warehouses" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "branch_id" UUID NOT NULL,
  "code" VARCHAR(40) NOT NULL,
  "name" VARCHAR(140) NOT NULL,
  "warehouse_kind" TEXT NOT NULL DEFAULT 'physical',
  "is_default" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for warehouses
ALTER TABLE "warehouses"
  ADD CONSTRAINT "warehouses_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "warehouses"
  ADD CONSTRAINT "warehouses_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "warehouses"
  ADD CONSTRAINT "warehouses_branch_id_ref"
  FOREIGN KEY ("branch_id") REFERENCES "branches" ("id");

-- Add composite unique keys for warehouses
CREATE UNIQUE INDEX IF NOT EXISTS "warehouses_tenant_company_id_unique_idx"
  ON "warehouses" ("tenant_id", "company_id", "id")
  WHERE "id" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "warehouses_tenant_company_branch_id_unique_idx"
  ON "warehouses" ("tenant_id", "company_id", "branch_id", "id")
  WHERE "id" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "warehouses_tenant_company_code_unique_idx"
  ON "warehouses" ("tenant_id", "company_id", "code")
  WHERE "code" IS NOT NULL;

