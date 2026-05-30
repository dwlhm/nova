package types

import (
	"fmt"
	"reflect"
)

func ValidateValueText(typeRef string, value any) error {
	typ, diagnostics := ParseText(typeRef)
	if len(diagnostics) > 0 {
		return fmt.Errorf("invalid type %s: %s", typeRef, diagnostics[0].Message)
	}
	return EmptyEnvironment().ValidateValue(typ, value)
}

func (env Environment) ValidateValue(typ Type, value any) error {
	if env.matches(typ, value) {
		return nil
	}
	return fmt.Errorf("expected %s", Format(typ))
}
func IsSerializable(value any) bool {
	if value == nil {
		return true
	}
	if opaque, ok := value.(OpaqueValue); ok {
		return IsSerializable(opaque.Value)
	}
	return isSerializableValue(reflect.ValueOf(value))
}

func (env Environment) matches(typ Type, value any) bool {
	typ = env.resolve(typ)
	switch typ.Kind {
	case KindInvalid:
		return false
	case KindUnknown:
		return IsSerializable(value)
	case KindVoid, KindNull:
		return value == nil
	case KindLiteral:
		return literalMatches(typ, value)
	case KindPrimitive:
		return primitiveMatches(typ.Name, value)
	case KindArray:
		return typ.Element != nil && env.arrayMatches(*typ.Element, value)
	case KindRecord:
		return env.recordMatches(typ, value)
	case KindUnion:
		for _, option := range typ.Options {
			if env.matches(option, value) {
				return true
			}
		}
		return false
	case KindOpaque:
		opaque, ok := value.(OpaqueValue)
		return ok && opaque.Type == typ.Name && IsSerializable(opaque.Value)
	case KindNamed:
		return env.AllowUnknownNamedSerializable && IsSerializable(value)
	default:
		return false
	}
}
