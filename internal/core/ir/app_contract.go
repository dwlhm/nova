package ir

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/capability"
	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/core/view"
)

func materializeAppContract(input LowerInput, sources map[string]parser.File, model loweredAppModel, viewIR view.IR, manifests []capability.Manifest) contract.App {
	stateNames := stateNamesFromModel(model)
	lifecycles := buildContractLifecycles(input.Modules, sources, collectStateNamesFromMap(sources, input.Modules))
	externalSummaries := externalOperationSummaries(input.Externals)
	return contract.App{
		V:                  contract.Version,
		Target:             input.Profile,
		Entry:              input.Entry,
		Permissions:        permissionStrings(input.Permissions),
		Model:              contractModel(model),
		View:               contractView(viewIR, stateNames),
		Effects:            contractEffects(externalSummaries),
		Events:             contractEvents(modulePathsFromRefs(input.Modules), manifests, lifecycles),
		Lifecycles:         lifecycles,
		ExternalOperations: contractExternalOperations(input.Externals),
	}
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

func contractView(ir view.IR, stateNames map[string]bool) contract.View {
	bindings := make([]contract.BindingMeta, 0, len(ir.Metadata.Bindings))
	for _, ref := range ir.Metadata.Bindings {
		bindings = append(bindings, contract.BindingMeta{
			At:     cloneIntPath(ref.NodePath),
			Prop:   ref.Prop,
			States: append([]string(nil), ref.States...),
		})
	}
	return contract.View{
		Nodes:    contractNodes(ir.Nodes, stateNames),
		Bindings: bindings,
	}
}

func contractNodes(nodes []view.Node, stateNames map[string]bool) []contract.Node {
	out := make([]contract.Node, len(nodes))
	for i, node := range nodes {
		out[i] = contractNode(node, stateNames)
	}
	return out
}

func contractNode(node view.Node, stateNames map[string]bool) contract.Node {
	props := make(map[string]string, len(node.Props))
	for name, binding := range node.Props {
		props[name] = bindingExpr(binding, stateNames)
	}
	events := make(map[string]contract.Event, len(node.Events))
	for slot, route := range node.Events {
		args := make([]string, 0, len(route.Args))
		for _, arg := range route.Args {
			if implicitValueBinding(arg) {
				args = append(args, "$value")
				continue
			}
			args = append(args, bindingExpr(arg, stateNames))
		}
		events[slot] = contract.Event{
			Name: string(route.Event),
			Args: args,
		}
	}
	out := contract.Node{
		Kind:     node.Kind,
		Props:    props,
		Events:   events,
		Children: contractNodes(node.Children, stateNames),
	}
	if node.Key != nil {
		out.Key = bindingExpr(*node.Key, stateNames)
	}
	return out
}

func bindingExpr(binding view.Binding, stateNames map[string]bool) string {
	return expressionToJS(binding.Tokens, stateNames, nil)
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
