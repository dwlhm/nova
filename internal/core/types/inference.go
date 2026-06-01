package types

import "github.com/dwlhm/nova/internal/core/lexer"

func (env Environment) InferExpression(scope Scope, tokens []lexer.Token) (Type, bool) {
	tokens = trimNoise(tokens)
	tokens = trimWrappedParens(tokens)
	if len(tokens) == 0 {
		return Type{}, false
	}
	if len(tokens) == 1 {
		return env.inferSingle(scope, tokens[0])
	}
	if fieldType, ok := env.inferFieldAccess(scope, tokens); ok {
		return fieldType, true
	}
	if typ, ok := env.inferRecordLiteral(scope, tokens); ok {
		return typ, true
	}
	if typ, ok := env.inferInfix(scope, tokens, lexer.PLUS); ok {
		return typ, true
	}
	for _, op := range []lexer.TokenType{lexer.MINUS, lexer.ASTERISK, lexer.SLASH} {
		if typ, ok := env.inferInfix(scope, tokens, op); ok {
			return typ, true
		}
	}
	return Type{}, false
}

func (env Environment) inferSingle(scope Scope, tok lexer.Token) (Type, bool) {
	switch tok.Type {
	case lexer.STRING:
		return Type{Kind: KindLiteral, LiteralKind: LiteralString, Literal: tok.Literal}, true
	case lexer.NUMBER:
		return Type{Kind: KindLiteral, LiteralKind: LiteralNumber, Literal: tok.Literal}, true
	case lexer.TRUE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: true}, true
	case lexer.FALSE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: false}, true
	case lexer.NULL:
		return Type{Kind: KindNull}, true
	case lexer.VOID:
		return Type{Kind: KindVoid}, true
	case lexer.IDENT:
		typ, ok := scope[tok.Literal]
		return typ, ok
	default:
		return Type{}, false
	}
}

func (env Environment) inferFieldAccess(scope Scope, tokens []lexer.Token) (Type, bool) {
	if len(tokens) != 3 || tokens[0].Type != lexer.IDENT || tokens[1].Type != lexer.DOT || tokens[2].Type != lexer.IDENT {
		return Type{}, false
	}
	base, ok := scope[tokens[0].Literal]
	if !ok {
		return Type{}, false
	}
	base = env.resolve(base)
	if base.Kind != KindRecord {
		return Type{}, false
	}
	for _, field := range base.Fields {
		if field.Name == tokens[2].Literal {
			return field.Type, true
		}
	}
	return Type{}, false
}
