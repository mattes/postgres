package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	_, err := Open(postgresURI)
	require.NoError(t, err)
}

type TestGetTable_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestGet(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestGetTable_Struct{})))

	// create some records
	_, err = db.Exec(context.Background(), "INSERT INTO test_get_table_struct (col1, col2) VALUES ('1', 'bar'), ('2', 'bar'), ('3', 'abc')")
	require.NoError(t, err)

	// get a record
	s := &TestGetTable_Struct{
		Col1: "2",
	}
	require.NoError(t, db.Get(context.Background(), s))

	expect := &TestGetTable_Struct{
		Col1: "2",
		Col2: "bar",
	}
	require.Equal(t, expect, s)

	log.Equal(t, "test_data/test_get.txt")
}

type BenchmarkGet_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func BenchmarkGet(b *testing.B) {
	db, err := Open(postgresURI)
	require.NoError(b, err)

	// create table
	require.NoError(b, db.ensureTable(mustNewMetaStruct(&BenchmarkGet_Struct{})))

	// create a new record
	s := &BenchmarkGet_Struct{
		Col1: "foo",
		Col2: "bar",
	}
	require.NoError(b, db.Insert(context.Background(), s))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		require.NoError(b, db.Get(context.Background(), s))
	}
}

func BenchmarkGet_Plain(b *testing.B) {
	db, err := Open(postgresURI)
	require.NoError(b, err)

	// create table
	require.NoError(b, db.ensureTable(mustNewMetaStruct(&BenchmarkGet_Struct{})))

	// create a new record
	s := &BenchmarkGet_Struct{
		Col1: "foo",
		Col2: "bar",
	}
	require.NoError(b, db.Insert(context.Background(), s))

	b.ResetTimer()

	var col1, col2 string
	for n := 0; n < b.N; n++ {
		row := db.DB().QueryRow("SELECT col1, col2 FROM benchmark_get_struct WHERE col1 = $1 LIMIT 1", "foo")
		require.NoError(b, row.Scan(&col1, &col2))
	}
}

type TestGetTable_CompositePrimaryKey_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"pk"`
	Col3 string
}

func TestGet_CompositePrimaryKey(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestGetTable_CompositePrimaryKey_Struct{})))

	// create some records
	_, err = db.Exec(context.Background(), "INSERT INTO test_get_table_composite_primary_key_struct (col1, col2, col3) VALUES ('1', '2', 'bar'), ('3', '4', 'bar'), ('5', '6', 'abc')")
	require.NoError(t, err)

	// get a record
	s := &TestGetTable_CompositePrimaryKey_Struct{
		Col1: "3",
		Col2: "4",
	}
	require.NoError(t, db.Get(context.Background(), s))

	expect := &TestGetTable_CompositePrimaryKey_Struct{
		Col1: "3",
		Col2: "4",
		Col3: "bar",
	}
	require.Equal(t, expect, s)

	log.Equal(t, "test_data/test_get_composite_primary_key.txt")
}

type TestInsert_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestInsert(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestInsert_Struct{})))

	// create a new record
	s := &TestInsert_Struct{
		Col1: "1",
		Col2: "foo",
		Col3: "bar",
	}
	require.NoError(t, db.Insert(context.Background(), s))

	expect := &TestInsert_Struct{
		Col1: "1",
		Col2: "foo",
		Col3: "bar",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestInsert_Struct{
		Col1: "1",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	// try to insert again
	requirePQError(t, db.Insert(context.Background(), s), "unique_violation")

	log.Equal(t, "test_data/test_insert.txt")
}

type BenchmarkInsert_Struct struct {
	Col1 string
	Col2 string
}

func BenchmarkInsert(b *testing.B) {
	db, err := Open(postgresURI)
	require.NoError(b, err)

	// create table
	require.NoError(b, db.ensureTable(mustNewMetaStruct(&BenchmarkInsert_Struct{})))

	// create a new record
	s := &BenchmarkInsert_Struct{
		Col1: "foo",
		Col2: "bar",
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		require.NoError(b, db.Insert(context.Background(), s))
	}
}

func BenchmarkInsert_Plain(b *testing.B) {
	db, err := Open(postgresURI)
	require.NoError(b, err)

	// create table
	require.NoError(b, db.ensureTable(mustNewMetaStruct(&BenchmarkInsert_Struct{})))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err = db.DB().ExecContext(context.Background(), "INSERT INTO benchmark_insert_struct (col1, col2) VALUES ($1, $2)", "foo", "bar")
		require.NoError(b, err)
	}
}

type TestInsert_CompositePrimaryKey_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"pk"`
	Col3 string
}

