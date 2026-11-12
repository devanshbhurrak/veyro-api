# Veyro Database Development

## Overview

This document provides comprehensive guidance for developers working with the Veyro database implementation. It covers setup procedures, development workflows, testing strategies, and maintenance practices.

## Development Setup

### Prerequisites

#### System Requirements

- **Operating System**: Windows 10/11, macOS 11+, Linux
- **Database**: PostgreSQL 18+ (local or remote)
- **Go Version**: 1.21+
- **Additional Tools**: Docker, git, text editor (VS Code, IntelliJ, etc.)

#### Development Environment Setup

```bash
# Clone the Veyro repository
 git clone https://github.com/veyro/backend.git
 cd vevro/backend

# Install Go dependencies
go mod tidy

# Start PostgreSQL infrastructure
make docker-up

# Wait for PostgreSQL to be ready
make db-check

# Apply database migrations
make migrate-up

# Generate SQLC code
make sqlc

# Set up environment variables (if needed)
cp .env.example .env.local
```

### Database Configuration

#### Local Development

```bash
# Environment variables for local development
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=veyro
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_SSL_MODE=disable
export DB_POOL_SIZE=5
```

#### Production Configuration

```bash
# Production environment variables
export DB_HOST=prod-db.example.com
export DB_PORT=5432
export DB_NAME=veyro_prod
export DB_USER=veyro_app
export DB_PASSWORD=<secure_password>
export DB_SSL_MODE=require
export DB_POOL_SIZE=20
```

### Development Tools

#### IDE Configuration

**Visual Studio Code Extensions**

```json
// .vscode/settings.json
{
    "go.toolsManagement.checkForUpdates": "none",
    "go.autocompleteUnimportedPackages": true,
    "go.useLanguageServer": true,
    "go.formatOnSave": true,
    "go.lintOnSave": "file",
    "go.testFlags": ["-v", "-tags=integration"],
    "go.buildTags": "integration",
    "search.excludeFiles": [
        "**/node_modules/**",
        "**/.git/**",
        "**/dist/**",
        "**/build/**"
    ]
}
```

#### Database Clients

**pgAdmin 4**

1. Download and install pgAdmin 4 from https://www.pgadmin.org/
2. Configure connection:
   - Host: localhost
   - Port: 5432
   - Database: vevro
   - Username: postgres
   - Password: postgres

**DBeaver**

1. Install DBeaver from https://dbeaver.io/
2. Create new connection:
   - Connection Name: Veyro Local
   - Host: localhost
   - Port: 5432
   - Database: vevro
   - User: postgres

## Development Workflow

### 1. Feature Development

#### Branching Strategy

```bash
# Feature branch for new development
# Use descriptive branch names
# Example: feature/document-tracking
# or bugfix/customer-isolation

# Create and switch to feature branch
git checkout -b feature/new-feature

# Make changes and commit
git add .
git commit -m "feat: implement new feature"

# Push to remote repository
git push origin feature/new-feature
```

#### Code Development

```go
// Standard development workflow
// 1. Create new files or modify existing ones
// 2. Write tests for new functionality
// 3. Implement the feature
// 4. Run tests and verify functionality
// 5. Commit changes with descriptive commit messages
```

#### Code Review Process

```markdown
# Code Review Checklist

## Prerequisites
- [ ] Code follows Veyro coding standards
- [ ] Tests written and passing
- [ ] Documentation updated
- [ ] Dependencies reviewed

## Code Quality
- [ ] Functions and methods focused and concise
- [ ] Error handling explicit and informative
- [ ] Type hints and struct tags complete
- [ ] Variable names descriptive and consistent
- [ ] Comments explain complex logic
- [ ] Imports ordered correctly

## Testing
- [ ] Unit tests cover all new code paths
- [ ] Integration tests verify database interactions
- [ ] Edge cases and error conditions tested
- [ ] Performance benchmarks established

## Security
- [ ] No hardcoded secrets or credentials
- [ ] SQL injection prevention
- [ ] Input validation implemented
- [ ] Access control respected

## Performance
- [ ] Query optimization considered
- [ ] Index usage appropriate
- [ ] Connection pooling configured
- [ ] Resource usage efficient

## Documentation
- [ ] Function and package documentation
- [ ] Usage examples provided
- [ ] API documentation updated
- [ ] Architecture decisions documented
```

