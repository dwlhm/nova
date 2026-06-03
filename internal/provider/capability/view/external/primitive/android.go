package primitive

import (
	"github.com/dwlhm/nova/internal/core/contract"
	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	irandroid "github.com/dwlhm/nova/internal/provider/capability/view/ir/android"
	androidregistry "github.com/dwlhm/nova/internal/provider/capability/view/registry/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func DefaultLower(_ external.Context, node contract.Node, path []int) (irandroid.Node, []shared.Diagnostic) {
	return irandroid.LowerNode(node, path), nil
}

type NodeRender func(renderer *androidcodegen.Renderer, node contract.Node, parent, indent string, path []int, key, name string) string

func RegisterAndroidLower(registry *androidregistry.LowerRegistry, kind string) {
	registry.Register(kind, DefaultLower)
}

func RegisterAndroidEmit(registry *androidregistry.EmitRegistry, kind string, render NodeRender) {
	registry.RegisterNode(kind, func(renderer *androidcodegen.Renderer, lowered irandroid.Node, parent, indent string, key, name string) string {
		return render(renderer, lowered.Source, parent, indent, lowered.Path, key, name)
	})
}
