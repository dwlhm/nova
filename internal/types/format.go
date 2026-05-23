package types

import (
	"fmt"
	"strconv"
	"strings"
)

func Format(typ Type) string {
	switch typ.Kind {
	case KindPrimitive, KindOpaque, KindNamed:
		return typ.Name
	case KindUnknown:
		return "unknown"
	case KindVoid:
		return "void"
	case KindNull:
		return "null"
	case KindLiteral:
		switch typ.LiteralKind {
		case LiteralString:
			return strconv.Quote(fmt.Sprint(typ.Literal))
		case LiteralNumber, LiteralBoolean:
			return fmt.Sprint(typ.Literal)
		case LiteralNull:
			return "null"
		default:
			return fmt.Sprint(typ.Literal)
		}
	case KindArray:
		if typ.Element == nil {
			return "invalid[]"
		}
		return Format(*typ.Element) + "[]"
	case KindUnion:
		parts := make([]string, 0, len(typ.Options))
		for _, option := range typ.Options {
			parts = append(parts, Format(option))
		}
		return strings.Join(parts, " | ")
	case KindRecord:
		if typ.Name != "" {
			return typ.Name
		}
		return "record"
	default:
		return "invalid"
	}
}
