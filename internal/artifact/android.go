package artifact

import (
	"bytes"
	_ "embed"
	"strconv"
	"strings"
	"text/template"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/routing"
	"github.com/dwlhm/nova/internal/view"
)

//go:embed templates/android/MainActivity.kt.tmpl
var androidMainActivityTemplateSource string

//go:embed templates/android/NovaRuntime.kt.tmpl
var androidRuntimeTemplateSource string

var androidMainActivityTemplate = template.Must(template.New("android-main-activity").Parse(androidMainActivityTemplateSource))
var androidRuntimeTemplate = template.Must(template.New("android-runtime").Parse(androidRuntimeTemplateSource))

type androidTargetConfig struct {
	ApplicationID         string
	Namespace             string
	CompileSDK            string
	MinSDK                string
	TargetSDK             string
	VersionCode           string
	VersionName           string
	GradlePlugin          string
	KotlinPlugin          string
	ComposeCompilerPlugin string
	ComposeBOM            string
	ActivityCompose       string
	Material3             string
	Theme                 string
	ThemeParent           string
	Label                 string
	JavaVersion           string
}

func androidConfig(manifest project.Manifest) (androidTargetConfig, []Diagnostic) {
	target := manifest.Targets["android"]
	options := target.Options
	required := []string{
		"application_id",
		"namespace",
		"compile_sdk",
		"min_sdk",
		"target_sdk",
		"version_code",
		"gradle_plugin",
		"kotlin_plugin",
		"compose_compiler_plugin",
		"compose_bom",
		"activity_compose",
		"material3",
		"theme",
		"theme_parent",
		"java_version",
	}
	diagnostics := make([]Diagnostic, 0)
	for _, key := range required {
		if strings.TrimSpace(options[key]) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Code:     "NVA-ANDROID-001",
				Severity: diagnostic.SeverityError,
				Message:  "targets.android." + key + " is required for Android artifact generation",
			})
		}
	}
	if len(diagnostics) > 0 {
		return androidTargetConfig{}, diagnostics
	}

	versionName := strings.TrimSpace(options["version_name"])
	if versionName == "" {
		versionName = manifest.Project.Version
	}
	label := strings.TrimSpace(options["label"])
	if label == "" {
		label = manifest.Project.Name
	}
	return androidTargetConfig{
		ApplicationID:         options["application_id"],
		Namespace:             options["namespace"],
		CompileSDK:            options["compile_sdk"],
		MinSDK:                options["min_sdk"],
		TargetSDK:             options["target_sdk"],
		VersionCode:           options["version_code"],
		VersionName:           versionName,
		GradlePlugin:          options["gradle_plugin"],
		KotlinPlugin:          options["kotlin_plugin"],
		ComposeCompilerPlugin: options["compose_compiler_plugin"],
		ComposeBOM:            options["compose_bom"],
		ActivityCompose:       options["activity_compose"],
		Material3:             options["material3"],
		Theme:                 options["theme"],
		ThemeParent:           options["theme_parent"],
		Label:                 label,
		JavaVersion:           options["java_version"],
	}, nil
}

func androidSettings(name string, config androidTargetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	return "pluginManagement {\n    repositories {\n        google()\n        mavenCentral()\n        gradlePluginPortal()\n    }\n}\n\ndependencyResolutionManagement {\n    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)\n    repositories {\n        google()\n        mavenCentral()\n    }\n}\n\nrootProject.name = " + quoteKotlin(name) + "\ninclude(\":app\")\n"
}

func androidGradleProperties() string {
	return "android.useAndroidX=true\nandroid.nonTransitiveRClass=true\n"
}

func androidGradle(name string, config androidTargetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	return "buildscript {\n    repositories {\n        google()\n        mavenCentral()\n    }\n    dependencies {\n        classpath(\"com.android.tools.build:gradle:" + escapeGradleComment(config.GradlePlugin) + "\")\n        classpath(\"org.jetbrains.kotlin:kotlin-gradle-plugin:" + escapeGradleComment(config.KotlinPlugin) + "\")\n        classpath(\"org.jetbrains.kotlin.plugin.compose:org.jetbrains.kotlin.plugin.compose.gradle.plugin:" + escapeGradleComment(config.ComposeCompilerPlugin) + "\")\n    }\n}\n\n// Generated Nova Android project for " + escapeGradleComment(name) + ".\n"
}

