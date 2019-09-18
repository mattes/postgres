package postgres

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// StructTag is the default struct tag
var StructTag = "db"

var (
	// structs contains all registered structs, where the key
	// of the map is the actual lowercase name of the struct.
	structs   = make(map[string]*metaStruct)
	structsMu sync.RWMutex
)

// Register registers a struct. Optional alias has to be globally unique.
// Optional prefixID is used in NewID().
func Register(s Struct, alias, prefixID string) {
	if s == nil {
		panic("Register: struct is nil")
	}

	structsMu.Lock()
	defer structsMu.Unlock()

	if _, dup := structs[globalStructsName(s)]; dup {
		panic(fmt.Sprintf("Register: called twice for struct %T", s))
	}

	x, err := newMetaStruct(s)
	if err != nil {
		panic(fmt.Sprintf("Register: %v", err))
	}

	// if alias is set, make sure it's globally unique
	if alias != "" {
		for _, sx := range structs {
			if strings.EqualFold(sx.name, alias) {
				panic(fmt.Sprintf("Register: alias '%v' for struct %T not globally unique", alias, s))
			}
		}

		// override name with alias
		x.name = alias
	}

	x.prefixID = prefixID

	structs[globalStructsName(s)] = x
}

// StructFieldName defines a struct's field name where interface{} must be
// "resolvable" as string.
type StructFieldName interface{}

// Struct is a Go struct, i.e. &User{}
type Struct interface{}

// StructSlice is a Go slice of structs, i.e. []User{}
type StructSlice interface{}

type metaStruct struct {
	name     string
	prefixID string
	fields   fields
}

type fields []*field

type field struct {
	// position is the index within the struct
	position int

	name  string
	value reflect.Value

	primary bool

	// unique(indexName)
	unique          bool
	uniqueIndexName string

	// index(indexName)
	index     bool
	indexName string

	// references(table.column)
	referencesStruct string
	referencesFields []string
}

func newMetaStruct(v interface{}) (*metaStruct, error) {
	r := &metaStruct{}
	r.name = structName(v)

	f, err := newFields(v, true)
	if err != nil {
		return nil, err
	}
	r.fields = f

	return r, nil
}

func newFields(v interface{}, withTags bool) (fields, error) {
	val := valueOf(v)
	typ := typeOf(v)

	f := make(fields, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		f[i] = &field{}
		f[i].position = i
		f[i].name = typ.Field(i).Name
		f[i].value = val.Field(i)

		if withTags {
			tags, err := parseTags(typ.Field(i).Tag.Get(StructTag))
			if err != nil {
				return nil, err
			}
			if err := f[i].assignTags(tags); err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}

func mustNewFields(v interface{}, withTags bool) fields {
	f, err := newFields(v, withTags)
	if err != nil {
		panic(err)
	}
	return f
}

// alias returns name (which could be an alias) from list of registered structs
func (m *metaStruct) alias() string {
	if x, ok := structs[globalStructsNameFromString(m.name)]; ok {
		return x.name
	}
	return m.name
}

// fieldMask returns fields based on given fieldmask
func (f fields) fieldMask(fieldMask []StructFieldName) []*field {
	out := make([]*field, 0, len(f))
	for i := 0; i < len(f); i++ {
		if fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i])
		}
	}
	return out
}

// names returns field's names based on given fieldmask
func (f fields) names(fieldMask ...StructFieldName) []string {
	out := make([]string, 0, len(f))
	for i := 0; i < len(f); i++ {
		if fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i].name)
		}
	}
	return out
}

// values returns field's values based on given fieldmask
func (f fields) values(fieldMask ...StructFieldName) []interface{} {
	out := make([]interface{}, 0, len(f))
	for i := 0; i < len(f); i++ {
		if fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i])
		}
	}
	return out
}

// primaryNames returns field's names where field is a primary key,
// based on given fieldmask
func (f fields) primaryNames(fieldMask ...StructFieldName) []string {
	out := make([]string, 0, len(f))
	for i := 0; i < len(f); i++ {
		if f[i].primary && fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i].name)
		}
	}
	return out
}

