--+ migration:000003_parties
--+ description: Party and group tables for customer/supplier management

-- parties: Tenant-wide party records (customers, suppliers, contacts)
CREATE TABLE IF NOT EXISTS "parties" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "party_code" VARCHAR(50),
  "display_name" VARCHAR(180) NOT NULL,
  "legal_name" VARCHAR(220),
  "party_kind" TEXT NOT NULL DEFAULT 'business',
  "email" CITEXT,
  "phone_e164" VARCHAR(16),
  "pan" VARCHAR(10),
  "notes" TEXT,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_by" UUID,
  "updated_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id"),
  FOREIGN KEY ("created_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("updated_by") REFERENCES "users" ("id")
);

-- Add tenant-safe composite foreign key target
ALTER TABLE "parties"
  ADD CONSTRAINT "parties_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

-- Add partial unique index for active tenant/party code uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS "parties_tenant_code_active_idx"
  ON "parties" ("tenant_id", "party_code")
  WHERE "party_code" IS NOT NULL AND "status" = 'active';

-- party_groups: Company-specific groups for parties (customer groups, supplier groups)
CREATE TABLE IF NOT EXISTS "party_groups" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "role" TEXT NOT NULL,
  "code" VARCHAR(50),
  "name" VARCHAR(120) NOT NULL,
  "description" TEXT,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for party groups
ALTER TABLE "party_groups"
  ADD CONSTRAINT "party_groups_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

-- Add tenant/company/role aware composite unique key
ALTER TABLE "party_groups"
  ADD CONSTRAINT "party_groups_tenant_company_role_id_unique"
  UNIQUE ("tenant_id", "company_id", "role", "id");

-- Add unique name constraints
CREATE UNIQUE INDEX IF NOT EXISTS "party_groups_tenant_company_role_name_active_idx"
  ON "party_groups" ("tenant_id", "company_id", "role", "name")
  WHERE "name" IS NOT NULL AND "status" = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS "party_groups_tenant_company_role_code_active_idx"
  ON "party_groups" ("tenant_id", "company_id", "role", "code")
  WHERE "code" IS NOT NULL AND "status" = 'active';

-- party_roles: Company-specific customer/supplier classification
CREATE TABLE IF NOT EXISTS "party_roles" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "party_id" UUID NOT NULL,
  "role" TEXT NOT NULL,
  "group_id" UUID,
  "is_active" BOOLEAN NOT NULL DEFAULT true,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for party roles
ALTER TABLE "party_roles"
  ADD CONSTRAINT "party_roles_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "party_roles"
  ADD CONSTRAINT "party_roles_party_id_ref"
  FOREIGN KEY ("party_id") REFERENCES "parties" ("id");

ALTER TABLE "party_roles"
  ADD CONSTRAINT "party_roles_group_id_ref"
  FOREIGN KEY ("group_id") REFERENCES "party_groups" ("id");

-- Add party-specific composite unique key
CREATE UNIQUE INDEX IF NOT EXISTS "party_roles_tenant_company_party_role_unique_idx"
  ON "party_roles" ("tenant_id", "company_id", "party_id", "role")
  WHERE "party_id" IS NOT NULL AND "is_active" = true;

