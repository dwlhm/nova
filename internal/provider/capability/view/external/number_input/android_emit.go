package numberinput

import (
	"strings"

	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	irandroid "github.com/dwlhm/nova/internal/provider/capability/view/ir/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func emitAndroid(renderer *androidcodegen.Renderer, node irandroid.Node, parent, indent string, key, name string) string {
	source := node.Source
	value := shared.QuoteCodeString("")
	if expr := strings.TrimSpace(source.Props["value"]); expr != "" {
		value = androidcodegen.JavaEvalStringExpr(expr)
	}
	hint := shared.QuoteCodeString("")
	if expr := strings.TrimSpace(source.Props["placeholder"]); expr != "" {
		hint = androidcodegen.JavaEvalStringExpr(expr)
	}
	var builder strings.Builder
	builder.WriteString(indent + "EditText " + name + " = new EditText(this);\n")
	builder.WriteString(indent + name + ".setSingleLine(true);\n")
	builder.WriteString(indent + name + ".setInputType(InputType.TYPE_CLASS_NUMBER | InputType.TYPE_NUMBER_FLAG_DECIMAL);\n")
	builder.WriteString(indent + name + ".setText(" + value + ");\n")
	builder.WriteString(indent + name + ".setHint(" + hint + ");\n")
	if route, ok := source.Events["on_change"]; ok {
		builder.WriteString(indent + name + ".addTextChangedListener(new TextWatcher() {\n")
		builder.WriteString(indent + "    @Override public void beforeTextChanged(CharSequence s, int start, int count, int after) {}\n")
		builder.WriteString(indent + "    @Override public void onTextChanged(CharSequence s, int start, int before, int count) {}\n")
		builder.WriteString(indent + "    @Override public void afterTextChanged(Editable editable) {\n")
		builder.WriteString(indent + "        dispatch(" + shared.QuoteCodeString(route.Name) + ", " + androidcodegen.JavaContractEventArgs(route.Args, "inputNumber(editable.toString())") + ");\n")
		builder.WriteString(indent + "    }\n")
		builder.WriteString(indent + "});\n")
	}
	builder.WriteString(renderer.RenderStaticStyles(source, key, name, indent))
	builder.WriteString(indent + "views.put(" + shared.QuoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	return builder.String()
}
