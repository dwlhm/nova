package internal

import (
	"testing"
)

func TestDefaultCapabilityRegistryIncludesIntegratedPrimitives(t *testing.T) {
	ids := make(map[string]bool)
	for _, id := range NewDefaultHost().CapabilityIDs() {
		ids[id] = true
	}
	for _, want := range []string{
		"style",
		"runtime",
		"text",
		"button",
		"page",
		"row",
		"scroll",
		"stack",
		"surface",
		"column",
		"text_input",
		"number_input",
		"select",
	} {
		if !ids[want] {
			t.Fatalf("registry missing capability %q", want)
		}
	}
}
