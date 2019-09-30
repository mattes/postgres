package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jpillora/backoff"
)

var (
	// ConnectTimeout is the max time to wait for Postgres to be reachable.
	ConnectTimeout = 5 * time.Second

	// MigrateKey is a random key for Postgres' advisory lock.
	// It must be the same for all running Migrate funcs.
	MigrateKey = 8267205493056421913

	// maxAdvisoryLockAttemtps is the max amount of times it tries
	// to acquire a lock before it panics
	maxAdvisoryLockAttemtps = 50
)

type rowScan interface {
	Scan(dest ...interface{}) error
}

type Postgres struct {
	uri    string
	db     *sql.DB
	Logger Logger

	// createTempTables can be set to true to just create temporary tables,
	// useful for tests
	createTempTables bool
}

// Open creates a new Postgres client.
//
// To set a schema, specify `search_path` in URI.
//
// To only create temporary tables, i.e. for testing purposes,
// specify `createTempTables=true` in URI.
func Open(uri string) (*Postgres, error) {
	// parse uri
	extra, err := parseExtraParams(&uri)
	if err != nil {
		return nil, err
	}

	// reset env vars
	resetEnv()

	// create sql.DB instance
	db, err := sql.Open("postgres", uri)
	if err != nil {
		return nil, err
	}

	// ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), ConnectTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	p := &Postgres{
		uri:              uri,
		db:               db,
		createTempTables: extra.createTempTables,
	}

	// verify min version
	minVersion, err := p.isMinVersion(11)
	if err != nil {
		return nil, err
	}

	if !minVersion {
		return nil, fmt.Errorf("Postgres version >= 11 required")
	}

	return p, nil
}

// clone creates a new instance of *Postgres.
// The new *Postgres instance has its own new connection pool.
func (p *Postgres) clone() (*Postgres, error) {
	px, err := Open(p.uri)
	if err != nil {
		return nil, err
	}

	px.Logger = p.Logger
	return px, nil
}

// Close closes the database and prevents new queries from starting.
// Close then waits for all queries that have started processing on the server
// to finish.
//
// It is rare to Close a DB, as the DB handle is meant to be
// long-lived and shared between many goroutines.
func (p *Postgres) Close() error {
	return p.db.Close()
}

// DB returns the underlying *sql.DB database.
func (p *Postgres) DB() *sql.DB {
	return p.db
}

// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
//
// Expired connections may be closed lazily before reuse.
//
// If d <= 0, connections are reused forever.
func (p *Postgres) SetConnMaxLifetime(d time.Duration) {
	p.db.SetConnMaxLifetime(d)
}

// SetMaxIdleConns sets the maximum number of connections in the idle
// connection pool.
//
// If MaxOpenConns is greater than 0 but less than the new MaxIdleConns,
// then the new MaxIdleConns will be reduced to match the MaxOpenConns limit.
//
// If n <= 0, no idle connections are retained.
//
// The default max idle connections is currently 2. This may change in
// a future release.
func (p *Postgres) SetMaxIdleConns(n int) {
	p.db.SetMaxIdleConns(n)
}

// SetMaxOpenConns sets the maximum number of open connections to the database.
//
// If MaxIdleConns is greater than 0 and the new MaxOpenConns is less than
// MaxIdleConns, then MaxIdleConns will be reduced to match the new
// MaxOpenConns limit.
//
// If n <= 0, then there is no limit on the number of open connections.
// The default is 0 (unlimited).
func (p *Postgres) SetMaxOpenConns(n int) {
	p.db.SetMaxOpenConns(n)
}

// Stats returns database statistics.
func (p *Postgres) Stats() sql.DBStats {
	return p.db.Stats()
}

// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
func (p *Postgres) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Get finds a record by its primary keys.
func (p *Postgres) Get(ctx context.Context, s Struct) error {
	return getStruct(p, ctx, s)
}

// Filter finds records based on QueryStmt. See QueryStmt for more details.
func (p *Postgres) Filter(ctx context.Context, s StructSlice, q *QueryStmt) error {
	return filterStruct(p, ctx, s, q)
}

// Insert creates a new record.
func (p *Postgres) Insert(ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	return insertStruct(p, ctx, s, fieldMask...)
}

