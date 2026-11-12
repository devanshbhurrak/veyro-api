--+ migration:000022_phase3_identity_tenancy
--+ description: Phase 3 additions for invitations, sessions, deferred constraints and permission catalogue

-- ---------------------------------------------------------------------------
-- Session access tokens.
-- Access tokens are opaque, high-entropy values. Only their SHA-256 hash is
-- stored. Refresh credentials continue to use refresh_token_hash.
-- ---------------------------------------------------------------------------
ALTER TABLE "user_sessions"
  ADD COLUMN IF NOT EXISTS "access_token_hash" BYTEA;

ALTER TABLE "user_sessions"
  ADD COLUMN IF NOT EXISTS "access_expires_at" TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS "user_sessions_access_token_hash_idx"
  ON "user_sessions" ("access_token_hash")
  WHERE "access_token_hash" IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Identity notifications (verification, password reset) are global and do not
-- belong to a tenant. Allow outbox rows without tenant scope.
-- ---------------------------------------------------------------------------
ALTER TABLE "outbox_events" ALTER COLUMN "tenant_id" DROP NOT NULL;

-- A deduplication key is unique across the outbox.
CREATE UNIQUE INDEX IF NOT EXISTS "outbox_events_deduplication_key_idx"
  ON "outbox_events" ("deduplication_key");

-- ---------------------------------------------------------------------------
-- Workspace creation happens before a tenant exists, and company creation
-- happens before a company exists. Allow those scopes to be null while keeping
-- the record itself globally unique per operation scope and key.
-- ---------------------------------------------------------------------------
ALTER TABLE "idempotency_records" ALTER COLUMN "tenant_id" DROP NOT NULL;
ALTER TABLE "idempotency_records" ALTER COLUMN "company_id" DROP NOT NULL;

DROP INDEX IF EXISTS "idempotency_records_tenant_company_scope_key_idx";

