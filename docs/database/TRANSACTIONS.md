# Veyro Database Transactions

## Overview

This document provides comprehensive guidance on database transaction management and transaction infrastructure in the Veyro application.

## Transaction Architecture

### Core Principles

Veyro uses PostgreSQL transactions as the foundation for data consistency and atomicity. Transaction management follows these key principles:

1. **Atomicity**: All operations in a transaction succeed or fail together
2. **Consistency**: Transactions bring the database from one valid state to another
3. **Isolation**: Concurrent transactions do not interfere with each other
4. **Durability**: Committed transactions persist even after system failures

### Transaction Strategy

```go
// Veyro uses pgxpool with transaction support for all data operations
// Transactions are managed at the application level with explicit boundaries
// All business operations execute within transactions
```

## Transaction Infrastructure

### Database Package Transactions

#### Transaction Type

```go
//go:build ignore

package database

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Transaction represents a database transaction with helper methods
 type Transaction struct {
    tx pgx.Tx
    db  *Database
 }
```

#### Transaction Helper Functions

```go
// NewTransaction creates a new database transaction
func (db *Database) NewTransaction(ctx context.Context, txOptions pgx.TxOptions) (*Transaction, error) {
    tx, err := db.Pool.BeginTx(ctx, txOptions)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }

    return &Transaction{tx: tx, db: db}, nil
 }

// WithTransaction provides a helper to execute multiple operations within a transaction
func (db *Database) WithTransaction(ctx context.Context, fn func(*Transaction) error, txOptions pgx.TxOptions) error {
    tx, err := db.NewTransaction(ctx, txOptions)
    if err != nil {
        return err
    }
    defer func() {
        if err := tx.Rollback(ctx); errors.Is(err, pgx.ErrTxClosed) {
            // Transaction already committed or rolled back
        } else if err != nil {
            // Log rollback error but don't fail the operation
        }
    }()

    if err := fn(tx); err != nil {
        // Rollback on any error unless it's already been committed
        if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
            return fmt.Errorf("transaction rollback failed: %w", rollbackErr)
        }
        return err
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("transaction commit failed: %w", err)
    }

    return nil
 }
```

### Transaction Isolation Levels

#### Available Isolation Levels

```go
// pgx.TxOptions defines transaction characteristics
// Veyro uses Serializable for financial and inventory operations

txOptions := pgx.TxOptions{
    IsoLevel: pgx.Serializable, // Strongest isolation
}

// Simpler transactions for non-critical operations
txOptionsSimple := pgx.TxOptions{
    IsoLevel: pgx.ReadCommitted,
}

// Default transaction options
txOptionsDefault := pgx.TxOptions{
    IsoLevel: pgx.ReadCommitted,
}
```

#### Isolation Level Selection

```go
func getTransactionOptions(operationType string) pgx.TxOptions {
    switch operationType {
    case "financial_posting", "journal_entry", "stock_movement", "payment_allocation":
        // Use Serializable for critical financial operations
        return pgx.TxOptions{IsoLevel: pgx.Serializable}
    case "document_creation", "item_update", "party_creation":
        // Use ReadCommitted for document and master data
        return pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
    case "report_generation", "data_export":
        // Use RepeatableRead for analytics queries
        return pgx.TxOptions{IsoLevel: pgx.RepeatableRead}
    default:
        // Default to ReadCommitted
        return pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
    }
}
```

## Transaction Conventions

### Application-Level Transaction Management

#### Transaction Scoping

```go
// Veyro follows strict transaction scoping rules:

// Rule 1: One transaction per business operation
func CreateInvoice(ctx context.Context, doc *Document) error {
    err := db.WithTransaction(ctx, func(tx *Transaction) error {
        // Create document
        if err := createDocumentTx(ctx, tx, doc); err != nil {
            return err
        }

        // Create document lines
        if err := createDocumentLinesTx(ctx, tx, doc); err != nil {
            return err
        }

        // Post to accounting
        if err := postInvoiceToAccountingTx(ctx, tx, doc); err != nil {
            return err
        }

        // Update inventory if applicable
        if doc.TrackInventory {
            if err := updateInventoryTx(ctx, tx, doc); err != nil {
                return err
            }
        }

        return nil
    });
    
    return err
}

// Rule 2: No nested transactions
// Transactions should not be nested within other transactions
// Use savepoints if nested behavior is required
```

