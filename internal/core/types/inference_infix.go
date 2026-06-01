package types

import "github.com/dwlhm/nova/internal/core/lexer"

func (env Environment) inferInfix(scope Scope, tokens []lexer.Token, op lexer.TokenType) (Type, bool) {
	parts := splitTopLevel(tokens, op)
	if len(parts) < 2 {
		return Type{}, false
	}
	allNumber := true
	allString := op == lexer.PLUS
	for _, part := range parts {
		typ, ok := env.InferExpression(scope, part)
		if !ok {
			return Type{}, false
		}
		if !isNumberLike(env.resolve(typ)) {
			allNumber = false
		}
		if !isStringLike(env.resolve(typ)) {
			allString = false
		}
	}
	if allNumber {
		return Type{Kind: KindPrimitive, Name: "number"}, true
	}
	if allString {
		return Type{Kind: KindPrimitive, Name: "string"}, true
	}
	return Type{}, false
}

func isNumberLike(typ Type) bool {
	return typ.Kind == KindPrimitive && typ.Name == "number" ||
		typ.Kind == KindLiteral && typ.LiteralKind == LiteralNumber
}

func isStringLike(typ Type) bool {
	return typ.Kind == KindPrimitive && typ.Name == "string" ||
		typ.Kind == KindLiteral && typ.LiteralKind == LiteralString
}
