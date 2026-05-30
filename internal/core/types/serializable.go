package types

import "reflect"

func isSerializableValue(value reflect.Value) bool {
	value = unwrap(value)
	if !value.IsValid() {
		return true
	}
	if value.CanInterface() {
		if opaque, ok := value.Interface().(OpaqueValue); ok {
			return IsSerializable(opaque.Value)
		}
	}
	switch value.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if !isSerializableValue(value.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return false
		}
		for _, key := range value.MapKeys() {
			if !isSerializableValue(value.MapIndex(key)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func unwrap(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}
