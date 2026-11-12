--+ migration:000012_document_templates
--+ description: Document template tables

-- document_templates: Document layout templates
CREATE TABLE IF NOT EXISTS "document_templates" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "document_type" TEXT NOT NULL,
  "name" VARCHAR(120) NOT NULL,
  "is_default" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for document_templates
ALTER TABLE "document_templates"
  ADD CONSTRAINT "document_templates_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_templates"
  ADD CONSTRAINT "document_templates_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

-- Add unique constraint for default templates
CREATE UNIQUE INDEX IF NOT EXISTS "document_templates_tenant_company_type_default_idx"
  ON "document_templates" ("tenant_id", "company_id", "document_type")
  WHERE "document_type" IS NOT NULL AND "is_default" = true;

-- document_template_versions: Immutable template versions
CREATE TABLE IF NOT EXISTS "document_template_versions" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "template_id" UUID NOT NULL,
  "version_number" INTEGER NOT NULL,
  "config_schema_version" SMALLINT NOT NULL,
  "layout_config" JSONB NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'draft',
  "created_by" UUID,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "published_at" TIMESTAMPTZ,
  "row_version" BIGINT NOT NULL DEFAULT 1,
  FOREIGN KEY ("created_by") REFERENCES "users" ("id")
);

-- Add composite foreign keys for document_template_versions
ALTER TABLE "document_template_versions"
  ADD CONSTRAINT "document_template_versions_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "document_template_versions"
  ADD CONSTRAINT "document_template_versions_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "document_template_versions"
  ADD CONSTRAINT "document_template_versions_template_id_ref"
  FOREIGN KEY ("template_id") REFERENCES "document_templates" ("id");

-- Add unique constraint for version numbers
CREATE UNIQUE INDEX IF NOT EXISTS "document_template_versions_template_version_idx"
  ON "document_template_versions" ("tenant_id", "company_id", "template_id", "version_number")
  WHERE "template_id" IS NOT NULL AND "version_number" IS NOT NULL;

