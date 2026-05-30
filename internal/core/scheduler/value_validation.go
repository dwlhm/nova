package scheduler

import "reflect"

func unwrapValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func isNumberValue(value reflect.Value) bool {
	value = unwrapValue(value)
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

func isArrayValue(value reflect.Value) bool {
	value = unwrapValue(value)
	if !value.IsValid() {
		return false
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

func isSerializable(value DataValue) bool {
	if value == nil {
		return true
	}
	return isSerializableValue(reflect.ValueOf(value))
}

func isSerializableValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Interface:
		if value.IsNil() {
			return true
		}
		return isSerializableValue(value.Elem())
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
