package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool connects to an isolated migrated database when TEST_DATABASE_URL is
// set. Integration tests never run against a production database.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping database integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
	return pool
}

func TestCriticalTablesExist(t *testing.T) {
	pool := testPool(t)
	for _, table := range []string{
		"tenants", "users", "user_sessions", "tenant_memberships", "roles", "permissions",
		"companies", "branches", "warehouses", "accounts", "financial_years",
		"accounting_periods", "document_number_series", "tenant_invitations",
		"outbox_events", "idempotency_records", "audit_events",
	} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS (SELECT 1 FROM information_schema.tables
			 WHERE table_schema='public' AND table_name=$1)`, table).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestTenantScopedQueriesAndPrimaryConstraint(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	var tenantA, tenantB string
	if err := tx.QueryRow(ctx, `INSERT INTO tenants (name, slug) VALUES ('A','isolation-a') RETURNING id`).Scan(&tenantA); err != nil {
		t.Fatalf("insert tenant A: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO tenants (name, slug) VALUES ('B','isolation-b') RETURNING id`).Scan(&tenantB); err != nil {
		t.Fatalf("insert tenant B: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO companies (tenant_id, trade_name, is_primary)
		VALUES ($1,'Company A',true)`, tenantA); err != nil {
		t.Fatalf("insert company: %v", err)
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM companies WHERE tenant_id=$1`, tenantB).Scan(&count); err != nil {
		t.Fatalf("count tenant B companies: %v", err)
	}
	if count != 0 {
		t.Fatalf("tenant B must not see tenant A's company, found %d", count)
	}

	// A tenant may have only one active primary company.
	if _, err := tx.Exec(ctx, `INSERT INTO companies (tenant_id, trade_name, is_primary)
		VALUES ($1,'Second primary',true)`, tenantA); err == nil {
		t.Fatal("expected a second primary company to violate the unique index")
	}
}
