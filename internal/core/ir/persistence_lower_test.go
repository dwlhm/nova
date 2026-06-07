package ir

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/contract"
)

func TestBuildHydrationManifest(t *testing.T) {
	lifecycles := []contract.Lifecycle{
		{Phase: "mount", Steps: []contract.LifecycleStep{{Emit: &contract.LifecycleEmit{Name: "@restore"}}}},
		{
			Phase: "after",
			Event: "@restore",
			Steps: []contract.LifecycleStep{{
				External: &contract.LifecycleExternal{
					EffectID:  "@env/storage#load",
					Input:     map[string]string{"key": `"a"`},
					OnSuccess: "@loaded_a",
					OnFailure: "@load_failed",
				},
			}},
		},
		{
			Phase: "after",
			Event: "@loaded_a",
			Steps: []contract.LifecycleStep{{
				External: &contract.LifecycleExternal{
					EffectID:  "@env/storage#load",
					Input:     map[string]string{"key": `"b"`},
					OnSuccess: "@loaded_b",
					OnFailure: "@load_failed",
				},
			}},
		},
		{
			Phase: "after",
			Event: "@loaded_b",
			Steps: []contract.LifecycleStep{{Emit: &contract.LifecycleEmit{Name: "@ready"}}},
		},
	}
	manifest := buildHydrationManifest(lifecycles)
	if manifest == nil {
		t.Fatal("expected hydration manifest")
	}
	if manifest.BootstrapEvent != "@restore" {
		t.Fatalf("bootstrap = %q", manifest.BootstrapEvent)
	}
	if len(manifest.Loads) != 2 {
		t.Fatalf("loads = %d, want 2", len(manifest.Loads))
	}
	if manifest.TerminalEvent != "@ready" {
		t.Fatalf("terminal = %q", manifest.TerminalEvent)
	}
	if len(manifest.SkipAfterEvents) != 3 {
		t.Fatalf("skipAfterEvents = %v", manifest.SkipAfterEvents)
	}
}

func TestBuildHydrationManifestFromFinanceShape(t *testing.T) {
	lifecycles := []contract.Lifecycle{
		{Phase: "mount", Steps: []contract.LifecycleStep{{Emit: &contract.LifecycleEmit{Name: "@restore"}}}},
		{
			Phase: "after",
			Event: "@restore",
			Steps: []contract.LifecycleStep{{
				External: &contract.LifecycleExternal{
					EffectID:  "@env/storage#load",
					Input:     map[string]string{"key": `"finance_ledger_v1"`},
					OnSuccess: "@ledger_hydrated",
					OnFailure: "@load_failed",
				},
			}},
		},
		{
			Phase: "after",
			Event: "@ledger_hydrated",
			Steps: []contract.LifecycleStep{{Emit: &contract.LifecycleEmit{Name: "@ledger_synced"}}},
		},
	}
	manifest := buildHydrationManifest(lifecycles)
	if manifest == nil {
		t.Fatal("expected hydration manifest")
	}
	if manifest.TerminalEvent != "@ledger_synced" {
		t.Fatalf("terminal = %q", manifest.TerminalEvent)
	}
}
