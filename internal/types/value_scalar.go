package types

import (
	"fmt"
	"reflect"
)

func primitiveMatches(name string, value any) bool {
	if value == nil {
		return false
	}
	ref := unwrap(reflect.ValueOf(value))
	if !ref.IsValid() {
		return false
	}
	switch name {
	case "string":
		return ref.Kind() == reflect.String
	case "number":
		return isNumberValue(ref)
	case "boolean":
		return ref.Kind() == reflect.Bool
	default:
		return false
	}
}

func literalMatches(typ Type, value any) bool {
	switch typ.LiteralKind {
	case LiteralString:
		ref := unwrap(reflect.ValueOf(value))
		return ref.IsValid() && ref.Kind() == reflect.String && ref.String() == typ.Literal
	case LiteralNumber:
		return fmt.Sprint(value) == typ.Literal
	case LiteralBoolean:
		ref := unwrap(reflect.ValueOf(value))
		return ref.IsValid() && ref.Kind() == reflect.Bool && ref.Bool() == typ.Literal
	case LiteralNull:
		return value == nil
	default:
		return false
	}
}

func isNumberValue(value reflect.Value) bool {
	value = unwrap(value)
	if !value.IsValid() {
		return false
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}
