package android

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/contract"
)

func TestAndroidManifestIncludesDeepLinkIntentWhenRouteStateExists(t *testing.T) {
	config := targetConfig{Namespace: "nova.generated", Theme: "Theme", Label: "demo"}
	withRoute := androidManifestXML(config, nil, true)
	if !strings.Contains(withRoute, `android:scheme="nova"`) || !strings.Contains(withRoute, "android.intent.action.VIEW") {
		t.Fatalf("route apps must declare nova:// deep link intent filter:\n%s", withRoute)
	}

	withoutRoute := androidManifestXML(config, nil, false)
	if strings.Contains(withoutRoute, `android:scheme="nova"`) {
		t.Fatalf("apps without route state must not declare deep link intent filter")
	}
}

func TestHasRouteState(t *testing.T) {
	if !hasRouteState(contract.App{Model: contract.Model{States: []contract.State{{Name: "route"}}}}) {
		t.Fatal("expected route state detection")
	}
	if hasRouteState(contract.App{Model: contract.Model{States: []contract.State{{Name: "count"}}}}) {
		t.Fatal("expected no route state")
	}
}
