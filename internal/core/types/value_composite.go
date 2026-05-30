package types

import "reflect"

func (env Environment) arrayMatches(element Type, value any) bool {
	ref := unwrap(reflect.ValueOf(value))
	if !ref.IsValid() {
		return false
	}
	switch ref.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < ref.Len(); i++ {
			if !env.matches(element, ref.Index(i).Interface()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (env Environment) recordMatches(typ Type, value any) bool {
	ref := unwrap(reflect.ValueOf(value))
	if !ref.IsValid() || ref.Kind() != reflect.Map || ref.Type().Key().Kind() != reflect.String {
		return false
	}
	for _, field := range typ.Fields {
		valueRef := ref.MapIndex(reflect.ValueOf(field.Name))
		if !valueRef.IsValid() {
			if field.Optional {
				continue
			}
			return false
		}
		if !env.matches(field.Type, valueRef.Interface()) {
			return false
		}
	}
	return IsSerializable(value)
}
