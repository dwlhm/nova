package rendererjava

import (
	"strings"
	"testing"
)

func TestSourceIncludesPrimitiveRegistry(t *testing.T) {
	source := Source("nova.generated.demo")
	if !strings.Contains(source, "definePrimitive") {
		t.Fatal("NovaRenderer must expose definePrimitive")
	}
	if !strings.Contains(source, "hasPrimitive") {
		t.Fatal("NovaRenderer must expose hasPrimitive")
	}
}
