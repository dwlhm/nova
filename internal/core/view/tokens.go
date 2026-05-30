package view

import (
	"strings"

	"github.com/dwlhm/nova/internal/lexer"
)

func isAttributeName(typ lexer.TokenType) bool {
	return isName(typ) || typ == lexer.SIGNAL
}

func isName(typ lexer.TokenType) bool {
	switch typ {
	case lexer.IDENT, lexer.TYPE_STRING, lexer.TYPE_NUMBER, lexer.TYPE_BOOLEAN, lexer.TYPE_UNKNOWN,
		lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL, lexer.OPERATION,
		lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS, lexer.TARGET,
		lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return true
	default:
		return false
	}
}

func nextDepth(depth int, typ lexer.TokenType) int {
	switch typ {
	case lexer.LPAREN, lexer.LBRACKET, lexer.LBRACE:
		return depth + 1
	case lexer.RPAREN, lexer.RBRACKET, lexer.RBRACE:
		if depth > 0 {
			return depth - 1
		}
	}
	return depth
}

func trimNoise(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && tokens[start].Type == lexer.COMMENT {
		start++
	}
	end := len(tokens)
	for end > start && tokens[end-1].Type == lexer.COMMENT {
		end--
	}
	return tokens[start:end]
}

func joinTokenLiterals(tokens []lexer.Token) string {
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if tok.Literal != "" {
			parts = append(parts, tok.Literal)
		}
	}
	return strings.Join(parts, "")
}

func cloneTokens(tokens []lexer.Token) []lexer.Token {
	out := make([]lexer.Token, len(tokens))
	copy(out, tokens)
	return out
}

func clonePath(path []int) []int {
	out := make([]int, len(path))
	copy(out, path)
	return out
}
