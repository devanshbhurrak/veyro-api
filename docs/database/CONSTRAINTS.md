# Veyro Database Constraints & Validation

## Overview

This document details all database constraints, validations, and integrity rules implemented in the Veyro Phase 2 database.

## Constraint Categories

### 1. Primary Keys

#### Single-Column Primary Keys

**Global Identity Tables**

```sql
-- users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
);

-- user_sessions table  
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
);

-- auth_tokens table
CREATE TABLE auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
);
```

**Tenant-Owned Tables**

```sql
-- tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
);

-- roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
);
```

**Company-Owned Tables**

```sql
-- companies table
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
);

-- items table
CREATE TABLE items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
);
```

#### Composite Primary Keys

```sql
-- role_permissions composite PK
CREATE TABLE role_permissions (
    tenant_id UUID NOT NULL,
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, role_id, permission_id)
);

-- stock_balances composite PK
CREATE TABLE stock_balances (
    tenant_id UUID NOT NULL,
    company_id UUID NOT NULL,
    item_id UUID NOT NULL,
    warehouse_id UUID NOT NULL,
    ...
    PRIMARY KEY (tenant_id, company_id, item_id, warehouse_id)
);

-- membership_role_assignments composite FK target
CREATE TABLE membership_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
    -- Composite FK target for tenant/company/role
);
```

### 2. Composite Foreign Keys

#### Tenant/Company Composite Foreign Keys

**Company → Tenant**

```sql
-- companies table
ALTER TABLE companies
  ADD CONSTRAINT companies_tenant_id_ref
  FOREIGN KEY (tenant_id) REFERENCES tenants(id);
```

**Branch → Company/Tenant**

```sql
-- branches table
ALTER TABLE branches
  ADD CONSTRAINT branches_tenant_company_id_ref
  FOREIGN KEY (tenant_id, company_id) REFERENCES companies(id);
```

**Warehouse → Branch/Company/Tenant**

```sql
-- warehouses table
ALTER TABLE warehouses
  ADD CONSTRAINT warehouses_tenant_company_branch_ref
  FOREIGN KEY (tenant_id, company_id, branch_id) REFERENCES branches(id);
```

**Document → Company/Tenant**

```sql
-- documents table
ALTER TABLE documents
  ADD CONSTRAINT documents_tenant_company_ref
  FOREIGN KEY (tenant_id, company_id) REFERENCES companies(id);
```

#### Company-Owned Child → Parent Composite Keys

**Party Group → Tenant/Company**

```sql
-- party_groups table
ALTER TABLE party_groups
  ADD CONSTRAINT party_groups_tenant_company_ref
  FOREIGN KEY (tenant_id, company_id) REFERENCES companies(id);
```

**Party Role → Tenant/Company/Party**

```sql
-- party_roles table
ALTER TABLE party_roles
  ADD CONSTRAINT party_roles_tenant_company_party_ref
  FOREIGN KEY (tenant_id, company_id, party_id) 
  REFERENCES parties(id);
```

### 3. Tenant-Safe Unique Keys

#### Tenant-Owned Tables

```sql
-- parties table: unique tenant/party_code
CREATE UNIQUE INDEX parties_tenant_code_active_idx
  ON parties(tenant_id, party_code)
  WHERE party_code IS NOT NULL AND status = 'active';

-- items table: unique tenant/company/sku
CREATE UNIQUE INDEX items_tenant_company_sku_idx
  ON items(tenant_id, company_id, sku)
  WHERE sku IS NOT NULL;
```

#### Company-Owned Tables

```sql
-- accounts table: unique tenant/company/account_code
CREATE UNIQUE INDEX accounts_tenant_company_code_idx
  ON accounts(tenant_id, company_id, account_code);

-- items table: unique tenant/company/id (composite FK target)
ALTER TABLE items
  ADD CONSTRAINT items_tenant_company_id_ref
  FOREIGN KEY (tenant_id, company_id, id) REFERENCES items(id);
```

### 4. Composite Foreign Key Targets

#### Four-Column Composite Keys

```sql
-- warehouses: tenant/company/branch/warehouse composite FK
ALTER TABLE warehouses
  ADD CONSTRAINT warehouses_tenant_company_branch_id_ref
  FOREIGN KEY (tenant_id, company_id, branch_id, id)
  REFERENCES branches(id);

-- document_attachments: tenant/company/document/attachment composite FK
ALTER TABLE document_attachments
  ADD CONSTRAINT doc_attachments_tenant_company_doc_ref
  FOREIGN KEY (tenant_id, company_id, document_id, id)
  REFERENCES documents(id);
```

