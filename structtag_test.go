package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStructTag(t *testing.T) {
	tt := []struct {
		in     string
		expect *structTag
	}{
		{
			// empty string returns "empty" structTag
			"",
			&structTag{},
		},

		{
			"foo",
			&structTag{
				Functions: []stFunction{{Name: "foo"}}},
		},
		{
			"foo,bar",
			&structTag{
				Functions: []stFunction{{Name: "foo"}, {Name: "bar"}}},
		},
		{
			"foo, bar",
			&structTag{
				Functions: []stFunction{{Name: "foo"}, {Name: "bar"}}},
		},
		{
			"foo , bar",
			&structTag{
				Functions: []stFunction{{Name: "foo"}, {Name: "bar"}}},
		},

		{
			"foo()",
			&structTag{
				Functions: []stFunction{{Name: "foo"}}},
		},

		{
			// test string value
			"foo(abc=def)",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{String: stringPtr("def")}}}}}},
		},
		{
			"foo(abc='d e f')",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{String: stringPtr("d e f")}}}}}},
		},
		{
			// test float value
			"foo(abc=1.0)",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{Float: float64Ptr(1.0)}}}}}},
		},
		{
			// test int value
			"foo(abc=1)",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{Int: intPtr(1)}}}}}},
		},
		{
			// test empty list value
			"foo(abc={})",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{}}}}}},
		},
		{
			// test list value with spacing
			"foo(abc = { def })",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{List: []string{"def"}}}}}}},
		},
		{
			// test list value with one element
			"foo(abc={def})",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{List: []string{"def"}}}}}}},
		},
		{
			"foo(abc={'d e f'})",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{List: []string{"d e f"}}}}}}},
		},
		{
			// test list value with two elements
			"foo(abc={def,ghi})",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{List: []string{"def", "ghi"}}}}}}},
		},
		{
			"foo(abc={ def , ghi })",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{List: []string{"def", "ghi"}}}}}}},
		},
		{
			"foo(abc={' d e f ', ghi })",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{List: []string{" d e f ", "ghi"}}}}}}},
		},
		{
			// test list value with three elements
			"foo(abc={def,ghi,jkl})",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{{Name: "abc", Value: stValue{List: []string{"def", "ghi", "jkl"}}}}}}},
		},

		{
			// test multiple args
			"foo(abc=def, ghi=jkl)",
			&structTag{
				Functions: []stFunction{
					{Name: "foo", Args: []stArg{
						{Name: "abc", Value: stValue{String: stringPtr("def")}},
						{Name: "ghi", Value: stValue{String: stringPtr("jkl")}},
					}}}},
		},

		// test actual struct tag
		{
			"index(method=btree, name=my_index, order=asc, composite={foo, bar})",
			&structTag{
				Functions: []stFunction{
					{Name: "index", Args: []stArg{
						{Name: "method", Value: stValue{String: stringPtr("btree")}},
						{Name: "name", Value: stValue{String: stringPtr("my_index")}},
						{Name: "order", Value: stValue{String: stringPtr("asc")}},
						{Name: "composite", Value: stValue{List: []string{"foo", "bar"}}},
					}}}},
		},
	}

	for _, x := range tt {
		st, err := parseStructTag(x.in)
		require.NoError(t, err)
		assert.Equal(t, x.expect, st, x.in)
	}

	// test a bunch of errors
	invalidStructTags := []string{
		"foo(bar)",
		"foo(abc: def)", "foo(abc: 'def')",
	}

	for _, x := range invalidStructTags {
		_, err := parseStructTag(x)
		assert.Error(t, err)
	}
}

func BenchmarkParseStructTag(b *testing.B) {
	tag := "index(method=btree, name=my_index, order=asc, composite={foo, bar})"
	for n := 0; n < b.N; n++ {
		_, err := parseStructTag(tag)
		if err != nil {
			panic(err)
		}
	}
}

func TestParseStructTag_PrimaryKey(t *testing.T) {
	tag := `pk(method=btree, name=abc, order=asc, composite={foo, bar})`

	f := &field{}
	require.NoError(t, f.parseStructTag(tag))

	expect := &primaryKeyStructTag{
		method:    "btree",
		name:      "abc",
		order:     "asc",
		composite: []string{"foo", "bar"},
	}
	require.Equal(t, expect, f.primaryKey)
}

func TestParseStructTag_Indexes(t *testing.T) {
	tag := `index(method=btree, name=abc, order=asc, composite={foo, bar}), index(name=def), unique(order=desc)`

	f := &field{}
	require.NoError(t, f.parseStructTag(tag))

	expect := []indexStructTag{
		{
			unique:    false,
			method:    "btree",
			name:      "abc",
			order:     "asc",
			composite: []string{"foo", "bar"},
		},
		{
			unique: false,
			name:   "def",
		},
		{
			unique: true,
			order:  "desc",
		},
	}
	require.Equal(t, expect, f.indexes)
}

func TestParseStructTag_ForeignKeys(t *testing.T) {
	tag := `references(struct=Foo, field=Bar), references(struct=a, fields={b,c})`

	f := field{}
	require.NoError(t, f.parseStructTag(tag))

	expect := []foreignKeyStructTag{
		{
			structName: "Foo",
			fieldNames: []string{"Bar"},
		},
		{
			structName: "a",
			fieldNames: []string{"b", "c"},
		},
	}
	require.Equal(t, expect, f.foreignKeys)
}

func TestParseStructTag_PartitionByRange(t *testing.T) {
	tag := `partitionByRange`

	f := field{}
	require.NoError(t, f.parseStructTag(tag))
	require.NotNil(t, f.partitionByRange)
}
