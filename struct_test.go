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
		{name: "a", indexes: []indexStructTag{{unique: true}}},
		{name: "b", indexes: []indexStructTag{{unique: true, name: "foo", composite: []string{"c"}}}},
		{name: "c", indexes: []indexStructTag{{unique: true, name: "bar", composite: []string{"b"}}}},
		{name: "d", indexes: []indexStructTag{{unique: true, composite: []string{"b"}}}},
	}

	out := fields.uniqueIndexes()
	expect := map[string][]string{
		"a_unique":   []string{"a"},
		"foo":        []string{"b", "c"},
		"bar":        []string{"c", "b"},
		"d_b_unique": []string{"d", "b"},
	}
	require.Equal(t, expect, out)
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

type TestGlobalStructsName_Struct struct{}

func TestGlobalStructsName(t *testing.T) {
	expect := "test_global_structs_name_struct"
	require.Equal(t, expect, globalStructsName(&TestGlobalStructsName_Struct{}))
	require.Equal(t, expect, globalStructsNameFromString("TestGlobalStructsName_Struct"))
}
