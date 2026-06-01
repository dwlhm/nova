package validator

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/capability"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	novatypes "github.com/dwlhm/nova/internal/core/types"
)

const unknownEventArity = -1

type symbolTable struct {
	StateNames         map[string]bool
	StateTypes         map[string]novatypes.Type
	ExternalOperations map[externalOperation]bool
	Events             eventTable
	TypeEnv            novatypes.Environment
}

type eventTable map[string]eventSignature

type eventSignature struct {
	Name    string
	Arity   int
	Payload []novatypes.Type
	Token   lexer.Token
}

func (events eventTable) Has(name string) bool {
	_, ok := events[name]
	return ok
}

func (events eventTable) Signature(name string) (eventSignature, bool) {
	signature, ok := events[name]
	return signature, ok
}

func buildSymbols(file parser.File) (symbolTable, []Diagnostic) {
	typeEnv, typeDiagnostics := novatypes.BuildEnvironment(file)
	symbols := symbolTable{
		StateNames:         collectStateNames(file),
		StateTypes:         make(map[string]novatypes.Type),
		ExternalOperations: collectExternalOperations(file),
		Events:             make(eventTable),
		TypeEnv:            typeEnv,
	}
	diagnostics := convertTypeDiagnostics(typeDiagnostics)
	symbols.StateTypes = collectStateTypes(file, typeEnv, &diagnostics)

	for _, decl := range file.Imports {
		if decl.Kind != parser.ImportEvent {
			continue
		}
		for _, item := range decl.Items {
			name := capability.EffectiveImportName(item)
			diagnostics = append(diagnostics, symbols.Events.add(eventSignature{
				Name:  name,
				Arity: unknownEventArity,
				Token: importItemToken(item, lexer.SIGNAL),
			})...)
		}
	}

	for _, contract := range file.ContractStates {
		for _, state := range contract.States {
			for _, transition := range state.Transitions {
				payload, payloadDiagnostics := eventPatternPayload(typeEnv, transition.Event)
				diagnostics = append(diagnostics, payloadDiagnostics...)
				diagnostics = append(diagnostics, symbols.Events.add(eventSignature{
					Name:    transition.Event.Name,
					Arity:   len(transition.Event.Params),
					Payload: payload,
					Token:   eventPatternToken(transition.Event),
				})...)
			}
		}
	}

	for _, contract := range file.ContractCapabilities {
		for _, emit := range contract.Emits {
			arity, payload, arityDiagnostics := capabilityEmitSignature(typeEnv, emit)
			diagnostics = append(diagnostics, arityDiagnostics...)
			diagnostics = append(diagnostics, symbols.Events.add(eventSignature{
				Name:    emit.Event,
				Arity:   arity,
				Payload: payload,
				Token:   emitToken(emit),
			})...)
		}
	}

	return symbols, diagnostics
}

func collectStateTypes(file parser.File, env novatypes.Environment, diagnostics *[]Diagnostic) map[string]novatypes.Type {
	stateTypes := make(map[string]novatypes.Type)
	for _, decl := range file.Imports {
		if decl.Kind != parser.ImportState {
			continue
		}
		for _, item := range decl.Items {
			stateTypes[capability.EffectiveImportName(item)] = novatypes.Type{Kind: novatypes.KindUnknown}
		}
	}
	for _, contract := range file.ContractStates {
		for _, state := range contract.States {
			typ, typeDiagnostics := novatypes.ParseRef(env, state.Type)
			*diagnostics = append(*diagnostics, convertTypeDiagnostics(typeDiagnostics)...)
			stateTypes[state.Name] = typ
		}
	}
	return stateTypes
}

func (events eventTable) add(signature eventSignature) []Diagnostic {
	if signature.Name == "" {
		return nil
	}

	existing, ok := events[signature.Name]
	if !ok {
		events[signature.Name] = signature
		return nil
	}
	if existing.Arity == unknownEventArity && signature.Arity != unknownEventArity {
		events[signature.Name] = signature
		return nil
	}
	if signature.Arity == unknownEventArity || existing.Arity == unknownEventArity {
		return nil
	}
	if existing.Arity == signature.Arity {
		return nil
	}

	return []Diagnostic{{
		Message: fmt.Sprintf("scheduler event %s has conflicting payload arity", signature.Name),
		Token:   signature.Token,
	}}
}

