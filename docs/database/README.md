# Veyro Database Setup

## Overview

This document provides comprehensive guidance for setting up, managing, and working with the PostgreSQL database for the Veyro business management platform.

## Database Architecture

Veyro uses PostgreSQL 18+ as its primary database backend with the following key architectural principles:

### 1. Tenant-Ready Design

- **Shared Database**: Single PostgreSQL instance serving multiple tenants
- **Row-Level Isolation**: Each tenant's data is isolated through composite foreign keys
- **No Cross-Tenant Access**: Database constraints prevent tenants from accessing each other's data

### 2. Company Model

- **Legal Entities**: Companies represent legal/business entities within a tenant
- **Nested Structure**: Companies belong to tenants, branches to companies, warehouses to branches
- **Financial Scope**: All financial data is scoped by tenant and company

### 3. Business Domain Structure

The database is organized into logical domains:

#### Identity & Access (Extension tables 1-2)
- `users`: Application user accounts
- `user_sessions`: Web session management
- `auth_tokens`: Email verification and password reset tokens
- `tenants`: Workspace definitions
- `tenant_memberships`: User-workspace relationships
- `roles`: Role definitions
- `permissions`: Permission catalogue
- `role_permissions`: Role-permission mappings

#### Organization & Setup (Extensions 3-6)
- `companies`: Company accounting entities
- `branches`: Company operational locations
- `warehouses`: Company stock locations
- `accounts`: Chart of accounts
- `item_categories`: Item categorization

#### Catalog & Masters (Extensions 7-10)
- `units`: Measurement units
- `items`: Product and service catalog
- `tax_codes`: Tax identification codes
- `tax_code_versions`: Historical tax rates
- `tax_code_components`: Component-level tax definitions

#### Business Documents (Extensions 11-15)
- `documents`: Document headers (invoices, bills, payments, etc.)
- `document_lines`: Commercial line items
- `document_party_snapshots`: Historical party information
- `document_line_tax_components`: Line-level tax breakdown
- `document_templates`: Document layout templates

#### Accounting & Settlement (Extensions 16-18)
- `posting_batches`: Accounting transactions
- `journal_entries`: Ledger entries
- `journal_lines`: Journal line details
- `party_open_items`: Outstanding receivables/payables
- `payment_allocations`: Payment application records

#### Inventory & Logistics (Extensions 19-21)
- `stock_movements`: Inventory movements
- `stock_balances`: Current stock projections
- `manual_journal_lines`: Manual accounting entries
- `media_assets`: File storage metadata
- `document_attachments`: Document file attachments

### 4. Data Integrity Features

#### Composite Foreign Keys

Most critical relationships use composite foreign keys that enforce both referential integrity and business scope:

```sql
-- Company belongs to tenant
ALTER TABLE companies
  ADD CONSTRAINT companies_tenant_id_ref
  FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Branch belongs to company AND tenant
ALTER TABLE branches
  ADD CONSTRAINT branches_tenant_company_id_ref
  FOREIGN KEY (tenant_id, company_id) REFERENCES companies(id);
```

#### Tenant-Safe Unique Keys

```sql
-- One primary company per tenant
CREATE UNIQUE INDEX companies_tenant_active_primary_idx
  ON companies WHERE is_primary = true AND status = 'active';

-- Unique item SKU per company
CREATE UNIQUE INDEX items_tenant_company_sku_idx
  ON items WHERE sku IS NOT NULL;
```

#### Deferred Validations

- Journal balance reconciliation
- Payment allocation limits
- Document total integrity

## Database Setup

### Prerequisites

- PostgreSQL 18+
- `citext` extension (for case-insensitive text)
- `pgcrypto` extension (for UUID generation)
- `btree_gist` extension (for exclusion constraints)

### Initial Setup Commands

```bash
# Start PostgreSQL infrastructure
make docker-up

# Check database readiness
make db-check

# Apply migrations (run golang-migrate from db/migrations)
migrations-up:
  golang-migrate -database "postgres://postgres:postgres@localhost:5432/veyro?sslmode=disable" -path ./db/migrations up
```

### Migration Strategy

Migrations follow a strict dependency order based on table dependencies:

1. **Identity Tables**: Users, sessions, auth tokens
2. **Tenant & RBAC**: Tenants, memberships, roles, permissions
3. **Organization**: Companies, branches, warehouses
4. **Catalog**: Items, units, tax codes
5. **Business Documents**: Documents, lines, tax details
6. **Accounting**: Journals, open items, allocations
7. **Inventory**: Stock movements, balances
8. **Supporting**: Idempotency, outbox, audit

### Configuration

#### Environment Variables

```bash
# Database connection
DB_HOST=localhost
DB_PORT=5432
DB_NAME=veyro
DB_USER=postgres
DB_PASSWORD=postgres
DB_SSL_MODE=disable
DB_POOL_SIZE=10

# Application
APP_NAME=Veyro
APP_ENV=development
APP_PORT=8080

# Storage
STORAGE_ENDPOINT=localhost:9000
STORAGE_BUCKET=veyro
STORAGE_ACCESS_KEY=minioadmin
STORAGE_SECRET_KEY=minioadmin
```

