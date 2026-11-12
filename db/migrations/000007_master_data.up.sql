--+ migration:000007_master_data
--+ description: Master data tables (units, item categories)

-- item_categories: Item categorization
CREATE TABLE IF NOT EXISTS "item_categories" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "parent_category_id" UUID,
  "code" VARCHAR(50),
  "name" VARCHAR(120) NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for item_categories
ALTER TABLE "item_categories"
  ADD CONSTRAINT "item_categories_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "item_categories"
  ADD CONSTRAINT "item_categories_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "item_categories"
  ADD CONSTRAINT "item_categories_parent_category_id_ref"
  FOREIGN KEY ("parent_category_id") REFERENCES "item_categories" ("id");

-- Add unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS "item_categories_tenant_company_id_unique_idx"
  ON "item_categories" ("tenant_id", "company_id", "id")
  WHERE "id" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "item_categories_tenant_company_name_unique_idx"
  ON "item_categories" ("tenant_id", "company_id", "name")
  WHERE "name" IS NOT NULL;

-- units: Measurement units
CREATE TABLE IF NOT EXISTS "units" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "unit_code" VARCHAR(30) NOT NULL,
  "name" VARCHAR(80) NOT NULL,
  "symbol" VARCHAR(20) NOT NULL,
  "precision_scale" SMALLINT NOT NULL,
  "is_system_seed" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add composite foreign keys for units
ALTER TABLE "units"
  ADD CONSTRAINT "units_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "units"
  ADD CONSTRAINT "units_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- Add unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS "units_tenant_company_id_unique_idx"
  ON "units" ("tenant_id", "company_id", "id")
  WHERE "id" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "units_tenant_company_unit_code_unique_idx"
  ON "units" ("tenant_id", "company_id", "unit_code")
  WHERE "unit_code" IS NOT NULL;