func collectStateNames(file parser.File) map[string]bool {
	names := make(map[string]bool)
	for _, decl := range file.Imports {
		if decl.Kind != parser.ImportState {
			continue
		}
		for _, item := range decl.Items {
			names[capability.EffectiveImportName(item)] = true
		}
	}
	for _, contract := range file.ContractStates {
		for _, state := range contract.States {
			names[state.Name] = true
		}
	}
	return names
}

func collectExternalOperations(file parser.File) map[externalOperation]bool {
	ops := make(map[externalOperation]bool)
	for _, external := range file.ExternalImports {
		for _, operation := range external.Operations {
			ops[externalOperation{
				Capability: external.Name,
				Operation:  operation.Name,
			}] = true
		}
	}
	return ops
}

func capabilityEmitSignature(env novatypes.Environment, emit parser.EmitDecl) (int, []novatypes.Type, []Diagnostic) {
	switch emit.Type.Text {
	case "void":
		return 0, nil, nil
	case "":
		return unknownEventArity, nil, []Diagnostic{{
			Message: fmt.Sprintf("emitted event %s must declare payload type or void", emit.Event),
			Token:   emitToken(emit),
		}}
	default:
		typ, diagnostics := novatypes.ParseRef(env, emit.Type)
		return 1, []novatypes.Type{typ}, convertTypeDiagnostics(diagnostics)
	}
}

func eventPatternPayload(env novatypes.Environment, pattern parser.EventPattern) ([]novatypes.Type, []Diagnostic) {
	payload := make([]novatypes.Type, 0, len(pattern.Params))
	diagnostics := make([]Diagnostic, 0)
	for _, param := range pattern.Params {
		typ, typeDiagnostics := novatypes.ParseRef(env, param.Type)
		diagnostics = append(diagnostics, convertTypeDiagnostics(typeDiagnostics)...)
		payload = append(payload, typ)
	}
	return payload, diagnostics
}

func eventParamScope(env novatypes.Environment, pattern parser.EventPattern) novatypes.Scope {
	scope := make(novatypes.Scope)
	for _, param := range pattern.Params {
		typ, diagnostics := novatypes.ParseRef(env, param.Type)
		if len(diagnostics) == 0 {
			scope[param.Name] = typ
		}
	}
	return scope
}

func fieldScope(env novatypes.Environment, fields []parser.FieldDecl) novatypes.Scope {
	scope := make(novatypes.Scope)
	for _, field := range fields {
		typ, diagnostics := novatypes.ParseRef(env, field.Type)
		if len(diagnostics) == 0 {
			scope[field.Name] = typ
		}
	}
	return scope
}

func mergeScopes(scopes ...novatypes.Scope) novatypes.Scope {
	merged := make(novatypes.Scope)
	for _, scope := range scopes {
		for name, typ := range scope {
			merged[name] = typ
		}
	}
	return merged
}

func convertTypeDiagnostics(diagnostics []novatypes.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, Diagnostic{
			Message: diagnostic.Message,
			Token:   diagnostic.Token,
		})
	}
	return out
}

func eventPatternToken(pattern parser.EventPattern) lexer.Token {
	if len(pattern.Tokens) == 0 {
		return lexer.Token{Type: lexer.SIGNAL, Literal: pattern.Name}
	}
	return pattern.Tokens[0]
}

func emitToken(emit parser.EmitDecl) lexer.Token {
	if len(emit.Tokens) == 0 {
		return lexer.Token{Type: lexer.SIGNAL, Literal: emit.Event}
	}
	return emit.Tokens[0]
}

func lifecycleEventToken(lifecycle parser.LifecycleDecl) lexer.Token {
	for _, tok := range lifecycle.Tokens {
		if tok.Type == lexer.SIGNAL && tok.Literal == lifecycle.Event {
			return tok
		}
	}
	return lexer.Token{Type: lexer.SIGNAL, Literal: lifecycle.Event}
}