// Update updates an existing record by looking at the orimary keys of a struct.
func (p *Postgres) Update(ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	return updateStruct(p, ctx, s, fieldMask...)
}

// Save creates a new record or updates an existing record by looking at
// the primary keys of a struct.
func (p *Postgres) Save(ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	return saveStruct(p, ctx, s, fieldMask...)
}

// Delete deletes a record by looking at the primary keys of a struct.
func (p *Postgres) Delete(ctx context.Context, s Struct) error {
	return deleteStruct(p, ctx, s)
}

// Migrate runs SQL migrations for structs registered with `Register`.
// Migrations are non-destructive and only backwards-compatible changes
// will be performed, in particular:
//  * New tables for structs are created (this includes creation of primary key)
//  * New columns for struct fields are created
//  * New indexes are created
//  * New unique indexes are created (if possible)
//  * New foreign keys are created (if possible)
//
// Migrate blocks until it successfully acquired a global lock using Postgres' advisory locks.
// This guarantees that only one Migrate function can run at a time across different processes.
//
// The performed migrations as mentioned above are idempotent.
func (p *Postgres) Migrate(ctx context.Context) error {
	// TODO implement context.Context

	// create new postgres instance and only allow it to have 1 connection
	px, err := p.clone()
	if err != nil {
		return err
	}

	// only allow 1 connection that is immediately closed when done
	px.SetMaxOpenConns(1)
	px.SetMaxIdleConns(0)

	// acquire new lock on connection
	px.waitForAdvisoryLock(MigrateKey)

	// when done, release lock on same connection and close the cloned postgres instance
	defer func() {
		px.advisoryUnlock(MigrateKey)
		px.Close()
	}()

	// run the following commands on same postgres connection via `px` ...

	for _, r := range structs {
		if err := px.ensureTable(r); err != nil {
			return err
		}
	}

	for _, r := range structs {
		if err := px.ensureForeignKeys(r); err != nil {
			return err
		}
	}

	return nil
}

