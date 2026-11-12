--+ migration:000002_tenancy_and_rbac
--+ description: Rollback tenant and RBAC foundation tables

-- Drop membership role assignments
DROP TABLE IF EXISTS "membership_role_assignments";
DROP TABLE IF EXISTS "role_permissions";
DROP TABLE IF EXISTS "permissions";
DROP TABLE IF EXISTS "roles";
DROP TABLE IF EXISTS "tenant_memberships";
DROP TABLE IF EXISTS "tenants";

-- Note: users, user_sessions, auth_tokens tables will be dropped by first migration
-- Note: companies table will be dropped by later migrations