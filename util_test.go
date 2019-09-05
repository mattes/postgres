package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSnake(t *testing.T) {
	tests := []struct{ in, out string }{
		{"foo", "foo"},
		{"Foo", "foo"},
		{"FOO", "foo"},
		{"FooBar", "foo_bar"},
		{"FooBarAbc", "foo_bar_abc"},
		{"FOOBarAbc", "foo_bar_abc"},
		{"Col1Index", "col1_index"},
		{"Col1_Index", "col1_index"},
		{"Col_1_Index", "col_1_index"},
		{"1Test", "1_test"},
		{"Col_1", "col_1"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.out, toSnake(tt.in))
	}
}

func TestIdentifier(t *testing.T) {
	require.Equal(t, `"foo"`, mustIdentifier("foo"))
	require.Equal(t, `"public"."foo"`, mustIdentifier("public.foo"))
	require.Panics(t, func() { mustIdentifier("") })
}

func TestEqualStringSlice(t *testing.T) {
	require.True(t, equalStringSlice([]string{"a", "b", "c"}, []string{"a", "b", "c"}))
	require.False(t, equalStringSlice([]string{"a", "c", "b"}, []string{"a", "b", "c"}))
	require.False(t, equalStringSlice([]string{"a", "b"}, []string{"a", "b", "c"}))
}

func TestEqualStringSliceIgnoreOrder(t *testing.T) {
	require.True(t, equalStringSliceIgnoreOrder([]string{"a", "b", "c"}, []string{"a", "b", "c"}))
	require.True(t, equalStringSliceIgnoreOrder([]string{"a", "c", "b"}, []string{"a", "b", "c"}))
	require.False(t, equalStringSliceIgnoreOrder([]string{"a", "b"}, []string{"a", "b", "c"}))
}