package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestGetStruct_UsesAlias_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestGetStruct_UsesAlias(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	log := &testLogger{}
	db.Logger = log

	getStruct(db, context.Background(), &TestGetStruct_UsesAlias_Struct{})

	// now register alias and do it again
	Register(&TestGetStruct_UsesAlias_Struct{}, "my_get_alias", "")
	getStruct(db, context.Background(), &TestGetStruct_UsesAlias_Struct{})

	log.Equal(t, "test_data/test_get_struct_uses_alias.txt")
}

type TestFilterStruct_UsesAlias_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestFilterStruct_UsesAlias(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	log := &testLogger{}
	db.Logger = log

	s := make([]TestFilterStruct_UsesAlias_Struct, 0)

	filterStruct(db, context.Background(), &s, Query("Col1 = $1", "foo"))

	// now register alias and do it again
	Register(&TestFilterStruct_UsesAlias_Struct{}, "my_filter_alias", "")
	filterStruct(db, context.Background(), &s, Query("Col1 = $1", "foo"))

	log.Equal(t, "test_data/test_filter_struct_uses_alias.txt")
}

type TestSaveStruct_UsesAlias_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestSaveStruct_UsesAlias(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	log := &testLogger{}
	db.Logger = log

	saveStruct(db, context.Background(), &TestSaveStruct_UsesAlias_Struct{})

	// now register alias and do it again
	Register(&TestSaveStruct_UsesAlias_Struct{}, "my_save_alias", "")
	saveStruct(db, context.Background(), &TestSaveStruct_UsesAlias_Struct{})

	log.Equal(t, "test_data/test_save_struct_uses_alias.txt")
}

type TestInsertStruct_UsesAlias_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestInsertStruct_UsesAlias(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	log := &testLogger{}
	db.Logger = log

	insertStruct(db, context.Background(), &TestInsertStruct_UsesAlias_Struct{})

	// now register alias and do it again
	Register(&TestInsertStruct_UsesAlias_Struct{}, "my_insert_alias", "")
	insertStruct(db, context.Background(), &TestInsertStruct_UsesAlias_Struct{})

	log.Equal(t, "test_data/test_insert_struct_uses_alias.txt")
}

type TestUpdateStruct_UsesAlias_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestUpdateStruct_UsesAlias(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	log := &testLogger{}
	db.Logger = log

	updateStruct(db, context.Background(), &TestUpdateStruct_UsesAlias_Struct{})

	// now register alias and do it again
	Register(&TestUpdateStruct_UsesAlias_Struct{}, "my_update_alias", "")
	updateStruct(db, context.Background(), &TestUpdateStruct_UsesAlias_Struct{})

	log.Equal(t, "test_data/test_update_struct_uses_alias.txt")
}

type TestDeleteStruct_UsesAlias_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestDeleteStruct_UsesAlias(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	log := &testLogger{}
	db.Logger = log

	deleteStruct(db, context.Background(), &TestDeleteStruct_UsesAlias_Struct{})

	// now register alias and do it again
	Register(&TestDeleteStruct_UsesAlias_Struct{}, "my_delete_alias", "")
	deleteStruct(db, context.Background(), &TestDeleteStruct_UsesAlias_Struct{})

	log.Equal(t, "test_data/test_delete_struct_uses_alias.txt")
}
