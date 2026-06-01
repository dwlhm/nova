package validator

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/lexer"
	novatypes "github.com/dwlhm/nova/internal/core/types"
)

type eventEmission struct {
	Name  string
	Arity int
	Args  [][]lexer.Token
	Token lexer.Token
}

func canReadState(tokens []lexer.Token, pos int, stateNames map[string]bool, paramNames map[string]bool) bool {
	if pos < 0 || pos >= len(tokens) {
		return false
	}
	tok := tokens[pos]
	if tok.Type != lexer.IDENT || !stateNames[tok.Literal] || paramNames[tok.Literal] {
		return false
	}
	if pos > 0 && tokens[pos-1].Type == lexer.DOT {
		return false
	}
	if pos+1 < len(tokens) && tokens[pos+1].Type == lexer.ASSIGN_IN {
		return false
	}
	return true
}

func findStateWrites(tokens []lexer.Token, stateNames map[string]bool) []lexer.Token {
	writes := make([]lexer.Token, 0)
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type == lexer.IDENT && stateNames[tokens[i].Literal] && tokens[i+1].Type == lexer.ASSIGN_IN {
			writes = append(writes, tokens[i])
		}
	}
	return writes
}

func findExternalCalls(tokens []lexer.Token, externalOps map[externalOperation]bool) []externalOperation {
	calls := make([]externalOperation, 0)
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].Type != lexer.IDENT || tokens[i+1].Type != lexer.DOT || tokens[i+2].Type != lexer.IDENT {
			continue
		}
		call := externalOperation{
			Capability: tokens[i].Literal,
			Operation:  tokens[i+2].Literal,
		}
		if externalOps[call] {
			calls = append(calls, call)
		}
	}
	return calls
}

func callToken(tokens []lexer.Token, call externalOperation) lexer.Token {
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].Type == lexer.IDENT &&
			tokens[i].Literal == call.Capability &&
			tokens[i+1].Type == lexer.DOT &&
			tokens[i+2].Type == lexer.IDENT &&
			tokens[i+2].Literal == call.Operation {
			return tokens[i]
		}
	}
	return lexer.Token{Type: lexer.EOF}
}

func findEventEmits(tokens []lexer.Token, sourcePayload bool) []eventEmission {
	emits := make([]eventEmission, 0)
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type != lexer.MAP_ARROW || tokens[i+1].Type != lexer.SIGNAL {
			continue
		}

		signalPos := i + 1
		arity := 0
		args := [][]lexer.Token(nil)
		if signalPos+1 < len(tokens) && tokens[signalPos+1].Type == lexer.LPAREN {
			args = eventArgs(tokens, signalPos+2)
			arity = len(args)
		} else if sourcePayload {
			source := sourceTokens(tokens[:i])
			arity = sourceArity(source)
			if arity == 1 {
				args = [][]lexer.Token{source}
			}
		} else if i > 0 && tokens[i-1].Type == lexer.SIGNAL {
			arity = unknownEventArity
		}

		emits = append(emits, eventEmission{
			Name:  tokens[signalPos].Literal,
			Arity: arity,
			Args:  args,
			Token: tokens[signalPos],
		})
	}
	return emits
}

func validateEventEmits(context string, emissions []eventEmission, events eventTable, env novatypes.Environment, scope novatypes.Scope) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for _, emission := range emissions {
		signature, ok := events.Signature(emission.Name)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("%s emits undeclared scheduler event %s", context, emission.Name),
				Token:   emission.Token,
			})
			continue
		}
		if signature.Arity == unknownEventArity || emission.Arity == unknownEventArity || signature.Arity == emission.Arity {
			diagnostics = append(diagnostics, validateEventPayloadTypes(context, emission, signature, env, scope)...)
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Message: fmt.Sprintf("%s emits scheduler event %s with %s, want %s", context, emission.Name, arityLabel(emission.Arity), arityLabel(signature.Arity)),
			Token:   emission.Token,
		})
	}
	return diagnostics
}

func validateEventPayloadTypes(context string, emission eventEmission, signature eventSignature, env novatypes.Environment, scope novatypes.Scope) []Diagnostic {
	if len(signature.Payload) == 0 || len(emission.Args) == 0 || len(signature.Payload) != len(emission.Args) {
		return nil
	}
	diagnostics := make([]Diagnostic, 0)
	for i, arg := range emission.Args {
		actual, ok := env.InferExpression(scope, arg)
		if !ok {
			continue
		}
		if env.Assignable(actual, signature.Payload[i]) {
			continue
		}
		token := emission.Token
		if len(arg) > 0 {
			token = arg[0]
		}
		diagnostics = append(diagnostics, Diagnostic{
			Message: fmt.Sprintf("%s emits scheduler event %s payload %d as %s, want %s", context, emission.Name, i+1, novatypes.Format(actual), novatypes.Format(signature.Payload[i])),
			Token:   token,
		})
	}
	return diagnostics
}

func sourceArity(tokens []lexer.Token) int {
	tokens = sourceTokens(tokens)
	if len(tokens) == 0 {
		return 0
	}
	if len(tokens) == 1 && tokens[0].Type == lexer.VOID {
		return 0
	}
	return 1
}

func sourceTokens(tokens []lexer.Token) []lexer.Token {
	tokens = trimComments(tokens)
	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

func eventArgs(tokens []lexer.Token, start int) [][]lexer.Token {
	args := make([][]lexer.Token, 0)
	pos := start
	for pos < len(tokens) && tokens[pos].Type == lexer.COMMENT {
		pos++
	}
	if pos >= len(tokens) || tokens[pos].Type == lexer.RPAREN {
		return nil
	}

	argStart := pos
	depth := 0
	for ; pos < len(tokens); pos++ {
		tok := tokens[pos]
		if tok.Type == lexer.RPAREN && depth == 0 {
			args = append(args, trimComments(tokens[argStart:pos]))
			return args
		}
		if tok.Type == lexer.COMMA && depth == 0 {
			args = append(args, trimComments(tokens[argStart:pos]))
			argStart = pos + 1
			continue
		}
		depth = eventArgDepth(depth, tok.Type)
	}
	return args
}

func eventArgDepth(depth int, typ lexer.TokenType) int {
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

func trimComments(tokens []lexer.Token) []lexer.Token {
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

func arityLabel(arity int) string {
	switch arity {
	case unknownEventArity:
		return "forwarded payload"
	case 0:
		return "void"
	case 1:
		return "1 payload value"
	default:
		return fmt.Sprintf("%d payload values", arity)
	}
}
