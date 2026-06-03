package androidcodegen

import "github.com/dwlhm/nova/internal/core/contract"

// BuildNodeFunc renders one view node into Java source for MainActivity.
type BuildNodeFunc func(renderer *Renderer, node contract.Node, parent, indent string, path []int, key, name string) string

// NodeRegistry maps view kinds to codegen handlers without switch-based dispatch.
type NodeRegistry struct {
	byKind map[string]BuildNodeFunc
}

func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{byKind: make(map[string]BuildNodeFunc)}
}

func (registry *NodeRegistry) Register(kind string, render BuildNodeFunc) {
	if kind == "" || render == nil {
		return
	}
	registry.byKind[kind] = render
}

func (registry *NodeRegistry) lookup(kind string) (BuildNodeFunc, bool) {
	if registry == nil {
		return nil, false
	}
	render, ok := registry.byKind[kind]
	return render, ok
}

// MergeInto copies all registered handlers into dst.
func (registry *NodeRegistry) MergeInto(dst *NodeRegistry) {
	if registry == nil || dst == nil {
		return
	}
	for kind, render := range registry.byKind {
		dst.Register(kind, render)
	}
}
