package ir

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/capability"
	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/expr"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/core/view"
)

func materializeAppContract(input LowerInput, sources map[string]parser.File, model loweredAppModel, viewIR view.IR, manifests []capability.Manifest) (contract.App, []Diagnostic) {
	stateNames := stateNamesFromModel(model)
	registry := buildExprRegistry(sources)
	lifecycles, lifecycleDiagnostics := buildContractLifecycles(input.Modules, sources, registry, collectStateNamesFromMap(sources, input.Modules))
	viewContract, viewDiagnostics := contractView(viewIR, registry, stateNames)
	diagnostics := append(lifecycleDiagnostics, viewDiagnostics...)
	externalSummaries := externalOperationSummaries(input.Externals)
	return contract.App{
		V:                  contract.Version,
		Target:             input.Profile,
		Entry:              input.Entry,
		Permissions:        permissionStrings(input.Permissions),
		Model:              contractModel(model),
		View:               viewContract,
		Effects:            contractEffects(externalSummaries),
		Events:             mergeNavigationPlatformEmitters(mergeAppLifecycleEvents(contractEvents(modulePathsFromRefs(input.Modules), manifests, lifecycles)), contractModel(model)),
		Lifecycles:         lifecycles,
		ExternalOperations: contractExternalOperations(input.Externals),
		Persistence:        buildHydrationManifest(lifecycles),
		AppLifecycle:       contract.StandardAppLifecycleEvents(),
	}, diagnostics
}

