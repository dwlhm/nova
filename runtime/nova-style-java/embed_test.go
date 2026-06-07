package stylejava

import (
	"strings"
	"testing"
)

func TestSourceIncludesApplyHelpers(t *testing.T) {
	source := Source("nova.generated.finance")
	if !strings.HasPrefix(source, "package nova.generated.finance;\n\n") {
		t.Fatalf("unexpected package header:\n%s", source[:80])
	}
	if !strings.Contains(source, "applyDynamicClasses") {
		t.Fatalf("NovaStyle must expose applyDynamicClasses")
	}
	if !strings.Contains(source, "applyWithStates") {
		t.Fatalf("NovaStyle must expose applyWithStates")
	}
}
