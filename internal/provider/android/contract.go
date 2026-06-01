package android

import (
	"github.com/dwlhm/nova/internal/provider/shared"
	"sort"
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/routing"
)

type androidContractRenderer struct {
	styles androidStyleSheet
}

func mainActivityFromContract(app contract.App, config targetConfig, styles []shared.StyleAsset) string {
	routePatterns := androidContractPagePaths(app.View)
	renderer := androidContractRenderer{styles: newAndroidStyleSheet(styles)}
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\n")
	builder.WriteString("import static " + config.Namespace + ".NovaRuntime.*;\n\n")
	builder.WriteString("import nova.scheduler.NovaScheduler;\n")
	builder.WriteString("import nova.scheduler.NovaTransition;\n\n")
	builder.WriteString("import android.app.Activity;\n")
	builder.WriteString("import android.graphics.Color;\n")
	builder.WriteString("import android.graphics.Typeface;\n")
	builder.WriteString("import android.graphics.drawable.GradientDrawable;\n")
	builder.WriteString("import android.os.Bundle;\n")
	builder.WriteString("import android.text.Editable;\n")
	builder.WriteString("import android.text.InputType;\n")
	builder.WriteString("import android.text.TextWatcher;\n")
	builder.WriteString("import android.util.TypedValue;\n")
	builder.WriteString("import android.view.Gravity;\n")
	builder.WriteString("import android.view.View;\n")
	builder.WriteString("import android.view.ViewGroup;\n")
	builder.WriteString("import android.widget.Button;\n")
	builder.WriteString("import android.widget.EditText;\n")
	builder.WriteString("import android.widget.FrameLayout;\n")
	builder.WriteString("import android.widget.GridLayout;\n")
	builder.WriteString("import android.widget.LinearLayout;\n")
	builder.WriteString("import android.widget.ScrollView;\n")
	builder.WriteString("import android.widget.TextView;\n")
	builder.WriteString("import java.util.ArrayList;\n")
	builder.WriteString("import java.util.Arrays;\n")
	builder.WriteString("import java.util.Collections;\n")
	builder.WriteString("import java.util.LinkedHashMap;\n")
	builder.WriteString("import java.util.LinkedHashSet;\n")
	builder.WriteString("import java.util.List;\n")
	builder.WriteString("import java.util.Map;\n")
	builder.WriteString("import java.util.Set;\n\n")
	builder.WriteString("public final class MainActivity extends Activity implements NovaScheduler.Host {\n")
	builder.WriteString("    private final Map<String, Object> state = new LinkedHashMap<>();\n")
	builder.WriteString("    private final Map<String, View> views = new LinkedHashMap<>();\n")
	builder.WriteString("    private final List<Object> routeBackStack = new ArrayList<>();\n")
	builder.WriteString("    private final NovaScheduler scheduler = new NovaScheduler(this);\n")
	builder.WriteString("    private final NovaPrimitiveRegistry primitiveRegistry = NovaRendererExtensions.register(new NovaPrimitiveRegistry());\n")
	builder.WriteString("    private boolean applyingSystemBack = false;\n")
	if len(app.Lifecycles) > 0 {
		builder.WriteString("    private boolean mounting = false;\n")
		builder.WriteString("    private final Set<String> mountInvalidations = new LinkedHashSet<>();\n")
	}
	builder.WriteString("\n")
	builder.WriteString("    @Override\n")
	builder.WriteString("    protected void onCreate(Bundle savedInstanceState) {\n")
	builder.WriteString("        super.onCreate(savedInstanceState);\n")
	builder.WriteString("        initializeState();\n")
	builder.WriteString("        initializeNavigationStack();\n")
	builder.WriteString("        schedulerMountLifecycles();\n")
	builder.WriteString("        setContentView(buildViewTree());\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    @Override\n")
	builder.WriteString("    public void onBackPressed() {\n")
	builder.WriteString("        if (canNavigateBack()) handleSystemBack(); else super.onBackPressed();\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void initializeState() {\n")
	builder.WriteString("        if (!state.isEmpty()) return;\n")
	builder.WriteString(androidContractStateInitializers(app.Model.States))
	builder.WriteString("    }\n\n")
	builder.WriteString("    private View buildViewTree() {\n")
	builder.WriteString("        views.clear();\n")
	builder.WriteString("        LinearLayout root = new LinearLayout(this);\n")
	builder.WriteString("        root.setOrientation(LinearLayout.VERTICAL);\n")
	builder.WriteString("        root.setGravity(Gravity.CENTER);\n")
	builder.WriteString("        root.setPadding(dp(24), dp(24), dp(24), dp(24));\n")
	builder.WriteString("        root.setLayoutParams(new ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));\n")
	builder.WriteString(renderer.renderBuildNodes(app.View.Nodes, "root", "        ", nil))
	builder.WriteString("        applyBindings(null);\n")
	builder.WriteString("        updatePageVisibility();\n")
	builder.WriteString("        return root;\n")
	builder.WriteString("    }\n\n")
	builder.WriteString(androidContractTransitionTable(app.Model.States))
	hydration, hydrationOK := detectStorageHydrationChain(app.Lifecycles)
	if hydrationOK {
		builder.WriteString(androidContractStorageHydration(hydration))
	}
	builder.WriteString(androidContractLifecycleHooks(app, hydration))
	builder.WriteString(androidJavaRoutePatternTable(routePatterns))
	builder.WriteString("    private List<String> stateNames() {\n")
	builder.WriteString("        return Arrays.asList(\n")
	for _, state := range app.Model.States {
		builder.WriteString("            " + shared.QuoteCodeString(state.Name) + ",\n")
	}
	builder.WriteString("            \"\"\n")
	builder.WriteString("        );\n")
	builder.WriteString("    }\n\n")
	builder.WriteString(renderer.renderApplyBindings(app.View.Nodes, app.View.Bindings))
	builder.WriteString(renderer.renderUpdatePageVisibility(app.View.Nodes))
	builder.WriteString(javaSchedulerActivityHost())
	if len(app.Lifecycles) == 0 {
		builder.WriteString(javaSchedulerLifecycleStubs())
	} else {
		builder.WriteString("    protected void schedulerDisposeLifecycles() {}\n\n")
	}
	builder.WriteString("    private void applyStateCommit(Set<String> invalidations) {\n")
	builder.WriteString("        if (invalidations.isEmpty()) return;\n")
	builder.WriteString("        applyBindings(invalidations);\n")
	builder.WriteString("        updatePageVisibility();\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private boolean hasRouteState() { return state.containsKey(\"route\"); }\n\n")
	builder.WriteString("    private boolean hasTransition(String eventName) {\n")
	builder.WriteString("        for (NovaTransition transition : transitions()) {\n")
	builder.WriteString("            if (transition.eventName.equals(eventName)) return true;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        return false;\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private boolean canNavigateBack() { return routeBackStack.size() > 1; }\n\n")
	builder.WriteString("    private void handleSystemBack() {\n")
	builder.WriteString("        if (!canNavigateBack()) return;\n")
	builder.WriteString("        Object targetRoute = cloneRoute(routeBackStack.get(routeBackStack.size() - 2));\n")
	builder.WriteString("        Object beforeRoute = cloneRoute(state.get(\"route\"));\n")
	builder.WriteString("        applyingSystemBack = true;\n")
	builder.WriteString("        try {\n")
	builder.WriteString("            if (hasTransition(\"@navigate\")) {\n")
	builder.WriteString("                dispatch(\"@navigate\", Collections.singletonList(record(entry(\"kind\", \"back\"))));\n")
	builder.WriteString("            } else {\n")
	builder.WriteString("                dispatch(\"@route_changed\", Collections.singletonList(targetRoute));\n")
	builder.WriteString("            }\n")
	builder.WriteString("        } finally {\n")
	builder.WriteString("            applyingSystemBack = false;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        reconcileAfterSystemBack(beforeRoute, state.get(\"route\"), targetRoute);\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void reconcileAfterSystemBack(Object beforeRoute, Object afterRoute, Object targetRoute) {\n")
	builder.WriteString("        if (routeKey(beforeRoute).equals(routeKey(afterRoute))) return;\n")
	builder.WriteString("        if (routeKey(afterRoute).equals(routeKey(targetRoute))) {\n")
	builder.WriteString("            routeBackStack.remove(routeBackStack.size() - 1);\n")
	builder.WriteString("            return;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        routeBackStack.remove(routeBackStack.size() - 1);\n")
	builder.WriteString("        Object last = routeBackStack.isEmpty() ? null : routeBackStack.get(routeBackStack.size() - 1);\n")
	builder.WriteString("        if (!routeKey(last).equals(routeKey(afterRoute))) routeBackStack.add(cloneRoute(afterRoute));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void reconcileRouteBackStack(Object beforeRoute, Object afterRoute) {\n")
	builder.WriteString("        if (!hasRouteState()) return;\n")
	builder.WriteString("        if (routeBackStack.isEmpty()) {\n")
	builder.WriteString("            routeBackStack.add(cloneRoute(afterRoute));\n")
	builder.WriteString("            return;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        if (routeKey(beforeRoute).equals(routeKey(afterRoute)) || applyingSystemBack) return;\n")
	builder.WriteString("        Object last = routeBackStack.get(routeBackStack.size() - 1);\n")
	builder.WriteString("        if (!routeKey(last).equals(routeKey(afterRoute))) routeBackStack.add(cloneRoute(afterRoute));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void initializeNavigationStack() {\n")
	builder.WriteString("        if (!hasRouteState() || !routeBackStack.isEmpty()) return;\n")
	builder.WriteString("        state.put(\"route\", routeValueForShape(state.get(\"route\"), routePatterns()));\n")
	builder.WriteString("        routeBackStack.add(cloneRoute(state.get(\"route\")));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private String activeRoutePath() { return pathOf(state.get(\"route\")); }\n\n")
	builder.WriteString("    private boolean shouldApply(Set<String> invalidations, List<String> states) {\n")
	builder.WriteString("        if (invalidations == null) return true;\n")
	builder.WriteString("        for (String state : states) {\n")
	builder.WriteString("            if (invalidations.contains(state)) return true;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        return false;\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private boolean booleanValue(Object value) { return Boolean.TRUE.equals(value) || \"true\".equals(String.valueOf(value)); }\n\n")
	builder.WriteString("    private double inputNumber(String value) { try { return Double.parseDouble(value); } catch (NumberFormatException ignored) { return 0.0; } }\n\n")
	builder.WriteString("    private int dp(int value) { return (int) (value * getResources().getDisplayMetrics().density); }\n")
	builder.WriteString("}\n")
	return builder.String()
}

