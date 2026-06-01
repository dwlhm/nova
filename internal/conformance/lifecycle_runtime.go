package conformance

import (
	"path"
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/scheduler"
	"github.com/dwlhm/nova/internal/provider/build"
)

type loweredLifecycle struct {
	Owner string
	Phase scheduler.LifecyclePhase
	Event scheduler.SchedulerEvent
	Steps []loweredLifecycleStep
}

type loweredLifecycleStep struct {
	Emit     *loweredLifecycleEmit
	External *loweredLifecycleExternal
}

type loweredLifecycleEmit struct {
	Name string
	Args [][]lexer.Token
}

type loweredLifecycleExternal struct {
	Capability string
	Operation  string
	Input      map[string][]lexer.Token
	OnSuccess  string
	OnFailure  string
}

func buildSchedulerLifecycles(plan build.BuildPlan, sources map[string]parser.File, stateNames map[string]bool) []scheduler.LifecycleHandler {
	lowered := make([]loweredLifecycle, 0)
	for _, module := range plan.Modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		owner := moduleOwner(module.Path)
		aliases := capabilityAliasMap(file)
		for _, lifecycle := range file.Lifecycles {
			steps := make([]loweredLifecycleStep, 0)
			for _, statement := range lifecycle.Statements {
				steps = append(steps, lowerSchedulerLifecycleSteps(statement.Tokens, stateNames, aliases)...)
			}
			if len(steps) == 0 {
				continue
			}
			lowered = append(lowered, loweredLifecycle{
				Owner: owner,
				Phase: scheduler.LifecyclePhase(strings.ToLower(lifecycle.Phase)),
				Event: scheduler.SchedulerEvent(lifecycle.Event),
				Steps: steps,
			})
		}
	}
	handlers := make([]scheduler.LifecycleHandler, 0, len(lowered))
	for _, lifecycle := range lowered {
		owner := scheduler.CapabilityRef(lifecycle.Owner)
		run := buildLifecycleRunner(lifecycle, stateNames, plan)
		switch lifecycle.Phase {
		case scheduler.PhaseMount:
			handlers = append(handlers, scheduler.Mount(owner, run))
		case scheduler.PhaseDispose:
			handlers = append(handlers, scheduler.Dispose(owner, run))
		case scheduler.PhaseBefore:
			handlers = append(handlers, scheduler.Before(owner, lifecycle.Event, run))
		case scheduler.PhaseAfter:
			handlers = append(handlers, scheduler.After(owner, lifecycle.Event, run))
		case scheduler.PhaseError:
			handlers = append(handlers, scheduler.OnError(owner, run))
		}
	}
	return handlers
}

func buildLifecycleRunner(lifecycle loweredLifecycle, stateNames map[string]bool, plan build.BuildPlan) scheduler.LifecycleFunc {
	return func(ctx scheduler.LifecycleContext) (scheduler.LifecycleOutput, error) {
		output := scheduler.LifecycleOutput{}
		paramNames := eventParamNamesFromEnvelope(ctx.Event)
		for _, step := range lifecycle.Steps {
			if step.Emit != nil {
				args := step.Emit.Args
				if len(args) == 0 {
					args = nil
				}
				payload, err := lifecycleEmitPayload(args, stateNames, paramNames, ctx)
				if err != nil {
					return scheduler.LifecycleOutput{}, err
				}
				output.Emit = append(output.Emit, scheduler.Emit(
					scheduler.CapabilityRef(lifecycle.Owner),
					scheduler.SchedulerEvent(step.Emit.Name),
					payload,
				))
				continue
			}
			if step.External != nil {
				input, err := evaluateExternalInput(step.External.Input, stateNames, paramNames, ctx)
				if err != nil {
					return scheduler.LifecycleOutput{}, err
				}
				output.External = append(output.External, scheduler.ExternalOperation(
					scheduler.CapabilityRef(lifecycle.Owner),
					step.External.Capability,
					step.External.Operation,
					input,
					externalOutputType(plan, step.External.Capability, step.External.Operation),
					scheduler.SchedulerEvent(step.External.OnSuccess),
					scheduler.SchedulerEvent(step.External.OnFailure),
				))
			}
		}
		return output, nil
	}
}

