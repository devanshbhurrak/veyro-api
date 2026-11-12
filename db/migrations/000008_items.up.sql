--+ migration:000008_items
--+ description: Item masters and related tables

-- items: Products and services
CREATE TABLE IF NOT EXISTS "items" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "item_type" TEXT NOT NULL,
  "sku" VARCHAR(80),
  "name" VARCHAR(180) NOT NULL,
  "description" TEXT,
  "category_id" UUID,
  "base_unit_id" UUID NOT NULL,
  "hsn_sac_code" VARCHAR(12),
  "default_sales_rate" NUMERIC(20,6),
  "default_purchase_rate" NUMERIC(20,6),
  "default_tax_code_id" UUID,
  "track_inventory" BOOLEAN NOT NULL DEFAULT false,
  "is_sellable" BOOLEAN NOT NULL DEFAULT true,
  "is_purchasable" BOOLEAN NOT NULL DEFAULT true,
  "reorder_level" NUMERIC(20,6),
  "sales_account_id" UUID,
  "purchase_account_id" UUID,
  "inventory_account_id" UUID,
  "cogs_account_id" UUID,
  "image_asset_id" UUID,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_by" UUID,
  "updated_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("updated_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for items
ALTER TABLE "items"
  ADD CONSTRAINT "items_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "items"
  ADD CONSTRAINT "items_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "items"
  ADD CONSTRAINT "items_base_unit_id_ref"
  FOREIGN KEY ("base_unit_id") REFERENCES "units" ("id");

-- Add partial unique index for SKU
CREATE UNIQUE INDEX IF NOT EXISTS "items_tenant_company_sku_idx"
  ON "items" ("tenant_id", "company_id", "sku")
  WHERE "sku" IS NOT NULL AND "status" = 'active';

-- item_unit_conversions: Item-specific unit conversions
CREATE TABLE IF NOT EXISTS "item_unit_conversions" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "item_id" UUID NOT NULL,
  "unit_id" UUID NOT NULL,
  "conversion_to_base" NUMERIC(20,8) NOT NULL,
  "is_default_sales_unit" BOOLEAN NOT NULL DEFAULT false,
  "is_default_purchase_unit" BOOLEAN NOT NULL DEFAULT false,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add composite foreign keys for conversions
ALTER TABLE "item_unit_conversions"
  ADD CONSTRAINT "item_unit_conversions_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "item_unit_conversions"
  ADD CONSTRAINT "item_unit_conversions_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "item_unit_conversions"
  ADD CONSTRAINT "item_unit_conversions_item_id_ref"
  FOREIGN KEY ("item_id") REFERENCES "items" ("id");

ALTER TABLE "item_unit_conversions"
  ADD CONSTRAINT "item_unit_conversions_unit_id_ref"
  FOREIGN KEY ("unit_id") REFERENCES "units" ("id");

-- Add unique constraint
CREATE UNIQUE INDEX IF NOT EXISTS "item_unit_conversions_tenant_company_item_unit_idx"
  ON "item_unit_conversions" ("tenant_id", "company_id", "item_id", "unit_id")
  WHERE "item_id" IS NOT NULL AND "unit_id" IS NOT NULL;

-- item_barcodes: Item barcode mappings
CREATE TABLE IF NOT EXISTS "item_barcodes" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "item_id" UUID NOT NULL,
  "barcode_value" VARCHAR(80) NOT NULL,
  "barcode_type" TEXT NOT NULL DEFAULT 'internal',
  "is_primary" BOOLEAN NOT NULL DEFAULT false,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "created_by" UUID,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for barcodes
ALTER TABLE "item_barcodes"
  ADD CONSTRAINT "item_barcodes_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "item_barcodes"
  ADD CONSTRAINT "item_barcodes_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "item_barcodes"
  ADD CONSTRAINT "item_barcodes_item_id_ref"
  FOREIGN KEY ("item_id") REFERENCES "items" ("id");

-- Add unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS "item_barcodes_tenant_company_barcode_idx"
  ON "item_barcodes" ("tenant_id", "company_id", "barcode_value")
  WHERE "barcode_value" IS NOT NULL;