CREATE UNIQUE INDEX IF NOT EXISTS "idempotency_records_scope_key_idx"
  ON "idempotency_records" (
    "operation_scope",
    "idempotency_key",
    COALESCE("tenant_id", '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE("company_id", '00000000-0000-0000-0000-000000000000'::uuid)
  );

-- ---------------------------------------------------------------------------
-- Deferred foreign keys that referenced tables created by later migrations.
-- ---------------------------------------------------------------------------
ALTER TABLE "party_groups"
  ADD CONSTRAINT "party_groups_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "party_roles"
  ADD CONSTRAINT "party_roles_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "branches"
  ADD CONSTRAINT "branches_default_tax_registration_id_ref"
  FOREIGN KEY ("default_tax_registration_id") REFERENCES "company_tax_registrations" ("id");

ALTER TABLE "company_settings"
  ADD CONSTRAINT "company_settings_default_cash_bank_account_id_ref"
  FOREIGN KEY ("default_cash_bank_account_id") REFERENCES "cash_bank_accounts" ("id");

ALTER TABLE "companies"
  ADD CONSTRAINT "companies_logo_asset_id_ref"
  FOREIGN KEY ("logo_asset_id") REFERENCES "media_assets" ("id");

-- ---------------------------------------------------------------------------
-- Uniqueness required for safe onboarding initialization and concurrent use.
-- ---------------------------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS "branches_tenant_company_default_idx"
  ON "branches" ("tenant_id", "company_id")
  WHERE "is_default" = true AND "status" = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS "warehouses_tenant_company_default_idx"
  ON "warehouses" ("tenant_id", "company_id")
  WHERE "is_default" = true AND "status" = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS "accounting_periods_year_number_idx"
  ON "accounting_periods" ("financial_year_id", "period_number");

CREATE UNIQUE INDEX IF NOT EXISTS "accounts_tenant_company_system_code_idx"
  ON "accounts" ("tenant_id", "company_id", "system_code")
  WHERE "system_code" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "cash_bank_accounts_tenant_company_default_idx"
  ON "cash_bank_accounts" ("tenant_id", "company_id")
  WHERE "is_default" = true AND "status" = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS "document_number_series_unique_idx"
  ON "document_number_series" (
    "tenant_id",
    "company_id",
    "document_type",
    "financial_year_id",
    COALESCE("branch_id", '00000000-0000-0000-0000-000000000000'::uuid)
  )
  WHERE "is_active" = true;

CREATE UNIQUE INDEX IF NOT EXISTS "membership_role_assignments_active_unique_idx"
  ON "membership_role_assignments" (
    "tenant_id",
    "membership_id",
    "role_id",
    "scope_type",
    COALESCE("company_id", '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE("branch_id", '00000000-0000-0000-0000-000000000000'::uuid)
  )
  WHERE "revoked_at" IS NULL;

-- ---------------------------------------------------------------------------
-- Staff invitations (P0 schema gap).
-- Invitation tokens are single-use, expire, and only their hash is stored.
-- The P0 invitation model grants a tenant-scoped default role.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS "tenant_invitations" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "email" CITEXT NOT NULL,
  "default_role_id" UUID NOT NULL,
  "invited_by" UUID NOT NULL,
  "token_hash" BYTEA NOT NULL UNIQUE,
  "message" TEXT,
  "status" TEXT NOT NULL DEFAULT 'pending',
  "expires_at" TIMESTAMPTZ NOT NULL,
  "accepted_at" TIMESTAMPTZ,
  "accepted_by" UUID,
  "revoked_at" TIMESTAMPTZ,
  "revoked_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id"),
  FOREIGN KEY ("tenant_id", "default_role_id") REFERENCES "roles" ("tenant_id", "id"),
  FOREIGN KEY ("invited_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("accepted_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("revoked_by") REFERENCES "users" ("id"),
  CONSTRAINT "tenant_invitations_status_check"
    CHECK ("status" IN ('pending', 'accepted', 'revoked', 'expired'))
);

CREATE UNIQUE INDEX IF NOT EXISTS "tenant_invitations_pending_email_idx"
  ON "tenant_invitations" ("tenant_id", "email")
  WHERE "status" = 'pending';

CREATE INDEX IF NOT EXISTS "tenant_invitations_tenant_idx"
  ON "tenant_invitations" ("tenant_id", "created_at" DESC);

-- ---------------------------------------------------------------------------
-- Stable platform permission catalogue.
-- ---------------------------------------------------------------------------
INSERT INTO "permissions" ("code", "module", "action", "description") VALUES
  ('workspace.view', 'workspace', 'view', 'View workspace'),
  ('workspace.update', 'workspace', 'update', 'Update workspace settings'),
  ('workspace.members.view', 'workspace', 'members.view', 'View workspace members'),
  ('workspace.members.invite', 'workspace', 'members.invite', 'Invite workspace members'),
  ('workspace.members.manage', 'workspace', 'members.manage', 'Manage workspace members and roles'),
  ('workspace.roles.view', 'workspace', 'roles.view', 'View workspace roles'),
  ('workspace.roles.manage', 'workspace', 'roles.manage', 'Manage workspace roles'),
  ('company.view', 'company', 'view', 'View company profile'),
  ('company.create', 'company', 'create', 'Create a company'),
  ('company.update', 'company', 'update', 'Update company profile'),
  ('company.settings.manage', 'company', 'settings.manage', 'Manage company settings'),
  ('company.tax.manage', 'company', 'tax.manage', 'Manage company tax registrations'),
  ('accounting.accounts.view', 'accounting', 'accounts.view', 'View chart of accounts'),
  ('accounting.settings.manage', 'accounting', 'settings.manage', 'Manage accounting settings'),
  ('sales.invoice.view', 'sales', 'invoice.view', 'View sales invoices'),
  ('sales.invoice.create', 'sales', 'invoice.create', 'Create sales invoices'),
  ('sales.invoice.post', 'sales', 'invoice.post', 'Post sales invoices'),
  ('sales.invoice.reverse', 'sales', 'invoice.reverse', 'Reverse sales invoices'),
  ('purchase.bill.view', 'purchase', 'bill.view', 'View purchase bills'),
  ('purchase.bill.create', 'purchase', 'bill.create', 'Create purchase bills'),
  ('purchase.bill.post', 'purchase', 'bill.post', 'Post purchase bills'),
  ('inventory.stock.view', 'inventory', 'stock.view', 'View inventory stock'),
  ('inventory.stock.adjust', 'inventory', 'stock.adjust', 'Adjust inventory stock'),
  ('reports.view', 'reports', 'view', 'View reports'),
  ('reports.export', 'reports', 'export', 'Export reports')
ON CONFLICT ("code") DO NOTHING;