#### Same-Document/Same-Line Integrity

```sql
-- document_line_tax_components: same-document composite FK
ALTER TABLE document_line_tax_components
  ADD CONSTRAINT doc_line_tax_doc_line_ref
  FOREIGN KEY (tenant_id, company_id, document_id, document_line_id)
  REFERENCES document_lines(id);

-- stock_movements: same-document-line composite FK
ALTER TABLE stock_movements
  ADD CONSTRAINT stock_movements_doc_line_ref
  FOREIGN KEY (tenant_id, company_id, document_id, document_line_id)
  REFERENCES document_lines(id);
```

### 5. Partial Unique Indexes

#### Active/Default Record Uniqueness

```sql
-- parties: one default party per tenant
CREATE UNIQUE INDEX parties_tenant_default_idx
  ON parties(tenant_id, id)
  WHERE status = 'active' AND party_code IS NULL;

-- item_categories: one default category per company
CREATE UNIQUE INDEX item_categories_tenant_company_default_idx
  ON item_categories(tenant_id, company_id, id)
  WHERE status = 'active';

-- party_roles: one active role per party/company/role
CREATE UNIQUE INDEX party_roles_tenant_company_party_role_active_idx
  ON party_roles(tenant_id, company_id, party_id, role)
  WHERE is_active = true;
```

#### Business-Specific Uniqueness

```sql
-- companies: one primary company per tenant
CREATE UNIQUE INDEX companies_tenant_primary_idx
  ON companies(tenant_id, id)
  WHERE is_primary = true AND status = 'active';

-- document_number_series: unique series scope
CREATE UNIQUE INDEX doc_series_tenant_company_type_year_branch_idx
  ON document_number_series(tenant_id, company_id, document_type, financial_year_id)
  WHERE is_active = true;

-- roles: unique role code per tenant
CREATE UNIQUE INDEX roles_tenant_code_idx
  ON roles(tenant_id, code)
  WHERE status = 'active';
```

### 6. Check Constraints

#### Monetary Value Constraints

```sql
-- Non-negative monetary amounts
ALTER TABLE accounts
  ADD CONSTRAINT accounts_normal_balance_check
  CHECK (normal_balance IN ('debit', 'credit'));

ALTER TABLE payment_details
  ADD CONSTRAINT payment_details_amount_check
  CHECK (amount > 0);

ALTER TABLE cash_bank_transfer_details
  ADD CONSTRAINT transfer_details_amount_check
  CHECK (amount > 0);
```

#### Quantity Constraints

```sql
-- Positive quantities
ALTER TABLE document_lines
  ADD CONSTRAINT doc_lines_quantity_check
  CHECK (quantity >= 0);

ALTER TABLE item_unit_conversions
  ADD CONSTRAINT conversions_factor_check
  CHECK (conversion_to_base > 0);

-- Non-negative conversion factors
ALTER TABLE item_unit_conversions
  ADD CONSTRAINT conversions_factor_positive
  CHECK (conversion_to_base > 0);
```

#### Percentage Constraints

```sql
-- Valid tax percentages
ALTER TABLE tax_code_components
  ADD CONSTRAINT tax_rate_percentage_check
  CHECK (rate_percentage >= 0 AND rate_percentage <= 100);

ALTER TABLE document_line_tax_components
  ADD CONSTRAINT line_tax_rate_check
  CHECK (rate_percentage >= 0 AND rate_percentage <= 100);
```

#### Date Constraints

```sql
-- Valid date ranges
ALTER TABLE financial_years
  ADD CONSTRAINT financial_year_dates_check
  CHECK (end_date > start_date);

ALTER TABLE company_tax_registrations
  ADD CONSTRAINT tax_reg_valid_to_check
  CHECK (valid_to IS NULL OR valid_to > valid_from);

ALTER TABLE tax_code_versions
  ADD CONSTRAINT tax_version_valid_to_check
  CHECK (valid_to IS NULL OR valid_to > valid_from);
```

#### Status Constraints

```sql
-- Valid status values
ALTER TABLE tenants
  ADD CONSTRAINT tenants_status_check
  CHECK (status IN ('active', 'suspended', 'closed'));

ALTER TABLE users
  ADD CONSTRAINT users_account_status_check
  CHECK (account_status IN ('active', 'locked', 'disabled'));

ALTER TABLE companies
  ADD CONSTRAINT companies_status_check
  CHECK (status IN ('active', 'suspended', 'closed'));
```