func lifecycleEmitPayload(args [][]lexer.Token, stateNames map[string]bool, paramNames map[string]bool, ctx scheduler.LifecycleContext) (scheduler.DataValue, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if len(args) == 1 {
		return evaluateExpression(args[0], stateNames, paramNames, ctx.Snapshot, ctx.Event)
	}
	values := make([]scheduler.DataValue, 0, len(args))
	for _, arg := range args {
		value, err := evaluateExpression(arg, stateNames, paramNames, ctx.Snapshot, ctx.Event)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func evaluateExternalInput(fields map[string][]lexer.Token, stateNames map[string]bool, paramNames map[string]bool, ctx scheduler.LifecycleContext) (map[string]scheduler.DataValue, error) {
	input := make(map[string]scheduler.DataValue, len(fields))
	for name, tokens := range fields {
		value, err := evaluateExpression(tokens, stateNames, paramNames, ctx.Snapshot, ctx.Event)
		if err != nil {
			return nil, err
		}
		input[name] = value
	}
	return input, nil
}

func eventParamNamesFromEnvelope(event scheduler.EventEnvelope) map[string]bool {
	names := make(map[string]bool)
	switch payload := event.Payload.(type) {
	case map[string]scheduler.DataValue:
		for name := range payload {
			names[name] = true
		}
	case map[string]any:
		for name := range payload {
			names[name] = true
		}
	}
	return names
}

func lowerSchedulerLifecycleSteps(tokens []lexer.Token, stateNames map[string]bool, aliases map[string]string) []loweredLifecycleStep {
	tokens = trimLifecycleTokens(tokens)
	if len(tokens) == 0 {
		return nil
	}
	if source, capability, operation, inputs, ok := parseLifecyclePipeStatement(tokens); ok {
		inputFields := make(map[string][]lexer.Token, len(inputs))
		for name, expr := range inputs {
			inputFields[name] = expr
		}
		if len(source) > 0 {
			if _, exists := inputFields["value"]; !exists {
				inputFields["value"] = source
			}
		}
		onSuccess := lifecycleCompletionEventTokens(inputFields, "onSuccess")
		onFailure := lifecycleCompletionEventTokens(inputFields, "onFailure")
		return []loweredLifecycleStep{{
			External: &loweredLifecycleExternal{
				Capability: capability,
				Operation:  operation,
				Input:      inputFields,
				OnSuccess:  onSuccess,
				OnFailure:  onFailure,
			},
		}}
	}
	return lowerSchedulerEmitSteps(tokens)
}

func lowerSchedulerEmitSteps(tokens []lexer.Token) []loweredLifecycleStep {
	steps := make([]loweredLifecycleStep, 0)
	for _, emission := range findSchedulerLifecycleEmits(tokens) {
		args := emission.Args
		if len(args) == 0 && emission.SourceArity == 1 {
			args = [][]lexer.Token{emission.Source}
		}
		steps = append(steps, loweredLifecycleStep{
			Emit: &loweredLifecycleEmit{
				Name: emission.Name,
				Args: args,
			},
		})
	}
	return steps
}

type schedulerLifecycleEmit struct {
	Name        string
	Source      []lexer.Token
	SourceArity int
	Args        [][]lexer.Token
}

func findSchedulerLifecycleEmits(tokens []lexer.Token) []schedulerLifecycleEmit {
	emits := make([]schedulerLifecycleEmit, 0)
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type != lexer.MAP_ARROW || tokens[i+1].Type != lexer.SIGNAL {
			continue
		}
		signalPos := i + 1
		emission := schedulerLifecycleEmit{Name: tokens[signalPos].Literal}
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
	return trimLifecycleTokens(tokens)
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
			args = append(args, trimLifecycleTokens(tokens[argStart:pos]))
			return args
		}
		if tok.Type == lexer.COMMA && depth == 0 {
			args = append(args, trimLifecycleTokens(tokens[argStart:pos]))
			argStart = pos + 1
			continue
		}
		depth = lifecycleExpressionDepth(depth, tok.Type)
	}
	return args
}

func parseLifecyclePipeStatement(tokens []lexer.Token) (source []lexer.Token, capability string, operation string, inputs map[string][]lexer.Token, ok bool) {
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
	source = trimLifecycleTokens(tokens[:pipeIdx])
	rest := trimLifecycleTokens(tokens[pipeIdx+1:])
	if len(rest) < 3 || rest[0].Type != lexer.IDENT || rest[1].Type != lexer.DOT || rest[2].Type != lexer.IDENT {
		return nil, "", "", nil, false
	}
	return source, rest[0].Literal, rest[2].Literal, parseLifecycleNamedInputs(rest[3:]), true
}

func parseLifecycleNamedInputs(tokens []lexer.Token) map[string][]lexer.Token {
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
		inputs[name] = trimLifecycleTokens(tokens[start:pos])
	}
	return inputs
}

func trimLifecycleTokens(tokens []lexer.Token) []lexer.Token {
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

func lifecycleExpressionDepth(depth int, typ lexer.TokenType) int {
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

func capabilityAliasMap(file parser.File) map[string]string {
	out := make(map[string]string, len(file.ExternalImports))
	for _, external := range file.ExternalImports {
		out[external.Name] = external.From
	}
	return out
}

func moduleOwner(modulePath string) string {
	base := path.Base(modulePath)
	return strings.TrimSuffix(base, path.Ext(base))
}

func externalOutputType(plan build.BuildPlan, capabilityAlias string, operation string) string {
	for _, resolved := range plan.ExternalOperations {
		if resolved.CapabilityName == capabilityAlias && resolved.Operation == operation {
			return resolved.Output
		}
	}
	return ""
}

func lifecycleCompletionEventTokens(input map[string][]lexer.Token, name string) string {
	tokens, ok := input[name]
	if !ok {
		return ""
	}
	delete(input, name)
	tokens = trimLifecycleTokens(tokens)
	if len(tokens) == 1 && tokens[0].Type == lexer.SIGNAL {
		return tokens[0].Literal
	}
	return ""
}