// EnsureTable creates table if it doesn't exist and makes sure
// primary keys, unique indexes and indexes are set correctly.
// It is non-destructive, and will not delete existing columns for example.
func (p *Postgres) ensureTable(r *metaStruct) error {
	// get details about table
	tbl, err := p.describeTable(toSnake(r.name))
	if isErrTableDoesNotExist(err) {

		// create table first
		if err := p.createTable(r); err != nil {
			return err
		}

		// load fresh details
		tbl, err = p.describeTable(toSnake(r.name))
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	// ensure table columns match struct fields
	// (only add new columns, existing columns are not deleted)
	for _, f := range r.fields {
		if !tbl.hasColumnByName(f.name) {
			if err := p.addColumn(toSnake(r.name), toSnake(f.name), f.columnType()); err != nil {
				return err
			}
		}
	}

	// ensure primary key
	primaryNames := r.fields.primaryNames()
	if len(primaryNames) > 0 {
		if !tbl.hasIndex(index{
			Name:      toSnake(r.name, "pk"),
			Type:      "btree",
			Columns:   primaryNames,
			IsUnique:  true,
			IsPrimary: true,
		}) {
			if err := p.createIndex(toSnake(r.name, "pk"), r.name, primaryNames, true, !r.fields.hasPartitionedField()); err != nil {
				return err
			}
			if err := p.addPrimaryKey(toSnake(r.name), toSnake(r.name, "pk"), toSnake(r.name, "pk")); err != nil {
				return err
			}
		}
	}

	// ensure unique indexes
	uniqueIndexes := r.fields.uniqueIndexes()
	if len(uniqueIndexes) > 0 {

		// add missing unique indexes
		for indexName, fieldNames := range uniqueIndexes {
			if !tbl.hasIndex(index{
				Name:     toSnake(r.name, indexName),
				Type:     "btree",
				Columns:  fieldNames,
				IsUnique: true,
			}) {
				if err := p.createIndex(toSnake(r.name, indexName), toSnake(r.name), fieldNames, true, !r.fields.hasPartitionedField()); err != nil {
					return err
				}
			}
		}
	}

	// ensure indexes
	indexes := r.fields.indexes()
	if len(indexes) > 0 {

		// add missing indexes
		for indexName, fieldNames := range indexes {
			if !tbl.hasIndex(index{
				Name:    toSnake(r.name, indexName),
				Type:    "btree",
				Columns: fieldNames,
			}) {
				if err := p.createIndex(toSnake(r.name, indexName), toSnake(r.name), fieldNames, false, !r.fields.hasPartitionedField()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// EnsureForeignKeys creates foreign keys if they don't already exist.
// This is a separate functon from EnsureTable as all tables have to exist first
// in order to create foreign keys.
func (p *Postgres) ensureForeignKeys(r *metaStruct) error {
	// ensure foreign keys
	for _, f := range r.fields {
		if f.foreignKeys != nil {
			for _, fk := range f.foreignKeys {

				refTbl, err := p.describeTable(toSnake(fk.structName))
				if err != nil {
					return err
				}

				// add unique index on referenced columns
				if !refTbl.hasUniqueIndexByColumns(fk.fieldNames) {
					if err := p.createIndex(toSnake(fk.structName, join(fk.fieldNames), "unique"), fk.structName, fk.fieldNames, true, !r.fields.hasPartitionedField()); err != nil {
						return err
					}
				}

				// add missing foreign keys
				exists, err := p.constraintExists(toSnake(r.name, f.name, "fk"))
				if err != nil {
					return err
				}
				if !exists {
					if err := p.addForeignKey(toSnake(r.name), toSnake(r.name, f.name, "fk"), []string{f.name}, fk.structName, fk.fieldNames); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// Exec executes a query that doesn't return rows. For example: an INSERT and UPDATE.
func (p *Postgres) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	query = formatQuery(query)
	start := time.Now()
	r, err := p.db.ExecContext(ctx, query, args...)
	p.logQuery(query, time.Since(start), args...)
	return r, err
}

// Query executes a query that returns rows, typically a SELECT.
func (p *Postgres) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	query = formatQuery(query)
	start := time.Now()
	r, err := p.db.QueryContext(ctx, query, args...)
	p.logQuery(query, time.Since(start), args...)
	return r, err
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (p *Postgres) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	query = formatQuery(query)
	start := time.Now()
	r := p.db.QueryRowContext(ctx, query, args...)
	p.logQuery(query, time.Since(start), args...)
	return r
}

func (p *Postgres) logQuery(query string, duration time.Duration, args ...interface{}) {
	queryLog(p.Logger, query, duration, args...)
}

func (p *Postgres) truncate(tableName string) error {
	_, err := p.Exec(context.Background(),
		fmt.Sprintf("TRUNCATE TABLE %v RESTART IDENTITY CASCADE", mustIdentifier(tableName)))
	return err
}

func (p *Postgres) createTable(r *metaStruct) error {
	q := queryf()
	q.Append("CREATE")

	if p.createTempTables {
		q.Append("TEMPORARY")
	}

	q.Appendf("TABLE IF NOT EXISTS %v", mustIdentifier(r.name))

	q.Append("(")

	// add columns
	fields := make([]string, 0)
	for _, f := range r.fields {
		fields = append(fields, fmt.Sprintf("%v %v", mustIdentifier(f.name), f.columnType()))
	}
	q.Append(join(fields))

	// add primary key
	if len(r.fields.primaryNames()) > 0 {
		q.Appendf(", CONSTRAINT %v PRIMARY KEY (%v)",
			mustIdentifier(toSnake(r.name, "pk")),
			mustJoinIdentifiers(r.fields.primaryNames()))
	}

	q.Append(")")

	// partition by range
	partitionByRangeFields := make([]string, 0)
	for _, f := range r.fields {
		if f.partitionByRange != nil {
			partitionByRangeFields = append(partitionByRangeFields, f.name)
		}
	}
	if len(partitionByRangeFields) > 0 {
		q.Appendf("PARTITION BY RANGE (%v)", mustJoinIdentifiers(partitionByRangeFields))
	}

	_, err := p.Exec(context.Background(), q.String())
	return err
}

func (p *Postgres) addColumn(tableName, columnName, dataType string) error {
	queryf := "ALTER TABLE %v ADD COLUMN %v %v"
	query := fmt.Sprintf(queryf, mustIdentifier(tableName), mustIdentifier(columnName), dataType)
	_, err := p.Exec(context.Background(), query)
	return err
}

func (p *Postgres) createIndex(indexName, tableName string, columns []string, unique, concurrently bool) error {
	q := queryf()
	q.Append("CREATE")

	if unique {
		q.Append("UNIQUE")
	}

	q.Append("INDEX")

	if concurrently {
		q.Append("CONCURRENTLY")
	}

	q.Append(mustIdentifier(indexName), "ON", mustIdentifier(tableName))
	q.Appendf("(%v)", mustJoinIdentifiers(columns))

	_, err := p.Exec(context.Background(), q.String())
	return err
}

func (p *Postgres) addForeignKey(tableName, constraintName string, columns []string, referenceTable string, referenceColumns []string) error {
	queryf := "ALTER TABLE %v ADD CONSTRAINT %v FOREIGN KEY (%v) REFERENCES %v (%v) MATCH SIMPLE ON DELETE CASCADE ON UPDATE CASCADE"
	query := fmt.Sprintf(queryf, mustIdentifier(tableName), mustIdentifier(constraintName), mustJoinIdentifiers(columns), mustIdentifier(referenceTable), mustJoinIdentifiers(referenceColumns))
	_, err := p.Exec(context.Background(), query)
	return err
}

func (p *Postgres) addPrimaryKey(tableName, constraintName, indexName string) error {
	queryf := "ALTER TABLE %v ADD CONSTRAINT %v PRIMARY KEY USING INDEX %v"
	query := fmt.Sprintf(queryf,
		mustIdentifier(tableName),
		mustIdentifier(constraintName),
		mustIdentifier(indexName))

	_, err := p.Exec(context.Background(), query)
	return err
}

func (p *Postgres) describeTable(tableName string) (*table, error) {
	// call describeTableIndexes first as it returns an actual error
	// if this table doesn't exist.

	indexes, err := p.describeTableIndexes(tableName)
	if err != nil {
		return nil, err
	}

	columns, err := p.describeTableColumns(tableName)
	if err != nil {
		return nil, err
	}

	return &table{
		Name:    tableName,
		Columns: columns,
		Indexes: indexes,
	}, nil
}

func (p *Postgres) describeTableIndexes(tableName string) ([]index, error) {
	// Copied this from Stackoverflow, lol. Is this really the only way?
	queryf := `
SELECT
  i.relname :: text AS name,
  am.amname :: text AS type,
  ARRAY(
    SELECT pg_get_indexdef(idx.indexrelid, k + 1, TRUE)
    FROM generate_subscripts(idx.indkey, 1) AS k ORDER BY k
  ) AS columns,
  idx.indisunique AS is_unique,
  idx.indisprimary AS is_primary,
  (idx.indexprs IS NOT NULL) OR (idx.indkey::int[] @> array[0]) AS is_functional,
  idx.indpred IS NOT NULL AS is_partial
FROM pg_index AS idx
JOIN pg_class AS i ON i.oid = idx.indexrelid
JOIN pg_am AS am ON i.relam = am.oid
JOIN pg_namespace AS ns ON i.relnamespace = ns.OID
WHERE idx.indrelid = %v :: REGCLASS`

	query := fmt.Sprintf(queryf, QuoteLiteral(tableName))
	rows, err := p.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	is := make([]index, 0)
	for rows.Next() {
		i := &index{}
		// note that order of returned columns needs to match order of index struct
		if err := mustNewFields(i, false).Scan(rows); err != nil {
			return nil, err
		}

		// unquote identifiers
		i.Name = unquoteIdentifier(i.Name)
		i.Columns = unquoteIdentifiers(i.Columns)

		is = append(is, *i)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return is, nil
}

func (p *Postgres) describeTableColumns(tableName string) ([]column, error) {
	queryf := "SELECT column_name, is_nullable, data_type FROM information_schema.columns WHERE table_name = %v"
	query := fmt.Sprintf(queryf, QuoteLiteral(tableName))
	rows, err := p.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cs := make([]column, 0)
	for rows.Next() {
		c := &column{}
		// note that order of returned columns needs to match order of colum struct
		if err := mustNewFields(c, false).Scan(rows); err != nil {
			return nil, err
		}

		// unquote identifier
		c.Name = unquoteIdentifier(c.Name)

		cs = append(cs, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cs, nil
}

func (p *Postgres) constraintExists(constraintName string) (bool, error) {
	queryf := "SELECT 1 FROM information_schema.constraint_column_usage WHERE constraint_name = %v"
	query := fmt.Sprintf(queryf, QuoteLiteral(constraintName))
	row := p.QueryRow(context.Background(), query)

	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return exists > 0, nil
}

var (
	ErrNoLock      = fmt.Errorf("no lock")
	ErrNotUnlocked = fmt.Errorf("not unlocked")
)

func (p *Postgres) waitForAdvisoryLock(key int) {
	d := &backoff.Backoff{
		Min:    3 * time.Second,
		Max:    15 * time.Second,
		Factor: 1.5,
		Jitter: true,
	}

	for {
		if err := p.advisoryLock(key); err == nil {
			return
		}

		if d.Attempt() > float64(maxAdvisoryLockAttemtps) {
			panic("unable to obtain advisory lock")
		}

		time.Sleep(d.Duration())
	}
}

func (p *Postgres) advisoryLock(key int) error {
	queryf := "SELECT pg_try_advisory_lock(%v)"
	query := fmt.Sprintf(queryf, key)
	row := p.QueryRow(context.Background(), query)

	var locked postgresBool
	if err := row.Scan(&locked); err != nil {
		return err
	}

	if locked {
		return nil
	}

	return ErrNoLock
}

// advisoryUnlock needs to run on same connection that acquired a lock before
func (p *Postgres) advisoryUnlock(key int) error {
	queryf := "SELECT pg_advisory_unlock(%v)"
	query := fmt.Sprintf(queryf, key)
	row := p.QueryRow(context.Background(), query)

	var unlocked postgresBool
	if err := row.Scan(&unlocked); err != nil {
		return err
	}

	if unlocked {
		return nil
	}

	return ErrNotUnlocked
}

func (p *Postgres) advisoryUnlockAll() error {
	query := "SELECT pg_advisory_unlock_all()"
	_, err := p.Exec(context.Background(), query)
	return err
}

func (p *Postgres) version() (int, error) {
	query := "SHOW server_version_num"
	row := p.QueryRow(context.Background(), query)

	var version int
	if err := row.Scan(&version); err != nil {
		return 0, err
	}

	return version, nil
}

func (p *Postgres) isMinVersion(version int) (bool, error) {
	x, err := p.version()
	if err != nil {
		return false, err
	}

	return x >= version*1000, nil
}

// scan calls row.Scan and returns a slice of pointers to interfaces
func scan(row rowScan, n int) ([]interface{}, error) {
	// construct slice of interfaces
	values := make([]interface{}, n)
	for i := 0; i < n; i++ {
		// re-assign value with pointer to interface, so we can
		// use it below in row.Scan which expects pointers to values
		values[i] = &values[i]
	}

	if err := row.Scan(values...); err != nil {
		return nil, err
	}
	return values, nil
}

type postgresBool bool

func (p *postgresBool) Scan(value interface{}) error {
	str, err := driver.String.ConvertValue(value)
	if err != nil {
		return err
	}

	if strings.EqualFold(str.(string), "YES") {
		*p = true
	} else if strings.EqualFold(str.(string), "TRUE") {
		*p = true
	} else if strings.EqualFold(str.(string), "ON") {
		*p = true
	}

	return nil
}

type extraParams struct {
	createTempTables bool
}

func parseExtraParams(uri *string) (*extraParams, error) {
	u, err := url.Parse(*uri)
	if err != nil {
		return nil, err
	}

	p := &extraParams{}
	cleanQuery := make(url.Values)

	for k, values := range u.Query() {
		switch k {

		case "createTempTables":
			p.createTempTables, err = strconv.ParseBool(firstUrlValue(values))
			if err != nil {
				return nil, err
			}

		default:
			cleanQuery[k] = values
		}
	}

	// remove extra params from query and set clean uri
	u.RawQuery = cleanQuery.Encode()
	*uri = u.String()

	return p, nil
}

func firstUrlValue(urlValues []string) string {
	if len(urlValues) > 0 {
		return urlValues[0]
	}
	return ""
}