func TestInsert_CompositePrimaryKey(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestInsert_CompositePrimaryKey_Struct{})))

	// create a new record
	s := &TestInsert_CompositePrimaryKey_Struct{
		Col1: "1",
		Col2: "2",
		Col3: "foo",
	}
	require.NoError(t, db.Insert(context.Background(), s))

	expect := &TestInsert_CompositePrimaryKey_Struct{
		Col1: "1",
		Col2: "2",
		Col3: "foo",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestInsert_CompositePrimaryKey_Struct{
		Col1: "1",
		Col2: "2",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	// try to insert again
	requirePQError(t, db.Insert(context.Background(), s), "unique_violation")

	log.Equal(t, "test_data/test_insert_composite_primary_key.txt")
}

type TestInsert_WithFieldMask_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestInsert_WithFieldMask(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestInsert_WithFieldMask_Struct{})))

	// create a new record
	s := &TestInsert_WithFieldMask_Struct{
		Col1: "1",
		Col2: "<not saved>",
		Col3: "bar",
	}
	require.NoError(t, db.Insert(context.Background(), s, "Col1", "Col3"))

	expect := &TestInsert_WithFieldMask_Struct{
		Col1: "1",
		Col2: "", // wasn't saved
		Col3: "bar",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestInsert_WithFieldMask_Struct{
		Col1: "1",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	// try to insert again
	requirePQError(t, db.Insert(context.Background(), s), "unique_violation")

	log.Equal(t, "test_data/test_insert_with_fieldmask.txt")
}

type TestUpdate_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestUpdate(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestUpdate_Struct{})))

	// create initial record
	_, err = db.Exec(context.Background(), "INSERT INTO test_update_struct (col1, col2, col3) VALUES ('1', 'a', 'b'), ('2', 'c', 'd'), ('3', 'e', 'f')")
	require.NoError(t, err)

	// update record
	s := &TestUpdate_Struct{
		Col1: "2",
		Col2: "x",
		Col3: "y",
	}
	require.NoError(t, db.Update(context.Background(), s))

	expect := &TestUpdate_Struct{
		Col1: "2",
		Col2: "x",
		Col3: "y",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestUpdate_Struct{
		Col1: "2",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_update.txt")
}

type TestUpdate_CompositePrimaryKey_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"pk"`
	Col3 string
}

func TestUpdate_CompositePrimaryKey(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestUpdate_CompositePrimaryKey_Struct{})))

	// create initial record
	_, err = db.Exec(context.Background(), "INSERT INTO test_update_composite_primary_key_struct (col1, col2, col3) VALUES ('1', 'a', 'b'), ('2', 'c', 'd'), ('3', 'e', 'f')")
	require.NoError(t, err)

	// update record
	s := &TestUpdate_CompositePrimaryKey_Struct{
		Col1: "2",
		Col2: "c",
		Col3: "x",
	}
	require.NoError(t, db.Update(context.Background(), s))

	expect := &TestUpdate_CompositePrimaryKey_Struct{
		Col1: "2",
		Col2: "c",
		Col3: "x",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestUpdate_CompositePrimaryKey_Struct{
		Col1: "2",
		Col2: "c",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_update_composite_key.txt")
}

type TestUpdate_WithFieldMask_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestUpdate_WithFieldMask(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestUpdate_WithFieldMask_Struct{})))

	// create initial record
	_, err = db.Exec(context.Background(), "INSERT INTO test_update_with_field_mask_struct (col1, col2, col3) VALUES ('1', 'a', 'b'), ('2', 'c', 'd'), ('3', 'e', 'f')")
	require.NoError(t, err)

	// update record
	s := &TestUpdate_WithFieldMask_Struct{
		Col1: "2",
		Col2: "<not saved>",
		Col3: "y",
	}
	require.NoError(t, db.Update(context.Background(), s, "Col3"))

	expect := &TestUpdate_WithFieldMask_Struct{
		Col1: "2",
		Col2: "c", // wasn't saved
		Col3: "y",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestUpdate_WithFieldMask_Struct{
		Col1: "2",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_update_with_field_mask.txt")
}

type TestSave_Insert_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestSave_Insert(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestSave_Insert_Struct{})))

	// create a new record
	s := &TestSave_Insert_Struct{
		Col1: "1",
		Col2: "foo",
		Col3: "bar",
	}
	require.NoError(t, db.Save(context.Background(), s))

	expect := &TestSave_Insert_Struct{
		Col1: "1",
		Col2: "foo",
		Col3: "bar",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestSave_Insert_Struct{
		Col1: "1",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_save_insert.txt")
}

type TestSave_Insert_CompositePrimaryKey_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"pk"`
	Col3 string
}

