package artifact

import (
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/routing"
	"github.com/dwlhm/nova/internal/view"
)

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
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + "\n\n")
	builder.WriteString("import android.os.Bundle\n")
	builder.WriteString("import androidx.activity.ComponentActivity\n")
	builder.WriteString("import androidx.activity.compose.BackHandler\n")
	builder.WriteString("import androidx.activity.compose.setContent\n")
	builder.WriteString("import androidx.compose.foundation.layout.Arrangement\n")
	builder.WriteString("import androidx.compose.foundation.layout.Box\n")
	builder.WriteString("import androidx.compose.foundation.layout.Column\n")
	builder.WriteString("import androidx.compose.foundation.layout.Row\n")
	builder.WriteString("import androidx.compose.foundation.layout.fillMaxSize\n")
	builder.WriteString("import androidx.compose.foundation.layout.padding\n")
	builder.WriteString("import androidx.compose.material3.Button\n")
	builder.WriteString("import androidx.compose.material3.MaterialTheme\n")
	builder.WriteString("import androidx.compose.material3.Surface\n")
	builder.WriteString("import androidx.compose.material3.Text\n")
	builder.WriteString("import androidx.compose.runtime.Composable\n")
	builder.WriteString("import androidx.compose.runtime.mutableStateListOf\n")
	builder.WriteString("import androidx.compose.runtime.mutableStateMapOf\n")
	builder.WriteString("import androidx.compose.ui.Modifier\n")
	builder.WriteString("import androidx.compose.ui.unit.dp\n")
	builder.WriteString("import java.net.URI\n")
	builder.WriteString("import java.net.URLDecoder\n")
	builder.WriteString("import java.net.URLEncoder\n\n")
	builder.WriteString("class MainActivity : ComponentActivity() {\n")
	builder.WriteString("    private val state = mutableStateMapOf<String, Any?>()\n")
	builder.WriteString("    private val routeBackStack = mutableStateListOf<Any?>()\n")
	builder.WriteString("    private var applyingSystemBack = false\n\n")
	builder.WriteString("    override fun onCreate(savedInstanceState: Bundle?) {\n")
	builder.WriteString("        super.onCreate(savedInstanceState)\n")
	builder.WriteString("        initializeState()\n")
	builder.WriteString("        initializeNavigationStack()\n")
	builder.WriteString("        setContent {\n")
	builder.WriteString("            MaterialTheme {\n")
	builder.WriteString("                BackHandler(enabled = canNavigateBack()) {\n")
	builder.WriteString("                    handleSystemBack()\n")
	builder.WriteString("                }\n")
	builder.WriteString("                Surface(modifier = Modifier.fillMaxSize()) {\n")
	builder.WriteString("                    RenderApp()\n")
	builder.WriteString("                }\n")
	builder.WriteString("            }\n")
	builder.WriteString("        }\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private fun initializeState() {\n")
	builder.WriteString("        if (state.isNotEmpty()) return\n")
	for _, state := range bundle.Model.States {
		builder.WriteString("        state[" + quoteKotlin(state.Name) + "] = " + androidInitialValue(state.Initial) + "\n")
	}
	builder.WriteString("    }\n\n")
	builder.WriteString("    @Composable\n")
	builder.WriteString("    private fun RenderApp() {\n")
	builder.WriteString("        Column(\n")
	builder.WriteString("            modifier = Modifier.fillMaxSize().padding(24.dp),\n")
	builder.WriteString("            verticalArrangement = Arrangement.Center\n")
	builder.WriteString("        ) {\n")
	builder.WriteString(renderer.renderNodes(bundle.ViewIR.Nodes, "            "))
	builder.WriteString("        }\n")
	builder.WriteString("    }\n\n")
	builder.WriteString(androidTransitionTable(bundle.Model))
	builder.WriteString(androidRoutePatternTable(routePatterns))
	builder.WriteString(androidComposeHelpers())
	builder.WriteString("}\n")
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
		return renderer.renderContainer(node, indent, "Row", "horizontalArrangement = Arrangement.spacedBy(8.dp)")
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

