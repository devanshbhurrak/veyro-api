--+ migration:000022_phase3_identity_tenancy
--+ description: Roll back Phase 3 invitations, session tokens and constraint additions

-- See the up migration for the full ordering rationale. Drop in reverse.
DROP TABLE IF EXISTS "tenant_invitations";

DROP INDEX IF EXISTS "membership_role_assignments_active_unique_idx";
DROP INDEX IF EXISTS "document_number_series_unique_idx";
DROP INDEX IF EXISTS "cash_bank_accounts_tenant_company_default_idx";
DROP INDEX IF EXISTS "accounts_tenant_company_system_code_idx";
DROP INDEX IF EXISTS "accounting_periods_year_number_idx";
DROP INDEX IF EXISTS "warehouses_tenant_company_default_idx";
DROP INDEX IF EXISTS "branches_tenant_company_default_idx";

ALTER TABLE "companies" DROP CONSTRAINT IF EXISTS "companies_logo_asset_id_ref";
ALTER TABLE "company_settings" DROP CONSTRAINT IF EXISTS "company_settings_default_cash_bank_account_id_ref";
ALTER TABLE "branches" DROP CONSTRAINT IF EXISTS "branches_default_tax_registration_id_ref";
ALTER TABLE "party_roles" DROP CONSTRAINT IF EXISTS "party_roles_company_id_ref";
ALTER TABLE "party_groups" DROP CONSTRAINT IF EXISTS "party_groups_company_id_ref";

DROP INDEX IF EXISTS "outbox_events_deduplication_key_idx";

DELETE FROM "permissions" WHERE "code" IN (
  'workspace.view', 'workspace.update', 'workspace.members.view', 'workspace.members.invite',
  'workspace.members.manage', 'workspace.roles.view', 'workspace.roles.manage',
  'company.view', 'company.create', 'company.update', 'company.settings.manage', 'company.tax.manage',
  'accounting.accounts.view', 'accounting.settings.manage',
  'sales.invoice.view', 'sales.invoice.create', 'sales.invoice.post', 'sales.invoice.reverse',
  'purchase.bill.view', 'purchase.bill.create', 'purchase.bill.post',
  'inventory.stock.view', 'inventory.stock.adjust', 'reports.view', 'reports.export'
);

DROP INDEX IF EXISTS "user_sessions_access_token_hash_idx";
ALTER TABLE "user_sessions" DROP COLUMN IF EXISTS "access_expires_at";
ALTER TABLE "user_sessions" DROP COLUMN IF EXISTS "access_token_hash";
