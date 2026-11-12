--+ migration:000017_document_relations
--+ description: Document relations and attachments

-- document_relations: Header-level document lineage
CREATE TABLE IF NOT EXISTS "document_relations" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "source_document_id" UUID NOT NULL,
  "target_document_id" UUID NOT NULL,
  "relation_type" TEXT NOT NULL,
  "created_by" UUID NOT NULL,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for document_relations
ALTER TABLE "document_relations"
  ADD CONSTRAINT "document_relations_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_relations"
  ADD CONSTRAINT "document_relations_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_relations"
  ADD CONSTRAINT "document_relations_source_document_id_ref"
  FOREIGN KEY ("source_document_id") REFERENCES "documents" ("id");

ALTER TABLE "document_relations"
  ADD CONSTRAINT "document_relations_target_document_id_ref"
  FOREIGN KEY ("target_document_id") REFERENCES "documents" ("id");

-- document_line_relations: Line-level document lineage
CREATE TABLE IF NOT EXISTS "document_line_relations" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "source_document_line_id" UUID NOT NULL,
  "target_document_line_id" UUID NOT NULL,
  "relation_type" TEXT NOT NULL,
  "source_quantity_consumed" NUMERIC(20,6),
  "source_base_quantity_consumed" NUMERIC(20,6),
  "amount_consumed" NUMERIC(20,2),
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for document_line_relations
ALTER TABLE "document_line_relations"
  ADD CONSTRAINT "document_line_relations_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_line_relations"
  ADD CONSTRAINT "document_line_relations_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_line_relations"
  ADD CONSTRAINT "document_line_relations_source_document_line_id_ref"
  FOREIGN KEY ("source_document_line_id") REFERENCES "document_lines" ("id");

ALTER TABLE "document_line_relations"
  ADD CONSTRAINT "document_line_relations_target_document_line_id_ref"
  FOREIGN KEY ("target_document_line_id") REFERENCES "document_lines" ("id");

-- document_attachments: Document file attachments
CREATE TABLE IF NOT EXISTS "document_attachments" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_id" UUID NOT NULL,
  "media_asset_id" UUID NOT NULL,
  "attachment_kind" TEXT NOT NULL,
  "display_name" VARCHAR(180),
  "created_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for document_attachments
ALTER TABLE "document_attachments"
  ADD CONSTRAINT "document_attachments_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_attachments"
  ADD CONSTRAINT "document_attachments_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_attachments"
  ADD CONSTRAINT "document_attachments_document_id_ref"
  FOREIGN KEY ("document_id") REFERENCES "documents" ("id");

ALTER TABLE "document_attachments"
  ADD CONSTRAINT "document_attachments_media_asset_id_ref"
  FOREIGN KEY ("media_asset_id") REFERENCES "media_assets" ("id");