#### Transaction Best Practices

```go
// Best Practice 1: Keep transactions focused and short
func CreateSalesInvoice(ctx context.Context, doc *Document) error {
    // Transaction scope should be limited to invoice creation
    // Avoid external network calls or long-running operations
    return db.WithTransactionForUpdate(ctx, func(tx *Transaction) error {
        // Invoice creation logic here
    })
}

// Best Practice 2: Fail fast on errors
func CreateComplexDocument(ctx context.Context, doc *Document) error {
    return db.WithTransaction(ctx, func(tx *Transaction) error {
        if err := validateDocumentTx(ctx, tx, doc); err != nil {
            return err // Rollback immediately
        }

        if err := createDocumentTx(ctx, tx, doc); err != nil {
            return err
        }

        // Continue with other operations...
        return nil
    })
}

// Best Practice 3: Proper error handling
func UpdateDocumentWithRelationships(ctx context.Context, doc *Document) error {
    return db.WithTransaction(ctx, func(tx *Transaction) error {
        // Attempt operations
        if err := performOperation1Tx(ctx, tx, doc); err != nil {
            return fmt.Errorf("operation 1 failed: %w", err)
        }

        if err := performOperation2Tx(ctx, tx, doc); err != nil {
            return fmt.Errorf("operation 2 failed: %w", err)
        }

        if err := performOperation3Tx(ctx, tx, doc); err != nil {
            return fmt.Errorf("operation 3 failed: %w", err)
        }

        return nil
    })
}
```

### Transaction Usage Patterns

#### Pattern 1: Simple Document Creation

```go
func CreateDocument(ctx context.Context, doc *Document) error {
    return db.WithTransaction(ctx, func(tx *Transaction) error {
        // Create document header
        if err := tx.ExecQuery(ctx, 
            "INSERT INTO documents (id, tenant_id, company_id, document_type, status) VALUES ($1, $2, $3, $4, $5)",
            doc.ID, doc.TenantID, doc.CompanyID, doc.DocumentType, doc.Status); err != nil {
            return err
        }

        // Create document lines
        for _, line := range doc.Lines {
            if err := tx.ExecQuery(ctx,
                "INSERT INTO document_lines (id, document_id, line_number, item_id, quantity, unit_rate) VALUES ($1, $2, $3, $4, $5, $6)",
                line.ID, doc.ID, line.LineNumber, line.ItemID, line.Quantity, line.UnitRate); err != nil {
                return err
            }
        }

        return nil
    })
}
```

#### Pattern 2: Financial Posting with Multiple Operations

```go
func PostSalesInvoice(ctx context.Context, doc *Document) error {
    return db.WithTransactionForUpdate(ctx, func(tx *Transaction) error {
        // 1. Create document
        if err := createDocumentTx(ctx, tx, doc); err != nil {
            return err
        }

        // 2. Create document lines
        if err := createDocumentLinesTx(ctx, tx, doc); err != nil {
            return err
        }

        // 3. Create journal entries
        if err := createJournalEntriesTx(ctx, tx, doc); err != nil {
            return err
        }

        // 4. Update party open items
        if err := updatePartyOpenItemsTx(ctx, tx, doc); err != nil {
            return err
        }

        // 5. Update stock balances
        if err := updateStockBalancesTx(ctx, tx, doc); err != nil {
            return err
        }

        // 6. Create posting batch
        if err := createPostingBatchTx(ctx, tx, doc); err != nil {
            return err
        }

        return nil
    })
}
```

#### Pattern 3: Compensation Operations

