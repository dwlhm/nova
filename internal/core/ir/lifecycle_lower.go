package ir

import (
	"path"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/expr"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/security"
)

func buildContractLifecycles(modules []ModuleRef, sources map[string]parser.File, registry *expr.Registry, stateNames map[string]bool) ([]contract.Lifecycle, []Diagnostic) {
	lifecycles := make([]contract.Lifecycle, 0)
	var diagnostics []Diagnostic
	for _, module := range modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		owner := moduleOwner(module.Path)
		aliases := capabilityAliasMap(file)
		for _, lifecycle := range file.Lifecycles {
			steps := make([]contract.LifecycleStep, 0)
			for _, statement := range lifecycle.Statements {
				loweredSteps, stepDiagnostics := lowerLifecycleSteps(statement.Tokens, registry, stateNames, aliases)
				steps = append(steps, loweredSteps...)
				diagnostics = append(diagnostics, stepDiagnostics...)
			}
			if len(steps) == 0 {
				continue
			}
			lifecycles = append(lifecycles, contract.Lifecycle{
				Owner: owner,
				Phase: strings.ToLower(lifecycle.Phase),
				Event: lifecycle.Event,
				Steps: steps,
			})
		}
	}
	sort.Slice(lifecycles, func(i, j int) bool {
		if lifecycles[i].Owner != lifecycles[j].Owner {
			return lifecycles[i].Owner < lifecycles[j].Owner
		}
		if lifecycles[i].Phase != lifecycles[j].Phase {
			return lifecycles[i].Phase < lifecycles[j].Phase
		}
		return lifecycles[i].Event < lifecycles[j].Event
	})
	return lifecycles, diagnostics
}

func lowerLifecycleSteps(tokens []lexer.Token, registry *expr.Registry, stateNames map[string]bool, aliases map[string]string) ([]contract.LifecycleStep, []Diagnostic) {
	tokens = trimExpressionTokens(tokens)
	if len(tokens) == 0 {
		return nil, nil
	}
	if source, capability, operation, inputs, ok := parsePipeStatement(tokens); ok {
		var diagnostics []Diagnostic
		onSuccess := lifecycleCompletionEventFromTokens(inputs, "onSuccess")
		onFailure := lifecycleCompletionEventFromTokens(inputs, "onFailure")
		inputExprs := make(map[string]string, len(inputs))
		for name, exprTokens := range inputs {
			exprValue, diags := lowerExpressionJS(exprTokens, registry, stateNames, nil)
			diagnostics = append(diagnostics, diags...)
			inputExprs[name] = exprValue
		}
		if len(source) > 0 {
			if _, exists := inputExprs["value"]; !exists {
				exprValue, diags := lowerExpressionJS(source, registry, stateNames, nil)
				diagnostics = append(diagnostics, diags...)
				inputExprs["value"] = exprValue
			}
		}
		capabilitySource := aliases[capability]
		if capabilitySource == "" {
			capabilitySource = capability
		}
		return []contract.LifecycleStep{{
			External: &contract.LifecycleExternal{
				EffectID:  effectID(capabilitySource, operation),
				Input:     inputExprs,
				OnSuccess: onSuccess,
				OnFailure: onFailure,
			},
		}}, diagnostics
	}
	return lowerEmitSteps(tokens, registry, stateNames)
}

func lowerEmitSteps(tokens []lexer.Token, registry *expr.Registry, stateNames map[string]bool) ([]contract.LifecycleStep, []Diagnostic) {
	steps := make([]contract.LifecycleStep, 0)
	var diagnostics []Diagnostic
	for _, emission := range findLifecycleEmits(tokens) {
		args := emission.Args
		if len(args) == 0 && emission.SourceArity == 1 {
			args = [][]lexer.Token{emission.Source}
		}
		argExprs := make([]string, 0, len(args))
		for _, arg := range args {
			exprValue, diags := lowerExpressionJS(arg, registry, stateNames, nil)
			diagnostics = append(diagnostics, diags...)
			argExprs = append(argExprs, exprValue)
		}
		steps = append(steps, contract.LifecycleStep{
			Emit: &contract.LifecycleEmit{
				Name: emission.Name,
				Args: argExprs,
			},
		})
	}
	return steps, diagnostics
}

type lifecycleEmit struct {
	Name        string
	Source      []lexer.Token
	SourceArity int
	Args        [][]lexer.Token
}