func (renderer androidContractRenderer) renderBuildNodes(nodes []contract.Node, parent string, indent string, path []int) string {
	var builder strings.Builder
	for index, node := range nodes {
		nodePath := append(shared.CloneIntPath(path), index)
		builder.WriteString(renderer.renderBuildNode(node, parent, indent, nodePath))
	}
	return builder.String()
}

func (renderer androidContractRenderer) renderBuildNode(node contract.Node, parent string, indent string, path []int) string {
	key := androidPathKey(path)
	name := androidJavaVar("node", path)
	switch node.Kind {
	case "page":
		return renderer.renderJavaContainer(node, parent, indent, path, key, name, "FrameLayout", "")
	case "text", "#text":
		value := shared.QuoteCodeString("")
		if expr := strings.TrimSpace(node.Props["value"]); expr != "" {
			value = androidJavaEvalStringExpr(expr)
		}
		var builder strings.Builder
		builder.WriteString(indent + "TextView " + name + " = new TextView(this);\n")
		builder.WriteString(indent + name + ".setText(" + value + ");\n")
		builder.WriteString(renderer.renderStaticStyles(node, name, indent))
		builder.WriteString(indent + "views.put(" + shared.QuoteCodeString(key) + ", " + name + ");\n")
		builder.WriteString(indent + parent + ".addView(" + name + ");\n")
		return builder.String()
	case "button":
		return renderer.renderJavaButton(node, parent, indent, path, key, name)
	case "text_input":
		return renderer.renderJavaInput(node, parent, indent, path, key, name, false)
	case "number_input":
		return renderer.renderJavaInput(node, parent, indent, path, key, name, true)
	case "row":
		return renderer.renderJavaRow(node, parent, indent, path, key, name)
	case "scroll":
		return renderer.renderJavaScroll(node, parent, indent, path, key, name)
	case "stack":
		return renderer.renderJavaContainer(node, parent, indent, path, key, name, "FrameLayout", "")
	default:
		return renderer.renderJavaContainer(node, parent, indent, path, key, name, "LinearLayout", "LinearLayout.VERTICAL")
	}
}