```go
func CreateCompensationTransaction(ctx context.Context, sourceDoc *Document, targetDoc *Document) error {
    return db.WithTransaction(ctx, func(tx *Transaction) error {
        // Create compensation document
        compDoc := createCompensationDocument(ctx, sourceDoc, targetDoc)
        if err := createDocumentTx(ctx, tx, compDoc); err != nil {
            return err
        }

        // Create compensation lines
        if err := createCompensationLinesTx(ctx, tx, compDoc); err != nil {
            return err
        }

        // Update original document with compensation reference
        if err := updateOriginalDocumentWithCompensationTx(ctx, tx, sourceDoc, compDoc); err != nil {
            return err
        }

        // Update target document with compensation reference
        if err := updateTargetDocumentWithCompensationTx(ctx, tx, targetDoc, compDoc); err != nil {
            return err
        }

        return nil
    })
}
```

## Transaction Testing

### Test Scenarios

#### Test 1: Successful Transaction

```go
func TestSuccessfulTransaction(t *testing.T) {
    db := getTestDatabase(t)
    defer db.Close()

    err := db.WithTransaction(context.Background(), func(tx *Transaction) error {
        // Perform operations that should succeed
        _, err := tx.ExecQuery(context.Background(), "INSERT INTO test_table (id, value) VALUES ($1, $2)", "test1", 100)
        if err != nil {
            return err
        }

        _, err = tx.ExecQuery(context.Background(), "INSERT INTO test_table (id, value) VALUES ($1, $2)", "test2", 200)
        if err != nil {
            return err
        }

        return nil
    })

    require.NoError(t, err)

    // Verify data was committed
    var count int
    err = db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM test_table").Scan(&count)
    require.NoError(t, err)
    assert.Equal(t, 2, count)
}
```

#### Test 2: Failed Transaction Rollback

```go
func TestFailedTransactionRollback(t *testing.T) {
    db := getTestDatabase(t)
    defer db.Close()

    // Create initial data
    _, err := db.Pool.Exec(context.Background(), "INSERT INTO test_table (id, value) VALUES ($1, $2)", "initial", 50)
    require.NoError(t, err)

    // Attempt transaction that will fail
    err = db.WithTransaction(context.Background(), func(tx *Transaction) error {
        // Successful operation
        _, err := tx.ExecQuery(context.Background(), "UPDATE test_table SET value = $1 WHERE id = $2", 75, "initial")
        if err != nil {
            return err
        }

        // Simulate an error
        return errors.New("simulated error for testing rollback")
    })

    require.Error(t, err)

    // Verify that the rollback occurred (data should be back to original state)
    var value int
    err = db.Pool.QueryRow(context.Background(), "SELECT value FROM test_table WHERE id = $1", "initial").Scan(&value)
    require.NoError(t, err)
    assert.Equal(t, 50, value) // Should still be 50, not 75
}
```

#### Test 3: Isolation Between Transactions

```go
func TestTransactionIsolation(t *testing.T) {
    db := getTestDatabase(t)
    defer db.Close()

    // Start first transaction
    err := db.WithTransaction(context.Background(), func(tx1 *Transaction) error {
        // Insert data in first transaction
        _, err := tx1.ExecQuery(context.Background(), "INSERT INTO shared_table (id, value) VALUES ($1, $2)", "tx1", 100)
        if err != nil {
            return err
        }

        // Read data in first transaction
        var value int
        err := tx1.QueryRow(context.Background(), "SELECT value FROM shared_table WHERE id = $1", "tx1").Scan(&value)
        if err != nil {
            return err
        }
        assert.Equal(t, 100, value)

        return nil
    })
    require.NoError(t, err)

    // Start second transaction
    err = db.WithTransaction(context.Background(), func(tx2 *Transaction) error {
        // Should not see data from first transaction (depending on isolation level)
        // With ReadCommitted, tx2 won't see uncommitted changes from tx1
        return nil
    })
    require.NoError(t, err)
}
```

#### Test 4: Nested Transaction Attempts

