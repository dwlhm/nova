package column

import (
	"github.com/dwlhm/nova/internal/core/contract"
	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/primitive"
)

var Module = primitive.Android(
	external.PrimitiveMetadata("column", []string{"column"}, 37),
	"column",
	func(renderer *androidcodegen.Renderer, node contract.Node, parent, indent string, path []int, key, name string) string {
		return renderer.RenderColumnNode(node, parent, indent, path, key, name)
	},
)