type localSymbol struct {
	Name  string
	Kind  string
	Token lexer.Token
}

func validateLocalSymbols(file parser.File) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	seen := make(map[string]localSymbol)

	for _, symbol := range collectLocalSymbols(file) {
		if symbol.Name == "" {
			continue
		}
		existing, ok := seen[symbol.Name]
		if !ok {
			seen[symbol.Name] = symbol
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Message: fmt.Sprintf("local symbol %s from %s conflicts with previous %s", symbol.Name, symbol.Kind, existing.Kind),
			Token:   symbol.Token,
		})
	}

	return diagnostics
}

func collectLocalSymbols(file parser.File) []localSymbol {
	symbols := make([]localSymbol, 0)

	for _, decl := range file.Imports {
		for _, item := range decl.Items {
			symbols = append(symbols, localSymbol{
				Name:  capability.EffectiveImportName(item),
				Kind:  importSymbolKind(decl.Kind),
				Token: importItemToken(item, defaultImportTokenType(decl.Kind)),
			})
		}
	}

	for _, external := range file.ExternalImports {
		symbols = append(symbols, localSymbol{
			Name:  external.Name,
			Kind:  "external import",
			Token: lexer.Token{Type: lexer.IDENT, Literal: external.Name},
		})
		for _, operation := range external.Operations {
			name := external.Name + "." + operation.Name
			symbols = append(symbols, localSymbol{
				Name:  name,
				Kind:  "external operation",
				Token: externalOperationToken(external.Name, operation),
			})
		}
	}

	for _, contract := range file.ContractTypes {
		symbols = append(symbols, localSymbol{
			Name:  contract.Name,
			Kind:  "contract type",
			Token: lexer.Token{Type: lexer.IDENT, Literal: contract.Name},
		})
	}
	for _, contract := range file.ContractStates {
		for _, state := range contract.States {
			symbols = append(symbols, localSymbol{
				Name:  state.Name,
				Kind:  "contract state",
				Token: lexer.Token{Type: lexer.IDENT, Literal: state.Name},
			})
		}
	}
	for _, contract := range file.ContractCapabilities {
		symbols = append(symbols, localSymbol{
			Name:  contract.Name,
			Kind:  "contract capability",
			Token: lexer.Token{Type: lexer.IDENT, Literal: contract.Name},
		})
	}
	for _, fn := range file.Funcs {
		symbols = append(symbols, localSymbol{
			Name:  fn.Name,
			Kind:  "func",
			Token: lexer.Token{Type: lexer.IDENT, Literal: fn.Name},
		})
	}
	for _, template := range file.Templates {
		if template.Target == "" {
			continue
		}
		symbols = append(symbols, localSymbol{
			Name:  template.Target,
			Kind:  "template target",
			Token: lexer.Token{Type: lexer.IDENT, Literal: template.Target},
		})
	}

	return symbols
}

func importSymbolKind(kind parser.ImportKind) string {
	switch kind {
	case parser.ImportState:
		return "state import"
	case parser.ImportEvent:
		return "event import"
	default:
		return "capability import"
	}
}

func defaultImportTokenType(kind parser.ImportKind) lexer.TokenType {
	if kind == parser.ImportEvent {
		return lexer.SIGNAL
	}
	return lexer.IDENT
}

func importItemToken(item parser.ImportItem, fallback lexer.TokenType) lexer.Token {
	name := capability.EffectiveImportName(item)
	for _, tok := range item.Tokens {
		if tok.Literal == name {
			return tok
		}
	}
	return lexer.Token{Type: fallback, Literal: name}
}

func externalOperationToken(capabilityName string, operation parser.ExternalOperationDecl) lexer.Token {
	for _, tok := range operation.Tokens {
		if tok.Type == lexer.IDENT && tok.Literal == operation.Name {
			return tok
		}
	}
	return lexer.Token{Type: lexer.IDENT, Literal: capabilityName + "." + operation.Name}
}
