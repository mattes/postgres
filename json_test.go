package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestJsonMarshalStructEmbedded struct {
	Bar string
}

type TestJsonMarshalStruct struct {
	FooBar     string
	Foo        []string
	Embedded   TestJsonMarshalStructEmbedded
	unexported bool
	Nil        interface{}
	Zero       string
}

func TestJsonMarshalAndUnmarshal(t *testing.T) {
	v := TestJsonMarshalStruct{
		FooBar: "foobar",
		Foo:    []string{"a", "b", "c"},
		Embedded: TestJsonMarshalStructEmbedded{
			Bar: "embedded",
		},
		unexported: true,
		Nil:        nil,
		Zero:       "",
	}

	out, err := jsonMarshal(v)
	require.NoError(t, err)

	expect := `{"foo_bar":"foobar","foo":["a","b","c"],"embedded":{"bar":"embedded"},"nil":null,"zero":""}`
	require.Equal(t, expect, string(out))

	var v2 TestJsonMarshalStruct
	err = jsonUnmarshal([]byte(expect), &v2)
	require.NoError(t, err)

	v2.unexported = true // set manually
	require.Equal(t, v, v2)
}

func TestJsonMarshal_NoEscapeHTML(t *testing.T) {
	out, err := jsonMarshal([]string{"<body>foobar</body>"})
	require.NoError(t, err)
	require.Equal(t, `["<body>foobar</body>"]`, string(out))
}

func TestJsonMarshal_EmptySlice(t *testing.T) {
	out, err := jsonMarshal([]string{})
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestJsonMarshal_Nil(t *testing.T) {
	out, err := jsonMarshal(nil)
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestJsonMarshal_EmptyMap(t *testing.T) {
	out, err := jsonMarshal(map[string]string{})
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestJsonMarshal_ZeroStruct(t *testing.T) {
	{
		out, err := jsonMarshal(struct{ Name string }{})
		require.NoError(t, err)
		require.Nil(t, out)
	}
	{
		out, err := jsonMarshal(&(struct{ Name string }{}))
		require.NoError(t, err)
		require.Nil(t, out)
	}
}
