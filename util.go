package postgres

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/azer/snakecase"
	"github.com/lib/pq"
)

func toSnake(in ...string) string {
	for i := 0; i < len(in); i++ {
		in[i] = strings.Map(toSnakeChars, in[i])
		in[i] = strings.Trim(in[i], "_")
		in[i] = snakecase.SnakeCase(in[i])
	}
	return strings.Join(in, "_")
}

func toSnakeChars(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r

	case r >= 'A' && r <= 'Z':
		return r

	case r >= '0' && r <= '9':
		return r

	default:
		return '_'
	}
}

func stringSliceToSnake(in []string) []string {
	out := make([]string, len(in))
	for i := 0; i < len(in); i++ {
		out[i] = toSnake(in[i])
	}
	return out
}

// equalStringSlice returns true if two string slices are exactly the same
func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// equalStringSliceIgnoreOrder returns true if two string slices are the same,
// ignoring the order of their content.
func equalStringSliceIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		found := false

		for j := 0; j < len(b); j++ {
			if a[i] == b[j] {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func isErrTableDoesNotExist(err error) bool {
	if err == nil {
		return false
	}

	if err, ok := err.(*pq.Error); ok {
		return err.Code.Name() == "undefined_table"
	}

	return false
}

func formatQuery(query string) string {
	return strings.TrimSpace(query)
}

func literal(in string) string {
	return pq.QuoteLiteral(in)
}

func mustIdentifier(in string) string {
	i, err := identifier(in)
	if err != nil {
		panic(err)
	}
	return i
}

func identifier(in string) (string, error) {
	if in == "" {
		panic("empty identifier")
	}

	parts := strings.SplitN(in, ".", 2)

	switch len(parts) {
	case 1:
		return quoteIdentifier(toSnake(parts[0]))

	case 2:
		if parts[0] == "" || parts[1] == "" {
			panic("empty identifier")
		}

		p0, err := quoteIdentifier(toSnake(parts[0]))
		if err != nil {
			return "", err
		}

		p1, err := quoteIdentifier(toSnake(parts[1]))
		if err != nil {
			return "", err
		}

		return p0 + "." + p1, nil

	default:
		return "", fmt.Errorf("invalid identifier")
	}
}

func mustQuoteIdentifier(in string) string {
	i, err := quoteIdentifier(in)
	if err != nil {
		panic(err)
	}
	return i
}

func quoteIdentifier(in string) (string, error) {
	in = strings.Trim(in, `"`)
	q := pq.QuoteIdentifier(in)
	return checkIdentifierLen(q)
}

func unquoteIdentifier(in string) string {
	return strings.Trim(in, `\"`)
}

func unquoteIdentifiers(in []string) []string {
	out := make([]string, len(in))
	for i := 0; i < len(in); i++ {
		out[i] = unquoteIdentifier(in[i])
	}
	return out
}

func checkIdentifierLen(in string) (string, error) {
	if len(in) > 63 {
		return "", fmt.Errorf("identifier too long: %v", in)
	}
	return in, nil
}

func mustJoinIdentifiers(in []string) string {
	out := make([]string, 0, len(in))
	for i := 0; i < len(in); i++ {
		if in[i] != "" {
			out = append(out, mustIdentifier(in[i]))
		}
	}
	return strings.Join(out, ", ")
}

func mustJoinIdentifiersWithPrefix(in []string, prefix string) string {
	p := mustIdentifier(prefix)
	out := make([]string, 0, len(in))
	for i := 0; i < len(in); i++ {
		if in[i] != "" {
			out = append(out, p+"."+mustIdentifier(in[i]))
		}
	}
	return strings.Join(out, ", ")
}

func join(in []string) string {
	out := make([]string, 0, len(in))
	for i := 0; i < len(in); i++ {
		if in[i] != "" {
			out = append(out, in[i])
		}
	}
	return strings.Join(out, ", ")
}

func toString(str interface{}) string {
	v := reflect.ValueOf(str)
	if v.Type().Kind() != reflect.String {
		panic("must be stringable type")
	}
	return v.String()
}

type queryFmt struct {
	elems []string
}

func queryf() *queryFmt {
	return &queryFmt{}
}

func (q *queryFmt) Append(args ...interface{}) *queryFmt {
	for i := 0; i < len(args); i++ {
		switch v := args[i].(type) {
		case string:
			if v != "" {
				q.elems = append(q.elems, v)
			}

		default:
			x := fmt.Sprintf("%v", v)
			if x != "" {
				q.elems = append(q.elems, x)
			}
		}
	}

	return q
}
func (q *queryFmt) Appendf(format string, args ...interface{}) *queryFmt {
	return q.Append(fmt.Sprintf(format, args...))
}

func (q *queryFmt) String() string {
	return strings.Join(q.elems, " ")
}
