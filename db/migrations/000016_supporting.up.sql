--+ migration:000016_supporting
--+ description: Supporting tables (idempotency, outbox, audit)

-- idempotency_records: Idempotency tracking
CREATE TABLE IF NOT EXISTS "idempotency_records" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "operation_scope" VARCHAR(160) NOT NULL,
  "idempotency_key" VARCHAR(160) NOT NULL,
  "request_hash" BYTEA NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'processing',
  "response_status" SMALLINT,
  "resource_type" VARCHAR(80),
  "resource_id" UUID,
  "created_by" UUID NOT NULL,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "completed_at" TIMESTAMPTZ,
  "expires_at" TIMESTAMPTZ NOT NULL,
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for idempotency_records
ALTER TABLE "idempotency_records"
  ADD CONSTRAINT "idempotency_records_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "idempotency_records"
  ADD CONSTRAINT "idempotency_records_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- Add unique constraint for idempotency key
CREATE UNIQUE INDEX IF NOT EXISTS "idempotency_records_tenant_company_scope_key_idx"
  ON "idempotency_records" ("tenant_id", "company_id", "operation_scope", "idempotency_key")
  WHERE "operation_scope" IS NOT NULL AND "idempotency_key" IS NOT NULL;

-- outbox_events: Durable event outbox
CREATE TABLE IF NOT EXISTS "outbox_events" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID,
  "aggregate_type" VARCHAR(80) NOT NULL,
  "aggregate_id" UUID NOT NULL,
  "event_type" VARCHAR(120) NOT NULL,
  "event_version" SMALLINT NOT NULL,
  "payload" JSONB NOT NULL,
  "deduplication_key" VARCHAR(200) NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'pending',
  "attempt_count" INTEGER NOT NULL DEFAULT 0,
  "available_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "locked_until" TIMESTAMPTZ,
  "locked_by" VARCHAR(120),
  "processed_at" TIMESTAMPTZ,
  "last_error_code" VARCHAR(100),
  "last_error_message" TEXT,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for outbox_events
ALTER TABLE "outbox_events"
  ADD CONSTRAINT "outbox_events_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "outbox_events"
  ADD CONSTRAINT "outbox_events_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- audit_events: Audit trail
CREATE TABLE IF NOT EXISTS "audit_events" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID,
  "company_id" UUID,
  "branch_id" UUID,
  "actor_user_id" UUID,
  "actor_type" TEXT NOT NULL,
  "action" VARCHAR(120) NOT NULL,
  "entity_type" VARCHAR(100) NOT NULL,
  "entity_id" UUID,
  "request_id" UUID,
  "ip_address" INET,
  "user_agent" TEXT,
  "changes" JSONB,
  "metadata" JSONB,
  "occurred_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("actor_user_id") REFERENCES "users" ("id")
);

-- Add composite foreign keys for audit_events
ALTER TABLE "audit_events"
  ADD CONSTRAINT "audit_events_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "audit_events"
  ADD CONSTRAINT "audit_events_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

