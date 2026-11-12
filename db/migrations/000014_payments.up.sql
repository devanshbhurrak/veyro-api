--+ migration:000014_payments
--+ description: Payment and allocation tables

-- payment_allocations: Payment application records
CREATE TABLE IF NOT EXISTS "payment_allocations" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "payment_document_id" UUID NOT NULL,
  "party_open_item_id" UUID NOT NULL,
  "posting_batch_id" UUID NOT NULL,
  "amount" NUMERIC(20,2) NOT NULL,
  "allocation_action" TEXT NOT NULL DEFAULT 'apply',
  "reverses_allocation_id" UUID,
  "created_by" UUID NOT NULL,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for payment_allocations
ALTER TABLE "payment_allocations"
  ADD CONSTRAINT "payment_allocations_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "payment_allocations"
  ADD CONSTRAINT "payment_allocations_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "payment_allocations"
  ADD CONSTRAINT "payment_allocations_payment_document_id_ref"
  FOREIGN KEY ("payment_document_id") REFERENCES "documents" ("id");

ALTER TABLE "payment_allocations"
  ADD CONSTRAINT "payment_allocations_party_open_item_id_ref"
  FOREIGN KEY ("party_open_item_id") REFERENCES "party_open_items" ("id");

ALTER TABLE "payment_allocations"
  ADD CONSTRAINT "payment_allocations_posting_batch_id_ref"
  FOREIGN KEY ("posting_batch_id") REFERENCES "posting_batches" ("id");

-- document_applications: Document application records
CREATE TABLE IF NOT EXISTS "document_applications" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "source_open_item_id" UUID NOT NULL,
  "target_open_item_id" UUID NOT NULL,
  "posting_batch_id" UUID NOT NULL,
  "amount" NUMERIC(20,2) NOT NULL,
  "application_action" TEXT NOT NULL DEFAULT 'apply',
  "reverses_application_id" UUID,
  "created_by" UUID NOT NULL,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for document_applications
ALTER TABLE "document_applications"
  ADD CONSTRAINT "document_applications_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_applications"
  ADD CONSTRAINT "document_applications_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_applications"
  ADD CONSTRAINT "document_applications_source_open_item_id_ref"
  FOREIGN KEY ("source_open_item_id") REFERENCES "party_open_items" ("id");

ALTER TABLE "document_applications"
  ADD CONSTRAINT "document_applications_target_open_item_id_ref"
  FOREIGN KEY ("target_open_item_id") REFERENCES "party_open_items" ("id");

ALTER TABLE "document_applications"
  ADD CONSTRAINT "document_applications_posting_batch_id_ref"
  FOREIGN KEY ("posting_batch_id") REFERENCES "posting_batches" ("id");

-- cash_bank_accounts: Cash and bank account details
CREATE TABLE IF NOT EXISTS "cash_bank_accounts" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "account_id" UUID NOT NULL,
  "account_kind" TEXT NOT NULL,
  "display_name" VARCHAR(120) NOT NULL,
  "bank_name" VARCHAR(120),
  "account_holder_name" VARCHAR(160),
  "account_number_encrypted" BYTEA,
  "account_number_key_version" VARCHAR(80),
  "account_number_last4" VARCHAR(4),
  "ifsc_code" VARCHAR(11),
  "upi_handle" VARCHAR(100),
  "is_default" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for cash_bank_accounts
ALTER TABLE "cash_bank_accounts"
  ADD CONSTRAINT "cash_bank_accounts_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "cash_bank_accounts"
  ADD CONSTRAINT "cash_bank_accounts_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "cash_bank_accounts"
  ADD CONSTRAINT "cash_bank_accounts_account_id_ref"
  FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