#### Database Configuration

The database configuration supports:

- **Connection Pooling**: Configurable pool sizes with health monitoring
- **Connection Health**: Automatic reconnection and health checks
- **Error Classification**: Detailed error categorization for different failure modes
- **Transaction Management**: Transaction helper functions with rollback guarantees

## Database Access

### Connection Pool

The application uses `pgxpool` for connection management:

```go
import "github.com/jackc/pgx/v5/pgxpool"

// Create database connection
db, err := database.NewDatabase(database.DatabaseConfig{
    Host:     "localhost",
    Port:     5432,
    Database: "veyro",
    Username: "postgres",
    Password: "postgres",
    SSLMode:  "disable",
    PoolSize: 10,
})

// Use database
err = db.HealthCheck(ctx)
```

### Transaction Management

```go
import "github.com/jackc/pgx/v5"

// WithTransaction helper for atomic operations
db.WithTransaction(ctx, func(tx *database.Transaction) error {
    // Execute multiple operations within a single transaction
    _, err := tx.ExecQuery(ctx, "INSERT INTO table1 VALUES ($1)", id)
    if err != nil {
        return err
    }

    _, err = tx.ExecQuery(ctx, "UPDATE table2 SET field = $1 WHERE id = $2", value, id)
    return err
})

// Transaction with row locking
db.WithTransactionForUpdate(ctx, func(tx *database.Transaction) error {
    // Operations that require serializable isolation
    // Used for financial posting operations
})
```

### Database Errors

Error handling provides detailed classification:

```go
func ClassifyDatabaseError(err error) *database.DatabaseError {
    // Classifies errors into specific types
    // ErrorCodeForeignKeyViolation
    // ErrorCodeUniqueViolation
    // ErrorCodeConnectionFailure
    // etc.
}

func IsConstraintError(err error) bool {
    // Checks if error is a constraint violation
}

func IsRetryableError(err error) bool {
    // Checks if error can be retried
}
```

## Query Organization

### SQL Generation

The project uses `sqlc` for type-safe SQL query generation:

```yaml
# db/sqlc.yaml
version: "3"
packages:
  - name: db
    path: backend/db/queries
    queries: "*.sql"
    schema: backend/schema.sql
    go_package: "backend/db/queries"
```

### Query Structure

Queries are organized by module:

```
backend/db/queries/
├── identity/
│   ├── users.sql
│   └── sessions.sql
├── tenancy/
│   ├── tenants.sql
│   └── memberships.sql
├── catalog/
│   ├── items.sql
│   └── categories.sql
├── accounting/
│   ├── journals.sql
│   └── lines.sql
└── inventory/
    ├── stock.sql
    └── balances.sql
```

## Security & Compliance

### Tenant Isolation

All tables have appropriate tenant and company scope enforcement:

```sql
-- Party belongs to tenant
ALTER TABLE parties
  ADD CONSTRAINT parties_tenant_id_ref
  FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Party has company scope
ALTER TABLE party_roles
  ADD CONSTRAINT party_roles_tenant_company_id_ref
  FOREIGN KEY (tenant_id, company_id) REFERENCES companies(id);
```

### Row-Level Security (Optional)

Veyro supports optional PostgreSQL Row-Level Security:

```sql
-- Enable RLS for tables that need additional security
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

-- Create policy to restrict access to tenant members
CREATE POLICY documents_tenant_policy ON documents
  FOR SELECT TO db_role
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

### Access Control

Access control is enforced through:

1. **Application-Level**: Role-based access control in Go code
2. **Database-Level**: Foreign key constraints
3. **Optional RLS**: Additional defense-in-depth security

## Monitoring & Observability

### Health Endpoints

#### Liveness Check

```http
GET /health
Response: 200 OK
{"status":"healthy"}
```

#### Readiness Check

```http
GET /ready
Response: 200 OK
{"status":"ready"}
```

#### Database-Specific Readiness

```http
GET /ready/db
Response: 200 OK
{"status":"ready"}
```

Response: 503 Service Unavailable
{"status":"not_ready","error":"database connection failed"}
```

### Monitoring Metrics

Database monitoring includes:

- **Connection Pool Stats**: Total, acquired, idle connections
- **Query Performance**: Execution times, error rates
- **Table Row Counts**: Critical table sizes
- **Database Size**: Approximate database storage usage

### Backups & Maintenance

#### Regular Backups

```bash
# Logical backup using pg_dump
pg_dump -h localhost -U postgres -d vevro > backup_$(date +%Y%m%d_%H%M%S).sql

# Stream backup for point-in-time recovery
pg_dump -h localhost -U postgres -d vevro -F c -f backup.dump
```

#### Schema Validation

```bash
# Verify database schema matches design
make db-verify

# Check for missing constraints
make db-check
```