func (renderer androidContractRenderer) renderJavaInput(node contract.Node, parent string, indent string, path []int, key string, name string, number bool) string {
	value := shared.QuoteCodeString("")
	if expr := strings.TrimSpace(node.Props["value"]); expr != "" {
		value = androidJavaEvalStringExpr(expr)
	}
	hint := shared.QuoteCodeString("")
	if expr := strings.TrimSpace(node.Props["placeholder"]); expr != "" {
		hint = androidJavaEvalStringExpr(expr)
	}
	var builder strings.Builder
	builder.WriteString(indent + "EditText " + name + " = new EditText(this);\n")
	builder.WriteString(indent + name + ".setSingleLine(true);\n")
	if number {
		builder.WriteString(indent + name + ".setInputType(InputType.TYPE_CLASS_NUMBER | InputType.TYPE_NUMBER_FLAG_DECIMAL);\n")
	}
	builder.WriteString(indent + name + ".setText(" + value + ");\n")
	builder.WriteString(indent + name + ".setHint(" + hint + ");\n")
	if route, ok := node.Events["on_change"]; ok {
		valueExpr := "editable.toString()"
		if number {
			valueExpr = "inputNumber(editable.toString())"
		}
		builder.WriteString(indent + name + ".addTextChangedListener(new TextWatcher() {\n")
		builder.WriteString(indent + "    @Override public void beforeTextChanged(CharSequence s, int start, int count, int after) {}\n")
		builder.WriteString(indent + "    @Override public void onTextChanged(CharSequence s, int start, int before, int count) {}\n")
		builder.WriteString(indent + "    @Override public void afterTextChanged(Editable editable) {\n")
		builder.WriteString(indent + "        dispatch(" + shared.QuoteCodeString(route.Name) + ", " + androidJavaContractEventArgs(route.Args, valueExpr) + ");\n")
		builder.WriteString(indent + "    }\n")
		builder.WriteString(indent + "});\n")
	}
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + shared.QuoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	return builder.String()
}

