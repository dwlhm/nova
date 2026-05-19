package artifact

import (
	"strings"

	"github.com/dwlhm/nova/internal/build"
)

func androidSettings(name string) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	return "pluginManagement {\n    repositories {\n        google()\n        mavenCentral()\n        gradlePluginPortal()\n    }\n}\n\ndependencyResolutionManagement {\n    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)\n    repositories {\n        google()\n        mavenCentral()\n    }\n}\n\nrootProject.name = " + quoteKotlin(name) + "\ninclude(\":app\")\n"
}

func androidGradle(name string) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	return "buildscript {\n    repositories {\n        google()\n        mavenCentral()\n    }\n    dependencies {\n        classpath(\"com.android.tools.build:gradle:8.12.3\")\n    }\n}\n\n// Generated Nova Android project for " + escapeGradleComment(name) + ".\n"
}

func androidAppGradle() string {
	return "import com.android.build.api.dsl.ApplicationExtension\n\napply(plugin = \"com.android.application\")\n\nextensions.configure<ApplicationExtension>(\"android\") {\n    namespace = \"nova.generated\"\n    compileSdk = 35\n\n    defaultConfig {\n        applicationId = \"nova.generated.counter\"\n        minSdk = 23\n        targetSdk = 35\n        versionCode = 1\n        versionName = \"1.0\"\n    }\n\n    compileOptions {\n        sourceCompatibility = JavaVersion.VERSION_17\n        targetCompatibility = JavaVersion.VERSION_17\n    }\n}\n"
}

func androidManifest() string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\">\n    <application android:theme=\"@style/Theme.Nova\" android:label=\"Nova Counter\">\n        <activity android:name=\".MainActivity\" android:exported=\"true\">\n            <intent-filter>\n                <action android:name=\"android.intent.action.MAIN\" />\n                <category android:name=\"android.intent.category.LAUNCHER\" />\n            </intent-filter>\n        </activity>\n    </application>\n</manifest>\n"
}

func androidStyles() string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<resources>\n    <style name=\"Theme.Nova\" parent=\"android:style/Theme.Material.Light.NoActionBar\">\n        <item name=\"android:windowActionBar\">false</item>\n        <item name=\"android:windowNoTitle\">true</item>\n    </style>\n</resources>\n"
}

func androidMainActivity(name string, bundle irBundle) string {
	counter := findCounterState(bundle.Model)
	if counter.Name == "" {
		return androidPlaceholderActivity(name, bundle.Target)
	}
	return "package nova.generated;\n\nimport android.app.Activity;\nimport android.graphics.Typeface;\nimport android.os.Bundle;\nimport android.view.Gravity;\nimport android.view.ViewGroup;\nimport android.widget.Button;\nimport android.widget.LinearLayout;\nimport android.widget.TextView;\n\npublic final class MainActivity extends Activity {\n    private int count = " + javaInitial(counter.Initial) + ";\n    private TextView valueText;\n\n    @Override\n    protected void onCreate(Bundle savedInstanceState) {\n        super.onCreate(savedInstanceState);\n\n        LinearLayout root = new LinearLayout(this);\n        root.setOrientation(LinearLayout.VERTICAL);\n        root.setGravity(Gravity.CENTER);\n        root.setPadding(48, 48, 48, 48);\n        root.setLayoutParams(new LinearLayout.LayoutParams(\n            ViewGroup.LayoutParams.MATCH_PARENT,\n            ViewGroup.LayoutParams.MATCH_PARENT\n        ));\n\n        TextView title = new TextView(this);\n        title.setText(" + quoteJava(displayName(name)) + ");\n        title.setTextSize(22);\n        title.setTypeface(Typeface.DEFAULT_BOLD);\n        title.setGravity(Gravity.CENTER);\n        root.addView(title, new LinearLayout.LayoutParams(\n            ViewGroup.LayoutParams.MATCH_PARENT,\n            ViewGroup.LayoutParams.WRAP_CONTENT\n        ));\n\n        valueText = new TextView(this);\n        valueText.setTextSize(56);\n        valueText.setTypeface(Typeface.DEFAULT_BOLD);\n        valueText.setGravity(Gravity.CENTER);\n        updateValue();\n        root.addView(valueText, new LinearLayout.LayoutParams(\n            ViewGroup.LayoutParams.MATCH_PARENT,\n            ViewGroup.LayoutParams.WRAP_CONTENT\n        ));\n\n        LinearLayout actions = new LinearLayout(this);\n        actions.setOrientation(LinearLayout.HORIZONTAL);\n        actions.setGravity(Gravity.CENTER);\n\n        Button decrement = actionButton(\"-\");\n        decrement.setOnClickListener(view -> {\n            count = count - 1;\n            updateValue();\n        });\n        actions.addView(decrement);\n\n        Button reset = actionButton(\"Reset\");\n        reset.setOnClickListener(view -> {\n            count = 0;\n            updateValue();\n        });\n        actions.addView(reset);\n\n        Button increment = actionButton(\"+\");\n        increment.setOnClickListener(view -> {\n            count = count + 1;\n            updateValue();\n        });\n        actions.addView(increment);\n\n        root.addView(actions, new LinearLayout.LayoutParams(\n            ViewGroup.LayoutParams.MATCH_PARENT,\n            ViewGroup.LayoutParams.WRAP_CONTENT\n        ));\n\n        setContentView(root);\n    }\n\n    private Button actionButton(String label) {\n        Button button = new Button(this);\n        button.setText(label);\n        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1);\n        params.setMargins(8, 24, 8, 0);\n        button.setLayoutParams(params);\n        return button;\n    }\n\n    private void updateValue() {\n        valueText.setText(Integer.toString(count));\n    }\n}\n"
}

func androidPlaceholderActivity(name string, target string) string {
	return "package nova.generated;\n\nimport android.app.Activity;\nimport android.os.Bundle;\nimport android.widget.TextView;\n\npublic final class MainActivity extends Activity {\n    @Override\n    protected void onCreate(Bundle savedInstanceState) {\n        super.onCreate(savedInstanceState);\n        TextView text = new TextView(this);\n        text.setText(" + quoteJava(displayName(name)+" ("+target+")") + ");\n        setContentView(text);\n    }\n}\n"
}

func androidApp(name string, target string) string {
	if strings.TrimSpace(name) == "" {
		name = "NovaApp"
	}
	return "package nova.generated\n\nclass NovaApp {\n    val name = " + quoteKotlin(name) + "\n    val target = " + quoteKotlin(target) + "\n}\n"
}

func androidRoutes() string {
	return "package nova.generated\n\nobject NovaRoutes {\n    const val root = \"/\"\n}\n"
}

func androidExternalBindings(operations []build.ResolvedExternalOperation) string {
	names := externalOperationNames(operations)
	var builder strings.Builder
	builder.WriteString("package nova.generated\n\nobject NovaExternalBindings {\n")
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

func findCounterState(model appModel) stateModel {
	for _, state := range model.States {
		if state.Name == "count" && state.Type == "number" {
			return state
		}
	}
	for _, state := range model.States {
		if state.Type == "number" {
			return state
		}
	}
	return stateModel{}
}

func kotlinInitial(expression string) string {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "0"
	}
	expression = strings.TrimPrefix(expression, "state.")
	if strings.ContainsAny(expression, "+-*/() ") {
		return "0"
	}
	return expression
}

func javaInitial(expression string) string {
	return kotlinInitial(expression)
}

func quoteJava(value string) string {
	return quoteKotlin(value)
}

func displayName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "Nova Counter"
	}
	return name
}
