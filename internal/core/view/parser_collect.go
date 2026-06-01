package view

import "github.com/dwlhm/nova/internal/core/lexer"

func (p *viewParser) collectAttributeExpression() []lexer.Token {
	start := p.pos
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 {
			if tok.Type == lexer.GT || tok.Type == lexer.PIPE_END {
				break
			}
			if isAttributeName(tok.Type) && (p.peekN(1).Type == lexer.ASSIGN_IN || p.peekN(1).Type == lexer.MAP_ARROW) {
				break
			}
		}
		depth = nextDepth(depth, tok.Type)
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}

func (p *viewParser) collectEventArg() []lexer.Token {
	start := p.pos
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 && (tok.Type == lexer.COMMA || tok.Type == lexer.RPAREN) {
			break
		}
		depth = nextDepth(depth, tok.Type)
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}

func (p *viewParser) collectText() []lexer.Token {
	start := p.pos
	for !p.done() && !p.at(lexer.LT) && !p.at(lexer.PIPE_END) {
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}