func (renderer androidContractRenderer) renderJavaScroll(node contract.Node, parent string, indent string, path []int, key string, name string) string {
	contentName := name + "Content"
	var builder strings.Builder
	builder.WriteString(indent + "ScrollView " + name + " = new ScrollView(this);\n")
	builder.WriteString(indent + "LinearLayout " + contentName + " = new LinearLayout(this);\n")
	builder.WriteString(indent + contentName + ".setOrientation(LinearLayout.VERTICAL);\n")
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + shared.QuoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	builder.WriteString(indent + name + ".addView(" + contentName + ");\n")
	builder.WriteString(renderer.renderBuildNodes(node.Children, contentName, indent, path))
	return builder.String()
}

func (renderer androidContractRenderer) renderJavaContainer(node contract.Node, parent string, indent string, path []int, key string, name string, className string, orientation string) string {
	var builder strings.Builder
	builder.WriteString(indent + className + " " + name + " = new " + className + "(this);\n")
	if orientation != "" {
		builder.WriteString(indent + name + ".setOrientation(" + orientation + ");\n")
	}
	if node.Kind == "surface" || node.Kind == "column" || node.Kind == "page" {
		builder.WriteString(indent + name + ".setPadding(0, dp(4), 0, dp(4));\n")
	}
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + shared.QuoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	builder.WriteString(renderer.renderBuildNodes(node.Children, name, indent, path))
	return builder.String()
}