func TestSave_Insert_CompositePrimaryKey(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestSave_Insert_CompositePrimaryKey_Struct{})))

	// create a new record
	s := &TestSave_Insert_CompositePrimaryKey_Struct{
		Col1: "1",
		Col2: "2",
		Col3: "foo",
	}
	require.NoError(t, db.Save(context.Background(), s))

	expect := &TestSave_Insert_CompositePrimaryKey_Struct{
		Col1: "1",
		Col2: "2",
		Col3: "foo",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestSave_Insert_CompositePrimaryKey_Struct{
		Col1: "1",
		Col2: "2",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_save_insert_composite_primary_key.txt")
}

type TestSave_Update_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestSave_Update(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestSave_Update_Struct{})))

	// create initial record
	_, err = db.Exec(context.Background(), "INSERT INTO test_save_update_struct (col1, col2, col3) VALUES ('1', 'a', 'b'), ('2', 'c', 'd'), ('3', 'e', 'f')")
	require.NoError(t, err)

	// update record
	s := &TestSave_Update_Struct{
		Col1: "2",
		Col2: "x",
		Col3: "y",
	}
	require.NoError(t, db.Save(context.Background(), s))

	expect := &TestSave_Update_Struct{
		Col1: "2",
		Col2: "x",
		Col3: "y",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestSave_Update_Struct{
		Col1: "2",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_save_update.txt")
}

type TestSave_Update_CompositePrimaryKey_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"pk"`
	Col3 string
}

func TestSave_Update_CompositePrimaryKey(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestSave_Update_CompositePrimaryKey_Struct{})))

	// create initial record
	_, err = db.Exec(context.Background(), "INSERT INTO test_save_update_composite_primary_key_struct (col1, col2, col3) VALUES ('1', 'a', 'b'), ('2', 'c', 'd'), ('3', 'e', 'f')")
	require.NoError(t, err)

	// update record
	s := &TestSave_Update_CompositePrimaryKey_Struct{
		Col1: "2",
		Col2: "c",
		Col3: "x",
	}
	require.NoError(t, db.Save(context.Background(), s))

	expect := &TestSave_Update_CompositePrimaryKey_Struct{
		Col1: "2",
		Col2: "c",
		Col3: "x",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestSave_Update_CompositePrimaryKey_Struct{
		Col1: "2",
		Col2: "c",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_save_update_composite_key.txt")
}

type TestSave_Insert_WithFieldMask_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestSave_Insert_WithFieldMask(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestSave_Insert_WithFieldMask_Struct{})))

	// create a new record
	s := &TestSave_Insert_WithFieldMask_Struct{
		Col1: "1",
		Col2: "<not saved>",
		Col3: "bar",
	}
	require.NoError(t, db.Save(context.Background(), s, "Col1", "Col3"))

	expect := &TestSave_Insert_WithFieldMask_Struct{
		Col1: "1",
		Col2: "", // wasn't saved
		Col3: "bar",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestSave_Insert_WithFieldMask_Struct{
		Col1: "1",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_save_insert_with_fieldmask.txt")
}

type TestSave_Update_WithFieldMask_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestSave_Update_WithFieldMask(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestSave_Update_WithFieldMask_Struct{})))

	// create initial record
	_, err = db.Exec(context.Background(), "INSERT INTO test_save_update_with_field_mask_struct (col1, col2, col3) VALUES ('1', 'a', 'b'), ('2', 'c', 'd'), ('3', 'e', 'f')")
	require.NoError(t, err)

	// update record
	s := &TestSave_Update_WithFieldMask_Struct{
		Col1: "2",
		Col2: "<not saved>",
		Col3: "y",
	}
	require.NoError(t, db.Save(context.Background(), s, "Col1", "Col3"))

	expect := &TestSave_Update_WithFieldMask_Struct{
		Col1: "2",
		Col2: "c", // wasn't saved
		Col3: "y",
	}
	require.Equal(t, expect, s)

	// read record from database
	s2 := &TestSave_Update_WithFieldMask_Struct{
		Col1: "2",
	}
	require.NoError(t, db.Get(context.Background(), s2))
	require.Equal(t, expect, s2)

	log.Equal(t, "test_data/test_save_update_with_field_mask.txt")
}

type TestDelete_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestDelete(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestDelete_Struct{})))

	// create a record
	_, err = db.Exec(context.Background(), "INSERT INTO test_delete_struct (col1, col2) VALUES ('1', 'a'), ('2', 'b'), ('3', 'c')")
	require.NoError(t, err)

	// delete record
	s := TestDelete_Struct{
		Col1: "2",
	}
	require.NoError(t, db.Delete(context.Background(), &s))

	log.Equal(t, "test_data/test_delete.txt")

	// did we delete the right record?
	x1 := &TestDelete_Struct{Col1: "1"}
	require.NoError(t, db.Get(context.Background(), x1))

	x2 := &TestDelete_Struct{Col1: "2"}
	require.Error(t, db.Get(context.Background(), x2))

	x3 := &TestDelete_Struct{Col1: "3"}
	require.NoError(t, db.Get(context.Background(), x3))
}

type TestDelete_CompositePrimaryKey_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"pk"`
	Col3 string
}

