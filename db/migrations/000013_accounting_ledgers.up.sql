--+ migration:000013_accounting_ledgers
--+ description: Accounting and settlement ledger tables

-- posting_batches: Atomic posting operations
CREATE TABLE IF NOT EXISTS "posting_batches" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "source_document_id" UUID NOT NULL,
  "posting_kind" TEXT NOT NULL,
  "posting_date" DATE NOT NULL,
  "posted_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "posted_by" UUID NOT NULL,
  "reverses_posting_batch_id" UUID,
  "request_id" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("posted_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for posting_batches
ALTER TABLE "posting_batches"
  ADD CONSTRAINT "posting_batches_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "posting_batches"
  ADD CONSTRAINT "posting_batches_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "posting_batches"
  ADD CONSTRAINT "posting_batches_source_document_id_ref"
  FOREIGN KEY ("source_document_id") REFERENCES "documents" ("id");

-- journal_entries: Journal entries
CREATE TABLE IF NOT EXISTS "journal_entries" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "posting_batch_id" UUID NOT NULL UNIQUE,
  "memo" TEXT,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for journal_entries
ALTER TABLE "journal_entries"
  ADD CONSTRAINT "journal_entries_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "journal_entries"
  ADD CONSTRAINT "journal_entries_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "journal_entries"
  ADD CONSTRAINT "journal_entries_posting_batch_id_ref"
  FOREIGN KEY ("posting_batch_id") REFERENCES "posting_batches" ("id");

-- journal_lines: Journal line details
CREATE TABLE IF NOT EXISTS "journal_lines" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "journal_entry_id" UUID NOT NULL,
  "line_number" INTEGER NOT NULL,
  "account_id" UUID NOT NULL,
  "party_id" UUID,
  "branch_id" UUID,
  "document_line_id" UUID,
  "source_manual_journal_line_id" UUID,
  "description" VARCHAR(300),
  "debit_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "credit_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for journal_lines
ALTER TABLE "journal_lines"
  ADD CONSTRAINT "journal_lines_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "journal_lines"
  ADD CONSTRAINT "journal_lines_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "journal_lines"
  ADD CONSTRAINT "journal_lines_journal_entry_id_ref"
  FOREIGN KEY ("journal_entry_id") REFERENCES "journal_entries" ("id");

ALTER TABLE "journal_lines"
  ADD CONSTRAINT "journal_lines_account_id_ref"
  FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

-- party_open_items: Outstanding receivables/payables
CREATE TABLE IF NOT EXISTS "party_open_items" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID NOT NULL UNIQUE,
  "party_id" UUID NOT NULL,
  "settlement_side" TEXT NOT NULL,
  "open_item_kind" TEXT NOT NULL,
  "polarity" SMALLINT NOT NULL,
  "original_amount" NUMERIC(20,2) NOT NULL,
  "remaining_amount" NUMERIC(20,2) NOT NULL,
  "due_date" DATE,
  "control_account_id" UUID NOT NULL,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for party_open_items
ALTER TABLE "party_open_items"
  ADD CONSTRAINT "party_open_items_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "party_open_items"
  ADD CONSTRAINT "party_open_items_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "party_open_items"
  ADD CONSTRAINT "party_open_items_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

ALTER TABLE "party_open_items"
  ADD CONSTRAINT "party_open_items_party_id_ref"
  FOREIGN KEY ("party_id") REFERENCES "parties" ("id");

ALTER TABLE "party_open_items"
  ADD CONSTRAINT "party_open_items_control_account_id_ref"
  FOREIGN KEY ("control_account_id") REFERENCES "accounts" ("id");