func (renderer androidContractRenderer) renderJavaRow(node contract.Node, parent string, indent string, path []int, key string, name string) string {
	var builder strings.Builder
	builder.WriteString(indent + "GridLayout " + name + " = new GridLayout(this);\n")
	builder.WriteString(indent + name + ".setColumnCount(" + javaInt(androidContractRowColumnCount(node)) + ");\n")
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + shared.QuoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	builder.WriteString(renderer.renderBuildNodes(node.Children, name, indent, path))
	return builder.String()
}

func (renderer androidContractRenderer) renderJavaButton(node contract.Node, parent string, indent string, path []int, key string, name string) string {
	var builder strings.Builder
	builder.WriteString(indent + "Button " + name + " = new Button(this);\n")
	builder.WriteString(indent + name + ".setAllCaps(false);\n")
	builder.WriteString(indent + name + ".setText(" + androidContractButtonLabel(node) + ");\n")
	if route, ok := node.Events["on_press"]; ok {
		builder.WriteString(indent + name + ".setOnClickListener(view -> dispatch(" + shared.QuoteCodeString(route.Name) + ", " + androidJavaContractEventArgs(route.Args, "") + "));\n")
	}
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + shared.QuoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	return builder.String()
}

func androidContractRowColumnCount(node contract.Node) int {
	count := len(node.Children)
	if count <= 0 {
		return 1
	}
	if count > 3 {
		return 3
	}
	return count
}

func (renderer androidContractRenderer) renderStaticStyles(node contract.Node, target string, indent string) string {
	classExpr, ok := node.Props["class"]
	if !ok {
		return ""
	}
	classList, ok := staticStringFromJSExpr(classExpr)
	if !ok {
		return ""
	}
	style := renderer.styles.StyleForClassList(classList)
	if style.Empty() {
		return ""
	}
	return androidJavaStyleApplication(target, node.Kind, style, indent)
}

func (renderer androidContractRenderer) renderApplyBindings(nodes []contract.Node, bindings []contract.BindingMeta) string {
	var builder strings.Builder
	builder.WriteString("    private void applyBindings(Set<String> invalidations) {\n")
	for _, binding := range bindings {
		if binding.Prop == "key" || strings.Contains(binding.Prop, "#arg") {
			continue
		}
		node, ok := contractNodeAtPath(nodes, binding.At)
		if !ok {
			continue
		}
		expr := strings.TrimSpace(node.Props[binding.Prop])
		if expr == "" {
			continue
		}
		pathKey := androidPathKey(binding.At)
		states := androidJavaStringList(binding.States)
		builder.WriteString("        if (shouldApply(invalidations, " + states + ")) {\n")
		builder.WriteString("            View target = views.get(" + shared.QuoteCodeString(pathKey) + ");\n")
		builder.WriteString("            if (target != null) {\n")
		switch {
		case binding.Prop == "value" && (node.Kind == "text" || node.Kind == "#text"):
			builder.WriteString("                ((TextView) target).setText(" + androidJavaEvalStringExpr(expr) + ");\n")
		case binding.Prop == "enabled":
			builder.WriteString("                target.setEnabled(booleanValue(" + androidJavaEvalValueExpr(expr) + "));\n")
		case binding.Prop == "label":
			builder.WriteString("                target.setContentDescription(" + androidJavaEvalStringExpr(expr) + ");\n")
		default:
			builder.WriteString("                target.setTag(" + androidJavaEvalStringExpr(expr) + ");\n")
		}
		builder.WriteString("            }\n")
		builder.WriteString("        }\n")
	}
	builder.WriteString("    }\n\n")
	return builder.String()
}

func (renderer androidContractRenderer) renderUpdatePageVisibility(nodes []contract.Node) string {
	var builder strings.Builder
	builder.WriteString("    private void updatePageVisibility() {\n")
	builder.WriteString("        String activePath = activeRoutePath();\n")
	builder.WriteString("        int selectedRouteScore = bestRouteScore(routePatterns(), activePath);\n")
	renderer.appendPageVisibility(&builder, nodes, nil)
	builder.WriteString("    }\n\n")
	return builder.String()
}