func androidAppGradle(config androidTargetConfig) string {
	javaVersion := "JavaVersion.VERSION_" + strings.ReplaceAll(config.JavaVersion, ".", "_")
	jvmTarget := "JvmTarget.JVM_" + strings.ReplaceAll(config.JavaVersion, ".", "_")
	return "import com.android.build.api.dsl.ApplicationExtension\nimport org.jetbrains.kotlin.gradle.dsl.JvmTarget\nimport org.jetbrains.kotlin.gradle.tasks.KotlinCompile\n\napply(plugin = \"com.android.application\")\napply(plugin = \"org.jetbrains.kotlin.android\")\napply(plugin = \"org.jetbrains.kotlin.plugin.compose\")\n\nextensions.configure<ApplicationExtension>(\"android\") {\n    namespace = " + quoteKotlin(config.Namespace) + "\n    compileSdk = " + config.CompileSDK + "\n\n    defaultConfig {\n        applicationId = " + quoteKotlin(config.ApplicationID) + "\n        minSdk = " + config.MinSDK + "\n        targetSdk = " + config.TargetSDK + "\n        versionCode = " + config.VersionCode + "\n        versionName = " + quoteKotlin(config.VersionName) + "\n    }\n\n    buildFeatures {\n        compose = true\n    }\n\n    compileOptions {\n        sourceCompatibility = " + javaVersion + "\n        targetCompatibility = " + javaVersion + "\n    }\n}\n\ntasks.withType<KotlinCompile>().configureEach {\n    compilerOptions.jvmTarget.set(" + jvmTarget + ")\n}\n\ndependencies {\n    add(\"implementation\", platform(\"androidx.compose:compose-bom:" + config.ComposeBOM + "\"))\n    add(\"implementation\", \"androidx.activity:activity-compose:" + config.ActivityCompose + "\")\n    add(\"implementation\", \"androidx.compose.material3:material3:" + config.Material3 + "\")\n}\n"
}

func androidManifest(config androidTargetConfig) string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\">\n    <application android:theme=\"@style/" + escapeXML(config.Theme) + "\" android:label=" + quoteXML(config.Label) + ">\n        <activity android:name=\"" + escapeXML(config.Namespace) + ".MainActivity\" android:exported=\"true\">\n            <intent-filter>\n                <action android:name=\"android.intent.action.MAIN\" />\n                <category android:name=\"android.intent.category.LAUNCHER\" />\n            </intent-filter>\n        </activity>\n    </application>\n</manifest>\n"
}

func androidStyles(config androidTargetConfig) string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<resources>\n    <style name=\"" + escapeXML(config.Theme) + "\" parent=\"" + escapeXML(config.ThemeParent) + "\">\n        <item name=\"android:windowActionBar\">false</item>\n        <item name=\"android:windowNoTitle\">true</item>\n    </style>\n</resources>\n"
}

func androidMainActivity(name string, bundle irBundle, config androidTargetConfig) string {
	routePatterns := androidPagePaths(bundle)
	renderer := androidComposeRenderer{stateNames: androidStateNames(bundle.Model), routePatterns: routePatterns}
	return renderAndroidMainActivity(androidMainActivityTemplateData{
		PackageName:       config.Namespace,
		StateInitializers: androidStateInitializers(bundle.Model),
		RenderBody:        renderer.renderNodes(bundle.ViewIR.Nodes, "            "),
		Transitions:       androidTransitionTable(bundle.Model),
		RoutePatterns:     androidRoutePatternTable(routePatterns),
	})
}

type androidMainActivityTemplateData struct {
	PackageName       string
	StateInitializers string
	RenderBody        string
	Transitions       string
	RoutePatterns     string
}

func renderAndroidMainActivity(data androidMainActivityTemplateData) string {
	var buffer bytes.Buffer
	if err := androidMainActivityTemplate.Execute(&buffer, data); err != nil {
		panic(err)
	}
	return buffer.String()
}

func androidRuntime(config androidTargetConfig) string {
	var buffer bytes.Buffer
	if err := androidRuntimeTemplate.Execute(&buffer, struct{ PackageName string }{PackageName: config.Namespace}); err != nil {
		panic(err)
	}
	return buffer.String()
}

func androidStateInitializers(model appModel) string {
	var builder strings.Builder
	for _, state := range model.States {
		builder.WriteString("        state[" + quoteKotlin(state.Name) + "] = " + androidInitialValue(state.Initial) + "\n")
	}
	return builder.String()
}

