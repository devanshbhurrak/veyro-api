--+ migration:000006_accounts
--+ description: Account management tables for chart of accounts

-- accounts: Chart of accounts
CREATE TABLE IF NOT EXISTS "accounts" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "account_code" VARCHAR(40) NOT NULL,
  "system_code" VARCHAR(60),
  "name" VARCHAR(160) NOT NULL,
  "account_type" TEXT NOT NULL,
  "account_subtype" TEXT,
  "parent_account_id" UUID,
  "normal_balance" TEXT NOT NULL,
  "is_control_account" BOOLEAN NOT NULL DEFAULT false,
  "control_account_role" TEXT,
  "allow_manual_posting" BOOLEAN NOT NULL DEFAULT true,
  "is_system_account" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for accounts
ALTER TABLE "accounts"
  ADD CONSTRAINT "accounts_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "accounts"
  ADD CONSTRAINT "accounts_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "accounts"
  ADD CONSTRAINT "accounts_parent_account_id_ref"
  FOREIGN KEY ("parent_account_id") REFERENCES "accounts" ("id");

-- Add unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS "accounts_tenant_company_id_unique_idx"
  ON "accounts" ("tenant_id", "company_id", "id")
  WHERE "id" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "accounts_tenant_company_code_unique_idx"
  ON "accounts" ("tenant_id", "company_id", "account_code")
  WHERE "account_code" IS NOT NULL;

