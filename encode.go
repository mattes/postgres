package postgres

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"

	"github.com/lib/pq"
)

// ColumnTyper is an interface to be implemented by a custom type
type ColumnTyper interface {

	// ColumnType returns postgres' column type
	ColumnType() string
}

// columnType returns a postgres column type for a given value
func columnType(value interface{}) string {
	if ct, err := callColumnTyper(value); err != nil && err != ErrColumnTyperNotImplemented {
		panic(err)
	} else if err == nil {
		return ct
	}

	if implementsScanner(reflect.ValueOf(value)) {
		return "text null"
	}

	switch value.(type) {
	case time.Time, *time.Time:
		// INFO: we map the zero type to null
		return "timestamp (6) without time zone null"

	case time.Duration:
		// see https://github.com/lib/pq/issues/78 why we can't use postgres' interval type
		return "bigint not null default 0"

	case *time.Duration:
		// see comment above in regards to interval type
		return "bigint null"

	case []string:
		return "text[] null"
	}

	switch typeOf(value).Kind() {
	case reflect.String:
		return "text not null default ''"

	case reflect.Bool:
		return "boolean not null default false"

	case reflect.Int:
		return "integer not null default 0"

	case reflect.Struct:
		return "jsonb null"

	case reflect.Slice:
		return "jsonb null"

	case reflect.Map:
		return "jsonb null"
	}

	panic(fmt.Sprintf("columnType: unsupported Go type %v", reflect.TypeOf(value)))
}

// encodeValue returns driver.Value, which is stored in postgres, for a given value
func encodeValue(value reflect.Value) (driver.Value, error) {
	if (value.Kind() == reflect.Slice || value.Kind() == reflect.Map) && value.Len() == 0 {
		return nil, nil
	}

	if value.Kind() == reflect.Struct && isZero(value.Interface()) {
		return nil, nil
	}

	if value.Kind() == reflect.Slice {
		switch value.Interface().(type) {
		case []string:
			return pq.Array(value.Interface()).Value()
		}
	}

	switch v := value.Interface().(type) {

	case time.Time:
		// please note that postgres only stores microsecond 1e+6 precision
		return v.UTC().Truncate(time.Microsecond), nil // always store UTC

	case *time.Time:
		if v == nil {
			return nil, nil
		}
		return v.UTC().Truncate(time.Microsecond), nil // always store UTC

	case time.Duration:
		return v.Nanoseconds(), nil

	case *time.Duration:
		if v == nil {
			return nil, nil
		}
		return v.Nanoseconds(), nil
	}

	if reflect.PtrTo(value.Type()).Implements(valuerReflectType) {
		return value.Addr().Interface().(driver.Valuer).Value()
	}

	v, err := driver.DefaultParameterConverter.ConvertValue(value.Interface())
	if err == nil {
		return v, nil
	}

	return jsonMarshal(value.Interface())
}

// decodeValue stores decoded value in dst
func decodeValue(dst reflect.Value, value interface{}) error {
	// if value is nil, we can skip further processing
	if value == nil {
		return nil
	}

	if err := callScanner(dst, value); err != nil && err != ErrScannerNotImplemented {
		return err
	} else if err == nil {
		return nil
	}

	switch x := value.(type) {
	case time.Time:
		switch dst.Interface().(type) {
		case time.Time:
			return setValue(dst, x.UTC())

		case *time.Time:
			n := x.UTC()
			return setValue(dst, &n)
		}
	}

	switch dst.Interface().(type) {
	case time.Duration:
		return setValue(dst, time.Duration(value.(int64)))
	case *time.Duration:
		x := time.Duration(value.(int64))
		return setValue(dst, &x)
	}

	// reverse pointer if any
	dstKind := dst.Type().Kind()
	if dstKind == reflect.Ptr {
		dstKind = dst.Type().Elem().Kind()
	}

	switch dstKind {

	case reflect.String:
		if _, ok := dst.Interface().(string); ok {
			return setValue(dst, value)
		}

		return setValue(dst, reflect.ValueOf(value).Convert(dst.Type()).Interface())

	case reflect.Bool:
		return setValue(dst, value.(bool))

	case reflect.Int:
		switch v := value.(type) {
		case int64:
			return setValue(dst, int(v))
		}

	case reflect.Map:
		n := reflect.New(dst.Type()).Interface()

		if err := jsonUnmarshal([]byte(value.([]byte)), n); err != nil {
			return err
		}
		return setValue(dst, n)

	case reflect.Slice:
		switch dst.Interface().(type) {
		case []string:
			a := pq.StringArray{}
			if err := a.Scan(value.([]byte)); err != nil {
				return err
			}
			return setValue(dst, a)
		}

		n := reflect.New(dst.Type()).Interface()
		if err := jsonUnmarshal([]byte(value.([]byte)), n); err != nil {
			return err
		}
		return setValue(dst, n)

	case reflect.Struct:
		n := reflect.New(dst.Type())
		if err := jsonUnmarshal([]byte(value.([]byte)), n.Interface()); err != nil {
			return err
		}
		return setValue(dst, n.Elem().Interface())
	}

	return nil
}
