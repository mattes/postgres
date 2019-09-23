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
func Register(s Struct, alias string) {
	RegisterWithPrefix(s, alias, "")
}

// RegisterWithPrefix registers a struct. Optional alias has to be globally unique.
// Optional prefixID is used in NewID().
func RegisterWithPrefix(s Struct, alias string, prefixID string) {
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
		alias = toSnake(alias)
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

	// data parsed from struct tag
	primaryKey       *primaryKeyStructTag
	foreignKeys      []foreignKeyStructTag
	indexes          []indexStructTag
	partitionByRange *partitionByRangeStructTag
}

func newMetaStruct(v interface{}) (*metaStruct, error) {
	r := &metaStruct{}
	r.name = toSnake(structName(v))

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
			if err := f[i].parseStructTag(typ.Field(i).Tag.Get(StructTag)); err != nil {
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

func (f fields) primaryFields(fieldMask ...StructFieldName) []*field {
	out := make([]*field, 0, len(f))

	for i := 0; i < len(f); i++ {
		if f[i].primaryKey != nil && fieldMaskMatch(fieldMask, f[i].name) {
			out = append(out, f[i])
		}

		// append composite primary keys
		if f[i].primaryKey != nil {
			for _, name := range f[i].primaryKey.composite {
				fx := f.mustFindByName(name)
				if fieldMaskMatch(fieldMask, fx.name) {
					out = append(out, fx)
				}
			}
		}
	}

	return out
}

// primaryNames returns field's names where field is a primary key,
// based on given fieldmask
func (f fields) primaryNames(fieldMask ...StructFieldName) []string {
	pf := f.primaryFields(fieldMask...)
	out := make([]string, 0, len(pf))
	for _, field := range pf {
		out = append(out, field.name)
	}
	return out
}

// primaryValues returns field's values where field is a primary key,
// based on given fieldmask
func (f fields) primaryValues(fieldMask ...StructFieldName) []interface{} {
	pf := f.primaryFields(fieldMask...)
	out := make([]interface{}, 0, len(pf))
	for _, field := range pf {
		out = append(out, field)
	}
	return out
}

// nonPrimaryFields returns fields where field is not a primary key,
// based on given fieldmask
func (f fields) nonPrimaryFields(fieldMask ...StructFieldName) []*field {
	out := make([]*field, 0, len(f))
	for i := 0; i < len(f); i++ {
		if f[i].primaryKey == nil && fieldMaskMatch(fieldMask, f[i].name) {

			// make sure field is not part of composite key
			found := false
			for j := 0; j < len(f); j++ {
				if f[j].primaryKey != nil && stringSliceContains(f[j].primaryKey.composite, f[i].name) {
					found = true
					break
				}
			}

			if !found {
				out = append(out, f[i])
			}
		}
	}

	return out
}

// nonPrimaryNames returns field's names where field is a not a primary key,
// based on given fieldmask
func (f fields) nonPrimaryNames(fieldMask ...StructFieldName) []string {
	npf := f.nonPrimaryFields(fieldMask...)
	out := make([]string, 0, len(npf))
	for _, field := range npf {
		out = append(out, field.name)
	}
	return out
}

func (f fields) wherePrimaryStr(p *placeholderMap) string {
	pf := f.primaryFields(nil)
	out := make([]string, 0, len(pf))
	for _, x := range pf {
		out = append(out, fmt.Sprintf("%v = %v", mustIdentifier(x.name), p.next(x)))
	}
	return strings.Join(out, " AND ")
}

// uniqueIndexes returns map of unique indexes and it columns/ fields
func (f fields) uniqueIndexes() map[string][]string {
	out := make(map[string][]string)
	for _, x := range f {

		for _, index := range x.indexes {
			if !index.unique {
				continue
			}

			// dynamically create index name if not set
			if index.name == "" {
				index.name = fmt.Sprintf("%v_unique", strings.Join(append([]string{x.name}, index.composite...), "_"))
			}

			out[index.name] = append([]string{x.name}, index.composite...)
		}
	}

	return out
}

func (f fields) indexes() map[string][]string {
	out := make(map[string][]string)
	for _, x := range f {

		for _, index := range x.indexes {
			if index.unique {
				continue
			}

			// dynamically create index name if not set
			if index.name == "" {
				index.name = fmt.Sprintf("%v_index", strings.Join(append([]string{x.name}, index.composite...), "_"))
			}

			if _, ok := out[index.name]; !ok {
				out[index.name] = []string{x.name}
			} else {
				out[index.name] = append(out[index.name], x.name)
			}

			if len(index.composite) > 0 {
				out[index.name] = append(out[index.name], index.composite...)
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

func (f fields) hasPartitionedField() bool {
	for _, x := range f {
		if x.partitionByRange != nil {
			return true
		}
	}
	return false
}

func (f fields) findByName(name string) *field {
	if name == "" {
		return nil
	}

	for i := 0; i < len(f); i++ {
		if strings.EqualFold(f[i].name, name) {
			return f[i]
		}
	}

	return nil
}

func (f fields) mustFindByName(name string) *field {
	x := f.findByName(name)
	if x == nil {
		panic(fmt.Sprintf("field '%v' does not exist", name))
	}

	return x
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

func globalStructsName(s Struct) string {
	return strings.ToLower(toSnake(structName(s)))
}

func globalStructsNameFromString(s string) string {
	return strings.ToLower(toSnake(s))
}