// primaryValues returns field's values where field is a primary key,
// based on given fieldmask
func (f fields) primaryValues(fieldMask ...StructFieldName) []interface{} {
	out := make([]interface{}, 0, len(f))
	for i := 0; i < len(f); i++ {
		if f[i].primary && fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i])
		}
	}
	return out
}

// nonPrimary returns fields where field is not a primary key,
// based on given fieldmask
func (f fields) nonPrimary(fieldMask ...StructFieldName) []*field {
	out := make([]*field, 0, len(f))
	for i := 0; i < len(f); i++ {
		if !f[i].primary && fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i])
		}
	}
	return out
}

// nonPrimaryNames returns field's names where field is a not a primary key,
// based on given fieldmask
func (f fields) nonPrimaryNames(fieldMask ...StructFieldName) []string {
	out := make([]string, 0, len(f))
	for i := 0; i < len(f); i++ {
		if !f[i].primary && fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i].name)
		}
	}
	return out
}

func (f fields) wherePrimaryStr(p *placeholderMap) string {
	out := make([]string, 0, len(f))
	for i := 0; i < len(f); i++ {
		if f[i].primary {
			out = append(out, fmt.Sprintf("%v = %v", mustIdentifier(f[i].name), p.next(f[i])))
		}
	}
	return strings.Join(out, " AND ")
}

func (f fields) uniqueIndexes() map[string][]string {
	out := make(map[string][]string)
	for _, x := range f {
		if x.unique {

			// dynamically create index name if not set
			if x.uniqueIndexName == "" {
				x.uniqueIndexName = fmt.Sprintf("%v_unique", x.name)
			}

			if _, ok := out[x.uniqueIndexName]; !ok {
				out[x.uniqueIndexName] = []string{x.name}
			} else {
				out[x.uniqueIndexName] = append(out[x.uniqueIndexName], x.name)
			}
		}
	}

	return out
}

func (f fields) indexes() map[string][]string {
	out := make(map[string][]string)
	for _, x := range f {
		if x.index {

			// dynamically create index name if not set
			if x.indexName == "" {
				x.indexName = fmt.Sprintf("%v_index", x.name)
			}

			if _, ok := out[x.indexName]; !ok {
				out[x.indexName] = []string{x.name}
			} else {
				out[x.indexName] = append(out[x.indexName], x.name)
			}
		}
	}
	return out
}

// Scan implements database/sql#Scanner on all fields
func (f fields) Scan(row rowScan) error {
	values, err := scan(row, len(f))
	if err != nil {
		return err
	}

	if len(f) != len(values) {
		// this should never happen, but adding this here as a safeguard
		panic("Scan: wrong number of scanned values")
	}

	for i := 0; i < len(f); i++ {
		if err := f[i].Scan(values[i]); err != nil {
			return err
		}
	}

	return nil
}

func (f *field) String() string {
	return fmt.Sprintf("%v = %v (%T)", f.name, f.value, f.value.Interface())
}

// Scan implements database/sql#Scanner on an individual field
func (f *field) Scan(value interface{}) error {
	err := decodeValue(f.value, value)
	if err != nil {
		return fmt.Errorf("field %v: %v", f.name, err)
	}
	return nil
}

// Value implements database/sql/driver#Valuer
func (f *field) Value() (driver.Value, error) {
	v, err := encodeValue(f.value)
	if err != nil {
		return nil, fmt.Errorf("field %v: %v", f.name, err)
	}
	return v, nil
}

// ColumnType returns the postgres column type for this fields' value
func (f *field) columnType() string {
	return columnType(f.value.Interface())
}

