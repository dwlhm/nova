package types

import "github.com/dwlhm/nova/internal/core/lexer"

type typeParser struct {
	tokens      []lexer.Token
	pos         int
	env         Environment
	diagnostics []Diagnostic
}

func (p *typeParser) parseUnion() Type {
	options := []Type{p.parseArray()}
	for p.match(lexer.PIPE) {
		options = append(options, p.parseArray())
	}
	if len(options) == 1 {
		return options[0]
	}
	return Type{Kind: KindUnion, Options: options}
}

func (p *typeParser) parseArray() Type {
	typ := p.parsePrimary()
	for p.match(lexer.LBRACKET) {
		p.expect(lexer.RBRACKET)
		element := typ
		typ = Type{Kind: KindArray, Element: &element}
	}
	return typ
}

func (p *typeParser) parsePrimary() Type {
	if p.done() {
		p.errorf(lexer.Token{Type: lexer.EOF}, "expected type")
		return Type{Kind: KindInvalid}
	}
	tok := p.advance()
	switch tok.Type {
	case lexer.TYPE_STRING:
		return Type{Kind: KindPrimitive, Name: "string"}
	case lexer.TYPE_NUMBER:
		return Type{Kind: KindPrimitive, Name: "number"}
	case lexer.TYPE_BOOLEAN:
		return Type{Kind: KindPrimitive, Name: "boolean"}
	case lexer.TYPE_UNKNOWN:
		return Type{Kind: KindUnknown}
	case lexer.VOID:
		return Type{Kind: KindVoid}
	case lexer.NULL:
		return Type{Kind: KindNull}
	case lexer.STRING:
		return Type{Kind: KindLiteral, LiteralKind: LiteralString, Literal: tok.Literal}
	case lexer.NUMBER:
		return Type{Kind: KindLiteral, LiteralKind: LiteralNumber, Literal: tok.Literal}
	case lexer.TRUE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: true}
	case lexer.FALSE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: false}
	case lexer.IDENT, lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL,
		lexer.OPERATION, lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS,
		lexer.TARGET, lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return Type{Kind: KindNamed, Name: tok.Literal}
	default:
		p.errorf(tok, "expected type, got %s", tok.Type)
		return Type{Kind: KindInvalid}
	}
}
