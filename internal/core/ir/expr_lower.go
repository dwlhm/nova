package ir

import (
	"encoding/json"
	"fmt"

	"github.com/dwlhm/nova/internal/core/expr"
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

func buildLoweredAppModel(modules []ModuleRef, sources map[string]parser.File) (loweredAppModel, []Diagnostic) {
	stateNames := collectModelStateNames(modules, sources)
	registry := buildExprRegistry(sources)
	states := make([]loweredStateModel, 0)
	var diagnostics []Diagnostic
	for _, module := range modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		for _, contract := range file.ContractStates {
			for _, state := range contract.States {
				initial, initialDiags := lowerExpressionJS(state.Initial, registry, stateNames, nil)
				diagnostics = append(diagnostics, initialDiags...)
				transitions, transitionDiags := transitionModels(state.Transitions, registry, stateNames)
				diagnostics = append(diagnostics, transitionDiags...)
				states = append(states, loweredStateModel{
					Owner:       contract.Name,
					Name:        state.Name,
					Type:        state.Type.Text,
					Initial:     initial,
					Transitions: transitions,
				})
			}
		}
	}
	return loweredAppModel{States: states}, diagnostics
}

func transitionModels(transitions []parser.TransitionRule, registry *expr.Registry, stateNames map[string]bool) ([]loweredTransitionModel, []Diagnostic) {
	out := make([]loweredTransitionModel, 0, len(transitions))
	var diagnostics []Diagnostic
	for _, transition := range transitions {
		params := eventParamNames(transition.Event)
		paramSet := stringSet(params)
		expression, diags := lowerExpressionJS(transition.Expr, registry, stateNames, paramSet)
		diagnostics = append(diagnostics, diags...)
		out = append(out, loweredTransitionModel{
			Event:      transition.Event.Name,
			Params:     params,
			Expression: expression,
		})
	}
	return out, diagnostics
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

func buildExprRegistry(sources map[string]parser.File) *expr.Registry {
	files := make([]parser.File, 0, len(sources))
	for _, file := range sources {
		files = append(files, file)
	}
	return expr.BuildRegistry(files)
}

func lowerExpressionJS(tokens []lexer.Token, registry *expr.Registry, stateNames map[string]bool, paramNames map[string]bool) (string, []Diagnostic) {
	tokens = trimExpressionTokens(tokens)
	if len(tokens) == 0 {
		return `""`, nil
	}
	lowered, err := expr.LowerJS(tokens, registry, stateNames, paramNames)
	if err != nil {
		return "", []Diagnostic{exprDiagnostic(err)}
	}
	return lowered, nil
}

func exprDiagnostic(err error) Diagnostic {
	return Diagnostic{
		Code:    "NVA-EXPR-001",
		Message: fmt.Sprintf("expression lowering failed: %v", err),
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
