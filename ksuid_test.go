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
	RegisterWithPrefix(&TestNewID_Struct{}, "test_new_id", "my_prefix")

	prefix, id, err := ParseID(NewID(&TestNewID_Struct{}))
	require.NoError(t, err)
	require.Equal(t, "my_prefix", prefix)
	require.Len(t, id.String(), 27)
}

type TestNewID_NoPrefix_Struct struct{}

func TestNewID_NoPrefix(t *testing.T) {
	RegisterWithPrefix(&TestNewID_NoPrefix_Struct{}, "test_new_id_no_prefix", "")

	prefix, id, err := ParseID(NewID(&TestNewID_NoPrefix_Struct{}))
	require.NoError(t, err)
	require.Equal(t, "", prefix)
	require.Len(t, id.String(), 27)
}