```go
func TestNestedTransactionPrevention(t *testing.T) {
    db := getTestDatabase(t)
    defer db.Close()

    err := db.WithTransaction(context.Background(), func(tx1 *Transaction) error {
        // Attempt nested transaction (should not be allowed)
        err := db.WithTransaction(context.Background(), func(tx2 *Transaction) error {
            return tx2.ExecQuery(context.Background(), "INSERT INTO nested_test (id) VALUES ($1)", "nested")
        })

        // This should either fail or the nested transaction should be absorbed
        // depending on the implementation
        return err
    })

    // The behavior should be documented
    t.Logf("Nested transaction behavior: %v", err)
}
```

## Transaction Monitoring

### Transaction Tracking

```go
// Transaction tracking for monitoring and debugging
type TransactionTracker struct {
    db *Database
    activeTransactions map[string]*ActiveTransaction
    mutex sync.RWMutex
}

func (tt *TransactionTracker) StartTransaction(id string, operation string, ctx context.Context) {
    tt.mutex.Lock()
    defer tt.mutex.Unlock()

    tt.activeTransactions[id] = &ActiveTransaction{
        ID: id,
        Operation: operation,
        StartTime: time.Now(),
        Context: ctx,
    }
}

func (tt *TransactionTracker) EndTransaction(id string, status string, duration time.Duration) {
    tt.mutex.Lock()
    defer tt.mutex.Unlock()

    if trans, exists := tt.activeTransactions[id]; exists {
        trans.EndTime = time.Now()
        trans.Status = status
        trans.Duration = duration
        delete(tt.activeTransactions, id)
    }
}

// Transaction metrics
type TransactionMetrics struct {
    ActiveCount    int
    CommittedCount int
    RolledBackCount int
    AverageDuration time.Duration
    ErrorCount      int
}

func (tt *TransactionTracker) GetMetrics() TransactionMetrics {
    tt.mutex.RLock()
    defer tt.mutex.RUnlock()

    metrics := TransactionMetrics{}
    for _, trans := range tt.activeTransactions {
        metrics.ActiveCount++
    }

    // Calculate additional metrics...
    return metrics
}
```

### Transaction Health Checks

```go
// Transaction health verification
func (db *Database) CheckTransactionHealth(ctx context.Context) error {
    // Check if there are any long-running transactions
    var longRunningTx int
    query := """
    SELECT COUNT(*) FROM pg_stat_activity 
    WHERE state = 'active' 
      AND now() - query_start > INTERVAL '5 minutes'
      AND datname = current_database()
    """

    err := db.Pool.QueryRow(ctx, query).Scan(&longRunningTx)
    if err != nil {
        return err
    }

    if longRunningTx > 0 {
        return fmt.Errorf("found %d long-running transactions", longRunningTx)
    }

    // Check connection pool health
    stats := db.ConnectionPoolStats()
    if stats.IdleConns() < 2 { // Minimum healthy connections
        return fmt.Errorf("connection pool has insufficient idle connections: %d", stats.IdleConns())
    }

    return nil
}
```

## Transaction Retry Logic

### Transient Error Handling

```go
func (db *Database) WithRetry(ctx context.Context, fn func() error, maxRetries int) error {
    var err error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        err = fn()
        if err == nil {
            return nil
        }

        // Check if error is retryable
        if !IsRetryableError(err) {
            return err // Not retryable, return immediately
        }

        // Exponential backoff
        backoffDuration := time.Duration(1<<attempt) * time.Second
        select {
        case <-time.After(backoffDuration):
            // Continue to next attempt
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return fmt.Errorf("max retries exceeded: %w", err)
}

func IsRetryableError(err error) bool {
    // Classification from database error handling
    dbErr := ClassifyDatabaseError(err)
    if dbErr == nil {
        return false
    }

    switch dbErr.Code {
    case ErrorCodeTimeout, ErrorCodeSerializationFailure, ErrorCodeDeadlockDetected:
        return true
    default:
        return false
    }
}
```

### Deadlock Handling

```go
func (db *Database) WithDeadlockHandling(ctx context.Context, fn func() error) error {
    err := fn()
    if err == nil {
        return nil
    }

    // Check for deadlock error
    dbErr := ClassifyDatabaseError(err)
    if dbErr == nil || dbErr.Code != ErrorCodeDeadlockDetected {
        return err // Not a deadlock
    }

    // Deadlock detected, wait and retry
    select {
    case <-time.After(100 * time.Millisecond):
        // Wait for deadlock to be resolved
    case <-ctx.Done():
        return ctx.Err()
    }

    // Retry the operation
    return fn()
}
```