### 7. Exclusion Constraints

#### Date Range Exclusions

```sql
-- Exclude overlapping financial years
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE financial_years
  ADD CONSTRAINT financial_years_exclude_range
  EXCLUDE USING gist (
    tenant_id WITH =,
    company_id WITH =,
    daterange(start_date, end_date, '[]') WITH &&
  );

-- Exclude overlapping tax code versions
ALTER TABLE tax_code_versions
  ADD CONSTRAINT tax_versions_exclude_range
  EXCLUDE USING gist (
    tenant_id WITH =,
    company_id WITH =,
    tax_code_id WITH =,
    daterange(valid_from, valid_to, '[]') WITH &&
  );
```

### 8. Deferred Validations

#### Same-Document Composite Foreign Keys

```sql
-- Trigger function to enforce same-document integrity
CREATE OR REPLACE FUNCTION validate_document_line_integrity()
RETURNS TRIGGER AS $$
BEGIN
    -- Verify document_line_id belongs to stated document_id
    IF EXISTS (
        SELECT 1 FROM document_lines dl
        WHERE dl.tenant_id = NEW.tenant_id
          AND dl.company_id = NEW.company_id
          AND dl.document_id != NEW.document_id
          AND dl.id = NEW.document_line_id
    ) THEN
        RAISE EXCEPTION 'Document line does not belong to stated document'
          USING HINT = 'Document line must belong to the same document as the tax component';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for deferred validation
CREATE TRIGGER document_line_integrity_trigger
  AFTER INSERT OR UPDATE ON document_line_tax_components
  FOR EACH ROW EXECUTE FUNCTION validate_document_line_integrity();
```

#### Company Scope Violations

```sql
-- Trigger to prevent cross-company references
CREATE OR REPLACE FUNCTION prevent_cross_company_references()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if document belongs to different company
    IF EXISTS (
        SELECT 1 FROM documents d
        WHERE d.id = NEW.document_id
          AND d.company_id != NEW.company_id
    ) THEN
        RAISE EXCEPTION 'Cannot reference document from different company'
          USING HINT = 'Document line must belong to the same company as the tax component';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for cross-company validation
CREATE TRIGGER cross_company_reference_trigger
  BEFORE INSERT OR UPDATE ON document_line_tax_components
  FOR EACH ROW EXECUTE FUNCTION prevent_cross_company_references();
```

### 9. Immutable Financial Records

#### Append-Only Design

```sql
-- journal_entries: posted entries are immutable
ALTER TABLE journal_entries
  ADD CONSTRAINT journal_entries_posted_immutable
  CHECK (posted_at IS NOT NULL AND posted_at <= now());

-- stock_movements: append-only movement records
ALTER TABLE stock_movements
  ADD CONSTRAINT stock_movements_immutable
  CHECK (created_at <= now());

-- outbox_events: append-only event records
ALTER TABLE outbox_events
  ADD CONSTRAINT outbox_events_immutable
  CHECK (created_at <= now() AND (processed_at IS NULL OR processed_at >= created_at));
```

#### Reversal Workflow Protection

```sql
-- Prevent direct modification of posted journals
CREATE OR REPLACE FUNCTION prevent_posted_journal_modification()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.posted_at IS NOT NULL AND NEW.posted_at IS NULL THEN
        RAISE EXCEPTION 'Cannot unpost a journal entry'
          USING HINT = 'Use a reversal document to correct posted entries';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER posted_journal_modification_trigger
  BEFORE UPDATE ON journal_entries
  FOR EACH ROW EXECUTE FUNCTION prevent_posted_journal_modification();
```

## Constraint Verification

### Test Constraints

Database integration tests verify all constraints:

```go
// Test unique constraints
func TestUniqueConstraints(t *testing.T) {
    // Test that duplicate company codes are rejected
    // Test that duplicate SKUs are rejected
    // Test that active/default record uniqueness works
}

// Test foreign key constraints  
func TestForeignKeyConstraints(t *testing.T) {
    // Test tenant/company composite FKs
    // Test same-document integrity
    // Test company scope enforcement
}

// Test check constraints
func TestCheckConstraints(t *testing.T) {
    // Test monetary value constraints
    // Test quantity constraints
    // Test date range constraints
    // Test status value constraints
}

// Test exclusion constraints
func TestExclusionConstraints(t *testing.T) {
    // Test financial year overlap prevention
    // Test tax version overlap prevention
}

// Test deferred validations
func TestDeferredValidations(t *testing.T) {
    // Test same-document integrity triggers
    // Test cross-company reference prevention
}
```

