--+ migration:000001_extensions_and_identity
--+ description: Rollback extensions and global identity tables

-- Drop tables
DROP TABLE IF EXISTS "auth_tokens";
DROP TABLE IF EXISTS "user_sessions";
DROP TABLE IF EXISTS "users";

-- Drop extensions
DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS citext;