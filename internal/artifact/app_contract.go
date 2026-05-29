package artifact

import (
	"fmt"
	"sort"

	"github.com/dwlhm/nova/internal/contract"
	"github.com/dwlhm/nova/internal/security"
	"github.com/dwlhm/nova/internal/view"
)

func buildAppContract(bundle irBundle, permissions []security.Permission) contract.App {
	stateNames := stateNamesFromModel(bundle.Model)
	return contract.App{
		V:           contract.Version,
		Target:      bundle.Target,
		Entry:       bundle.Entry,
		Permissions: permissionStrings(permissions),
		Model:       contract.Model{States: cloneStateModels(bundle.Model.States)},
		View:        contractView(bundle.ViewIR, stateNames),
		Effects:     contractEffects(bundle.ExternalOperations),
	}
}

func buildManifest(input GenerateInput, bundle irBundle, metadata contractBuildVersions) contract.BuildManifest {
	return contract.BuildManifest{
		ContractVersion:    contract.Version,
		LanguageVersion:    metadata.LanguageVersion,
		SchedulerVersion:   metadata.SchedulerVersion,
		RuntimeVersion:     metadata.RuntimeVersion,
		Target:             bundle.Target,
		Entry:              bundle.Entry,
		Modules:            bundle.Modules,
		TemplateFile:       input.Plan.Template.SourceFile,
		TemplateIndex:      input.Plan.Template.Index,
		ExternalOperations: externalOperationNames(input.Plan.ExternalOperations),
		Permissions:        permissionStrings(input.Plan.Permissions),
	}
}

type contractBuildVersions struct {
	LanguageVersion  string
	SchedulerVersion string
	RuntimeVersion   string
}

func cloneStateModels(states []stateModel) []contract.State {
	out := make([]contract.State, len(states))
	for i, state := range states {
		transitions := make([]contract.Transition, len(state.Transitions))
		for j, transition := range state.Transitions {
			transitions[j] = contract.Transition{
				Event:      transition.Event,
				Params:     append([]string(nil), transition.Params...),
				Expression: transition.Expression,
			}
		}
		out[i] = contract.State{
			Owner:       state.Owner,
			Name:        state.Name,
			Type:        state.Type,
			Initial:     state.Initial,
			Transitions: transitions,
		}
	}
	return out
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

func stateNamesFromModel(model appModel) map[string]bool {
	names := make(map[string]bool)
	for _, state := range model.States {
		names[state.Name] = true
	}
	return names
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
