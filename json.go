package postgres

import (
	"unicode"

	"github.com/azer/snakecase"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.Config{
	EscapeHTML:                    false,
	SortMapKeys:                   true,
	ValidateJsonRawMessage:        true,
	CaseSensitive:                 false,
	ObjectFieldMustBeSimpleString: true,
}.Froze()

func jsonMarshal(v interface{}) ([]byte, error) {
	if isNilOrZero(v) {
		return nil, nil
	}

	return json.Marshal(v)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, &v)
}

func init() {
	jsoniter.RegisterExtension(
		&namingStrategyExtension{jsoniter.DummyExtension{}, snakecase.SnakeCase})
}

type namingStrategyExtension struct {
	jsoniter.DummyExtension
	translate func(string) string
}

func (extension *namingStrategyExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		exported := unicode.IsUpper(rune(binding.Field.Name()[0]))
		if !exported {
			continue
		}

		binding.ToNames = []string{extension.translate(binding.Field.Name())}
		binding.FromNames = []string{extension.translate(binding.Field.Name())}
	}
}
