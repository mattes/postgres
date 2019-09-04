package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPrefixID(t *testing.T) {
	{
		n := NewPrefixID("")
		require.Len(t, n, 27)
		t.Logf("Example of KSUID: %v", n)
	}

	{
		n := NewPrefixID("customer")
		t.Logf("Example of KSUID with prefix: %v", n)

		prefix, id, err := ParseID(n)
		require.NoError(t, err)
		require.Equal(t, "customer", prefix)
		require.Len(t, id.String(), 27)
	}

	{
		n := NewPrefixID("customer_with_underscores")
		t.Logf("Example of KSUID with prefix: %v", n)

		prefix, id, err := ParseID(n)
		require.NoError(t, err)
		require.Equal(t, "customer_with_underscores", prefix)
		require.Len(t, id.String(), 27)
	}

	{
		n := NewPrefixID("CustomerCamelcase")
		t.Logf("Example of KSUID with prefix: %v", n)

		prefix, id, err := ParseID(n)
		require.NoError(t, err)
		require.Equal(t, "customer_camelcase", prefix)
		require.Len(t, id.String(), 27)
	}
}

type TestNewID_Struct struct{}

func TestNewID(t *testing.T) {
	Register("test_new_id_foobar", &TestNewID_Struct{})

	prefix, id, err := ParseID(NewID(&TestNewID_Struct{}))
	require.NoError(t, err)
	require.Equal(t, "test_new_id_foobar", prefix)
	require.Len(t, id.String(), 27)
}