func mergeAppLifecycleEvents(events []contract.EventContract) []contract.EventContract {
	byName := make(map[string]contract.EventContract, len(events)+len(contract.StandardAppLifecycleEvents()))
	for _, event := range events {
		byName[event.Name] = event
	}
	for _, name := range contract.StandardAppLifecycleEvents() {
		existing, ok := byName[name]
		if !ok {
			byName[name] = contract.EventContract{Name: name, Emitters: []string{"@nova/app"}}
			continue
		}
		emitters := make(map[string]bool, len(existing.Emitters)+1)
		for _, emitter := range existing.Emitters {
			emitters[emitter] = true
		}
		emitters["@nova/app"] = true
		merged := make([]string, 0, len(emitters))
		for emitter := range emitters {
			merged = append(merged, emitter)
		}
		sort.Strings(merged)
		existing.Emitters = merged
		byName[name] = existing
	}
	out := make([]contract.EventContract, 0, len(byName))
	for _, event := range byName {
		out = append(out, event)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func mergeNavigationPlatformEmitters(events []contract.EventContract, model contract.Model) []contract.EventContract {
	if !hasRouteStateContract(model) {
		return events
	}
	platformEmitters := contract.StandardNavigationPlatformEmitters()
	navigationEvents := map[string]bool{
		"@route_changed":      true,
		"@navigate":           true,
		"@navigation_failed":  true,
	}
	byName := make(map[string]contract.EventContract, len(events))
	for _, event := range events {
		byName[event.Name] = event
	}
	for name := range navigationEvents {
		existing, ok := byName[name]
		if !ok {
			existing = contract.EventContract{Name: name}
		}
		emitters := make(map[string]bool, len(existing.Emitters)+len(platformEmitters))
		for _, emitter := range existing.Emitters {
			emitters[emitter] = true
		}
		for _, emitter := range platformEmitters {
			emitters[emitter] = true
		}
		merged := make([]string, 0, len(emitters))
		for emitter := range emitters {
			merged = append(merged, emitter)
		}
		sort.Strings(merged)
		existing.Emitters = merged
		byName[name] = existing
	}
	out := make([]contract.EventContract, 0, len(byName))
	for _, event := range byName {
		out = append(out, event)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func hasRouteStateContract(model contract.Model) bool {
	for _, state := range model.States {
		if state.Name == "route" {
			return true
		}
	}
	return false
}

func contractEvents(modules []string, manifests []capability.Manifest, lifecycles []contract.Lifecycle) []contract.EventContract {
	manifestByModule := make(map[string]capability.Manifest, len(manifests))
	for _, manifest := range manifests {
		manifestByModule[string(manifest.Ref)] = manifest
	}
	emittersByEvent := make(map[string]map[string]bool)
	for _, modulePath := range modules {
		manifest, ok := manifestByModule[modulePath]
		if !ok {
			continue
		}
		for _, event := range manifest.Events {
			name := string(event.Ref.Name)
			if emittersByEvent[name] == nil {
				emittersByEvent[name] = make(map[string]bool)
			}
			emittersByEvent[name][string(event.Ref.Module)] = true
		}
	}
	for _, lifecycle := range lifecycles {
		for _, step := range lifecycle.Steps {
			if step.Emit != nil {
				addLifecycleEventEmitter(emittersByEvent, step.Emit.Name, lifecycle.Owner)
			}
			if step.External == nil {
				continue
			}
			addLifecycleEventEmitter(emittersByEvent, step.External.OnSuccess, lifecycle.Owner)
			addLifecycleEventEmitter(emittersByEvent, step.External.OnFailure, lifecycle.Owner)
		}
	}
	events := make([]contract.EventContract, 0, len(emittersByEvent))
	for name, emitters := range emittersByEvent {
		moduleEmitters := make([]string, 0, len(emitters))
		for module := range emitters {
			moduleEmitters = append(moduleEmitters, module)
		}
		sort.Strings(moduleEmitters)
		events = append(events, contract.EventContract{
			Name:     name,
			Emitters: moduleEmitters,
		})
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Name < events[j].Name })
	return events
}

func addLifecycleEventEmitter(emittersByEvent map[string]map[string]bool, eventName string, owner string) {
	if strings.TrimSpace(eventName) == "" || strings.TrimSpace(owner) == "" {
		return
	}
	if emittersByEvent[eventName] == nil {
		emittersByEvent[eventName] = make(map[string]bool)
	}
	emittersByEvent[eventName][owner] = true
}

func contractModel(model loweredAppModel) contract.Model {
	states := make([]contract.State, len(model.States))
	for i, state := range model.States {
		transitions := make([]contract.Transition, len(state.Transitions))
		for j, transition := range state.Transitions {
			transitions[j] = contract.Transition{
				Event:      transition.Event,
				Params:     append([]string(nil), transition.Params...),
				Expression: transition.Expression,
			}
		}
		if state.Name == "route" && !hasTransitionEvent(transitions, "@navigate") {
			transitions = append(transitions, contract.Transition{
				Event:      "@navigate",
				Params:     []string{"action"},
				Expression: contract.NavigationApplyExpression,
			})
		}
		states[i] = contract.State{
			Owner:       state.Owner,
			Name:        state.Name,
			Type:        state.Type,
			Initial:     state.Initial,
			Transitions: transitions,
		}
	}
	return contract.Model{States: states}
}

func hasTransitionEvent(transitions []contract.Transition, event string) bool {
	for _, transition := range transitions {
		if transition.Event == event {
			return true
		}
	}
	return false
}

func contractView(ir view.IR, registry *expr.Registry, stateNames map[string]bool) (contract.View, []Diagnostic) {
	bindings := make([]contract.BindingMeta, 0, len(ir.Metadata.Bindings))
	for _, ref := range ir.Metadata.Bindings {
		bindings = append(bindings, contract.BindingMeta{
			At:     cloneIntPath(ref.NodePath),
			Prop:   ref.Prop,
			States: append([]string(nil), ref.States...),
		})
	}
	nodes, diagnostics := contractNodes(ir.Nodes, registry, stateNames)
	return contract.View{
		Nodes:    nodes,
		Bindings: bindings,
	}, diagnostics
}

func contractNodes(nodes []view.Node, registry *expr.Registry, stateNames map[string]bool) ([]contract.Node, []Diagnostic) {
	out := make([]contract.Node, len(nodes))
	var diagnostics []Diagnostic
	for i, node := range nodes {
		contractNode, nodeDiagnostics := contractNode(node, registry, stateNames)
		out[i] = contractNode
		diagnostics = append(diagnostics, nodeDiagnostics...)
	}
	return out, diagnostics
}

func contractNode(node view.Node, registry *expr.Registry, stateNames map[string]bool) (contract.Node, []Diagnostic) {
	var diagnostics []Diagnostic
	props := make(map[string]string, len(node.Props))
	for name, binding := range node.Props {
		exprValue, diags := bindingExpr(binding, registry, stateNames)
		diagnostics = append(diagnostics, diags...)
		props[name] = exprValue
	}
	events := make(map[string]contract.Event, len(node.Events))
	for slot, route := range node.Events {
		args := make([]string, 0, len(route.Args))
		for _, arg := range route.Args {
			if implicitValueBinding(arg) {
				args = append(args, "$value")
				continue
			}
			exprValue, diags := bindingExpr(arg, registry, stateNames)
			diagnostics = append(diagnostics, diags...)
			args = append(args, exprValue)
		}
		events[slot] = contract.Event{
			Name: string(route.Event),
			Args: args,
		}
	}
	children, childDiagnostics := contractNodes(node.Children, registry, stateNames)
	diagnostics = append(diagnostics, childDiagnostics...)
	out := contract.Node{
		Kind:     node.Kind,
		Props:    props,
		Events:   events,
		Children: children,
	}
	if node.Key != nil {
		keyExpr, diags := bindingExpr(*node.Key, registry, stateNames)
		diagnostics = append(diagnostics, diags...)
		out.Key = keyExpr
	}
	return out, diagnostics
}

func bindingExpr(binding view.Binding, registry *expr.Registry, stateNames map[string]bool) (string, []Diagnostic) {
	return lowerExpressionJS(binding.Tokens, registry, stateNames, nil)
}

func implicitValueBinding(binding view.Binding) bool {
	tokens := trimExpressionTokens(binding.Tokens)
	return len(tokens) == 1 && tokens[0].Literal == "value"
}

func contractEffects(summaries []externalOperationSummary) []contract.Effect {
	if len(summaries) == 0 {
		return nil
	}
	effects := make([]contract.Effect, 0, len(summaries))
	seen := make(map[string]bool)
	for _, summary := range summaries {
		id := fmt.Sprintf("%s#%s", summary.CapabilitySource, summary.Operation)
		if seen[id] {
			continue
		}
		seen[id] = true
		perms := permissionStrings(summary.Permissions)
		effects = append(effects, contract.Effect{
			ID:          id,
			Permissions: perms,
		})
	}
	sort.Slice(effects, func(i, j int) bool { return effects[i].ID < effects[j].ID })
	return effects
}

func collectStateNamesFromMap(sources map[string]parser.File, modules []ModuleRef) map[string]bool {
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
		for _, decl := range file.Imports {
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

func cloneIntPath(values []int) []int {
	out := make([]int, len(values))
	copy(out, values)
	return out
}

func modulePathsFromRefs(modules []ModuleRef) []string {
	out := make([]string, 0, len(modules))
	for _, module := range modules {
		out = append(out, module.Path)
	}
	return out
}

type externalOperationSummary struct {
	CapabilitySource string
	Operation        string
	Permissions      []security.Permission
}

func externalOperationSummaries(operations []ResolvedExternal) []externalOperationSummary {
	summaries := make([]externalOperationSummary, 0, len(operations))
	for _, operation := range operations {
		summaries = append(summaries, externalOperationSummary{
			CapabilitySource: operation.CapabilitySource,
			Operation:        operation.Operation,
			Permissions:      append([]security.Permission(nil), operation.Permissions...),
		})
	}
	return summaries
}
