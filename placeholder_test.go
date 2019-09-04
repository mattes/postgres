package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestPlaceholder_Struct struct {
	Field1 string
	Field2 string
	Field3 string
	Field4 string
	Field5 string
	Field6 string
}

func TestPlaceholder(t *testing.T) {
	r := mustNewMetaStruct(&TestPlaceholder_Struct{"a", "b", "c", "d", "e", "f"})

	// verify positions
	require.Equal(t, 0, r.fields[0].position)
	require.Equal(t, 1, r.fields[1].position)
	require.Equal(t, 2, r.fields[2].position)
	require.Equal(t, 3, r.fields[3].position)
	require.Equal(t, 4, r.fields[4].position)
	require.Equal(t, 5, r.fields[5].position)

	p := newPlaceholderMap()

	// test assign and next
	require.Equal(t, "$1", p.next(r.fields[4]))
	require.Equal(t, []string{"$2", "$3"}, p.assign(r.fields[1:3]...))

	// test args
	args := p.args(r.fields)
	require.Len(t, args, 3)
	require.Equal(t, 4, args[0].(*field).position)
	require.Equal(t, 1, args[1].(*field).position)
	require.Equal(t, 2, args[2].(*field).position)
	require.Equal(t, "e", args[0].(*field).value.String())
	require.Equal(t, "b", args[1].(*field).value.String())
	require.Equal(t, "c", args[2].(*field).value.String())
}
