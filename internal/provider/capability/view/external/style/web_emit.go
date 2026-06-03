package style

import (
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func emitWebArtifacts(ctx external.Context) ([]shared.File, []shared.Diagnostic) {
	bundle := webStyleBundle(ctx.Input.StyleBundle)
	out := make([]shared.File, 0, len(bundle.Files)+1)
	out = append(out, shared.File{Path: "build/web/style-manifest.json", Content: shared.MustJSON(bundle.Manifest)})
	out = append(out, bundle.Files...)
	return out, nil
}
