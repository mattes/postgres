package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestRegister_Struct struct{}

func TestRegister(t *testing.T) {
	require.NotPanics(t, func() { RegisterWithPrefix(&TestRegister_Struct{}, "", "") })

	// try again and it panics because we cannot register same struct twice
	require.Panics(t, func() { RegisterWithPrefix(&TestRegister_Struct{}, "", "") })
}

type TestRegister_Alias_Struct1 struct{}
type TestRegister_Alias_Struct2 struct{}

func TestRegister_Alias(t *testing.T) {
	require.NotPanics(t, func() { RegisterWithPrefix(&TestRegister_Alias_Struct1{}, "test_register_same_alias", "") })

	// panics because alias is already registered
	require.Panics(t, func() { RegisterWithPrefix(&TestRegister_Alias_Struct2{}, "test_register_same_alias", "") })
}

func Test_UniqueIndexes(t *testing.T) {
	fields := fields{
		{name: "a", unique: true},
		{name: "b", unique: true, uniqueIndexName: "foo"},
		{name: "c", unique: true, uniqueIndexName: "foo"},
	}

	out := fields.uniqueIndexes()
	expect := map[string][]string{
		"a_unique": []string{"a"},
		"foo":      []string{"b", "c"},
	}
	require.Equal(t, expect, out)
}

type newFieldsStruct struct {
	a string `db:"unique"`
}

func Test_NewFields(t *testing.T) {
	fields := mustNewFields(&newFieldsStruct{}, true)
	require.True(t, fields[0].unique)
	require.Empty(t, fields[0].uniqueIndexName)
}

type TestFieldsExample struct {
	Col1 string
	Col2 string
	Col3 string
}

func TestFieldsNames(t *testing.T) {
	s := &TestFieldsExample{"a", "b", "c"}
	fs := mustNewFields(s, false)

	require.Equal(t, []string{"Col1", "Col2", "Col3"}, fs.names())
	require.Equal(t, []string{"Col2"}, fs.names("Col2"))
}

type FieldNameType string

const FieldNameConst FieldNameType = "Col2"

func TestFieldMaskMatch(t *testing.T) {
	require.False(t, fieldMaskMatch([]StructFieldName{"Col1", "Col2"}, "Col3"))
	require.True(t, fieldMaskMatch([]StructFieldName{"Col1", "Col2"}, "Col1"))
	require.True(t, fieldMaskMatch([]StructFieldName{"Col1", FieldNameConst}, "Col1"))
	require.True(t, fieldMaskMatch([]StructFieldName{}, "Col1"))
	require.True(t, fieldMaskMatch([]StructFieldName{nil}, "Col1"))
	require.True(t, fieldMaskMatch(nil, "Col1"))
}

func TestParseTag(t *testing.T) {
	// basic test
	{
		expect := []tag{{key: "pk"}, {key: "unique", values: []string{"foo", "bar"}}}
		tags, err := parseTags("pk,unique(foo,bar)")
		require.NoError(t, err)
		require.Equal(t, expect, tags)
	}

	// empty values
	{
		expect := []tag{{key: "pk"}, {key: "unique"}}
		tags, err := parseTags("pk,unique()")
		require.NoError(t, err)
		require.Equal(t, expect, tags)
	}

	// add some whitespace
	{
		expect := []tag{{key: "pk"}, {key: "unique", values: []string{"f o o", "b a r"}}}
		tags, err := parseTags(" pk , unique ( f o o , b a r )")
		require.NoError(t, err)
		require.Equal(t, expect, tags)
	}

	// add some whitespace inside single quotes
	{
		expect := []tag{{key: " pk "}, {key: " unique ", values: []string{" foo ", " bar "}}}
		tags, err := parseTags(" ' pk ' , ' unique ' (' foo ', ' bar ' )  ")
		require.NoError(t, err)
		require.Equal(t, expect, tags)
	}

	// don't split by comma inside value when inside single quotes
	{
		expect := []tag{{key: "p,k"}, {key: "uni,que", values: []string{"foo", "ba,r"}}}
		tags, err := parseTags(`'p,k','uni,que'(foo,'ba,r')`)
		require.NoError(t, err)
		require.Equal(t, expect, tags)
	}

	// bogus
	_, err := parseTags(`pk)`)
	require.Error(t, err)

	_, err = parseTags(`pk,)`)
	require.Error(t, err)

	_, err = parseTags(`pk(foo(`)
	require.Error(t, err)

	_, err = parseTags(`pk(())`)
	require.Error(t, err)
}

type TestGlobalStructsName_Struct struct{}

func TestGlobalStructsName(t *testing.T) {
	expect := "test_global_structs_name_struct"
	require.Equal(t, expect, globalStructsName(&TestGlobalStructsName_Struct{}))
	require.Equal(t, expect, globalStructsNameFromString("TestGlobalStructsName_Struct"))
}
