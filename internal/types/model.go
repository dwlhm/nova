package types

import "github.com/dwlhm/nova/internal/lexer"

type Kind string

const (
	KindPrimitive Kind = "primitive"
	KindUnknown   Kind = "unknown"
	KindVoid      Kind = "void"
	KindNull      Kind = "null"
	KindLiteral   Kind = "literal"
	KindArray     Kind = "array"
	KindRecord    Kind = "record"
	KindUnion     Kind = "union"
	KindOpaque    Kind = "opaque"
	KindNamed     Kind = "named"
	KindInvalid   Kind = "invalid"
)

type LiteralKind string

const (
	LiteralString  LiteralKind = "string"
	LiteralNumber  LiteralKind = "number"
	LiteralBoolean LiteralKind = "boolean"
	LiteralNull    LiteralKind = "null"
)

type Type struct {
	Kind        Kind
	Name        string
	Element     *Type
	Options     []Type
	Fields      []Field
	LiteralKind LiteralKind
	Literal     any
}

type Field struct {
	Name     string
	Optional bool
	Type     Type
}

type Environment struct {
	Definitions                   map[string]Type
	AllowUnknownNamedSerializable bool
}

type Diagnostic struct {
	Message string
	Token   lexer.Token
}

type Scope map[string]Type

type OpaqueValue struct {
	Type  string
	Value any
}
