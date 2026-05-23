package scheduler

import (
	"reflect"
	"strings"
)

func matchesSingleStateType(typeRef string, value DataValue) bool {
	switch typeRef {
	case "", "unknown":
		return true
	case "void", "null":
		return value == nil
	}
	if value == nil {
		return false
	}
	if strings.HasSuffix(typeRef, "[]") {
		return isArrayValue(reflect.ValueOf(value))
	}

	valueRef := unwrapValue(reflect.ValueOf(value))
	switch typeRef {
	case "string":
		return valueRef.IsValid() && valueRef.Kind() == reflect.String
	case "number":
		return isNumberValue(valueRef)
	case "boolean":
		return valueRef.IsValid() && valueRef.Kind() == reflect.Bool
	default:
		return isSerializable(value)
	}
}
