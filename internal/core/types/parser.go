package types

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func ParseRef(env Environment, ref parser.TypeRef) (Type, []Diagnostic) {
	tokens := ref.Tokens
	if len(tokens) == 0 && strings.TrimSpace(ref.Text) != "" {
		tokens = tokenizeTypeText(ref.Text)
	}
	return ParseTokens(env, tokens)
}

func ParseText(text string) (Type, []Diagnostic) {
	return ParseTokens(EmptyEnvironment(), tokenizeTypeText(text))
}

func ParseTokens(env Environment, tokens []lexer.Token) (Type, []Diagnostic) {
	p := typeParser{tokens: trimEOF(tokens), env: env}
	typ := p.parseUnion()
	if typ.Kind == "" {
		typ = Type{Kind: KindInvalid}
	}
	for !p.done() {
		if p.peek().Type != lexer.COMMENT {
			p.errorf(p.peek(), "unexpected token in type expression %s", p.peek().Literal)
		}
		p.advance()
	}
	return typ, p.diagnostics
}