func (renderer androidContractRenderer) appendPageVisibility(builder *strings.Builder, nodes []contract.Node, path []int) {
	for index, node := range nodes {
		nodePath := append(shared.CloneIntPath(path), index)
		if node.Kind == "page" {
			expr := strings.TrimSpace(node.Props["path"])
			if expr == "" {
				expr = `"/"`
			}
			key := androidPathKey(nodePath)
			scoreVar := "pageScore" + androidJavaPathSuffix(nodePath)
			builder.WriteString("        View page" + androidJavaPathSuffix(nodePath) + " = views.get(" + shared.QuoteCodeString(key) + ");\n")
			builder.WriteString("        int " + scoreVar + " = routeMatchScore(textValue(" + androidJavaEvalValueExpr(expr) + "), activePath);\n")
			builder.WriteString("        if (page" + androidJavaPathSuffix(nodePath) + " != null) page" + androidJavaPathSuffix(nodePath) + ".setVisibility(" + scoreVar + " >= 0 && " + scoreVar + " == selectedRouteScore ? View.VISIBLE : View.GONE);\n")
		}
		renderer.appendPageVisibility(builder, node.Children, nodePath)
	}
}

func androidContractButtonLabel(node contract.Node) string {
	for _, child := range node.Children {
		if child.Kind == "text" || child.Kind == "#text" {
			if expr := strings.TrimSpace(child.Props["value"]); expr != "" {
				return androidJavaEvalStringExpr(expr)
			}
		}
	}
	if expr := strings.TrimSpace(node.Props["label"]); expr != "" {
		return androidJavaEvalStringExpr(expr)
	}
	return shared.QuoteCodeString("Button")
}

func androidContractStateInitializers(states []contract.State) string {
	var builder strings.Builder
	for _, state := range states {
		builder.WriteString("        state.put(" + shared.QuoteCodeString(state.Name) + ", " + androidJavaInitialValue(state.Initial) + ");\n")
	}
	return builder.String()
}

func androidContractTransitionTable(states []contract.State) string {
	var builder strings.Builder
	builder.WriteString("    private List<NovaTransition> transitions() {\n")
	builder.WriteString("        return Arrays.asList(\n")
	for _, state := range states {
		for _, transition := range state.Transitions {
			builder.WriteString("            new NovaTransition(")
			builder.WriteString(shared.QuoteCodeString(state.Name))
			builder.WriteString(", ")
			builder.WriteString(shared.QuoteCodeString(transition.Event))
			builder.WriteString(", ")
			builder.WriteString(androidJavaStringList(transition.Params))
			builder.WriteString(", ")
			builder.WriteString(shared.QuoteCodeString(transition.Expression))
			builder.WriteString("),\n")
		}
	}
	builder.WriteString("            new NovaTransition(\"\", \"\", Collections.emptyList(), \"\")\n")
	builder.WriteString("        );\n")
	builder.WriteString("    }\n\n")
	return builder.String()
}

func androidContractPagePaths(view contract.View) []string {
	seen := make(map[string]bool)
	paths := make([]string, 0)
	var walk func(nodes []contract.Node)
	walk = func(nodes []contract.Node) {
		for _, node := range nodes {
			if node.Kind == "page" {
				if path, ok := staticStringFromJSExpr(node.Props["path"]); ok {
					normalized := routing.DescribePattern(path).Pattern
					if !seen[normalized] {
						seen[normalized] = true
						paths = append(paths, normalized)
					}
				}
			}
			walk(node.Children)
		}
	}
	walk(view.Nodes)
	sort.Strings(paths)
	return paths
}

