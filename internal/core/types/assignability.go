package types

func (env Environment) Assignable(source Type, target Type) bool {
	source = env.resolve(source)
	target = env.resolve(target)

	if target.Kind == KindUnknown {
		return true
	}
	if source.Kind == KindUnknown {
		return true
	}
	if source.Kind == KindInvalid || target.Kind == KindInvalid {
		return false
	}
	if target.Kind == KindUnion {
		for _, option := range target.Options {
			if env.Assignable(source, option) {
				return true
			}
		}
		return false
	}
	if source.Kind == KindUnion {
		for _, option := range source.Options {
			if !env.Assignable(option, target) {
				return false
			}
		}
		return true
	}
	if source.Kind == KindLiteral {
		switch source.LiteralKind {
		case LiteralString:
			return target.Kind == KindPrimitive && target.Name == "string" ||
				target.Kind == KindLiteral && target.LiteralKind == LiteralString && target.Literal == source.Literal
		case LiteralNumber:
			return target.Kind == KindPrimitive && target.Name == "number" ||
				target.Kind == KindLiteral && target.LiteralKind == LiteralNumber && target.Literal == source.Literal
		case LiteralBoolean:
			return target.Kind == KindPrimitive && target.Name == "boolean" ||
				target.Kind == KindLiteral && target.LiteralKind == LiteralBoolean && target.Literal == source.Literal
		case LiteralNull:
			return target.Kind == KindNull
		}
	}
	if source.Kind == KindNull {
		return target.Kind == KindNull
	}
	if source.Kind != target.Kind {
		return false
	}
	switch source.Kind {
	case KindPrimitive, KindOpaque, KindNamed:
		return source.Name == target.Name
	case KindUnknown, KindVoid, KindNull:
		return true
	case KindArray:
		return source.Element != nil && target.Element != nil && env.Assignable(*source.Element, *target.Element)
	case KindRecord:
		return env.recordAssignable(source, target)
	case KindLiteral:
		return source.LiteralKind == target.LiteralKind && source.Literal == target.Literal
	default:
		return false
	}
}
func (env Environment) resolve(typ Type) Type {
	if typ.Kind != KindNamed {
		return typ
	}
	resolved, ok := env.Definitions[typ.Name]
	if !ok {
		return typ
	}
	if resolved.Kind == KindNamed && resolved.Name == typ.Name {
		return typ
	}
	return env.resolve(resolved)
}

func (env Environment) recordAssignable(source Type, target Type) bool {
	sourceFields := make(map[string]Field, len(source.Fields))
	for _, field := range source.Fields {
		sourceFields[field.Name] = field
	}
	for _, targetField := range target.Fields {
		sourceField, ok := sourceFields[targetField.Name]
		if !ok {
			if targetField.Optional {
				continue
			}
			return false
		}
		if !env.Assignable(sourceField.Type, targetField.Type) {
			return false
		}
	}
	return true
}
