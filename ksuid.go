package postgres

import (
	"fmt"
	"strings"

	"github.com/segmentio/ksuid"
)

var ErrInvalidId = fmt.Errorf("invalid ID")

// NewID returns a new ramdon ID, prefixed with the registered name of the given struct.
// Example: `user_1R0D8rn6jP870lrtSpgb1y6M5tG`
func NewID(s Struct) string {
	if s == nil {
		panic(fmt.Sprintf("nil struct %T", s))
	}

	x, ok := structs[globalStructsName(s)]
	if !ok {
		panic(fmt.Sprintf("unknown prefix for struct %T", s))
	}

	return NewPrefixID(x.prefixID)
}

// NewPrefixID returns a new random ID, prefixed with the given prefix.
// Example: `user_1R0D8rn6jP870lrtSpgb1y6M5tG`
func NewPrefixID(prefix string) string {
	if prefix != "" {
		return toSnake(prefix) + "_" + ksuid.New().String()
	}
	return ksuid.New().String()
}

// ParseID parses a ID like `user_1R0D8rn6jP870lrtSpgb1y6M5tG`.
func ParseID(id string) (prefix string, kid ksuid.KSUID, err error) {
	parts := strings.Split(id, "_")
	if len(parts) == 1 {
		k, err := ksuid.Parse(parts[0])
		return "", k, err

	} else if len(parts) > 1 {
		k, err := ksuid.Parse(parts[len(parts)-1])
		if err != nil {
			return "", ksuid.Nil, err
		}
		return strings.Join(parts[0:len(parts)-1], "_"), k, nil
	}

	return "", ksuid.Nil, ErrInvalidId
}

// NewID is a convenience function calling NewID. It exists
// so that the package doesn't have to be imported if just a *Postgres
// instance is passed around.
func (p *Postgres) NewID(s Struct) string {
	return NewID(s)
}
