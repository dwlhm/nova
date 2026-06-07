package ir

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/capability"
	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func TestContractEventsIncludesLifecycleOwnersForCompletionAndEmit(t *testing.T) {
	store := parser.File{
		ContractStates: []parser.ContractStateDecl{{
			Name: "AppState",
			States: []parser.StateDecl{{
				Name:    "payload",
				Type:    parser.TypeRef{Text: "string"},
				Initial: []lexer.Token{{Type: lexer.STRING, Literal: "demo"}},
				Transitions: []parser.TransitionRule{{
					Event: parser.EventPattern{Name: "@ledger_hydrated", Params: []parser.FieldDecl{{Name: "value", Type: parser.TypeRef{Text: "unknown"}}}},
					Expr:  []lexer.Token{{Type: lexer.IDENT, Literal: "payload"}},
				}},
			}},
		}},
	}
	persistence := parser.File{
		ExternalImports: []parser.ExternalImportDecl{{
			Name: "storage",
			From: "@env/storage",
		}},
		Lifecycles: []parser.LifecycleDecl{{
			Phase: "after",
			Event: "@restore",
			Statements: []parser.Statement{{
				Tokens: []lexer.Token{
					{Type: lexer.VOID}, {Type: lexer.PIPE_FWD},
					{Type: lexer.IDENT, Literal: "storage"}, {Type: lexer.DOT}, {Type: lexer.IDENT, Literal: "load"},
					{Type: lexer.IDENT, Literal: "key"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.STRING, Literal: "demo"},
					{Type: lexer.IDENT, Literal: "onSuccess"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.SIGNAL, Literal: "@ledger_hydrated"},
					{Type: lexer.IDENT, Literal: "onFailure"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.SIGNAL, Literal: "@load_failed"},
				},
			}},
		}, {
			Phase: "after",
			Event: "@ledger_hydrated",
			Statements: []parser.Statement{{
				Tokens: []lexer.Token{
					{Type: lexer.VOID}, {Type: lexer.MAP_ARROW}, {Type: lexer.SIGNAL, Literal: "@ledger_synced"},
				},
			}},
		}},
	}
	manifests := []capability.Manifest{
		capability.BuildManifest("src/FinanceStore.nova", store),
		capability.BuildManifest("src/FinancePersistence.nova", persistence),
	}
	sources := map[string]parser.File{
		"src/FinanceStore.nova":       store,
		"src/FinancePersistence.nova": persistence,
	}
	lifecycles, diagnostics := buildContractLifecycles(
		[]ModuleRef{{Path: "src/FinanceStore.nova"}, {Path: "src/FinancePersistence.nova"}},
		sources,
		buildExprRegistry(sources),
		map[string]bool{"payload": true},
	)
	if len(diagnostics) > 0 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
	events := contractEvents(
		[]string{"src/FinanceStore.nova", "src/FinancePersistence.nova"},
		manifests,
		lifecycles,
	)

	emitters := map[string][]string{}
	for _, event := range events {
		emitters[event.Name] = append([]string(nil), event.Emitters...)
	}
	if !containsString(emitters["@ledger_hydrated"], "FinancePersistence") {
		t.Fatalf("@ledger_hydrated emitters = %v, want FinancePersistence", emitters["@ledger_hydrated"])
	}
	if !containsString(emitters["@ledger_hydrated"], "src/FinanceStore.nova") {
		t.Fatalf("@ledger_hydrated emitters = %v, want FinanceStore", emitters["@ledger_hydrated"])
	}
	if !containsString(emitters["@ledger_synced"], "FinancePersistence") {
		t.Fatalf("@ledger_synced emitters = %v, want FinancePersistence", emitters["@ledger_synced"])
	}
}

func TestMergeNavigationPlatformEmittersAddsRuntimeSources(t *testing.T) {
	events := mergeNavigationPlatformEmitters([]contract.EventContract{{
		Name:     "@route_changed",
		Emitters: []string{"src/App.nova"},
	}}, contract.Model{States: []contract.State{{Name: "route"}}})

	var routeChanged contract.EventContract
	for _, event := range events {
		if event.Name == "@route_changed" {
			routeChanged = event
			break
		}
	}
	if routeChanged.Name == "" {
		t.Fatalf("missing @route_changed in %+v", events)
	}
	for _, want := range []string{"platform", "renderer", "@nova/navigation", "src/App.nova"} {
		if !containsString(routeChanged.Emitters, want) {
			t.Fatalf("emitters = %v, want %s", routeChanged.Emitters, want)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