func androidComposeHelpers() string {
	return `    private data class NovaTransition(
        val stateName: String,
        val eventName: String,
        val params: List<String>,
        val expression: String
    )

    private fun initializeNavigationStack() {
        if (!hasRouteState() || routeBackStack.isNotEmpty()) return
        state["route"] = routeValueForShape(state["route"])
        routeBackStack.add(cloneRoute(state["route"]))
    }

    private fun dispatch(eventName: String, args: List<Any?>) {
        val beforeRoute = cloneRoute(state["route"])
        for (transition in transitions()) {
            if (transition.eventName != eventName) continue
            val payload = transition.params.mapIndexed { index, name -> name to args.getOrNull(index) }.toMap()
            state[transition.stateName] = evaluate(transition.expression, payload)
        }
        if (hasRouteState()) {
            state["route"] = routeValueForShape(state["route"])
        }
        reconcileRouteBackStack(beforeRoute, state["route"])
    }

    private fun hasRouteState(): Boolean = state.containsKey("route")

    private fun hasTransition(eventName: String): Boolean {
        return transitions().any { it.eventName == eventName }
    }

    private fun canNavigateBack(): Boolean = routeBackStack.size > 1

    private fun handleSystemBack() {
        if (!canNavigateBack()) return
        val targetRoute = cloneRoute(routeBackStack[routeBackStack.lastIndex - 1])
        val beforeRoute = cloneRoute(state["route"])
        applyingSystemBack = true
        try {
            if (hasTransition("@navigate")) {
                dispatch("@navigate", listOf(record("kind" to "back")))
            } else {
                dispatch("@route_changed", listOf(targetRoute))
            }
        } finally {
            applyingSystemBack = false
        }
        reconcileAfterSystemBack(beforeRoute, state["route"], targetRoute)
    }

    private fun reconcileAfterSystemBack(beforeRoute: Any?, afterRoute: Any?, targetRoute: Any?) {
        if (routeKey(beforeRoute) == routeKey(afterRoute)) return
        if (routeKey(afterRoute) == routeKey(targetRoute)) {
            routeBackStack.removeAt(routeBackStack.lastIndex)
            return
        }
        routeBackStack.removeAt(routeBackStack.lastIndex)
        if (routeKey(routeBackStack.lastOrNull()) != routeKey(afterRoute)) {
            routeBackStack.add(cloneRoute(afterRoute))
        }
    }

    private fun reconcileRouteBackStack(beforeRoute: Any?, afterRoute: Any?) {
        if (!hasRouteState()) return
        if (routeBackStack.isEmpty()) {
            routeBackStack.add(cloneRoute(afterRoute))
            return
        }
        if (routeKey(beforeRoute) == routeKey(afterRoute) || applyingSystemBack) return
        if (routeKey(routeBackStack.lastOrNull()) != routeKey(afterRoute)) {
            routeBackStack.add(cloneRoute(afterRoute))
        }
    }

    private data class RouteMatch(
        val matched: Boolean,
        val pattern: String,
        val params: Map<String, String>,
        val score: Int,
        val fallback: Boolean
    )

    private fun routeKey(value: Any?): String {
        val route = routeObject(value)
        val query = queryString(route["query"])
        val fragment = route["fragment"]?.toString()?.trimStart('#').orEmpty()
        return route["path"].toString() + query + if (fragment.isBlank()) "" else "#$fragment"
    }

    private fun cloneRoute(value: Any?): Any? {
        return when (value) {
            is Map<*, *> -> value.entries.associate { it.key.toString() to it.value }
            else -> value
        }
    }

    private fun routeValueForShape(value: Any?): Any? {
        val route = routeObject(value)
        if (value is String) return route["path"]
        val next = if (value is Map<*, *>) {
            value.entries.associate { it.key.toString() to it.value }.toMutableMap()
        } else {
            mutableMapOf<String, Any?>()
        }
        next["path"] = route["path"]
        val query = route["query"]
        if (query is Map<*, *> && query.isNotEmpty()) next["query"] = query else next.remove("query")
        val fragment = route["fragment"]?.toString().orEmpty()
        if (fragment.isNotBlank()) next["fragment"] = fragment else next.remove("fragment")
        val best = bestRouteMatch(routePatterns(), route["path"].toString())
        if (best.params.isNotEmpty()) next["params"] = best.params else next.remove("params")
        return next
    }

    private fun evaluate(expression: String, payload: Map<String, Any?>): Any? {
        val trimmed = expression.trim()
        splitBinary(trimmed, "+")?.let { (left, right) ->
            val leftValue = evaluate(left, payload)
            val rightValue = evaluate(right, payload)
            if (leftValue is Number && rightValue is Number) {
                return numberValue(leftValue) + numberValue(rightValue)
            }
            return textValue(leftValue) + textValue(rightValue)
        }
        splitBinary(trimmed, "-")?.let { (left, right) ->
            return numberValue(evaluate(left, payload)) - numberValue(evaluate(right, payload))
        }
        return evaluateAtom(trimmed, payload)
    }

    private fun evaluateAtom(expression: String, payload: Map<String, Any?>): Any? {
        if (expression.startsWith("({") && expression.endsWith("})")) {
            return evaluateRecord(expression, payload)
        }
        if (expression.startsWith("payload.")) return payload[expression.removePrefix("payload.")]
        if (expression.startsWith("state.")) return state[expression.removePrefix("state.")]
        if (expression.startsWith("\"") && expression.endsWith("\"")) return expression.substring(1, expression.length - 1)
        expression.toDoubleOrNull()?.let { return it }
        return when (expression) {
            "true" -> true
            "false" -> false
            "null", "undefined", "" -> null
            else -> expression
        }
    }

    private fun evaluateRecord(expression: String, payload: Map<String, Any?>): Map<String, Any?> {
        val body = expression.removePrefix("({").removeSuffix("})").trim()
        if (body.isEmpty()) return emptyMap()
        return body.split(",").mapNotNull { field ->
            val parts = field.split(":", limit = 2)
            if (parts.size != 2) return@mapNotNull null
            parts[0].trim() to evaluate(parts[1].trim(), payload)
        }.toMap()
    }

    private fun splitBinary(expression: String, operator: String): Pair<String, String>? {
        val marker = " $operator "
        val index = expression.indexOf(marker)
        if (index < 0) return null
        return expression.substring(0, index) to expression.substring(index + marker.length)
    }

    private fun activeRoutePath(): String = pathOf(state["route"])

    private fun pathOf(value: Any?): String {
        return routeObject(value)["path"]?.toString() ?: "/"
    }

    private fun routeObject(value: Any?): Map<String, Any?> {
        if (value is Map<*, *>) {
            val parsed = routeObjectFromString(value["path"]?.toString() ?: "/")
            val query = value["query"] ?: parsed["query"]
            val fragment = value["fragment"] ?: parsed["fragment"]
            return mapOf("path" to parsed["path"], "query" to query, "fragment" to fragment)
        }
        return routeObjectFromString(value?.toString() ?: "/")
    }

    private fun routeObjectFromString(value: String): Map<String, Any?> {
        val text = value.trim().ifBlank { "/" }
        try {
            val uri = URI(text)
            if (uri.isAbsolute) {
                return mapOf(
                    "path" to normalizePath(uri.rawPath ?: "/"),
                    "query" to queryMap(uri.rawQuery.orEmpty()),
                    "fragment" to safeDecodeComponent(uri.rawFragment.orEmpty())
                )
            }
        } catch (ignored: Exception) {
        }
        val hashIndex = text.indexOf('#')
        val beforeHash = if (hashIndex >= 0) text.substring(0, hashIndex) else text
        val fragment = if (hashIndex >= 0) safeDecodeComponent(text.substring(hashIndex + 1)) else ""
        val queryIndex = beforeHash.indexOf('?')
        val rawPath = if (queryIndex >= 0) beforeHash.substring(0, queryIndex) else beforeHash
        val rawQuery = if (queryIndex >= 0) beforeHash.substring(queryIndex + 1) else ""
        return mapOf("path" to normalizePath(rawPath), "query" to queryMap(rawQuery), "fragment" to fragment)
    }

    private fun normalizePath(value: String): String {
        var cleaned = value.trim().ifBlank { "/" }
        cleaned = cleaned.substringBefore('#').substringBefore('?')
        if (!cleaned.startsWith("/")) cleaned = "/$cleaned"
        val segments = mutableListOf<String>()
        cleaned.split("/").forEach { rawSegment ->
            val segment = safeDecodePathSegment(rawSegment)
            when {
                segment.isBlank() || segment == "." -> Unit
                segment == ".." -> {
                    if (segments.isNotEmpty()) segments.removeAt(segments.lastIndex)
                }
                else -> segments.add(segment)
            }
        }
        return if (segments.isEmpty()) "/" else "/" + segments.joinToString("/")
    }

    private fun routeMatches(pattern: String, value: String): Boolean = routeMatch(pattern, value).matched

    private fun bestRouteMatch(patterns: List<String>, value: String): RouteMatch {
        var best = RouteMatch(false, "", emptyMap(), -1, false)
        for (pattern in patterns) {
            val match = routeMatch(pattern, value)
            if (match.matched && (!best.matched || match.score > best.score)) {
                best = match
            }
        }
        return best
    }

    private fun routeMatch(pattern: String, value: String): RouteMatch {
        val normalizedPattern = normalizePattern(pattern)
        val normalizedPath = normalizePath(value)
        if (normalizedPattern == "*") return RouteMatch(true, normalizedPattern, emptyMap(), 0, true)
        val patternSegments = routeSegments(normalizedPattern)
        val pathSegments = routeSegments(normalizedPath)
        val params = mutableMapOf<String, String>()
        var score = 0
        for (index in patternSegments.indices) {
            val segment = patternSegments[index]
            if (segment == "*") {
                return if (index == patternSegments.lastIndex) {
                    RouteMatch(true, normalizedPattern, params, score + 10, false)
                } else {
                    RouteMatch(false, normalizedPattern, emptyMap(), -1, false)
                }
            }
            if (index >= pathSegments.size) return RouteMatch(false, normalizedPattern, emptyMap(), -1, false)
            val name = dynamicSegmentName(segment)
            if (name != null) {
                params[name] = pathSegments[index]
                score += 50
                continue
            }
            if (segment != pathSegments[index]) return RouteMatch(false, normalizedPattern, emptyMap(), -1, false)
            score += 100
        }
        if (patternSegments.size != pathSegments.size) return RouteMatch(false, normalizedPattern, emptyMap(), -1, false)
        return RouteMatch(true, normalizedPattern, params, score + 1000, false)
    }

    private fun routeParams(pattern: String, value: String): Map<String, String> = routeMatch(pattern, value).params

    private fun normalizePattern(pattern: String): String {
        val trimmed = pattern.trim()
        if (trimmed == "*" || trimmed == "/*") return "*"
        if (trimmed.endsWith("/*")) return normalizePath(trimmed.removeSuffix("/*")) + "/*"
        return normalizePath(trimmed)
    }

    private fun routeSegments(value: String): List<String> {
        val normalized = normalizePath(value)
        return if (normalized == "/") emptyList() else normalized.trim('/').split("/")
    }

    private fun dynamicSegmentName(segment: String): String? {
        if (segment.startsWith(":") && segment.length > 1) return segment.substring(1)
        if (segment.startsWith("{") && segment.endsWith("}") && segment.length > 2) return segment.substring(1, segment.length - 1)
        return null
    }

    private fun queryMap(query: String): Map<String, String> {
        if (query.isBlank()) return emptyMap()
        return query.split("&").mapNotNull { part ->
            if (part.isBlank()) return@mapNotNull null
            val pieces = part.split("=", limit = 2)
            safeDecodeComponent(pieces[0]) to safeDecodeComponent(pieces.getOrElse(1) { "" })
        }.toMap()
    }

    private fun queryString(value: Any?): String {
        if (value !is Map<*, *> || value.isEmpty()) return ""
        return value.entries
            .map { it.key.toString() to it.value?.toString().orEmpty() }
            .sortedBy { it.first }
            .joinToString("&", prefix = "?") { "${encodeComponent(it.first)}=${encodeComponent(it.second)}" }
    }

    private fun safeDecodePathSegment(value: String): String {
        return safeDecodeComponent(value.replace("+", "%2B"))
    }

    private fun safeDecodeComponent(value: String): String {
        return try {
            URLDecoder.decode(value, "UTF-8")
        } catch (ignored: Exception) {
            value
        }
    }

    private fun encodeComponent(value: String): String {
        return URLEncoder.encode(value, "UTF-8").replace("+", "%20")
    }

    private fun record(vararg fields: Pair<String, Any?>): Map<String, Any?> = mapOf(*fields)

    private fun textValue(value: Any?): String {
        if (value == null) return ""
        if (value is Double && value % 1.0 == 0.0) return value.toLong().toString()
        return value.toString()
    }

    private fun numberValue(value: Any?): Double {
        return when (value) {
            is Number -> value.toDouble()
            is String -> value.toDoubleOrNull() ?: 0.0
            else -> 0.0
        }
    }
`
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
