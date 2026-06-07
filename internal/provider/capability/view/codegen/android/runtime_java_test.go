package androidcodegen

import (
	"strings"
	"testing"

	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
)

func TestJavaRuntimeEvaluateRecordWithoutOuterParens(t *testing.T) {
	source := JavaRuntime(androidtarget.Config{Namespace: "nova.generated"})
	if !strings.Contains(source, `trimmed.startsWith("{") && trimmed.endsWith("}")`) {
		t.Fatalf("NovaRuntime.evaluate must accept braced record literals after outer parens are stripped")
	}
	if !strings.Contains(source, `expression.startsWith("{") && expression.endsWith("}")`) {
		t.Fatalf("NovaRuntime.evaluateRecord must parse braced record literals")
	}
}

func TestJavaRuntimeRouteValueForShapeDoesNotReturnBarePathString(t *testing.T) {
	source := JavaRuntime(androidtarget.Config{Namespace: "nova.generated"})
	if strings.Contains(source, `if (value instanceof String) return route.get("path");`) {
		t.Fatalf("routeValueForShape must normalize string routes to map shape, not return bare path")
	}
}