type androidComposeRenderer struct {
	stateNames    map[string]bool
	routePatterns []string
}

func (renderer androidComposeRenderer) renderNodes(nodes []view.Node, indent string) string {
	var builder strings.Builder
	for _, node := range nodes {
		builder.WriteString(renderer.renderNode(node, indent))
	}
	return builder.String()
}

func (renderer androidComposeRenderer) renderNode(node view.Node, indent string) string {
	switch node.Kind {
	case "page":
		return renderer.renderPage(node, indent)
	case "text", "#text":
		return renderer.renderText(node, indent)
	case "button":
		return renderer.renderButton(node, indent)
	case "row":
		return renderer.renderContainer(node, indent, "FlowRow", "horizontalArrangement = Arrangement.spacedBy(8.dp),\n"+indent+"    verticalArrangement = Arrangement.spacedBy(8.dp)")
	case "stack":
		return renderer.renderContainer(node, indent, "Box", "")
	default:
		return renderer.renderContainer(node, indent, "Column", "verticalArrangement = Arrangement.spacedBy(8.dp)")
	}
}

func (renderer androidComposeRenderer) renderPage(node view.Node, indent string) string {
	path, ok := androidKotlinValueExpression(node.Props["path"].Tokens, renderer.stateNames)
	if !ok {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(indent + "if (routeMatches(textValue(" + path + "), activeRoutePath())) {\n")
	builder.WriteString(renderer.renderNodes(node.Children, indent+"    "))
	builder.WriteString(indent + "}\n")
	return builder.String()
}

func (renderer androidComposeRenderer) renderText(node view.Node, indent string) string {
	value := androidTextExpression(node.Props["value"], renderer.stateNames)
	return indent + "Text(text = " + value + ", modifier = Modifier.padding(vertical = 4.dp))\n"
}

func (renderer androidComposeRenderer) renderButton(node view.Node, indent string) string {
	onClick := "{}"
	if route, ok := node.Events["on_press"]; ok {
		onClick = "{ dispatch(" + quoteKotlin(string(route.Event)) + ", " + androidEventArgs(route.Args, renderer.stateNames) + ") }"
	}
	var builder strings.Builder
	builder.WriteString(indent + "Button(onClick = " + onClick + ", modifier = Modifier.padding(4.dp)) {\n")
	builder.WriteString(indent + "    Text(text = " + androidButtonLabel(node, renderer.stateNames) + ")\n")
	builder.WriteString(indent + "}\n")
	return builder.String()
}

func (renderer androidComposeRenderer) renderContainer(node view.Node, indent string, composable string, arrangement string) string {
	var builder strings.Builder
	builder.WriteString(indent + composable + "(\n")
	builder.WriteString(indent + "    modifier = Modifier.padding(vertical = 4.dp)")
	if arrangement != "" {
		builder.WriteString(",\n" + indent + "    " + arrangement + "\n")
	} else {
		builder.WriteString("\n")
	}
	builder.WriteString(indent + ") {\n")
	builder.WriteString(renderer.renderNodes(node.Children, indent+"    "))
	builder.WriteString(indent + "}\n")
	return builder.String()
}

func androidApp(name string, target string, config androidTargetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "NovaApp"
	}
	return "package " + config.Namespace + "\n\nclass NovaApp {\n    val name = " + quoteKotlin(name) + "\n    val target = " + quoteKotlin(target) + "\n}\n"
}