func (f *field) assignTags(tags []tag) error {
	for _, t := range tags {
		switch t.key {
		case "pk":
			f.primary = true

		case "unique":
			f.unique = true

			if len(t.values) == 1 {
				f.uniqueIndexName = t.values[0]
			} else if len(t.values) > 1 {
				return fmt.Errorf("field %v has too many values for tag 'unique'", f.name)
			}

		case "index":
			f.index = true
			if len(t.values) == 1 {
				f.indexName = t.values[0]

			} else if len(t.values) > 1 {
				return fmt.Errorf("field %v has too many values for tag 'index'", f.name)
			}

		case "references":
			if len(t.values) != 1 {
				return fmt.Errorf("field %v has too many values for tag 'references'", f.name)
			}

			parts := strings.SplitN(t.values[0], ".", 2)
			if len(parts) != 2 {
				return fmt.Errorf("field %v has invalid tag 'references'", f.name)
			}

			f.referencesStruct = parts[0]
			f.referencesFields = []string{parts[1]}
		}
	}

	return nil
}

// fieldMaskMatch returns true if given fieldMask is empty or
// if searched field is present in fieldMask.
func fieldMaskMatch(fieldMask []StructFieldName, name string) bool {
	if isEmptyFieldMask(fieldMask) {
		return true
	}

	x := toSnake(name)

	for i := 0; i < len(fieldMask); i++ {
		if strings.EqualFold(toSnake(toString(fieldMask[i])), x) {
			return true
		}
	}

	return false
}

// isEmptyFieldMask returns true if fieldMask is nil, has no elements
// or if it has a nil element.
func isEmptyFieldMask(fieldMask []StructFieldName) bool {
	return fieldMask == nil || len(fieldMask) == 0 || (len(fieldMask) == 1 && fieldMask[0] == nil)
}

type tag struct {
	key    string
	values []string
}

// parseTag parses the part within the quotes of a struct tag like:
// `db:"pk,unique(foo,bar)"`
func parseTags(in string) ([]tag, error) {
	if len(in) == 0 {
		return []tag{}, nil
	}

	tags := make([]tag, 0)
	var splitErr error

	// helper func to split key/value pairs,
	// essentially ignore commas inside key/value pair
	insideValues := false
	insideKey := false
	splitPairs := func(c rune) bool {
		switch c {
		case '\'':
			insideKey = !insideKey

		case '(':
			if insideValues {
				splitErr = fmt.Errorf("invalid tag")
				return false
			}
			insideValues = true

		case ')':
			if !insideValues {
				splitErr = fmt.Errorf("invalid tag")
				return false
			}
			insideValues = false

		case ',':
			return !insideValues && !insideKey

		}
		return false
	}

	insideValue := false
	splitValues := func(c rune) bool {
		switch c {
		case '\'':
			insideValue = !insideValue

		case ',':
			return !insideValue

		}
		return false
	}

	// split key/value pairs
	for _, pair := range strings.FieldsFunc(in, splitPairs) {
		pair = strings.TrimSpace(pair)

		// split a single key/value pair
		kv := strings.Split(pair, "(")
		if len(kv) == 1 {
			// key only
			key := strings.Trim(strings.TrimSpace(kv[0]), "'")
			tags = append(tags, tag{key: key})

		} else if len(kv) == 2 {
			// key and value
			values := strings.FieldsFunc(strings.TrimRight(kv[1], ")"), splitValues)
			trimmedValues := make([]string, 0, len(values))
			for _, v := range values {
				v = strings.TrimSpace(v)
				if len(v) > 0 {
					v = strings.Trim(v, "'")
					trimmedValues = append(trimmedValues, v)
				}
			}

			if len(trimmedValues) == 0 {
				trimmedValues = nil
			}

			key := strings.Trim(strings.TrimSpace(kv[0]), "'")
			tags = append(tags, tag{key: key, values: trimmedValues})

		} else {
			return nil, fmt.Errorf("invalid tag")
		}
	}

	// catching error from split funcs
	if splitErr != nil {
		return nil, splitErr
	}

	return tags, nil
}

func globalStructsName(s Struct) string {
	return strings.ToLower(toSnake(structName(s)))
}

func globalStructsNameFromString(s string) string {
	return strings.ToLower(toSnake(s))
}
