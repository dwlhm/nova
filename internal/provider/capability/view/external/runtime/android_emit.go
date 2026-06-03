package runtime

import (
	"strings"

	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func emitAndroidArtifacts(ctx external.Context) ([]shared.File, []shared.Diagnostic) {
	if ctx.Android == nil || ctx.AndroidNodes == nil {
		return nil, nil
	}
	app := ctx.Input.Bundle.App
	config := ctx.Android.Config
	sourceRoot := ctx.Android.SourceRoot
	return []shared.File{
		{Path: sourceRoot + "/MainActivity.java", Content: androidcodegen.MainActivityFromContract(app, config, ctx.Input.StyleBundle, ctx.AndroidNodes)},
		{Path: sourceRoot + "/NovaRuntime.java", Content: androidcodegen.JavaRuntime(config)},
		{Path: sourceRoot + "/NovaPrimitiveRegistry.java", Content: androidcodegen.PrimitiveRegistryJava(config)},
		{Path: sourceRoot + "/NovaRendererExtensions.java", Content: androidcodegen.RendererExtensionsJava(ctx.Input.Plan.Renderer.Extensions, config)},
		{Path: "build/android/generated/NovaApp.java", Content: androidcodegen.AppFromContract(ctx.Input.Project.Project.Name, app, config)},
		{Path: "build/android/generated/NovaRoutes.java", Content: androidcodegen.RoutesFromContract(app.View, config)},
		{Path: "build/android/generated/NovaExternalBindings.java", Content: androidcodegen.ExternalBindingsJava(ctx.Input.Plan.ExternalOperations, config)},
	}, nil
}

func AndroidSourceRoot(namespace string) string {
	return "build/android/app/src/main/java/" + strings.ReplaceAll(namespace, ".", "/")
}