func androidRoutes(bundle irBundle, config androidTargetConfig) string {
	paths := androidPagePaths(bundle)
	if len(paths) == 0 {
		paths = []string{"/"}
	}
	names := make(map[string]int)
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + "\n\nobject NovaRoutes {\n")
	for _, path := range paths {
		baseName := androidRouteConstName(path)
		names[baseName]++
		name := baseName
		if names[baseName] > 1 {
			name = baseName + "_" + javaInt(names[baseName])
		}
		builder.WriteString("    const val ")
		builder.WriteString(name)
		builder.WriteString(" = ")
		builder.WriteString(quoteKotlin(path))
		builder.WriteString("\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

func androidExternalBindings(operations []build.ResolvedExternalOperation, config androidTargetConfig) string {
	names := externalOperationNames(operations)
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + "\n\nobject NovaExternalBindings {\n")
	builder.WriteString("    val operations = listOf(\n")
	for _, name := range names {
		builder.WriteString("        ")
		builder.WriteString(quoteKotlin(name))
		builder.WriteString(",\n")
	}
	builder.WriteString("    )\n")
	builder.WriteString("}\n")
	return builder.String()
}

func androidPagePaths(bundle irBundle) []string {
	seen := make(map[string]bool)
	paths := make([]string, 0, len(bundle.ViewIR.Metadata.Pages))
	for _, page := range bundle.ViewIR.Metadata.Pages {
		path, ok := staticBindingString(page.Path)
		if !ok {
			continue
		}
		path = routing.DescribePattern(path).Pattern
		if seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths
}

func staticBindingString(binding view.Binding) (string, bool) {
	tokens := trimExpressionTokens(binding.Tokens)
	if len(tokens) == 1 && tokens[0].Type == lexer.STRING {
		return tokens[0].Literal, true
	}
	if binding.Text != "" {
		return binding.Text, true
	}
	return "", false
}

func androidTransitionTable(model appModel) string {
	var builder strings.Builder
	builder.WriteString("    private fun transitions(): List<NovaTransition> = listOf(\n")
	for _, state := range model.States {
		for _, transition := range state.Transitions {
			builder.WriteString("        NovaTransition(")
			builder.WriteString(quoteKotlin(state.Name))
			builder.WriteString(", ")
			builder.WriteString(quoteKotlin(transition.Event))
			builder.WriteString(", listOf(")
			for i, param := range transition.Params {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(quoteKotlin(param))
			}
			builder.WriteString("), ")
			builder.WriteString(quoteKotlin(transition.Expression))
			builder.WriteString("),\n")
		}
	}
	builder.WriteString("    )\n\n")
	return builder.String()
}

func androidRoutePatternTable(patterns []string) string {
	var builder strings.Builder
	builder.WriteString("    private fun routePatterns(): List<String> = listOf(\n")
	for _, pattern := range patterns {
		builder.WriteString("        ")
		builder.WriteString(quoteKotlin(pattern))
		builder.WriteString(",\n")
	}
	builder.WriteString("    )\n\n")
	return builder.String()
}

func androidInitialValue(expression string) string {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "null"
	}
	if strings.HasPrefix(expression, "({") && strings.HasSuffix(expression, "})") {
		return androidInitialRecord(expression)
	}
	if strings.HasPrefix(expression, "\"") && strings.HasSuffix(expression, "\"") {
		return quoteKotlin(strings.Trim(expression, "\""))
	}
	if _, err := strconv.ParseFloat(expression, 64); err == nil {
		if strings.Contains(expression, ".") {
			return expression
		}
		return expression + ".0"
	}
	switch expression {
	case "true", "false":
		return expression
	case "null", "undefined":
		return "null"
	default:
		return "null"
	}
}

func androidInitialRecord(expression string) string {
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "({"), "})"))
	if body == "" {
		return "emptyMap<String, Any?>()"
	}
	fields := splitAndroidRecordFields(body)
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		name, value, ok := strings.Cut(field, ":")
		if !ok {
			continue
		}
		parts = append(parts, quoteKotlin(strings.TrimSpace(name))+" to "+androidInitialValue(strings.TrimSpace(value)))
	}
	if len(parts) == 0 {
		return "emptyMap<String, Any?>()"
	}
	return "record(" + strings.Join(parts, ", ") + ")"
}

func splitAndroidRecordFields(body string) []string {
	fields := make([]string, 0)
	start := 0
	inString := false
	for i, ch := range body {
		if ch == '"' {
			inString = !inString
		}
		if ch == ',' && !inString {
			fields = append(fields, strings.TrimSpace(body[start:i]))
			start = i + 1
		}
	}
	fields = append(fields, strings.TrimSpace(body[start:]))
	return fields
}

func androidStateNames(model appModel) map[string]bool {
	names := make(map[string]bool, len(model.States))
	for _, state := range model.States {
		names[state.Name] = true
	}
	return names
}

func androidTextExpression(binding view.Binding, stateNames map[string]bool) string {
	tokens := trimExpressionTokens(binding.Tokens)
	if len(tokens) == 0 {
		return quoteKotlin(binding.Text)
	}
	if expression, ok := androidKotlinStringExpression(tokens, stateNames); ok {
		return expression
	}
	return quoteKotlin(binding.Text)
}

