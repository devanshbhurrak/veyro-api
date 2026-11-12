--+ migration:000005_media
--+ description: Media assets for document attachments

-- media_assets: Storage metadata for uploaded files and assets
CREATE TABLE IF NOT EXISTS "media_assets" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID,
  "storage_provider" VARCHAR(40) NOT NULL,
  "bucket_name" VARCHAR(120) NOT NULL,
  "object_key" TEXT NOT NULL,
  "original_filename" VARCHAR(255) NOT NULL,
  "content_type" VARCHAR(120) NOT NULL,
  "file_size_bytes" BIGINT NOT NULL,
  "sha256_checksum" BYTEA NOT NULL,
  "asset_kind" TEXT NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'pending_scan',
  "uploaded_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "deleted_at" TIMESTAMPTZ,
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for media assets
ALTER TABLE "media_assets"
  ADD CONSTRAINT "media_assets_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "media_assets"
  ADD CONSTRAINT "media_assets_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "media_assets"
  ADD CONSTRAINT "media_assets_uploaded_by_ref"
  FOREIGN KEY ("uploaded_by") REFERENCES "users" ("id");

-- Add composite unique indexes for media assets
CREATE UNIQUE INDEX IF NOT EXISTS "media_assets_tenant_id_unique_idx"
  ON "media_assets" ("tenant_id", "id")
  WHERE "id" IS NOT NULL;