func routesFromContract(view contract.View, config targetConfig) string {
	paths := androidContractPagePaths(view)
	if len(paths) == 0 {
		paths = []string{"/"}
	}
	names := make(map[string]int)
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\npublic final class NovaRoutes {\n    private NovaRoutes() {}\n")
	for _, path := range paths {
		baseName := androidRouteConstName(path)
		names[baseName]++
		name := baseName
		if names[baseName] > 1 {
			name = baseName + "_" + javaInt(names[baseName])
		}
		builder.WriteString("    public static final String ")
		builder.WriteString(strings.ToUpper(name))
		builder.WriteString(" = ")
		builder.WriteString(shared.QuoteCodeString(path))
		builder.WriteString(";\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

func appFromContract(name string, app contract.App, config targetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "NovaApp"
	}
	perms := androidJavaStringList(app.Permissions)
	return "package " + config.Namespace + ";\n\n" +
		"public final class NovaApp {\n" +
		"    public static final int CONTRACT_VERSION = " + strconv.Itoa(app.V) + ";\n" +
		"    public final String name = " + shared.QuoteCodeString(name) + ";\n" +
		"    public final String target = " + shared.QuoteCodeString(app.Target) + ";\n" +
		"    public final String entry = " + shared.QuoteCodeString(app.Entry) + ";\n" +
		"    public final java.util.List<String> permissions = " + perms + ";\n" +
		"    private NovaApp() {}\n" +
		"}\n"
}

func contractNodeAtPath(nodes []contract.Node, path []int) (contract.Node, bool) {
	list := nodes
	var node contract.Node
	for _, index := range path {
		if index < 0 || index >= len(list) {
			return contract.Node{}, false
		}
		node = list[index]
		list = node.Children
	}
	return node, true
}

func staticStringFromJSExpr(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", false
	}
	if unquoted, err := strconv.Unquote(expr); err == nil {
		return unquoted, true
	}
	return "", false
}

func androidJavaEvalStringExpr(jsExpr string) string {
	return "textValue(evaluate(" + shared.QuoteCodeString(jsExpr) + ", state, Collections.emptyMap()))"
}

func androidJavaEvalValueExpr(jsExpr string) string {
	return "evaluate(" + shared.QuoteCodeString(jsExpr) + ", state, Collections.emptyMap())"
}

func androidJavaContractEventArgs(args []string, implicitValueExpr string) string {
	if len(args) == 0 {
		return "Collections.emptyList()"
	}
	values := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "$value" {
			if implicitValueExpr == "" {
				values = append(values, "null")
			} else {
				values = append(values, implicitValueExpr)
			}
			continue
		}
		values = append(values, androidJavaEvalValueExpr(arg))
	}
	return "Arrays.<Object>asList(" + strings.Join(values, ", ") + ")"
}

func androidContractLifecycleHooks(app contract.App, hydration storageHydrationChain) string {
	if len(app.Lifecycles) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("    @Override\n")
	builder.WriteString("    public void schedulerApplyStateCommit(Set<String> invalidations) {\n")
	builder.WriteString("        if (mounting) {\n")
	builder.WriteString("            mountInvalidations.addAll(invalidations);\n")
	builder.WriteString("            return;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        applyStateCommit(invalidations);\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    protected void schedulerMountLifecycles() {\n")
	builder.WriteString("        mounting = true;\n")
	builder.WriteString("        mountInvalidations.clear();\n")
	builder.WriteString("        try {\n")
	builder.WriteString("            runContractLifecycles(\"mount\", \"\", Collections.emptyMap());\n")
	builder.WriteString("            scheduler.drain();\n")
	if hydration.AfterSkipEvents != nil {
		builder.WriteString("            hydratePersistedState();\n")
	}
	builder.WriteString("        } finally {\n")
	builder.WriteString("            mounting = false;\n")
	builder.WriteString("            applyStateCommit(mountInvalidations);\n")
	builder.WriteString("            mountInvalidations.clear();\n")
	builder.WriteString("        }\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    @Override\n")
	builder.WriteString("    public void schedulerBeforeEvent(String eventName, List<Object> args) {\n")
	builder.WriteString("        runContractLifecycles(\"before\", eventName, schedulerPayload(eventName, args));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    @Override\n")
	builder.WriteString("    public void schedulerAfterEvent(String eventName, List<Object> args) {\n")
	if hydration.AfterSkipEvents != nil {
		builder.WriteString("        if (mounting && HYDRATION_SKIP_AFTER.contains(eventName)) return;\n")
	}
	builder.WriteString("        runContractLifecycles(\"after\", eventName, schedulerPayload(eventName, args));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private Map<String, Object> schedulerPayload(String eventName, List<Object> args) {\n")
	builder.WriteString("        Map<String, Object> payload = new LinkedHashMap<>();\n")
	for _, state := range app.Model.States {
		for _, transition := range state.Transitions {
			builder.WriteString("        if (\"" + transition.Event + "\".equals(eventName)) {\n")
			for index, param := range transition.Params {
				builder.WriteString("            payload.put(" + shared.QuoteCodeString(param) + ", ")
				builder.WriteString("args.size() > " + strconv.Itoa(index) + " ? args.get(" + strconv.Itoa(index) + ") : null);\n")
			}
			builder.WriteString("        }\n")
		}
	}
	builder.WriteString("        return payload;\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void runContractLifecycles(String phase, String eventName, Map<String, Object> payload) {\n")
	builder.WriteString("        Map<String, Object> snapshot = new LinkedHashMap<>(state);\n")
	for _, lifecycle := range app.Lifecycles {
		builder.WriteString("        if (\"" + lifecycle.Phase + "\".equals(phase)")
		if lifecycle.Event != "" {
			builder.WriteString(" && \"" + lifecycle.Event + "\".equals(eventName)")
		}
		builder.WriteString(") {\n")
		for _, step := range lifecycle.Steps {
			if step.Emit != nil {
				builder.WriteString("            scheduler.enqueueLifecycle(" + shared.QuoteCodeString(lifecycle.Owner) + ", ")
				builder.WriteString(shared.QuoteCodeString(step.Emit.Name) + ", ")
				builder.WriteString(androidJavaContractLifecycleEventArgs(step.Emit.Args) + ");\n")
				continue
			}
			if step.External != nil {
				builder.WriteString("            invokeExternal(" + shared.QuoteCodeString(lifecycle.Owner) + ", ")
				builder.WriteString(shared.QuoteCodeString(step.External.EffectID) + ", ")
				builder.WriteString(androidJavaExternalInputMap(step.External.Input) + ", ")
				builder.WriteString(shared.QuoteCodeString(step.External.OnSuccess) + ", ")
				builder.WriteString(shared.QuoteCodeString(step.External.OnFailure) + ");\n")
			}
		}
		builder.WriteString("        }\n")
	}
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void invokeExternal(String owner, String effectId, Map<String, Object> input, String onSuccess, String onFailure) {\n")
	builder.WriteString("        try {\n")
	builder.WriteString("            Object output = NovaExternalAdapters.invoke(this, effectId, input);\n")
	builder.WriteString("            if (onSuccess != null && !onSuccess.isEmpty()) {\n")
	builder.WriteString("                scheduler.enqueueLifecycle(owner, onSuccess, output == null ? Collections.emptyList() : Collections.singletonList(output));\n")
	builder.WriteString("            }\n")
	builder.WriteString("        } catch (Exception error) {\n")
	builder.WriteString("            if (onFailure != null && !onFailure.isEmpty()) {\n")
	builder.WriteString("                scheduler.enqueueLifecycle(owner, onFailure, Collections.singletonList(error.getMessage()));\n")
	builder.WriteString("            }\n")
	builder.WriteString("        }\n")
	builder.WriteString("    }\n\n")
	return builder.String()
}

func androidJavaEvalLifecycleExpr(jsExpr string) string {
	return "evaluate(" + shared.QuoteCodeString(jsExpr) + ", snapshot, payload)"
}

func androidJavaContractLifecycleEventArgs(args []string) string {
	if len(args) == 0 {
		return "Collections.emptyList()"
	}
	values := make([]string, 0, len(args))
	for _, arg := range args {
		values = append(values, androidJavaEvalLifecycleExpr(arg))
	}
	return "Arrays.<Object>asList(" + strings.Join(values, ", ") + ")"
}

func androidJavaExternalInputMap(input map[string]string) string {
	if len(input) == 0 {
		return "Collections.emptyMap()"
	}
	names := make([]string, 0, len(input))
	for name := range input {
		names = append(names, name)
	}
	sort.Strings(names)
	pairs := make([]string, 0, len(names))
	for _, name := range names {
		pairs = append(pairs, "entry("+shared.QuoteCodeString(name)+", "+androidJavaEvalLifecycleExpr(input[name])+")")
	}
	return "record(" + strings.Join(pairs, ", ") + ")"
}