func TestDelete_CompositePrimaryKey(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestDelete_CompositePrimaryKey_Struct{})))

	// create a record
	_, err = db.Exec(context.Background(), "INSERT INTO test_delete_composite_primary_key_struct (col1, col2, col3) VALUES ('1', 'a', 'b'), ('2', 'c', 'd'), ('3', 'e', 'f')")
	require.NoError(t, err)

	// delete record
	s := &TestDelete_CompositePrimaryKey_Struct{
		Col1: "2",
		Col2: "c",
	}
	require.NoError(t, db.Delete(context.Background(), s))

	log.Equal(t, "test_data/test_delete_composite_primary_key.txt")

	// did we delete the right record?
	x1 := &TestDelete_CompositePrimaryKey_Struct{Col1: "1", Col2: "a"}
	require.NoError(t, db.Get(context.Background(), x1))

	x2 := &TestDelete_CompositePrimaryKey_Struct{Col1: "2", Col2: "c"}
	require.Error(t, db.Get(context.Background(), x2))

	x3 := &TestDelete_CompositePrimaryKey_Struct{Col1: "3", Col2: "e"}
	require.NoError(t, db.Get(context.Background(), x3))
}

type TestFilter_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
	Col3 string
}

func TestFilter(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{debug: false}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestFilter_Struct{})))

	// create a record
	_, err = db.Exec(context.Background(), "INSERT INTO test_filter_struct (col1, col2, col3) VALUES ('1', 'a', 'x'), ('2', 'c', 'y'), ('3', 'e', 'x')")
	require.NoError(t, err)

	// create query and filter ...
	s := []TestFilter_Struct{}
	require.NoError(t,
		db.Filter(context.Background(), &s,
			Query("Col3 = $1", "x").Limit(10).Asc("Col2")))
	require.Len(t, s, 2)
	require.Equal(t, []TestFilter_Struct{{"1", "a", "x"}, {"3", "e", "x"}}, s)

	log.Equal(t, "test_data/test_filter.txt")
}