### 2. Testing

#### Testing Strategy

Veyro employs a comprehensive testing approach:

```go
// 1. Unit Tests: Individual component testing
// 2. Integration Tests: Database and external service interactions
// 3. Contract Tests: API and business rule validation
// 4. Performance Tests: Database and application performance
```

#### Test Framework Configuration

```yaml
# go.mod (testing configuration)
module backend

go 1.21

require (
    // Testing dependencies
    github.com/stretchr/testify v1.8.1
    github.com/ory/dockertest v2.4.1
    // Other dependencies...
)
```

#### Test Environment Setup

```go
// Test environment setup
func TestMain(t *testing.T) {
    // Setup test database
    db := setupTestDatabase(t)
    defer db.Close()

    // Run integration tests
    testDatabaseConnectivity(t, db)
    testMigrationApplication(t, db)
    testConstraintEnforcement(t, db)
    testTransactionIntegrity(t, db)
}
```

#### Test Commands

```bash
# Run all tests
make test

# Run tests with verbose output
make test -v

# Run tests with race detector
make test-race

# Run integration tests specifically
make test-integration

# Run tests in specific packages
make test internal/platform/database
make test cmd/api
```

### 3. Database Operations

#### Database Migration Commands

```bash
# View migration status
# (requires golang-migrate)

golang-migrate -database "postgres://postgres:postgres@localhost:5432/veyro?sslmode=disable" \
    -path ./db/migrations status

# Apply all migrations
make migrate-up

# Roll back one migration (use with caution)
make migrate-down

# Check migration history
golang-migrate -database "postgres://..." -path ./db/migrations history
```

#### SQLC Code Generation

```bash
# Generate SQLC code
make sqlc

# Verify generated code
make db-verify

# Check if generation is up to date
golangci-lint run --enable=sqlc
```

#### Database Backup and Restore

```bash
# Create database backup
pg_dump -h localhost -U postgres -d vevro > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore from backup
pg_restore -h localhost -U postgres -d vevro backup_file.sql
```

### 4. Deployment

#### Deployment Checklist

```markdown
# Deployment Checklist

## Environment Preparation
- [ ] Target database environment prepared
- [ ] Application configuration updated
- [ ] Environment variables set
- [ ] Security permissions configured

## Database Migration
- [ ] Backup existing data
- [ ] Test migrations on staging environment
- [ ] Apply migrations to production
- [ ] Verify migration success

## Application Deployment
- [ ] Build application binaries
- [ ] Deploy to application server
- [ ] Update application configuration
- [ ] Start application services

## Testing and Validation
- [ ] Run integration tests in production environment
- [ ] Verify data integrity
- [ ] Test application functionality
- [ ] Monitor system performance

## Post-Deployment
- [ ] Monitor system logs
- [ ] Check database performance
- [ ] Validate user access
- [ ] Document deployment process
```

## Development Best Practices

### Code Quality

#### Naming Conventions

```go
// Package names
// Use lowercase, descriptive names
// Example: internal/platform/database

// Function names  
// Use camelCase for exported functions
// Example: CreateDocument, GetDatabaseStats

// Variable names
// Use camelCase for variables and parameters
// Example: tenantID, documentType, connectionPool

// Constant names
// Use UPPER_SNAKE_CASE for constants
// Example: MAX_CONNECTIONS, API_TIMEOUT
```

#### Error Handling

