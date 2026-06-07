package runtime

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/expr"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/provider/build"
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
	registry := expr.BuildRegistry(parserFilesFromSources(ctx.Input.Sources))
	novaExpr, err := expr.EmitJavaNovaExpr(config.Namespace, registry)
	if err != nil {
		return nil, []shared.Diagnostic{{
			Code:    "NVA-EXPR-002",
			Message: err.Error(),
		}}
	}
	files := []shared.File{
		{Path: sourceRoot + "/MainActivity.java", Content: androidcodegen.MainActivityFromContract(app, config, ctx.Input.StyleBundle, ctx.AndroidNodes)},
		{Path: sourceRoot + "/NovaRuntime.java", Content: androidcodegen.JavaRuntime(config)},
		{Path: sourceRoot + "/NovaHydration.java", Content: androidcodegen.JavaHydration(config)},
		{Path: sourceRoot + "/NovaStyle.java", Content: androidcodegen.JavaStyle(config)},
		{Path: sourceRoot + "/NovaExternal.java", Content: androidcodegen.JavaExternal(config)},
		{Path: sourceRoot + "/NovaExpr.java", Content: novaExpr},
		{Path: sourceRoot + "/NovaRenderer.java", Content: androidcodegen.JavaRenderer(config)},
		{Path: sourceRoot + "/NovaRendererExtensions.java", Content: androidcodegen.RendererExtensionsJava(ctx.Input.Plan.Renderer.Extensions, config)},
		{Path: "build/android/generated/NovaApp.java", Content: androidcodegen.AppFromContract(ctx.Input.Project.Project.Name, app, config)},
		{Path: "build/android/generated/NovaRoutes.java", Content: androidcodegen.RoutesFromContract(app.View, config)},
		{Path: "build/android/generated/NovaExternalBindings.java", Content: androidcodegen.ExternalBindingsJava(ctx.Input.Plan.ExternalOperations, ctx.Input.Plan.Permissions, config)},
	}
	if rules := androidcodegen.NovaStyleRulesFromBundle(ctx.Input.StyleBundle); rules != "" || androidcodegen.ContractHasDynamicClassBindings(app.View.Bindings) {
		content := rules
		if content == "" {
			content = "public final class NovaStyleRules {\n    public static final java.util.Map<String, java.util.Map<String, String>> CLASS_RULES = java.util.Collections.emptyMap();\n    private NovaStyleRules() {}\n}\n"
		}
		files = append(files, shared.File{
			Path:    sourceRoot + "/NovaStyleRules.java",
			Content: "package " + config.Namespace + ";\n\n" + content,
		})
	}
	return files, nil
}

func AndroidSourceRoot(namespace string) string {
	return "build/android/app/src/main/java/" + strings.ReplaceAll(namespace, ".", "/")
}

func parserFilesFromSources(sources []build.SourceFile) []parser.File {
	files := make([]parser.File, 0, len(sources))
	for _, source := range sources {
		files = append(files, source.File)
	}
	return files
}
