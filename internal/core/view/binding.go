package view

import "github.com/dwlhm/nova/internal/lexer"

func binding(tokens []lexer.Token) Binding {
	tokens = trimNoise(tokens)
	return Binding{
		Tokens: cloneTokens(tokens),
		Text:   joinTokenLiterals(tokens),
	}
}

func bindingStates(tokens []lexer.Token, stateNames map[string]bool) []string {
	seen := make(map[string]bool)
	states := make([]string, 0)
	for i, tok := range tokens {
		if tok.Type != lexer.IDENT || !stateNames[tok.Literal] {
			continue
		}
		if i > 0 && tokens[i-1].Type == lexer.DOT {
			continue
		}
		if seen[tok.Literal] {
			continue
		}
		seen[tok.Literal] = true
		states = append(states, tok.Literal)
	}
	return states
}
