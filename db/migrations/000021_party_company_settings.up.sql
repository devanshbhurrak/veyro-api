--+ migration:000021_party_company_settings
--+ description: Party company settings and additional party tables

-- party_company_settings: Company-specific party settings
CREATE TABLE IF NOT EXISTS "party_company_settings" (
  "tenant_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "party_id" UUID NOT NULL,
  "default_tax_registration_id" UUID,
  "credit_limit" NUMERIC(20,2),
  "credit_days" INTEGER NOT NULL DEFAULT 0,
  "credit_limit_action" TEXT NOT NULL DEFAULT 'warn',
  "default_payment_terms_note" VARCHAR(160),
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1,
  PRIMARY KEY ("tenant_id", "company_id", "party_id")
);

-- Add composite foreign keys for party_company_settings
ALTER TABLE "party_company_settings"
  ADD CONSTRAINT "party_company_settings_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "party_company_settings"
  ADD CONSTRAINT "party_company_settings_company_id_ref"
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id");

ALTER TABLE "party_company_settings"
  ADD CONSTRAINT "party_company_settings_party_id_ref"
  FOREIGN KEY ("party_id") REFERENCES "parties" ("id");

-- party_addresses: Party billing/shipping addresses
CREATE TABLE IF NOT EXISTS "party_addresses" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "party_id" UUID NOT NULL,
  "address_type" TEXT NOT NULL,
  "attention" VARCHAR(120),
  "line1" VARCHAR(180) NOT NULL,
  "line2" VARCHAR(180),
  "landmark" VARCHAR(120),
  "city" VARCHAR(100),
  "district" VARCHAR(100),
  "state_code" VARCHAR(2),
  "postal_code" VARCHAR(12),
  "country_code" VARCHAR(2) NOT NULL DEFAULT 'IN',
  "is_default_billing" BOOLEAN NOT NULL DEFAULT false,
  "is_default_shipping" BOOLEAN NOT NULL DEFAULT false,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for party_addresses
ALTER TABLE "party_addresses"
  ADD CONSTRAINT "party_addresses_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "party_addresses"
  ADD CONSTRAINT "party_addresses_party_id_ref"
  FOREIGN KEY ("party_id") REFERENCES "parties" ("id");

-- party_tax_registrations: Party GST registrations
CREATE TABLE IF NOT EXISTS "party_tax_registrations" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL,
  "party_id" UUID NOT NULL,
  "registration_type" TEXT NOT NULL,
  "registration_number" VARCHAR(30),
  "gstin" VARCHAR(15),
  "state_code" VARCHAR(2),
  "legal_name_snapshot" VARCHAR(220),
  "valid_from" DATE NOT NULL,
  "valid_to" DATE,
  "is_default" BOOLEAN NOT NULL DEFAULT false,
  "status" TEXT NOT NULL DEFAULT 'active',
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "row_version" BIGINT NOT NULL DEFAULT 1
);

-- Add composite foreign keys for party_tax_registrations
ALTER TABLE "party_tax_registrations"
  ADD CONSTRAINT "party_tax_registrations_tenant_id_ref"
  FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id");

ALTER TABLE "party_tax_registrations"
  ADD CONSTRAINT "party_tax_registrations_party_id_ref"
  FOREIGN KEY ("party_id") REFERENCES "parties" ("id");

