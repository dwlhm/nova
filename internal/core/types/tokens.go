package types

import "github.com/dwlhm/nova/internal/core/lexer"

func trimNoise(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && tokens[start].Type == lexer.COMMENT {
		start++
	}
	end := len(tokens)
	for end > start && (tokens[end-1].Type == lexer.COMMENT || tokens[end-1].Type == lexer.SEMICOLON) {
		end--
	}
	return tokens[start:end]
}

func trimWrappedParens(tokens []lexer.Token) []lexer.Token {
	for len(tokens) >= 2 && tokens[0].Type == lexer.LPAREN && tokens[len(tokens)-1].Type == lexer.RPAREN && wrapsAll(tokens) {
		tokens = tokens[1 : len(tokens)-1]
	}
	return tokens
}

func wrapsAll(tokens []lexer.Token) bool {
	depth := 0
	for i, tok := range tokens {
		switch tok.Type {
		case lexer.LPAREN:
			depth++
		case lexer.RPAREN:
			depth--
			if depth == 0 && i != len(tokens)-1 {
				return false
			}
		}
	}
	return depth == 0
}

func splitTopLevel(tokens []lexer.Token, delimiter lexer.TokenType) [][]lexer.Token {
	segments := make([][]lexer.Token, 0)
	start := 0
	depth := 0
	for i, tok := range tokens {
		if depth == 0 && tok.Type == delimiter {
			segments = append(segments, trimNoise(tokens[start:i]))
			start = i + 1
			continue
		}
		depth = nextDepth(depth, tok.Type)
	}
	if start == 0 {
		return nil
	}
	segments = append(segments, trimNoise(tokens[start:]))
	return segments
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
