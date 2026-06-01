package types

import "github.com/dwlhm/nova/internal/core/lexer"

func (env Environment) inferRecordLiteral(scope Scope, tokens []lexer.Token) (Type, bool) {
	fields, ok := parseRecordLiteralFields(tokens)
	if !ok {
		return Type{}, false
	}

	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		typ, ok := env.InferExpression(scope, field.Value)
		if !ok {
			return Type{}, false
		}
		out = append(out, Field{
			Name: field.Name.Literal,
			Type: typ,
		})
	}
	return Type{Kind: KindRecord, Fields: out}, true
}

type recordLiteralField struct {
	Name  lexer.Token
	Value []lexer.Token
}

func parseRecordLiteralFields(tokens []lexer.Token) ([]recordLiteralField, bool) {
	if len(tokens) < 2 || tokens[0].Type != lexer.LBRACE || tokens[len(tokens)-1].Type != lexer.RBRACE {
		return nil, false
	}

	body := tokens[1 : len(tokens)-1]
	fields := make([]recordLiteralField, 0)
	pos := 0
	for pos < len(body) {
		for pos < len(body) && isRecordFieldSeparator(body[pos].Type) {
			pos++
		}
		if pos >= len(body) {
			break
		}

		name := body[pos]
		if !isRecordFieldName(name.Type) {
			return nil, false
		}
		pos++
		if pos >= len(body) || body[pos].Type != lexer.ASSIGN_IN {
			return nil, false
		}
		pos++

		start := pos
		depth := 0
		for pos < len(body) {
			tok := body[pos]
			if depth == 0 && isRecordFieldSeparator(tok.Type) {
				break
			}
			depth = nextDepth(depth, tok.Type)
			pos++
		}
		value := trimNoise(body[start:pos])
		if len(value) == 0 {
			return nil, false
		}
		fields = append(fields, recordLiteralField{Name: name, Value: value})
	}
	return fields, len(fields) > 0
}

func isRecordFieldSeparator(typ lexer.TokenType) bool {
	return typ == lexer.SEMICOLON || typ == lexer.COMMA || typ == lexer.COMMENT
}

func isRecordFieldName(typ lexer.TokenType) bool {
	switch typ {
	case lexer.IDENT, lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL,
		lexer.OPERATION, lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS,
		lexer.TARGET, lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return true
	default:
		return false
	}
}
