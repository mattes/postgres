package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

type db interface {
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func getStruct(db db, ctx context.Context, s Struct) error {
	if !isPointer(s) {
		panic(fmt.Sprintf("expect *%T not %T", s, s))
	}

	r, err := newMetaStruct(s) // don't use registered metaStruct here
	if err != nil {
		return err
	}

	p := newPlaceholderMap()

	queryf := "SELECT %v FROM %v WHERE %v LIMIT 1"
	query := fmt.Sprintf(queryf,
		mustJoinIdentifiers(r.fields.names()),
		mustIdentifier(r.alias()),
		r.fields.wherePrimaryStr(p))

	row := db.QueryRow(ctx, query, p.args(r.fields)...)
	if err := r.fields.Scan(row); err != nil {
		return err
	}

	return nil
}

func filterStruct(db db, ctx context.Context, s StructSlice, q *QueryStmt) error {
	if !isPointer(s) {
		panic(fmt.Sprintf("expect *%T not %T", s, s))
	}

	sval := reflect.ValueOf(s)

	// check if s is a slice
	if sval.Elem().Kind() != reflect.Slice {
		panic(fmt.Sprintf("expect []%T not %T", s, s))
	}

	// create empty struct based on s slice
	typ := sval.Elem().Type().Elem()
	sx := reflect.New(typ).Interface()

	// get meta struct from slice'd type
	r, err := newMetaStruct(sx) // don't use registered metaStruct here
	if err != nil {
		return err
	}

	// verify the query, which is important if query contains untrusted user input
	if err := q.validate(r); err != nil {
		return err
	}

	qx := queryf()
	qx.Append("SELECT", mustJoinIdentifiers(r.fields.names()))
	qx.Append("FROM", mustIdentifier(r.alias()))
	qx.Append(q.queryStr()) // WHERE
	qx.Append(q.orderStr()) // ORDER BY
	qx.Append("LIMIT", q.limit)

	rows, err := db.Query(ctx, qx.String(), q.args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		sx := reflect.New(typ).Interface()
		if err := mustNewFields(sx, false).Scan(rows); err != nil {
			return err
		}

		// append to slice s
		sval.Elem().Set(reflect.Append(sval.Elem(), reflect.ValueOf(sx).Elem()))
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func saveStruct(db db, ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	if !isPointer(s) {
		panic(fmt.Sprintf("expect *%T not %T", s, s))
	}

	r, err := newMetaStruct(s) // don't use registered metaStruct here
	if err != nil {
		return err
	}

	p := newPlaceholderMap()

	queryf := "INSERT INTO %v (%v) VALUES (%v) ON CONFLICT (%v) DO UPDATE SET (%v) = ROW(%v) RETURNING %v"
	query := fmt.Sprintf(queryf,
		mustIdentifier(r.alias()),
		mustJoinIdentifiers(r.fields.names(fieldMask...)),
		join(p.assign(r.fields.fieldMask(fieldMask)...)),
		mustJoinIdentifiers(r.fields.primaryNames()),
		mustJoinIdentifiers(r.fields.nonPrimaryNames(fieldMask...)),
		mustJoinIdentifiersWithPrefix(r.fields.nonPrimaryNames(fieldMask...), "EXCLUDED"),
		mustJoinIdentifiers(r.fields.names()),
	)

	row := db.QueryRow(ctx, query, p.args(r.fields)...)
	if err := r.fields.Scan(row); err != nil {
		return err
	}

	return nil
}

func insertStruct(db db, ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	if !isPointer(s) {
		panic(fmt.Sprintf("expect *%T not %T", s, s))
	}

	r, err := newMetaStruct(s) // don't use registered metaStruct here
	if err != nil {
		return err
	}

	p := newPlaceholderMap()

	queryf := "INSERT INTO %v (%v) VALUES (%v) RETURNING %v"
	query := fmt.Sprintf(queryf,
		mustIdentifier(r.alias()),
		mustJoinIdentifiers(r.fields.names(fieldMask...)),
		join(p.assign(r.fields.fieldMask(fieldMask)...)),
		mustJoinIdentifiers(r.fields.names()),
	)

	row := db.QueryRow(ctx, query, p.args(r.fields)...)
	if err := r.fields.Scan(row); err != nil {
		return err
	}

	return nil
}

func updateStruct(db db, ctx context.Context, s Struct, fieldMask ...StructFieldName) error {
	if !isPointer(s) {
		panic(fmt.Sprintf("expect *%T not %T", s, s))
	}

	r, err := newMetaStruct(s) // don't use registered metaStruct here
	if err != nil {
		return err
	}

	p := newPlaceholderMap()

	queryf := "UPDATE %v SET (%v) = ROW(%v) WHERE %v RETURNING %v"
	query := fmt.Sprintf(queryf,
		mustIdentifier(r.alias()),
		mustJoinIdentifiers(r.fields.nonPrimaryNames(fieldMask...)),
		join(p.assign(r.fields.nonPrimaryFields(fieldMask...)...)),
		r.fields.wherePrimaryStr(p),
		mustJoinIdentifiers(r.fields.names()),
	)

	row := db.QueryRow(ctx, query, p.args(r.fields)...)
	if err := r.fields.Scan(row); err != nil {
		return err
	}

	return nil
}

func deleteStruct(db db, ctx context.Context, s Struct) error {
	if !isPointer(s) {
		panic(fmt.Sprintf("expect *%T not %T", s, s))
	}

	r, err := newMetaStruct(s) // don't use registered metaStruct here
	if err != nil {
		return err
	}

	p := newPlaceholderMap()

	queryf := "DELETE FROM %v WHERE %v RETURNING %v"
	query := fmt.Sprintf(queryf,
		mustIdentifier(r.alias()),
		r.fields.wherePrimaryStr(p),
		mustJoinIdentifiers(r.fields.names()))

	row := db.QueryRow(ctx, query, p.args(r.fields)...)
	if err := r.fields.Scan(row); err != nil {
		return err
	}

	return nil
}
