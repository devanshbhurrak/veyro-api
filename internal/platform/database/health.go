package database

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ReadyHandler returns an HTTP handler that checks database readiness
func ReadyHandler(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set content type
		w.Header().Set("Content-Type", "application/json")

		// Check database readiness with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := db.HealthCheck(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready"}`))
			return
		}

		// Check if we can access specific critical tables
		if err := db.CheckCriticalTables(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready"}`))
			return
		}

		// Return ready status
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}
}

// HealthCheck verifies basic database connectivity
func (db *Database) HealthCheck(ctx context.Context) error {
	// Test connection with a timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Try to execute a simple query
	var result int
	query := "SELECT 1"
	if err := db.Pool.QueryRow(ctx, query).Scan(&result); err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	return nil
}

// CheckCriticalTables verifies that all critical tables exist and are accessible
func (db *Database) CheckCriticalTables(ctx context.Context) error {
	criticalTables := []string{
		"tenants",
		"users",
		"companies",
		"accounts",
		"documents",
	}

	for _, table := range criticalTables {
		var exists bool
		query := `
            SELECT EXISTS (
                SELECT 1 
                FROM information_schema.tables 
                WHERE table_schema = 'public' 
                AND table_name = $1
            )
        `
		if err := db.Pool.QueryRow(ctx, query, table).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}

		if !exists {
			return fmt.Errorf("critical table %s does not exist", table)
		}
	}

	return nil
}

// GetDatabaseStats returns detailed database statistics for monitoring
func (db *Database) GetDatabaseStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get basic pool statistics
	if db.Pool != nil {
		poolStats := db.Pool.Stat()
		stats["total_conns"] = poolStats.TotalConns()
		stats["acquired_conns"] = poolStats.AcquiredConns()
		stats["idle_conns"] = poolStats.IdleConns()
		stats["max_conns"] = poolStats.MaxConns()
	}

	// Try to get table row counts for critical tables
	criticalTables := []string{
		"tenants",
		"users",
		"companies",
		"accounts",
		"documents",
	}

	rowCounts := make(map[string]int64)
	for _, table := range criticalTables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := db.Pool.QueryRow(ctx, query).Scan(&count); err == nil {
			rowCounts[table] = count
		}
		// Continue even if some tables can't be counted (permissions, etc.)
	}

	stats["table_row_counts"] = rowCounts

	// Get database size (approximate)
	var dbSizeBytes int64
	sizeQuery := "SELECT pg_database_size(current_database())"
	if err := db.Pool.QueryRow(ctx, sizeQuery).Scan(&dbSizeBytes); err == nil {
		stats["database_size_bytes"] = dbSizeBytes
	}

	return stats, nil
}

// ConnectionPoolStats returns connection pool statistics for monitoring
func (db *Database) ConnectionPoolStats() map[string]interface{} {
	if db.Pool == nil {
		return nil
	}

	stats := db.Pool.Stat()
	return map[string]interface{}{
		"total_conns":    stats.TotalConns(),
		"acquired_conns": stats.AcquiredConns(),
		"idle_conns":     stats.IdleConns(),
		"max_conns":      stats.MaxConns(),
	}
}