func androidKotlinStringExpression(tokens []lexer.Token, stateNames map[string]bool) (string, bool) {
	parts := splitAndroidTopLevel(tokens, lexer.PLUS)
	if len(parts) > 1 {
		expressions := make([]string, 0, len(parts))
		for _, part := range parts {
			expression, ok := androidKotlinStringExpression(part, stateNames)
			if !ok {
				return "", false
			}
			expressions = append(expressions, expression)
		}
		return strings.Join(expressions, " + "), true
	}
	if len(tokens) == 1 && tokens[0].Type == lexer.STRING {
		return quoteKotlin(tokens[0].Literal), true
	}
	if value, ok := androidKotlinValueExpression(tokens, stateNames); ok {
		return "textValue(" + value + ")", true
	}
	return "", false
}

func androidKotlinValueExpression(tokens []lexer.Token, stateNames map[string]bool) (string, bool) {
	tokens = trimExpressionTokens(tokens)
	if len(tokens) == 0 {
		return "null", true
	}
	if fields, ok := parseRecordExpressionFields(tokens); ok {
		parts := make([]string, 0, len(fields))
		for _, field := range fields {
			value, ok := androidKotlinValueExpression(field.Value, stateNames)
			if !ok {
				return "", false
			}
			parts = append(parts, quoteKotlin(field.Name.Literal)+" to "+value)
		}
		return "record(" + strings.Join(parts, ", ") + ")", true
	}
	if len(tokens) == 1 {
		switch tokens[0].Type {
		case lexer.STRING:
			return quoteKotlin(tokens[0].Literal), true
		case lexer.NUMBER:
			return tokens[0].Literal, true
		case lexer.TRUE:
			return "true", true
		case lexer.FALSE:
			return "false", true
		case lexer.NULL, lexer.VOID:
			return "null", true
		case lexer.IDENT:
			if stateNames[tokens[0].Literal] {
				return "state[" + quoteKotlin(tokens[0].Literal) + "]", true
			}
		}
	}
	if isRoutePathExpression(tokens) {
		return "pathOf(state[\"route\"])", true
	}
	return "", false
}

func splitAndroidTopLevel(tokens []lexer.Token, delimiter lexer.TokenType) [][]lexer.Token {
	segments := make([][]lexer.Token, 0)
	start := 0
	depth := 0
	for i, tok := range tokens {
		if depth == 0 && tok.Type == delimiter {
			segments = append(segments, trimExpressionTokens(tokens[start:i]))
			start = i + 1
			continue
		}
		depth = expressionDepth(depth, tok.Type)
	}
	if start == 0 {
		return nil
	}
	segments = append(segments, trimExpressionTokens(tokens[start:]))
	return segments
}

func isRoutePathExpression(tokens []lexer.Token) bool {
	return len(tokens) == 3 &&
		tokens[0].Type == lexer.IDENT && tokens[0].Literal == "route" &&
		tokens[1].Type == lexer.DOT &&
		tokens[2].Type == lexer.IDENT && tokens[2].Literal == "path"
}

func androidEventArgs(args []view.Binding, stateNames map[string]bool) string {
	if len(args) == 0 {
		return "emptyList()"
	}
	values := make([]string, 0, len(args))
	for _, arg := range args {
		value, ok := androidKotlinValueExpression(arg.Tokens, stateNames)
		if !ok {
			value = "null"
		}
		values = append(values, value)
	}
	return "listOf(" + strings.Join(values, ", ") + ")"
}

func androidButtonLabel(node view.Node, stateNames map[string]bool) string {
	for _, child := range node.Children {
		if child.Kind == "text" || child.Kind == "#text" {
			if value, ok := child.Props["value"]; ok {
				return androidTextExpression(value, stateNames)
			}
		}
	}
	if label, ok := node.Props["label"]; ok {
		return androidTextExpression(label, stateNames)
	}
	return quoteKotlin("Button")
}

func androidRouteConstName(path string) string {
	if strings.TrimSpace(path) == "*" || strings.TrimSpace(path) == "/*" {
		return "fallback"
	}
	name := strings.Trim(strings.ToLower(path), "/")
	if name == "" {
		return "root"
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
			lastUnderscore = false
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "route"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "route_" + out
	}
	return out
}

func javaInt(value int) string {
	return strconv.Itoa(value)
}

func quoteXML(value string) string {
	return "\"" + escapeXML(value) + "\""
}

func escapeXML(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "\"", "&quot;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	return value
}
