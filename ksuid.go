package postgres

import (
	"fmt"
	"strings"

	"github.com/segmentio/ksuid"
)

var ErrInvalidId = fmt.Errorf("invalid ID")

// NewID returns a new ramdon ID, prefixed with the registered name of the given struct.
func NewID(s Struct) string {
	name := structName(s)

	for prefix, x := range structs {
		if strings.EqualFold(x.name, name) {
			return toSnake(prefix) + "_" + ksuid.New().String()
		}
	}

	panic(fmt.Sprintf("unknown struct %T", s))
}

// NewPrefixID returns a new random ID, prefixed with the given prefix.
func NewPrefixID(prefix string) string {
	if prefix != "" {
		return toSnake(prefix) + "_" + ksuid.New().String()
	}
	return ksuid.New().String()
}

// ParseID parses a ID.
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
