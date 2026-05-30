package view

import (
	"fmt"

	"github.com/dwlhm/nova/internal/lexer"
)

func (p *viewParser) match(typ lexer.TokenType) bool {
	if !p.at(typ) {
		return false
	}
	p.advance()
	return true
}

func (p *viewParser) expect(typ lexer.TokenType) lexer.Token {
	if p.at(typ) {
		return p.advance()
	}
	tok := p.peek()
	p.errorf(tok, "expected %s", typ)
	return tok
}

func (p *viewParser) expectName(label string) lexer.Token {
	if isName(p.peek().Type) {
		return p.advance()
	}
	tok := p.peek()
	p.errorf(tok, "expected %s", label)
	return tok
}

func (p *viewParser) at(typ lexer.TokenType) bool {
	return p.peek().Type == typ
}

func (p *viewParser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *viewParser) peek() lexer.Token {
	if p.done() {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *viewParser) peekN(offset int) lexer.Token {
	pos := p.pos + offset
	if pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[pos]
}

func (p *viewParser) advance() lexer.Token {
	tok := p.peek()
	if !p.done() {
		p.pos++
	}
	return tok
}

func (p *viewParser) errorf(tok lexer.Token, message string, args ...any) {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	p.diagnostics = append(p.diagnostics, Diagnostic{Message: message, Token: tok})
}
