--+ migration:000002_tenancy_and_rbac
--+ description: Tenant and RBAC foundation tables

-- tenants: SaaS workspaces and primary data-isolation boundary
CREATE TABLE IF NOT EXISTS "tenants" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "name" VARCHAR(160) NOT NULL,
  "slug" CITEXT NOT NULL UNIQUE,
  "status" TEXT NOT NULL DEFAULT 'active',
  "timezone" TEXT NOT NULL DEFAULT 'Asia/Kolkata',
  "locale" VARCHAR(12) NOT NULL DEFAULT 'en-IN',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- tenant_memberships: Connect users to tenants with role-based access
CREATE TABLE IF NOT EXISTS "tenant_memberships" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "user_id" UUID NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'active',
  "joined_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "removed_at" TIMESTAMPTZ,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id"),
  FOREIGN KEY ("user_id") REFERENCES "users" ("id")
);

-- Add composite tenant-safe unique key for member uniqueness
ALTER TABLE "tenant_memberships"
  ADD CONSTRAINT "tenant_memberships_tenant_user_unique"
  UNIQUE ("tenant_id", "user_id");

-- Note: tenant_id FK to tenants will be added after tenants table is created

-- roles: Tenant-customizable named roles like Owner, Admin, Accountant, Manager
CREATE TABLE IF NOT EXISTS "roles" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "code" VARCHAR(60) NOT NULL,
  "name" VARCHAR(100) NOT NULL,
  "description" TEXT,
  "is_system" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id")
);

-- Add tenant-safe unique constraints for roles
ALTER TABLE "roles"
  ADD CONSTRAINT "roles_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "roles"
  ADD CONSTRAINT "roles_tenant_code_unique"
  UNIQUE ("tenant_id", "code");

ALTER TABLE "roles"
  ADD CONSTRAINT "roles_tenant_id_unique"
  UNIQUE ("tenant_id", "id");

-- permissions: Platform-seeded permission catalogue
CREATE TABLE IF NOT EXISTS "permissions" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "code" VARCHAR(120) NOT NULL UNIQUE,
  "module" VARCHAR(60) NOT NULL,
  "action" VARCHAR(60) NOT NULL,
  "description" TEXT,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- role_permissions: Many-to-many mapping of tenant roles to permissions
CREATE TABLE IF NOT EXISTS "role_permissions" (
  "tenant_id" UUID NOT NULL,
  "role_id" UUID NOT NULL,
  "permission_id" UUID NOT NULL,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  PRIMARY KEY ("tenant_id", "role_id", "permission_id"),
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id"),
  FOREIGN KEY ("role_id") REFERENCES "roles" ("id"),
  FOREIGN KEY ("permission_id") REFERENCES "permissions" ("id")
);

-- membership_role_assignments: Assign roles at tenant, company or branch scope
CREATE TABLE IF NOT EXISTS "membership_role_assignments" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "membership_id" UUID NOT NULL,
  "role_id" UUID NOT NULL,
  "scope_type" TEXT NOT NULL DEFAULT 'tenant',
  "company_id" UUID,
  "branch_id" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "revoked_at" TIMESTAMPTZ,
  "revoked_by" UUID,
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("revoked_by") REFERENCES "users" ("id"),
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id"),
  FOREIGN KEY ("membership_id") REFERENCES "tenant_memberships" ("id"),
  FOREIGN KEY ("role_id") REFERENCES "roles" ("id")
);

