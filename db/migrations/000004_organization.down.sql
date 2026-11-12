--+ migration:000004_organization
--+ description: Rollback company and organization structure tables

-- Drop warehouses
DROP TABLE IF EXISTS "warehouses";
DROP TABLE IF EXISTS "branches";
DROP TABLE IF EXISTS "companies";
DROP TABLE IF EXISTS "tenants";
DROP TABLE IF EXISTS "tenant_memberships";
DROP TABLE IF EXISTS "roles";
DROP TABLE IF EXISTS "permissions";
DROP TABLE IF EXISTS "role_permissions";
DROP TABLE IF EXISTS "membership_role_assignments";
DROP TABLE IF EXISTS "users";
DROP TABLE IF EXISTS "user_sessions";
DROP TABLE IF EXISTS "auth_tokens";