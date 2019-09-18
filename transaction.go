package postgres

import (
	"context"
	"database/sql"
	"time"
)

type Transaction struct {
	tx     *sql.Tx
	logger Logger // inherited from parent Postgres instance
}

// NewTransaction starts a new transaction.
func (p *Postgres) NewTransaction() (*Transaction, error) {
	tx, err := p.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		logger: p.Logger,
		tx:     tx,
	}, nil
}

// TransactionFunc is a callback called by Postgres.Transaction.
type TransactionFunc func(*Transaction) error

// Transaction starts a new transaction and automatically commits or
// rolls back the transaction if TransactionFunc returns an error.
func (p *Postgres) Transaction(fn TransactionFunc) error {
	tx, err := p.NewTransaction()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback() // TODO what if this errors?
		return err
	}

	return tx.Commit()
}

// Commit commits the transaction. Commit or Rollback must be called at least once,
// so the connection can be returned to the connection pool.
func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

// Rollback aborts the transaction. Rollback or Commit must be called at least once,
// so the connection can be returned to the connection pool.
// See Commit for an example.
func (t *Transaction) Rollback() error {
	return t.tx.Rollback()
}

// Get finds a record by its primary keys. See Postgres.Get for more details.
func (t *Transaction) Get(ctx context.Context, s Struct) error {
	return getStruct(t, ctx, s)
}

// Filter finds records based on QueryStmt. See Postgres.Filter for more details.
func (t *Transaction) Filter(ctx context.Context, s StructSlice, q *QueryStmt) error {
	return filterStruct(t, ctx, s, q)
}

// Insert creates a new record. See Postgres.Insert for more details.
func (t *Transaction) Insert(ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	return insertStruct(t, ctx, s, fieldMask...)
}

// Update updates an existing record by looking at the orimary keys of a struct.
// See Postgres.Update for more details.
func (t *Transaction) Update(ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	return updateStruct(t, ctx, s, fieldMask...)
}

// Save creates a new record or updates an existing record by looking at
// the primary keys of a struct. See Postgres.Filter for more details.
func (t *Transaction) Save(ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	return saveStruct(t, ctx, s, fieldMask...)
}

// Delete deletes a record by looking at the primary keys of a struct.
// See Postgres.Delete for more details.
func (t *Transaction) Delete(ctx context.Context, s Struct) error {
	return deleteStruct(t, ctx, s)
}

// Exec executes a query that doesn't return rows. For example: an INSERT and UPDATE.
func (t *Transaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	query = formatQuery(query)
	start := time.Now()
	r, err := t.tx.ExecContext(ctx, query, args...)
	t.logQuery(query, time.Since(start), args...)
	return r, err
}

// Query executes a query that returns rows, typically a SELECT.
func (t *Transaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	query = formatQuery(query)
	start := time.Now()
	r, err := t.tx.QueryContext(ctx, query, args...)
	t.logQuery(query, time.Since(start), args...)
	return r, err
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	query = formatQuery(query)
	start := time.Now()
	r := t.tx.QueryRowContext(ctx, query, args...)
	t.logQuery(query, time.Since(start), args...)
	return r
}

func (t *Transaction) logQuery(query string, duration time.Duration, args ...interface{}) {
	queryLog(t.logger, query, duration, args...)
}
