package app

import (
	"testing"
)

func TestParseNavigationActionBack(t *testing.T) {
	action, err := ParseNavigationAction(map[string]any{"kind": "back"})
	if err != nil {
		t.Fatalf("parse back action: %v", err)
	}
	if action.Kind != NavigateBack {
		t.Fatalf("kind = %s, want back", action.Kind)
	}
}

func TestParseNavigationActionPushRequiresRoute(t *testing.T) {
	_, err := ParseNavigationAction(map[string]any{"kind": "push"})
	if err == nil {
		t.Fatal("expected push without route to fail")
	}
}

func TestActionFromEventPayloadReadsNamedActionField(t *testing.T) {
	action, err := ActionFromEventPayload(map[string]any{
		"action": map[string]any{"kind": "replace", "route": map[string]any{"path": "/settings"}},
	})
	if err != nil {
		t.Fatalf("ActionFromEventPayload: %v", err)
	}
	if action.Kind != NavigateReplace || action.Route == nil || action.Route.Path != "/settings" {
		t.Fatalf("action = %+v", action)
	}
}
