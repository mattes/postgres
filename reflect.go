package postgres

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
)

var (
	valuerReflectType      = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	scannerReflectType     = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	columnTyperReflectType = reflect.TypeOf((*ColumnTyper)(nil)).Elem()
)

var (
	ErrScannerNotImplemented     = fmt.Errorf("type does not implement sql.Scanner")
	ErrColumnTyperNotImplemented = fmt.Errorf("type does not implement ColumnTyper")
)

func implementsScanner(v reflect.Value) bool {
	return v.Type().Implements(scannerReflectType) ||
		reflect.PtrTo(v.Type()).Implements(scannerReflectType)
}

func callScanner(dst reflect.Value, value interface{}) error {
	if dst.Type().Implements(scannerReflectType) {
		n := reflect.New(dst.Type().Elem())
		if err := n.Interface().(sql.Scanner).Scan(value); err != nil {
			return err
		}
		dst.Set(n)
		return nil
	}

	if reflect.PtrTo(dst.Type()).Implements(scannerReflectType) {
		if err := dst.Addr().Interface().(sql.Scanner).Scan(value); err != nil {
			return err
		}
		return nil
	}

	return ErrScannerNotImplemented
}

func callColumnTyper(value interface{}) (string, error) {
	switch x := value.(type) {
	case ColumnTyper:
		return x.ColumnType(), nil
	}

	return "", ErrColumnTyperNotImplemented
}

// isZero returns true if the given value is zero
func isZero(value interface{}) bool {
	if value == nil {
		return false // is nil, not zero
	}

	v := valueOf(value)
	if (v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.Len() == 0 {
		return true
	}

	return reflect.DeepEqual(
		v.Interface(),
		reflect.Zero(typeOf(value)).Interface())
}

func isNilOrZero(value interface{}) bool {
	return value == nil || isZero(value)
}

// typeOf returns reflect.Type. If a pointer is given,
// it returns type of pointer's value.
func typeOf(x interface{}) reflect.Type {
	typ := reflect.TypeOf(x)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

// valueOf returns reflect.Value. If pointer is given,
// it returns the pointer's value.
func valueOf(x interface{}) reflect.Value {
	val := reflect.ValueOf(x)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}

func isPointer(v interface{}) bool {
	return reflect.ValueOf(v).Kind() == reflect.Ptr
}

func structName(v interface{}) string {
	return typeOf(v).Name()
}

func setValue(dst reflect.Value, value interface{}) error {
	if !dst.CanSet() {
		return fmt.Errorf("field can't be set")
	}

	x := reflect.ValueOf(value)

	if dst.Kind() == reflect.Ptr && x.Kind() != reflect.Ptr {
		// dst is pointer, but x is not not a pointer
		// TODO
		panic("TODO")

	} else if dst.Kind() != reflect.Ptr && x.Kind() == reflect.Ptr {
		// dst is not a pointer, but x is pointer
		x = x.Elem()
	}

	if !x.Type().AssignableTo(dst.Type()) {
		return fmt.Errorf("field expects %T not %T", dst.Interface(), value)
	}

	dst.Set(x)
	return nil
}
