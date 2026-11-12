--+ migration:000001_extensions_and_identity
--+ description: Required PostgreSQL extensions and global identity tables

-- Extensions for citext (case-insensitive text), pgcrypto (UUID generation)
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- users: Global login identity table
CREATE TABLE IF NOT EXISTS "users" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "email" CITEXT NOT NULL UNIQUE,
  "phone_e164" VARCHAR(16),
  "password_hash" TEXT,
  "full_name" VARCHAR(160) NOT NULL,
  "account_status" TEXT NOT NULL DEFAULT 'active',
  "email_verified_at" TIMESTAMPTZ,
  "last_login_at" TIMESTAMPTZ,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- user_sessions: Web session tracking with refresh token rotation
CREATE TABLE IF NOT EXISTS "user_sessions" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "user_id" UUID NOT NULL,
  "refresh_token_hash" BYTEA NOT NULL UNIQUE,
  "token_family_id" UUID NOT NULL,
  "replaced_by_session_id" UUID,
  "expires_at" TIMESTAMPTZ NOT NULL,
  "last_used_at" TIMESTAMPTZ,
  "revoked_at" TIMESTAMPTZ,
  "revoke_reason" TEXT,
  "ip_address" INET,
  "user_agent" TEXT,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("user_id") REFERENCES "users" ("id"),
  FOREIGN KEY ("replaced_by_session_id") REFERENCES "user_sessions" ("id")
);

-- auth_tokens: Single-use tokens for email verification and password reset
CREATE TABLE IF NOT EXISTS "auth_tokens" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "user_id" UUID NOT NULL,
  "purpose" TEXT NOT NULL,
  "token_hash" BYTEA NOT NULL UNIQUE,
  "expires_at" TIMESTAMPTZ NOT NULL,
  "consumed_at" TIMESTAMPTZ,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("user_id") REFERENCES "users" ("id")
);

