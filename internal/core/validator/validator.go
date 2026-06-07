package validator

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	novatypes "github.com/dwlhm/nova/internal/core/types"
	"github.com/dwlhm/nova/internal/core/view"
)

type Diagnostic struct {
	Message string
	Token   lexer.Token
}

type externalOperation struct {
	Capability string
	Operation  string
}

func Validate(file parser.File) []Diagnostic {
	return ValidateWithImports(file, nil)
}

func ValidateWithImports(file parser.File, imports []parser.File) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	diagnostics = append(diagnostics, validateLocalSymbols(file)...)
	symbols, symbolDiagnostics := buildSymbolsWithImports(file, imports)
	diagnostics = append(diagnostics, symbolDiagnostics...)
	stateScope := novatypes.Scope(symbols.StateTypes)

	for _, contract := range file.ContractStates {
		for _, state := range contract.States {
			diagnostics = append(diagnostics, validateExpressionAssignable(
				fmt.Sprintf("state %s initial value", state.Name),
				symbols.TypeEnv,
				stateScope,
				state.Initial,
				symbols.StateTypes[state.Name],
			)...)
			for _, call := range findExternalCalls(state.Initial, symbols.ExternalOperations) {
				diagnostics = append(diagnostics, Diagnostic{
					Message: fmt.Sprintf("state contract cannot call external operation %s.%s", call.Capability, call.Operation),
					Token:   callToken(state.Initial, call),
				})
			}
			for _, tok := range findStateWrites(state.Initial, symbols.StateNames) {
				diagnostics = append(diagnostics, Diagnostic{
					Message: fmt.Sprintf("state initial value cannot write state %s", tok.Literal),
					Token:   tok,
				})
			}
			for _, emit := range findEventEmits(state.Initial, false) {
				diagnostics = append(diagnostics, Diagnostic{
					Message: fmt.Sprintf("state initial value cannot emit scheduler event %s", emit.Name),
					Token:   emit.Token,
				})
			}
			for _, transition := range state.Transitions {
				transitionScope := mergeScopes(stateScope, eventParamScope(symbols.TypeEnv, transition.Event))
				diagnostics = append(diagnostics, validateExpressionAssignable(
					fmt.Sprintf("state transition for %s", state.Name),
					symbols.TypeEnv,
					transitionScope,
					transition.Expr,
					symbols.StateTypes[state.Name],
				)...)
				for _, call := range findExternalCalls(transition.Expr, symbols.ExternalOperations) {
					diagnostics = append(diagnostics, Diagnostic{
						Message: fmt.Sprintf("state transition cannot call external operation %s.%s", call.Capability, call.Operation),
						Token:   callToken(transition.Expr, call),
					})
				}
				for _, tok := range findStateWrites(transition.Expr, symbols.StateNames) {
					diagnostics = append(diagnostics, Diagnostic{
						Message: fmt.Sprintf("state transition cannot write state %s", tok.Literal),
						Token:   tok,
					})
				}
				for _, emit := range findEventEmits(transition.Expr, true) {
					diagnostics = append(diagnostics, Diagnostic{
						Message: fmt.Sprintf("state transition cannot emit scheduler event %s", emit.Name),
						Token:   emit.Token,
					})
				}
			}
		}
	}

	for _, contract := range file.ContractTypes {
		for _, call := range findExternalCalls(contract.Tokens, symbols.ExternalOperations) {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("type contract cannot call external operation %s.%s", call.Capability, call.Operation),
				Token:   callToken(contract.Tokens, call),
			})
		}
	}

	for _, contract := range file.ContractCapabilities {
		for _, call := range findExternalCalls(contract.Tokens, symbols.ExternalOperations) {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("capability contract cannot call external operation %s.%s", call.Capability, call.Operation),
				Token:   callToken(contract.Tokens, call),
			})
		}
	}

	for _, fn := range file.Funcs {
		paramNames := collectParamNames(fn.Params)
		fnScope := fieldScope(symbols.TypeEnv, fn.Params)
		returnType, returnTypeDiagnostics := novatypes.ParseRef(symbols.TypeEnv, fn.Return)
		diagnostics = append(diagnostics, convertTypeDiagnostics(returnTypeDiagnostics)...)
		diagnostics = append(diagnostics, validateExpressionAssignable(
			fmt.Sprintf("func %s return", fn.Name),
			symbols.TypeEnv,
			fnScope,
			fn.Body,
			returnType,
		)...)
		for _, call := range findExternalCalls(fn.Body, symbols.ExternalOperations) {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("func cannot call external operation %s.%s", call.Capability, call.Operation),
				Token:   callToken(fn.Body, call),
			})
		}
		for i, tok := range fn.Body {
			if canReadState(fn.Body, i, symbols.StateNames, paramNames) {
				diagnostics = append(diagnostics, Diagnostic{
					Message: fmt.Sprintf("func cannot read state %s", tok.Literal),
					Token:   tok,
				})
			}
			if tok.Type == lexer.SIGNAL {
				diagnostics = append(diagnostics, Diagnostic{
					Message: fmt.Sprintf("func cannot emit scheduler event %s", tok.Literal),
					Token:   tok,
				})
			}
		}
		for _, tok := range findStateWrites(fn.Body, symbols.StateNames) {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("func cannot write state %s", tok.Literal),
				Token:   tok,
			})
		}
	}

	for _, template := range file.Templates {
		for _, call := range findExternalCalls(template.Tokens, symbols.ExternalOperations) {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("template cannot call external operation %s.%s", call.Capability, call.Operation),
				Token:   callToken(template.Tokens, call),
			})
		}
		diagnostics = append(diagnostics, validateTemplateView(template, symbols)...)
		diagnostics = append(diagnostics, validateEventEmits("template", findEventEmits(template.Tokens, false), symbols.Events, symbols.TypeEnv, stateScope)...)
	}

	for _, lifecycle := range file.Lifecycles {
		if lifecycle.Event != "" && !symbols.Events.Has(lifecycle.Event) {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("lifecycle listens to undeclared scheduler event %s", lifecycle.Event),
				Token:   lifecycleEventToken(lifecycle),
			})
		}
		for _, statement := range lifecycle.Statements {
			for _, tok := range findStateWrites(statement.Tokens, symbols.StateNames) {
				diagnostics = append(diagnostics, Diagnostic{
					Message: fmt.Sprintf("lifecycle cannot write state %s", tok.Literal),
					Token:   tok,
				})
			}
			diagnostics = append(diagnostics, validateEventEmits("lifecycle", findEventEmits(statement.Tokens, true), symbols.Events, symbols.TypeEnv, stateScope)...)
		}
	}

	return diagnostics
}

