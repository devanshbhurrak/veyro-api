--+ migration:000016_supporting
--+ description: Rollback supporting tables

-- Drop supporting tables
DROP TABLE IF EXISTS "audit_events";
DROP TABLE IF EXISTS "outbox_events";
DROP TABLE IF EXISTS "idempotency_records";