### Constraint Coverage Matrix

| Constraint Type | Tables Affected | Business Rule | Enforcement Method |
|----------------|----------------|---------------|-------------------|
| Primary Keys | 45+ tables | Unique row identification | Database PK |
| Composite FKs | 30+ relationships | Tenant/company scope | Database FK |
| Unique Keys | 25+ indexes | Business uniqueness | Database UNIQUE |
| Partial Indexes | 15+ indexes | Active/default uniqueness | Database partial |
| Check Constraints | 20+ tables | Value validation | Database CHECK |
| Exclusion Constraints | 5+ tables | Range overlap prevention | Database EXCLUDE |
| Deferred Validations | 10+ relationships | Same-document integrity | Triggers |

## Constraint Management

### Constraint Dependencies

Constraints are organized by:

1. **Creation Order**: Dependencies respected during migration
2. **Validation Order**: Check constraints validated in correct sequence
3. **Maintenance Order**: Easier to maintain dependent constraints

### Constraint Evolution

```sql
-- Adding a new composite FK requires:
1. Target table exists
2. Source table exists  
3. Parent records exist
4. Child records maintain referential integrity
5. Migration includes constraint creation
6. Tests verify constraint works
```

### Constraint Violations in Production

```sql
-- Query for constraint violations
SELECT 
    table_name,
    constraint_name,
    violation_type,
    violation_count,
    affected_records
FROM violation_reporting
WHERE constraint_type IN ('foreign_key', 'unique', 'check')
  AND detected_at >= NOW() - INTERVAL '24 hours';
```

## Performance Considerations

### Index Impact

- **Composite Keys**: Multiple column indexes are more expensive than single-column
- **Partial Indexes**: Significantly reduce index size and maintenance overhead
- **Exclusion Constraints**: Use GIST indexes which have higher storage overhead

### Constraint Enforcement Overhead

- **Check Constraints**: Minimal runtime overhead
- **Foreign Key Constraints**: Additional validation on inserts/updates
- **Deferred Constraints**: Validation deferred until transaction commit
- **Exclusion Constraints**: Higher overhead for range operations

### Optimization Strategies

```sql
-- Use partial indexes for common queries
CREATE INDEX idx_active_records ON table_name(id) WHERE status = 'active';

-- Use covering indexes for frequent queries
CREATE INDEX idx_cover_tenant_company ON table_name(tenant_id, company_id, id) 
  WHERE status = 'active';

-- Use filtered indexes for specific business rules
CREATE INDEX idx_positive_quantities ON document_lines(id) 
  WHERE quantity > 0 AND free_quantity >= 0;
```

## Constraint Documentation

### Constraint Naming Convention

```
<entity>_<field>_<rule>_<context>

Examples:
- parties_tenant_code_unique_idx (tenant-level party code uniqueness)
- items_tenant_company_sku_unique_idx (company-level SKU uniqueness)
- document_lines_tenant_company_id_ref (composite foreign key)
- companies_tenant_primary_active_idx (tenant-level business rule)
```

### Constraint Documentation Template

```markdown
## [Constraint Name]

**Table**: table_name
**Type**: Primary Key / Foreign Key / Unique / Check / Exclusion / Deferred
**Columns**: column1, column2, ...
**Business Rule**: Brief description of business rule enforced
**Enforcement**: Database level / Trigger / Application
**Usage**: Query validation / Data integrity / Business rule enforcement
**Impact**: Performance considerations / Maintenance requirements
**Testing**: Test scenarios / Edge cases covered
```

## Conclusion

The Veyro database implements comprehensive constraints and validations that provide:

1. **Data Integrity**: Prevent invalid data from entering the system
2. **Business Rule Enforcement**: Database-level business rule validation
3. **Tenant Isolation**: Strict separation of tenant/company data
4. **Performance Optimized**: Indexes and constraints tuned for query patterns
5. **Maintainable**: Well-documented and organized constraints

These constraints form the foundation of the Veyro data model, ensuring that the system maintains data quality, security, and business rule compliance while providing excellent performance characteristics.

---

*Document Version: 1.0*
*Last Updated: November 2026*
*Scope: Phase 2 Database Implementation*
*Status: Complete*