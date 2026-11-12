--+ migration:000015_inventory
--+ description: Stock ledger and balance projection tables

-- stock_movements: Immutable quantity and inventory-value movements
CREATE TABLE IF NOT EXISTS "stock_movements" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "posting_batch_id" UUID NOT NULL,
  "document_id" UUID NOT NULL,
  "document_line_id" UUID NOT NULL,
  "item_id" UUID NOT NULL,
  "warehouse_id" UUID NOT NULL,
  "movement_type" TEXT NOT NULL,
  "effective_date" DATE NOT NULL,
  "movement_sequence" BIGINT NOT NULL,
  "quantity_delta" NUMERIC(20,6) NOT NULL,
  "unit_cost" NUMERIC(20,8) NOT NULL,
  "value_delta" NUMERIC(20,2) NOT NULL,
  "related_movement_id" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for stock_movements
ALTER TABLE "stock_movements"
  ADD CONSTRAINT "stock_movements_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "stock_movements"
  ADD CONSTRAINT "stock_movements_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "stock_movements"
  ADD CONSTRAINT "stock_movements_posting_batch_id_ref"
  FOREIGN KEY ("posting_batch_id") REFERENCES "posting_batches" ("id");

ALTER TABLE "stock_movements"
  ADD CONSTRAINT "stock_movements_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

ALTER TABLE "stock_movements"
  ADD CONSTRAINT "stock_movements_document_line_id_ref"
  FOREIGN KEY ("document_line_id") REFERENCES "document_lines" ("id");

ALTER TABLE "stock_movements"
  ADD CONSTRAINT "stock_movements_item_id_ref"
  FOREIGN KEY ("item_id") REFERENCES "items" ("id");

ALTER TABLE "stock_movements"
  ADD CONSTRAINT "stock_movements_warehouse_id_ref"
  FOREIGN KEY ("warehouse_id") REFERENCES "warehouses" ("id");

-- stock_balances: Transactional projection of current stock
CREATE TABLE IF NOT EXISTS "stock_balances" (
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "item_id" UUID NOT NULL,
  "warehouse_id" UUID NOT NULL,
  "quantity_on_hand" NUMERIC(20,6) NOT NULL,
  "inventory_value" NUMERIC(20,2) NOT NULL,
  "last_movement_date" DATE,
  "last_movement_sequence" BIGINT NOT NULL,
  "row_version" BIGINT NOT NULL DEFAULT 1,
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY ("tenant_id", "company_id", "item_id", "warehouse_id")
);

-- Add composite foreign keys for stock_balances
ALTER TABLE "stock_balances"
  ADD CONSTRAINT "stock_balances_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "stock_balances"
  ADD CONSTRAINT "stock_balances_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "stock_balances"
  ADD CONSTRAINT "stock_balances_item_id_ref"
  FOREIGN KEY ("item_id") REFERENCES "items" ("id");

ALTER TABLE "stock_balances"
  ADD CONSTRAINT "stock_balances_warehouse_id_ref"
  FOREIGN KEY ("warehouse_id") REFERENCES "warehouses" ("id");

