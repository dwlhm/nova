package runtime

import (
	webcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/web"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func emitWebArtifacts(ctx external.Context) ([]shared.File, []shared.Diagnostic) {
	app := ctx.Input.Bundle.App
	return []shared.File{
		{Path: "build/web/assets/nova-runtime.js", Content: webcodegen.WebRuntime()},
		{Path: "build/web/app.bundle.js", Content: webcodegen.WebBundle(app)},
	}, nil
}
