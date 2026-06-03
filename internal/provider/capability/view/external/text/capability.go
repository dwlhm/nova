package text

import (
	"github.com/dwlhm/nova/internal/core/contract"
	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/primitive"
)

var Module = primitive.AndroidKinds(
	external.PrimitiveMetadata("text", []string{"text", "#text"}, 30),
	[]string{"text", "#text"},
	renderText,
)

func renderText(renderer *androidcodegen.Renderer, node contract.Node, parent, indent string, path []int, key, name string) string {
	return renderer.RenderTextNode(node, parent, indent, path, key, name)
}