```go
// Error handling patterns
// 1. Check for errors immediately after I/O operations
// 2. Wrap errors with context
// 3. Return errors to caller
// 4. Use custom error types for application-specific errors

func CreateDocument(ctx context.Context, doc *Document) error {
    // Validate input
    if err := validateDocument(doc); err != nil {
        return fmt.Errorf("invalid document: %w", err)
    }

    // Attempt database operation
    if err := db.WithTransaction(ctx, func(tx *Transaction) error {
        // Database operations
        return tx.ExecQuery(ctx, "INSERT INTO documents VALUES ($1, $2)", doc.ID, doc.Data)
    }); err != nil {
        return fmt.Errorf("database operation failed: %w", err)
    }

    return nil
}
```

#### Documentation Standards

```markdown
# Documentation Standards

## Go Comments
// Comment style: Go standard library
// Package-level: Describes package purpose and dependencies
// Function-level: Describes function purpose, parameters, returns
// Method-level: Describes method behavior and side effects

## API Documentation
// Use go doc for package-level documentation
// Add parameter and return documentation
// Include examples where helpful
// Document error conditions

## README Files
// Provide high-level overview
// Document setup and configuration
// Include usage examples
// List available commands
```

### Security Practices

#### Database Security

```go
// Security patterns
// 1. Never log sensitive data
// 2. Use parameterized queries
// 3. Validate and sanitize inputs
// 4. Implement proper access controls

func CreateDocument(ctx context.Context, doc *Document) error {
    // Sanitize input data
    if err := validateAndSanitizeDocument(doc); err != nil {
        return fmt.Errorf("invalid document data: %w", err)
    }

    // Use parameterized queries
    query := "INSERT INTO documents (id, data, tenant_id) VALUES ($1, $2, $3)"
    _, err := db.Pool.Exec(ctx, query, doc.ID, doc.Data, doc.TenantID)
    if err != nil {
        return fmt.Errorf("database operation failed: %w", err)
    }

    return nil
}
```

#### Application Security

```bash
# Environment security
# Never hardcode credentials in source code
# Use environment variables or secret management
# Example .env file (never commit to version control)

DB_PASSWORD=prod_database_password
API_KEY=prod_api_key_12345
ENCRYPTION_KEY=base64_encoded_key

# Use .gitignore to exclude sensitive files
# Example .gitignore entries
.env
.env.local
*.pem
*.key
```

### Performance Optimization

#### Query Optimization

```sql
-- Use appropriate indexes
CREATE INDEX idx_documents_tenant_company_status 
  ON documents(tenant_id, company_id, status);

CREATE INDEX idx_document_lines_document_line_number
  ON document_lines(document_id, line_number);

-- Use partial indexes for filtered queries
CREATE INDEX idx_active_documents 
  ON documents(id) WHERE status = 'active';
```

#### Connection Pool Optimization

```go
// Connection pool configuration
func GetDatabaseConfig() database.DatabaseConfig {
    return database.DatabaseConfig{
        Host:     getEnv("DB_HOST", "localhost"),
        Port:     getEnvInt("DB_PORT", 5432),
        Database: getEnv("DB_NAME", "veyro"),
        Username: getEnv("DB_USER", "postgres"),
        Password: getEnv("DB_PASSWORD", ""),
        SSLMode:  getEnv("DB_SSL_MODE", "require"),
        PoolSize: getEnvInt("DB_POOL_SIZE", 10),
    }
}
```

### Troubleshooting

#### Common Issues and Solutions

**Issue: Database Connection Failed**

```bash
# Troubleshooting steps
# 1. Check if PostgreSQL is running
make docker-up
make db-check

# 2. Check connection parameters
# Verify environment variables
# Check firewall settings

# 3. Check database logs
docker compose logs postgres

# 4. Test connection manually
psql -h localhost -U postgres -c "SELECT 1"
```

**Issue: Migration Failed**

```bash
# Troubleshooting steps
# 1. Check migration logs
# Migration output is typically logged to stderr

# 2. Verify migration syntax
# Check for SQL syntax errors
# Ensure referenced tables exist

# 3. Test migrations on staging
# Before applying to production
```

**Issue: Query Performance**

