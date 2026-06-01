package ir

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/capability"
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
					Event: parser.EventPattern{Name: "@loaded"},
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
					{Type: lexer.IDENT, Literal: "onSuccess"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.SIGNAL, Literal: "@loaded"},
					{Type: lexer.IDENT, Literal: "onFailure"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.SIGNAL, Literal: "@load_failed"},
				},
			}},
		}, {
			Phase: "after",
			Event: "@loaded_all_line4",
			Statements: []parser.Statement{{
				Tokens: []lexer.Token{
					{Type: lexer.VOID}, {Type: lexer.MAP_ARROW}, {Type: lexer.SIGNAL, Literal: "@ledger_ready"},
				},
			}},
		}},
	}
	manifests := []capability.Manifest{
		capability.BuildManifest("src/FinanceStore.nova", store),
		capability.BuildManifest("src/FinancePersistence.nova", persistence),
	}
	lifecycles := buildContractLifecycles(
		[]ModuleRef{{Path: "src/FinanceStore.nova"}, {Path: "src/FinancePersistence.nova"}},
		map[string]parser.File{
			"src/FinanceStore.nova":       store,
			"src/FinancePersistence.nova": persistence,
		},
		map[string]bool{"payload": true},
	)
	events := contractEvents(
		[]string{"src/FinanceStore.nova", "src/FinancePersistence.nova"},
		manifests,
		lifecycles,
	)

	emitters := map[string][]string{}
	for _, event := range events {
		emitters[event.Name] = append([]string(nil), event.Emitters...)
	}
	if !containsString(emitters["@loaded"], "FinancePersistence") {
		t.Fatalf("@loaded emitters = %v, want FinancePersistence", emitters["@loaded"])
	}
	if !containsString(emitters["@loaded"], "src/FinanceStore.nova") {
		t.Fatalf("@loaded emitters = %v, want FinanceStore", emitters["@loaded"])
	}
	if !containsString(emitters["@ledger_ready"], "FinancePersistence") {
		t.Fatalf("@ledger_ready emitters = %v, want FinancePersistence", emitters["@ledger_ready"])
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