type BenchmarkFilter_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"index"`
}

func BenchmarkFilter(b *testing.B) {
	db, err := Open(postgresURI)
	require.NoError(b, err)

	// create table
	require.NoError(b, db.ensureTable(mustNewMetaStruct(&BenchmarkFilter_Struct{})))

	// create a bunch of new records
	for i := 0; i < 25; i++ {
		s := &BenchmarkFilter_Struct{
			Col1: NewPrefixID(""),
			Col2: "foobar",
		}
		require.NoError(b, db.Insert(context.Background(), s))
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		s := []BenchmarkFilter_Struct{}
		require.NoError(b,
			db.Filter(context.Background(), &s,
				Query("Col2 = $1", "foobar").Asc("Col2").Limit(10)))
	}
}

func BenchmarkFilter_Plain(b *testing.B) {
	db, err := Open(postgresURI)
	require.NoError(b, err)

	// create table
	require.NoError(b, db.ensureTable(mustNewMetaStruct(&BenchmarkFilter_Struct{})))

	// create a bunch of new records
	for i := 0; i < 25; i++ {
		s := &BenchmarkFilter_Struct{
			Col1: NewPrefixID(""),
			Col2: "foobar",
		}
		require.NoError(b, db.Insert(context.Background(), s))
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		rows, err := db.DB().Query("SELECT col1, col2 FROM benchmark_filter_struct WHERE col2 = $1 ORDER BY col2 ASC LIMIT 10", "foobar")
		require.NoError(b, err)

		var col1, col2 string
		for rows.Next() {
			require.NoError(b, rows.Scan(&col1, &col2))
		}

		require.NoError(b, rows.Err())
		rows.Close()
	}
}

type TestEnsureTable_PrimaryKeys_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestEnsureTable_PrimaryKeys(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_PrimaryKeys_Struct{})))

	// make sure table was created correctly
	tbl, err := db.describeTable("test_ensure_table_primary_keys_struct")
	require.NoError(t, err)

	expectTable := &table{
		Name: "test_ensure_table_primary_keys_struct",
		Columns: []column{
			{Name: "col1", IsNullable: false, DataType: "text"},
			{Name: "col2", IsNullable: false, DataType: "text"},
		},
		Indexes: []index{
			{Name: "test_ensure_table_primary_keys_struct_pk", Type: "btree", Columns: []string{"col1"}, IsUnique: true, IsPrimary: true, IsFunctional: false, IsPartial: false},
		},
	}
	require.Equal(t, expectTable, tbl)

	// table already exists
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_PrimaryKeys_Struct{})))

	log.Equal(t, "test_data/test_ensure_table_primary_keys.txt")
}

type TestEnsureTable_CompositePrimaryKeys_Struct struct {
	Col1 string `db:"pk"`
	Col2 string `db:"pk"`
	Col3 string
}

func TestEnsureTable_CompositePrimaryKeys(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_CompositePrimaryKeys_Struct{})))

	// make sure table was created correctly
	tbl, err := db.describeTable("test_ensure_table_composite_primary_keys_struct")
	require.NoError(t, err)

	expectTable := &table{
		Name: "test_ensure_table_composite_primary_keys_struct",
		Columns: []column{
			{Name: "col1", IsNullable: false, DataType: "text"},
			{Name: "col2", IsNullable: false, DataType: "text"},
			{Name: "col3", IsNullable: false, DataType: "text"},
		},
		Indexes: []index{
			{Name: "test_ensure_table_composite_primary_keys_struct_pk", Type: "btree", Columns: []string{"col1", "col2"}, IsUnique: true, IsPrimary: true, IsFunctional: false, IsPartial: false},
		},
	}
	require.Equal(t, expectTable, tbl)

	// table already exists
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_CompositePrimaryKeys_Struct{})))

	log.Equal(t, "test_data/test_ensure_table_composite_primary_keys.txt")
}

type TestEnsureTable_UniqueIndex_Struct struct {
	Col1 string `db:"unique"`
	Col2 string
}

func TestEnsureTable_UniqueIndex(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_UniqueIndex_Struct{})))

	// make sure table was created correctly
	tbl, err := db.describeTable("test_ensure_table_unique_index_struct")
	require.NoError(t, err)

	expectTable := &table{
		Name: "test_ensure_table_unique_index_struct",
		Columns: []column{
			{Name: "col1", IsNullable: false, DataType: "text"},
			{Name: "col2", IsNullable: false, DataType: "text"},
		},
		Indexes: []index{
			{Name: "test_ensure_table_unique_index_struct_col1_unique", Type: "btree", Columns: []string{"col1"}, IsUnique: true, IsPrimary: false, IsFunctional: false, IsPartial: false},
		},
	}
	require.Equal(t, expectTable, tbl)

	// table already exists
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_UniqueIndex_Struct{})))

	log.Equal(t, "test_data/test_ensure_table_unique_index.txt")
}

type TestEnsureTable_CompositeUniqueIndex_Struct struct {
	Col1 string `db:"unique(foobar)"`
	Col2 string `db:"unique(foobar)"`
	Col3 string
}

func TestEnsureTable_CompositeUniqueIndex(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_CompositeUniqueIndex_Struct{})))

	// make sure table was created correctly
	tbl, err := db.describeTable("test_ensure_table_composite_unique_index_struct")
	require.NoError(t, err)

	expectTable := &table{
		Name: "test_ensure_table_composite_unique_index_struct",
		Columns: []column{
			{Name: "col1", IsNullable: false, DataType: "text"},
			{Name: "col2", IsNullable: false, DataType: "text"},
			{Name: "col3", IsNullable: false, DataType: "text"},
		},
		Indexes: []index{
			{Name: "test_ensure_table_composite_unique_index_struct_foobar",
				Type: "btree", Columns: []string{"col1", "col2"}, IsUnique: true, IsPrimary: false, IsFunctional: false, IsPartial: false},
		},
	}
	require.Equal(t, expectTable, tbl)

	// table already exists
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_CompositeUniqueIndex_Struct{})))

	log.Equal(t, "test_data/test_ensure_table_composite_unique_index.txt")
}

type TestEnsureTable_Index_Struct struct {
	Col1 string `db:"index"`
	Col2 string
}

func TestEnsureTable_Index(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_Index_Struct{})))

	// make sure table was created correctly
	tbl, err := db.describeTable("test_ensure_table_index_struct")
	require.NoError(t, err)

	expectTable := &table{
		Name: "test_ensure_table_index_struct",
		Columns: []column{
			{Name: "col1", IsNullable: false, DataType: "text"},
			{Name: "col2", IsNullable: false, DataType: "text"},
		},
		Indexes: []index{
			{Name: "test_ensure_table_index_struct_col1_index", Type: "btree", Columns: []string{"col1"}},
		},
	}
	require.Equal(t, expectTable, tbl)

	// table already exists
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_Index_Struct{})))

	log.Equal(t, "test_data/test_ensure_table_index.txt")
}

type TestEnsureTable_CompositeIndex_Struct struct {
	Col1 string `db:"index(foobar)"`
	Col2 string `db:"index(foobar)"`
	Col3 string
}

func TestEnsureTable_CompositeIndex(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_CompositeIndex_Struct{})))

	// make sure table was created correctly
	tbl, err := db.describeTable("test_ensure_table_composite_index_struct")
	require.NoError(t, err)

	expectTable := &table{
		Name: "test_ensure_table_composite_index_struct",
		Columns: []column{
			{Name: "col1", IsNullable: false, DataType: "text"},
			{Name: "col2", IsNullable: false, DataType: "text"},
			{Name: "col3", IsNullable: false, DataType: "text"},
		},
		Indexes: []index{
			{Name: "test_ensure_table_composite_index_struct_foobar",
				Type: "btree", Columns: []string{"col1", "col2"}},
		},
	}
	require.Equal(t, expectTable, tbl)

	// table already exists
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_CompositeIndex_Struct{})))

	log.Equal(t, "test_data/test_ensure_table_composite_index.txt")
}

type TestEnsureTable_ForeignKey_StructA struct {
	Col1 string `db:"pk"`
	Col2 string `db:"references(TestEnsureTable_ForeignKey_StructB.Col2)"`
}

type TestEnsureTable_ForeignKey_StructB struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestEnsureTable_ForeignKey(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{debug: false}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_ForeignKey_StructA{})))
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestEnsureTable_ForeignKey_StructB{})))

	require.NoError(t, db.ensureForeignKeys(mustNewMetaStruct(&TestEnsureTable_ForeignKey_StructA{})))
	require.NoError(t, db.ensureForeignKeys(mustNewMetaStruct(&TestEnsureTable_ForeignKey_StructB{})))

	// foreign key already exists
	require.NoError(t, db.ensureForeignKeys(mustNewMetaStruct(&TestEnsureTable_ForeignKey_StructA{})))
	require.NoError(t, db.ensureForeignKeys(mustNewMetaStruct(&TestEnsureTable_ForeignKey_StructB{})))

	log.Equal(t, "test_data/test_ensure_table_foreign_key.txt")
}

type TestRegisterAndMigrate_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestRegisterAndMigrate(t *testing.T) {
	Register("test_register_and_migrate", &TestRegisterAndMigrate_Struct{})

	db, err := Open(postgresURI)
	require.NoError(t, err)

	require.NoError(t, db.advisoryUnlockAll()) // just to make sure
	require.NoError(t, db.Migrate(context.Background()))

	// run again, no errors expected
	require.NoError(t, db.Migrate(context.Background()))
}

func TestAdvisoryLocks(t *testing.T) {
	db1, err := Open(postgresURI)
	require.NoError(t, err)

	db2, err := Open(postgresURI)
	require.NoError(t, err)

	db1.advisoryUnlockAll() // just to make sure
	db2.advisoryUnlockAll() // just to make sure

	key := MigrateKey - 2 // random key

	// db1 alone is doing some locking and unlocking
	require.NoError(t, db1.advisoryLock(key))
	require.NoError(t, db1.advisoryLock(key))
	require.NoError(t, db1.advisoryUnlock(key))
	require.NoError(t, db1.advisoryUnlock(key))
	require.Equal(t, ErrNotUnlocked, db1.advisoryUnlock(key))

	// db1 and db2 fighting over lock
	require.NoError(t, db1.advisoryLock(key))
	require.Equal(t, ErrNoLock, db2.advisoryLock(key))
	require.NoError(t, db1.advisoryUnlock(key))
	require.NoError(t, db2.advisoryLock(key))
	require.Equal(t, ErrNoLock, db1.advisoryLock(key))
	require.NoError(t, db2.advisoryUnlock(key))

	// db1 dies while having a lock
	require.NoError(t, db1.advisoryLock(key))
	require.NoError(t, db1.Close())
	time.Sleep(250 * time.Millisecond) // make sure it's closed
	require.NoError(t, db2.advisoryLock(key))
	require.NoError(t, db2.advisoryUnlock(key))
}

type TestDescribeTableIndexes_Struct struct {
	Col1      string `db:"pk"`
	Timestamp string `db:"pk"` // timestamp is also postgres identifier
}

func TestDescribeTableIndexes(t *testing.T) {
	Register("test_describe_table_indexes", &TestDescribeTableIndexes_Struct{})

	db, err := Open(postgresURI)
	require.NoError(t, err)

	require.NoError(t, db.advisoryUnlockAll()) // just to make sure
	require.NoError(t, db.Migrate(context.Background()))

	indexes, err := db.describeTableIndexes("test_describe_table_indexes_struct")
	require.NoError(t, err)

	require.Len(t, indexes, 1)
	require.Equal(t, "test_describe_table_indexes_struct_pk", indexes[0].Name)
	require.Equal(t, "btree", indexes[0].Type)
	require.Equal(t, []string{"col1", "timestamp"}, indexes[0].Columns)
}

type TestDescribeTableColumns_Struct struct {
	Col1      string `db:"pk"`
	Timestamp string `db:"pk"` // timestamp is also postgres identifier
}

func TestDescribeTableColumns(t *testing.T) {
	Register("test_describe_table_columns", &TestDescribeTableColumns_Struct{})

	db, err := Open(postgresURI)
	require.NoError(t, err)

	require.NoError(t, db.advisoryUnlockAll()) // just to make sure
	require.NoError(t, db.Migrate(context.Background()))

	cols, err := db.describeTableColumns("test_describe_table_columns_struct")
	require.NoError(t, err)

	require.Len(t, cols, 2)
	require.Equal(t, "col1", cols[0].Name)
	require.Equal(t, "timestamp", cols[1].Name)
}
