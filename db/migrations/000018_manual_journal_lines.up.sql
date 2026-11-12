--+ migration:000018_manual_journal_lines
--+ description: Manual journal lines table

-- manual_journal_lines: Manual accounting entries
CREATE TABLE IF NOT EXISTS "manual_journal_lines" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID NOT NULL,
  "line_number" INTEGER NOT NULL,
  "account_id" UUID NOT NULL,
  "party_id" UUID,
  "branch_id" UUID,
  "description" VARCHAR(300),
  "debit_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "credit_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for manual_journal_lines
ALTER TABLE "manual_journal_lines"
  ADD CONSTRAINT "manual_journal_lines_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "manual_journal_lines"
  ADD CONSTRAINT "manual_journal_lines_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "manual_journal_lines"
  ADD CONSTRAINT "manual_journal_lines_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

ALTER TABLE "manual_journal_lines"
  ADD CONSTRAINT "manual_journal_lines_account_id_ref"
  FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