func findLifecycleEmits(tokens []lexer.Token) []lifecycleEmit {
	emits := make([]lifecycleEmit, 0)
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type != lexer.MAP_ARROW || tokens[i+1].Type != lexer.SIGNAL {
			continue
		}
		signalPos := i + 1
		emission := lifecycleEmit{Name: tokens[signalPos].Literal}
		if signalPos+1 < len(tokens) && tokens[signalPos+1].Type == lexer.LPAREN {
			emission.Args = lifecycleEventArgs(tokens, signalPos+2)
		} else {
			emission.Source = lifecycleSourceTokens(tokens[:i])
			emission.SourceArity = lifecycleSourceArity(emission.Source)
		}
		emits = append(emits, emission)
	}
	return emits
}

func lifecycleSourceArity(tokens []lexer.Token) int {
	tokens = lifecycleSourceTokens(tokens)
	if len(tokens) == 0 {
		return 0
	}
	if len(tokens) == 1 && tokens[0].Type == lexer.VOID {
		return 0
	}
	return 1
}

func lifecycleSourceTokens(tokens []lexer.Token) []lexer.Token {
	tokens = trimExpressionTokens(tokens)
	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

func lifecycleEventArgs(tokens []lexer.Token, start int) [][]lexer.Token {
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
			args = append(args, trimExpressionTokens(tokens[argStart:pos]))
			return args
		}
		if tok.Type == lexer.COMMA && depth == 0 {
			args = append(args, trimExpressionTokens(tokens[argStart:pos]))
			argStart = pos + 1
			continue
		}
		depth = expressionDepth(depth, tok.Type)
	}
	return args
}

func parsePipeStatement(tokens []lexer.Token) (source []lexer.Token, capability string, operation string, inputs map[string][]lexer.Token, ok bool) {
	pipeIdx := -1
	for i, tok := range tokens {
		if tok.Type == lexer.PIPE_FWD {
			pipeIdx = i
			break
		}
	}
	if pipeIdx < 0 {
		return nil, "", "", nil, false
	}
	source = trimExpressionTokens(tokens[:pipeIdx])
	rest := trimExpressionTokens(tokens[pipeIdx+1:])
	if len(rest) < 3 || rest[0].Type != lexer.IDENT || rest[1].Type != lexer.DOT || rest[2].Type != lexer.IDENT {
		return nil, "", "", nil, false
	}
	capability = rest[0].Literal
	operation = rest[2].Literal
	return source, capability, operation, parseNamedInputs(rest[3:]), true
}

func parseNamedInputs(tokens []lexer.Token) map[string][]lexer.Token {
	inputs := make(map[string][]lexer.Token)
	pos := 0
	for pos < len(tokens) {
		if tokens[pos].Type != lexer.IDENT || pos+1 >= len(tokens) || tokens[pos+1].Type != lexer.ASSIGN_IN {
			break
		}
		name := tokens[pos].Literal
		pos += 2
		start := pos
		for pos < len(tokens) {
			if tokens[pos].Type == lexer.IDENT && pos+1 < len(tokens) && tokens[pos+1].Type == lexer.ASSIGN_IN {
				break
			}
			pos++
		}
		inputs[name] = trimExpressionTokens(tokens[start:pos])
	}
	return inputs
}

func effectID(source string, operation string) string {
	return source + "#" + operation
}

func lifecycleCompletionEventFromTokens(inputs map[string][]lexer.Token, name string) string {
	tokens, ok := inputs[name]
	if !ok {
		return ""
	}
	delete(inputs, name)
	tokens = trimExpressionTokens(tokens)
	if len(tokens) != 1 {
		return ""
	}
	switch tokens[0].Type {
	case lexer.SIGNAL, lexer.STRING:
		return tokens[0].Literal
	default:
		return ""
	}
}

func capabilityAliasMap(file parser.File) map[string]string {
	out := make(map[string]string, len(file.ExternalImports))
	for _, external := range file.ExternalImports {
		out[external.Name] = external.From
	}
	return out
}

func contractExternalOperations(operations []ResolvedExternal) []contract.ExternalOperation {
	if len(operations) == 0 {
		return nil
	}
	out := make([]contract.ExternalOperation, 0, len(operations))
	seen := make(map[string]bool)
	for _, operation := range operations {
		id := operation.CapabilitySource + "#" + operation.Operation
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, contract.ExternalOperation{
			ID:             id,
			Capability:     operation.CapabilityName,
			Source:         operation.CapabilitySource,
			Operation:      operation.Operation,
			Output:         operation.Output,
			Permissions:    permissionStrings(operation.Permissions),
			Implementation: operation.Implementation,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func moduleOwner(modulePath string) string {
	base := path.Base(modulePath)
	return strings.TrimSuffix(base, path.Ext(base))
}

func permissionStrings(permissions []security.Permission) []string {
	if len(permissions) == 0 {
		return nil
	}
	out := make([]string, len(permissions))
	for i, permission := range permissions {
		out[i] = string(permission)
	}
	sort.Strings(out)
	return out
}