```sql
-- Check query execution plan
EXPLAIN ANALYZE SELECT * FROM documents WHERE tenant_id = $1;

-- Create missing indexes
CREATE INDEX idx_documents_tenant_company_status ON documents(tenant_id, company_id, status);

-- Optimize query
-- Use specific columns instead of *
-- Add filters to reduce result set
```

### Development Environment Management

#### Docker Development Environment

```dockerfile
# Dockerfile for Veyro backend
FROM golang:1.21-alpine

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN go build -o app ./...

# Run application
CMD ["/app"]
```

#### Docker Compose Configuration

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:18
    environment:
      POSTGRES_DB: vevro
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_HOST_AUTH_METHOD: trust
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

## Code Reviews and Standards

### Pre-commit Checklist

```bash
# Before committing code
# 1. Check for any syntax errors
ggo vet ./...

# 2. Run linting
make lint

# 3. Run formatting
make fmt

# 4. Run tests
make test

# 5. Check git status
ggit status

# 6. Stage changes
ggit add .

# 7. Create commit message
ggit commit -m "feat: add new feature with proper description"

# 8. Push to remote branch
ggit push origin <branch-name>
```

### Pull Request Standards

```markdown
# Pull Request Standards

## Title Format
<type>: <description>

Types:
- feat: New feature
- fix: Bug fix
- docs: Documentation changes
- style: Code style changes
- refactor: Code refactoring
- test: Adding tests
- chore: Maintenance tasks

## Description
- Clear description of changes
- Links to related issues or PRs
- Screenshots or examples if helpful
- Testing performed

## Checklist
- [ ] Code follows project standards
- [ ] Tests added or updated
- [ ] Documentation updated
- [ ] No linting errors
- [ ] All tests passing
- [ ] Code review comments addressed
```

## Continuous Integration

### GitHub Actions Configuration

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - name: Set up PostgreSQL
        uses: ./.github/actions/setup-postgres
      - name: Install dependencies
        run: go mod download
      - name: Run tests
        run: make test
      - name: Run linting
        run: make lint
      - name: Run sqlc generation
        run: make sqlc
      - name: Verify generated code
        run: git diff --exit-code
```

### Local CI Environment

```bash
# Local CI simulation
# 1. Ensure clean state
git clean -fd

# 2. Check linting
make lint

# 3. Run tests
make test

# 4. Build application
make build

# 5. Run integration tests
make test-integration
```

## Development Resources

### Learning Resources

#### Go Language Resources
- [Official Go Documentation](https://golang.org/doc/)
- [Go by Example](https://gobyexample.com/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Good Practices](https://github.com/golang-standards/project-structure)

#### Database Resources
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [pgx Documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [sqlc Documentation](https://github.com/sqlc-dev/sqlc)
- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)

#### Development Tools
- [Docker](https://docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Visual Studio Code](https://code.visualstudio.com/)
- [Git](https://git-scm.com/)

### Support

#### Getting Help
- **GitHub Issues**: Report bugs, request features
- **Stack Overflow**: Database and Go programming questions
- **Discord/Slack**: Community support
- **Documentation**: In-code comments and markdown files

#### Contributing
- Fork the repository
- Create a feature branch
- Implement your changes
- Write tests
- Submit a pull request

## Conclusion

This document provides comprehensive guidance for developing, testing, and deploying the Veyro backend application. By following these standards and best practices, developers can:

1. **Write High-Quality Code**: Adhere to consistent coding standards and practices
2. **Ensure Code Quality**: Implement comprehensive testing and code review
3. **Maintain Development Environment**: Set up and manage development tools effectively
4. **Troubleshoot Issues**: Identify and resolve common development problems
5. **Deploy Applications**: Deploy changes reliably and safely

The Veyro development process emphasizes code quality, testing, and maintainability while supporting rapid development and continuous delivery.

---

*Document Version: 1.0*
*Last Updated: November 2026*
*Scope: Phase 2 Database Implementation*
*Status: Complete*