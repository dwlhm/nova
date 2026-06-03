package androidcodegen

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/contract"
)

func TestDetectStorageHydrationChain(t *testing.T) {
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
	chain, ok := detectStorageHydrationChain(lifecycles)
	if !ok {
		t.Fatal("expected hydration chain")
	}
	if len(chain.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(chain.Steps))
	}
	if chain.TerminalEvent != "@ready" {
		t.Fatalf("terminal = %q", chain.TerminalEvent)
	}
	if !strings.Contains(androidContractStorageHydration(chain), "HYDRATION_SKIP_AFTER") {
		t.Fatal("expected hydration skip set in generated Java")
	}
}

func TestDetectStorageHydrationChainFromFixtureApp(t *testing.T) {
	app := contract.App{
		Lifecycles: []contract.Lifecycle{
			{Phase: "mount", Steps: []contract.LifecycleStep{{Emit: &contract.LifecycleEmit{Name: "@restore"}}}},
			{
				Phase: "after",
				Event: "@restore",
				Steps: []contract.LifecycleStep{{
					External: &contract.LifecycleExternal{
						EffectID:  "@env/storage#load",
						Input:     map[string]string{"key": `"state"`},
						OnSuccess: "@loaded",
					},
				}},
			},
			{
				Phase: "after",
				Event: "@loaded",
				Steps: []contract.LifecycleStep{{Emit: &contract.LifecycleEmit{Name: "@ready"}}},
			},
		},
	}
	chain, ok := detectStorageHydrationChain(app.Lifecycles)
	if !ok {
		t.Fatal("expected hydration chain")
	}
	if chain.TerminalEvent != "@ready" {
		t.Fatalf("terminal = %q", chain.TerminalEvent)
	}
}
