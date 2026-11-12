--+ migration:000019_company_settings
--+ description: Company settings and financial years

-- company_settings: Company business defaults
CREATE TABLE IF NOT EXISTS "company_settings" (
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "default_branch_id" UUID NOT NULL,
  "default_warehouse_id" UUID NOT NULL,
  "default_cash_bank_account_id" UUID,
  "default_price_includes_tax" BOOLEAN NOT NULL DEFAULT false,
  "negative_stock_policy" TEXT NOT NULL DEFAULT 'block',
  "inventory_cost_method" TEXT NOT NULL DEFAULT 'moving_average',
  "default_credit_days" INTEGER NOT NULL DEFAULT 0,
  "credit_limit_action" TEXT NOT NULL DEFAULT 'warn',
  "amount_display_scale" SMALLINT NOT NULL DEFAULT 2,
  "quantity_display_scale" SMALLINT NOT NULL DEFAULT 2,
  "date_format" VARCHAR(20) NOT NULL DEFAULT 'dd/MM/yyyy',
  "locale" VARCHAR(12) NOT NULL DEFAULT 'en-IN',
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  PRIMARY KEY ("tenant_id", "company_id")
);

-- Add composite foreign keys for company_settings
ALTER TABLE "company_settings"
  ADD CONSTRAINT "company_settings_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "company_settings"
  ADD CONSTRAINT "company_settings_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "company_settings"
  ADD CONSTRAINT "company_settings_default_branch_id_ref"
  FOREIGN KEY ("default_branch_id") REFERENCES "branches" ("id");

ALTER TABLE "company_settings"
  ADD CONSTRAINT "company_settings_default_warehouse_id_ref"
  FOREIGN KEY ("default_warehouse_id") REFERENCES "warehouses" ("id");

-- financial_years: Company accounting periods
CREATE TABLE IF NOT EXISTS "financial_years" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "name" VARCHAR(40) NOT NULL,
  "start_date" DATE NOT NULL,
  "end_date" DATE NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'open',
  "is_current" BOOLEAN NOT NULL DEFAULT false,
  "closed_at" TIMESTAMPTZ,
  "closed_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("closed_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for financial_years
ALTER TABLE "financial_years"
  ADD CONSTRAINT "financial_years_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "financial_years"
  ADD CONSTRAINT "financial_years_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- Add unique constraint for financial year name
CREATE UNIQUE INDEX IF NOT EXISTS "financial_years_tenant_company_name_idx"
  ON "financial_years" ("tenant_id", "company_id", "name")
  WHERE "name" IS NOT NULL;

-- accounting_periods: Month/period-level posting locks
CREATE TABLE IF NOT EXISTS "accounting_periods" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "financial_year_id" UUID NOT NULL,
  "period_number" SMALLINT NOT NULL,
  "name" VARCHAR(40) NOT NULL,
  "start_date" DATE NOT NULL,
  "end_date" DATE NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'open',
  "locked_at" TIMESTAMPTZ,
  "locked_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("locked_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for accounting_periods
ALTER TABLE "accounting_periods"
  ADD CONSTRAINT "accounting_periods_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "accounting_periods"
  ADD CONSTRAINT "accounting_periods_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "accounting_periods"
  ADD CONSTRAINT "accounting_periods_financial_year_id_ref"
  FOREIGN KEY ("financial_year_id") REFERENCES "financial_years" ("id");

