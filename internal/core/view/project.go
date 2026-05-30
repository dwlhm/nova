package view

import "github.com/dwlhm/nova/internal/parser"

func Project(template parser.TemplateDecl, stateNames map[string]bool) (IR, []Diagnostic) {
	p := viewParser{tokens: template.Tokens}
	nodes := p.parseNodes(false)
	ir := IR{
		Target: template.Target,
		Nodes:  nodes,
	}
	ir.Metadata = BuildMetadata(ir.Nodes, stateNames)
	return ir, p.diagnostics
}

func BuildMetadata(nodes []Node, stateNames map[string]bool) DependencyMetadata {
	metadata := DependencyMetadata{}
	for i, node := range nodes {
		metadata = appendNodeMetadata(metadata, node, []int{i}, stateNames)
	}
	return metadata
}

func appendNodeMetadata(metadata DependencyMetadata, node Node, path []int, stateNames map[string]bool) DependencyMetadata {
	if node.Key != nil {
		metadata = appendBindingMetadata(metadata, path, "key", *node.Key, stateNames)
	}
	if node.Kind == "page" {
		if pagePath, ok := node.Props["path"]; ok {
			metadata.Pages = append(metadata.Pages, PageRef{
				NodePath: clonePath(path),
				Path:     pagePath,
			})
		}
	}
	for prop, binding := range node.Props {
		metadata = appendBindingMetadata(metadata, path, prop, binding, stateNames)
	}
	for slot, route := range node.Events {
		metadata.EventRoutes = append(metadata.EventRoutes, EventRouteRef{
			NodePath: clonePath(path),
			Slot:     slot,
			Event:    route.Event,
			Arity:    len(route.Args),
		})
		for i, arg := range route.Args {
			metadata = appendBindingMetadata(metadata, path, slot+"#arg"+string(rune('0'+i)), arg, stateNames)
		}
	}
	for i, child := range node.Children {
		childPath := append(clonePath(path), i)
		metadata = appendNodeMetadata(metadata, child, childPath, stateNames)
	}
	return metadata
}

func appendBindingMetadata(metadata DependencyMetadata, path []int, prop string, binding Binding, stateNames map[string]bool) DependencyMetadata {
	states := bindingStates(binding.Tokens, stateNames)
	if len(states) == 0 {
		return metadata
	}
	metadata.Bindings = append(metadata.Bindings, BindingRef{
		NodePath: clonePath(path),
		Prop:     prop,
		States:   states,
	})
	return metadata
}