#### Migration Verification

```bash
# Verify migration order and dependencies
make migrate-up

# Rollback test
make migrate-down
```

## Testing

### Test Infrastructure

Database tests use a real PostgreSQL instance with the following approach:

1. **Test Database Creation**: Separate test database
2. **Migration Application**: Apply all migrations to test database
3. **Constraint Testing**: Verify all constraints work correctly
4. **Isolation Testing**: Test tenant/company isolation
5. **Transaction Testing**: Verify atomic behavior

### Test Commands

```bash
# Run database integration tests
make test-integration

# Run all tests
make test

# Run tests with race detector
make test-race
```

### Test Scenarios

1. **Connection Tests**: Basic database connectivity
2. **Migration Tests**: Schema application and verification
3. **Constraint Tests**: Unique keys, foreign keys, check constraints
4. **Isolation Tests**: Tenant and company data separation
5. **Transaction Tests**: Atomic operation guarantees
6. **Performance Tests**: Query execution benchmarks

## Migration Management

### Migration Process

1. **Review**: Review migration changes before application
2. **Apply**: Apply migrations using `golang-migrate`
3. **Verify**: Verify migration success and data integrity
4. **Rollback**: Test rollback capability for critical migrations

### Migration Naming

```bash
000001_extensions_and_identity.up.sql      # Extensions and global identity
000001_extensions_and_identity.down.sql    # Rollback script

000002_tenancy_and_rbac.up.sql              # Tenant and RB
... (subsequent migration groups)
```

### Migration Dependencies

```sql
-- Migration group 2 depends on group 1
CREATE TABLE role_permissions (
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  role_id UUID NOT NULL REFERENCES roles(id),
  permission_id UUID NOT NULL REFERENCES permissions(id),
  ...
);

-- Migration group 3 depends on group 2
CREATE TABLE companies (
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  ...
);
```

## Performance Considerations

### Connection Pool Configuration

```go
DatabaseConfig{
    PoolSize: 10, // Adjust based on expected concurrent users
}
```

### Query Optimization

1. **Index Usage**: All composite foreign keys include indexes
2. **Partial Indexes**: Used for active/default record uniqueness
3. **Exclusion Constraints**: Prevent data conflicts
4. **Transaction Duration**: Keep transactions short and focused

### Scaling Considerations

- **Read Replicas**: Configure read replicas for reporting queries
- **Connection Pool Tuning**: Adjust pool sizes based on workload
- **Query Optimization**: Regularly review slow query logs

## Troubleshooting

### Common Issues

#### Connection Refused

```bash
# Check if PostgreSQL is running
make docker-up
make db-check
```

#### Constraint Violations

```sql
-- Check for existing data that violates new constraints
SELECT * FROM table1 WHERE constraint_column = violation_value;
```

#### Migration Failures

```bash
# Review migration logs
# Check PostgreSQL error logs
# Test migrations on a copy of the database
```

### Diagnostic Commands

```bash
# Check PostgreSQL status
docker compose exec postgres pg_isready

# Check connection pool status
# (application-specific monitoring)

# Review slow queries
sql [database] "SHOW pg_stat_statements;"
```

## Migration Guide

### From Previous Versions

#### Upgrade to Phase 2

1. **Backup Current Database**: `pg_dump current_db > backup.sql`
2. **Stop Application**: Ensure no active connections
3. **Apply Migrations**: Use `make migrate-up` or manual migration commands
4. **Verify Data**: Check for any data integrity issues
5. **Test Application**: Run full application test suite
6. **Monitor**: Monitor application and database health

### Rollback Procedure

```bash
# Rollback database migrations
make migrate-down

# Check database state
make db-check

# Verify application can start
make run-api
```

## Future Enhancements

### Planned Improvements

1. **Read Replicas**: Database read replicas for reporting workloads
2. **Advanced Caching**: Application-level caching for frequent queries
3. **Additional Extensions**: More PostgreSQL extensions as needed
4. **Enhanced Monitoring**: Real-time database metrics and alerting
5. **Automated Migrations**: Automated migration testing and deployment

### Compliance Considerations

- **Data Encryption**: Consider encrypting sensitive data at rest
- **Audit Logging**: Comprehensive audit trail for all data changes
- **Backup Verification**: Regular backup integrity checks
- **Security Scanning**: Periodic vulnerability assessments

## Conclusion

The Veyro database provides a robust, secure foundation for the business management platform with:

- **Strong Data Integrity**: Comprehensive constraints and validations
- **Tenant Isolation**: Proper separation of business data
- **Performance Optimized**: Indexes and connection pooling
- **Secure Design**: Multiple layers of security controls
- **Maintainable**: Well-documented migrations and procedures

This database foundation supports the complex requirements of a multi-tenant business management system while ensuring data integrity, security, and performance.

---

*Last Updated: November 2026*
*Database Version: Phase 2 Implementation*
*Architecture: PostgreSQL 18+ with pgx + sqlc*