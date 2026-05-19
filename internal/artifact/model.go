package artifact

import (
	"encoding/json"
	"strings"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
)

type appModel struct {
	States []stateModel `json:"states"`
}

type stateModel struct {
	Owner       string            `json:"owner"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Initial     string            `json:"initial"`
	Transitions []transitionModel `json:"transitions"`
}

type transitionModel struct {
	Event      string   `json:"event"`
	Params     []string `json:"params"`
	Expression string   `json:"expression"`
}

func buildAppModel(modules []build.ModuleRef, sources map[string]parser.File) appModel {
	stateNames := collectModelStateNames(modules, sources)
	states := make([]stateModel, 0)
	for _, module := range modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		for _, contract := range file.ContractStates {
			for _, state := range contract.States {
				states = append(states, stateModel{
					Owner:       contract.Name,
					Name:        state.Name,
					Type:        state.Type.Text,
					Initial:     expressionToJS(state.Initial, stateNames, nil),
					Transitions: transitionModels(state.Transitions, stateNames),
				})
			}
		}
	}
	return appModel{States: states}
}

func transitionModels(transitions []parser.TransitionRule, stateNames map[string]bool) []transitionModel {
	out := make([]transitionModel, 0, len(transitions))
	for _, transition := range transitions {
		params := eventParamNames(transition.Event)
		paramSet := stringSet(params)
		out = append(out, transitionModel{
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

func collectModelStateNames(modules []build.ModuleRef, sources map[string]parser.File) map[string]bool {
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

func expressionToJS(tokens []lexer.Token, stateNames map[string]bool, paramNames map[string]bool) string {
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if tok.Type == lexer.EOF || tok.Type == lexer.COMMENT || tok.Type == lexer.SEMICOLON {
			continue
		}
		parts = append(parts, tokenToJS(tok, stateNames, paramNames))
	}
	return strings.Join(parts, " ")
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
