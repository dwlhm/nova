package types

import (
	"fmt"

	"github.com/dwlhm/nova/internal/lexer"
)

func (p *typeParser) match(typ lexer.TokenType) bool {
	if p.done() || p.peek().Type != typ {
		return false
	}
	p.pos++
	return true
}

func (p *typeParser) expect(typ lexer.TokenType) {
	if p.match(typ) {
		return
	}
	p.errorf(p.peek(), "expected %s", typ)
}

func (p *typeParser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *typeParser) peek() lexer.Token {
	if p.done() {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *typeParser) advance() lexer.Token {
	tok := p.peek()
	if !p.done() {
		p.pos++
	}
	return tok
}

func (p *typeParser) errorf(tok lexer.Token, format string, args ...any) {
	p.diagnostics = append(p.diagnostics, Diagnostic{
		Message: fmt.Sprintf(format, args...),
		Token:   tok,
	})
}

func tokenizeTypeText(text string) []lexer.Token {
	tokens := lexer.Tokenize(text)
	return trimEOF(tokens)
}

func trimEOF(tokens []lexer.Token) []lexer.Token {
	out := tokens
	for len(out) > 0 && out[len(out)-1].Type == lexer.EOF {
		out = out[:len(out)-1]
	}
	return out
}
