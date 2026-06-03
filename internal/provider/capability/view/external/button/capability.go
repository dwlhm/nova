package button

import (
	"github.com/dwlhm/nova/internal/core/contract"
	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/primitive"
)

var Module = primitive.Android(
	external.PrimitiveMetadata("button", []string{"button"}, 31),
	"button",
	func(renderer *androidcodegen.Renderer, node contract.Node, parent, indent string, path []int, key, name string) string {
		return renderer.RenderButtonNode(node, parent, indent, path, key, name)
	},
)