func validateExpressionAssignable(context string, env novatypes.Environment, scope novatypes.Scope, expr []lexer.Token, target novatypes.Type) []Diagnostic {
	if target.Kind == "" || target.Kind == novatypes.KindInvalid {
		return nil
	}
	actual, ok := env.InferExpression(scope, expr)
	if !ok {
		return nil
	}
	if env.Assignable(actual, target) {
		return nil
	}
	return []Diagnostic{{
		Message: fmt.Sprintf("%s has type %s, want %s", context, novatypes.Format(actual), novatypes.Format(target)),
		Token:   firstToken(expr),
	}}
}

func validateTemplateView(template parser.TemplateDecl, symbols symbolTable) []Diagnostic {
	ir, viewDiagnostics := view.Project(template, symbols.StateNames)
	diagnostics := make([]Diagnostic, 0, len(viewDiagnostics))
	for _, diagnostic := range viewDiagnostics {
		diagnostics = append(diagnostics, Diagnostic{Message: diagnostic.Message, Token: diagnostic.Token})
	}
	diagnostics = append(diagnostics, validateTextNodes(ir.Nodes, symbols.TypeEnv, novatypes.Scope(symbols.StateTypes))...)
	return diagnostics
}

func validateTextNodes(nodes []view.Node, env novatypes.Environment, scope novatypes.Scope) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for _, node := range nodes {
		if node.Kind == "text" {
			if value, ok := node.Props["value"]; ok {
				diagnostics = append(diagnostics, validateExpressionAssignable(
					"text value binding",
					env,
					scope,
					value.Tokens,
					novatypes.Type{Kind: novatypes.KindPrimitive, Name: "string"},
				)...)
			}
		}
		diagnostics = append(diagnostics, validateTextNodes(node.Children, env, scope)...)
	}
	return diagnostics
}

func firstToken(tokens []lexer.Token) lexer.Token {
	for _, tok := range tokens {
		if tok.Type != lexer.COMMENT {
			return tok
		}
	}
	return lexer.Token{Type: lexer.EOF}
}

func collectParamNames(params []parser.FieldDecl) map[string]bool {
	names := make(map[string]bool)
	for _, param := range params {
		names[param.Name] = true
	}
	return names
}