## Transaction Documentation

### Transaction Guidelines

#### When to Use Different Isolation Levels

```go
// Serializable: Financial postings, inventory operations
func PostSalesInvoice(ctx context.Context, doc *Document) error {
    return db.WithTransactionForUpdate(ctx, func(tx *Transaction) error {
        // Financial operations requiring strongest isolation
    })
}

// ReadCommitted: Document creation, updates
func CreateDocument(ctx context.Context, doc *Document) error {
    return db.WithTransaction(ctx, func(tx *Transaction) error {
        // Document operations with moderate isolation
    })
}

// RepeatableRead: Analytics, reporting
func GenerateFinancialReport(ctx context.Context, reportID string) error {
    txOptions := pgx.TxOptions{IsoLevel: pgx.RepeatableRead}
    return db.WithTransaction(ctx, func(tx *Transaction) error {
        // Read-only analytics operations
    }, txOptions)
}
```

#### Transaction Size Guidelines

```go
// Small transactions (recommended)
func CreateInvoice(ctx context.Context, doc *Document) error {
    // Should complete quickly (< 1 second)
    return db.WithTransaction(ctx, func(tx *Transaction) error {
        // Invoice creation logic
    })
}

// Medium transactions (acceptable)
func PostSalesInvoice(ctx context.Context, doc *Document) error {
    // May take longer but still manageable (< 5 seconds)
    return db.WithTransactionForUpdate(ctx, func(tx *Transaction) error {
        // Multiple related operations
    })
}

// Large transactions (discouraged)
// Complex multi-step operations spanning multiple business modules
// Consider splitting into smaller, focused transactions
```

### Transaction Best Practices

1. **Keep Transactions Short**: Minimize transaction duration to improve concurrency
2. **Error Handling**: Implement proper error handling and rollback logic
3. **Isolation Level Selection**: Choose appropriate isolation levels for different operations
4. **Retry Logic**: Implement retry for transient failures
5. **Monitoring**: Track transaction metrics for performance optimization
6. **Deadlock Handling**: Implement deadlock detection and resolution
7. **Resource Management**: Ensure proper connection pool management

## Transaction Recovery

### Recovery Strategies

```go
// Connection recovery
func (db *Database) EnsureConnectionHealth(ctx context.Context) error {
    // Check connection pool health
    stats := db.ConnectionPoolStats()

    // If connection pool is unhealthy, try to recover
    if stats.IdleConns() <= 0 {
        // Attempt to recreate connection pool
        if err := db.reconnect(ctx); err != nil {
            return fmt.Errorf("connection recovery failed: %w", err)
        }
    }

    return nil
}

// Transaction recovery after crash
func (db *Database) RecoverTransactions(ctx context.Context) error {
    // Identify incomplete transactions
    incompleteTxs, err := db.identifyIncompleteTransactions(ctx)
    if err != nil {
        return err
    }

    // Attempt to rollback incomplete transactions
    for _, txID := range incompleteTxs {
        if err := db.forceRollback(ctx, txID); err != nil {
            // Log error but continue with other transactions
            continue
        }
    }

    return nil
}
```

## Conclusion

Transaction management is critical for the reliability and consistency of the Veyro application. By implementing:

1. **Robust Transaction Infrastructure**: Transaction helper functions and isolation management
2. **Comprehensive Error Handling**: Error classification and retry logic
3. **Performance Optimization**: Efficient transaction sizing and isolation levels
4. **Monitoring and Alerting**: Transaction tracking and health monitoring
5. **Recovery Mechanisms**: Connection and transaction recovery strategies

The Veyro database ensures data integrity, consistency, and performance for all business operations while maintaining high availability and operational excellence.

---

*Document Version: 1.0*
*Last Updated: November 2026*
*Scope: Phase 2 Database Implementation*
*Status: Complete*