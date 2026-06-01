package ir

import (
	"encoding/json"
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/view"
)

type loweredAppModel struct {
	States []loweredStateModel `json:"states"`
}

type loweredStateModel struct {
	Owner       string                   `json:"owner"`
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Initial     string                   `json:"initial"`
	Transitions []loweredTransitionModel `json:"transitions"`
}

type loweredTransitionModel struct {
	Event      string   `json:"event"`
	Params     []string `json:"params"`
	Expression string   `json:"expression"`
}

func buildLoweredAppModel(modules []ModuleRef, sources map[string]parser.File) loweredAppModel {
	stateNames := collectModelStateNames(modules, sources)
	states := make([]loweredStateModel, 0)
	for _, module := range modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		for _, contract := range file.ContractStates {
			for _, state := range contract.States {
				states = append(states, loweredStateModel{
					Owner:       contract.Name,
					Name:        state.Name,
					Type:        state.Type.Text,
					Initial:     expressionToJS(state.Initial, stateNames, nil),
					Transitions: transitionModels(state.Transitions, stateNames),
				})
			}
		}
	}
	return loweredAppModel{States: states}
}

func transitionModels(transitions []parser.TransitionRule, stateNames map[string]bool) []loweredTransitionModel {
	out := make([]loweredTransitionModel, 0, len(transitions))
	for _, transition := range transitions {
		params := eventParamNames(transition.Event)
		paramSet := stringSet(params)
		out = append(out, loweredTransitionModel{
			Event:      transition.Event.Name,
			Params:     params,
			Expression: expressionToJS(transition.Expr, stateNames, paramSet),
		})
	}
	return out
}

func eventParamNames(pattern parser.EventPattern) []string {
	out := make([]string, 0, len(pattern.Params))
	for _, param := range pattern.Params {
		out = append(out, param.Name)
	}
	return out
}

func collectModelStateNames(modules []ModuleRef, sources map[string]parser.File) map[string]bool {
	names := make(map[string]bool)
	for _, module := range modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		for _, contract := range file.ContractStates {
			for _, state := range contract.States {
				names[state.Name] = true
			}
		}
	}
	return names
}

func collectStateNames(sources []SourceFile) map[string]bool {
	names := make(map[string]bool)
	for _, source := range sources {
		for _, contract := range source.File.ContractStates {
			for _, state := range contract.States {
				names[state.Name] = true
			}
		}
		for _, decl := range source.File.Imports {
			if decl.Kind != parser.ImportState {
				continue
			}
			for _, item := range decl.Items {
				name := item.Name
				if item.Alias != "" {
					name = item.Alias
				}
				names[name] = true
			}
		}
	}
	return names
}

func expressionToJS(tokens []lexer.Token, stateNames map[string]bool, paramNames map[string]bool) string {
	tokens = trimExpressionTokens(tokens)
	if record, ok := recordLiteralToJS(tokens, stateNames, paramNames); ok {
		return record
	}

	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		parts = append(parts, tokenToJS(tok, stateNames, paramNames))
	}
	return strings.Join(parts, " ")
}

func recordLiteralToJS(tokens []lexer.Token, stateNames map[string]bool, paramNames map[string]bool) (string, bool) {
	fields, ok := parseRecordExpressionFields(tokens)
	if !ok {
		return "", false
	}

	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, field.Name.Literal+": "+expressionToJS(field.Value, stateNames, paramNames))
	}
	return "({ " + strings.Join(parts, ", ") + " })", true
}

type recordExpressionField struct {
	Name  lexer.Token
	Value []lexer.Token
}

func parseRecordExpressionFields(tokens []lexer.Token) ([]recordExpressionField, bool) {
	if len(tokens) < 2 || tokens[0].Type != lexer.LBRACE || tokens[len(tokens)-1].Type != lexer.RBRACE {
		return nil, false
	}

	body := tokens[1 : len(tokens)-1]
	fields := make([]recordExpressionField, 0)
	pos := 0
	for pos < len(body) {
		for pos < len(body) && isRecordExpressionSeparator(body[pos].Type) {
			pos++
		}
		if pos >= len(body) {
			break
		}

		name := body[pos]
		if !isRecordExpressionName(name.Type) {
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
			if depth == 0 && isRecordExpressionSeparator(tok.Type) {
				break
			}
			depth = expressionDepth(depth, tok.Type)
			pos++
		}
		value := trimExpressionTokens(body[start:pos])
		if len(value) == 0 {
			return nil, false
		}
		fields = append(fields, recordExpressionField{Name: name, Value: value})
	}
	return fields, len(fields) > 0
}

func trimExpressionTokens(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && (tokens[start].Type == lexer.EOF || tokens[start].Type == lexer.COMMENT || tokens[start].Type == lexer.SEMICOLON) {
		start++
	}
	end := len(tokens)
	for end > start && (tokens[end-1].Type == lexer.EOF || tokens[end-1].Type == lexer.COMMENT || tokens[end-1].Type == lexer.SEMICOLON) {
		end--
	}
	return tokens[start:end]
}

func isRecordExpressionSeparator(typ lexer.TokenType) bool {
	return typ == lexer.SEMICOLON || typ == lexer.COMMA || typ == lexer.COMMENT
}

func isRecordExpressionName(typ lexer.TokenType) bool {
	switch typ {
	case lexer.IDENT, lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL,
		lexer.OPERATION, lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS,
		lexer.TARGET, lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return true
	default:
		return false
	}
}

func expressionDepth(depth int, typ lexer.TokenType) int {
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

func tokenToJS(tok lexer.Token, stateNames map[string]bool, paramNames map[string]bool) string {
	switch tok.Type {
	case lexer.STRING:
		return quoteJS(tok.Literal)
	case lexer.IDENT:
		switch {
		case stateNames[tok.Literal]:
			return "state." + tok.Literal
		case paramNames[tok.Literal]:
			return "payload." + tok.Literal
		default:
			return tok.Literal
		}
	case lexer.NUMBER:
		return tok.Literal
	case lexer.TRUE:
		return "true"
	case lexer.FALSE:
		return "false"
	case lexer.NULL:
		return "null"
	case lexer.VOID:
		return "undefined"
	default:
		return tok.Literal
	}
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func quoteJS(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func staticBindingString(binding view.Binding) (string, bool) {
	tokens := trimExpressionTokens(binding.Tokens)
	if len(tokens) == 1 && tokens[0].Type == lexer.STRING {
		return tokens[0].Literal, true
	}
	return "", false
}

func stateNamesFromModel(model loweredAppModel) map[string]bool {
	names := make(map[string]bool)
	for _, state := range model.States {
		names[state.Name] = true
	}
	return names
}
