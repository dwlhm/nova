package ir

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func TestBuildContractLifecyclesLowersEmitAndExternal(t *testing.T) {
	file := parser.File{
		ExternalImports: []parser.ExternalImportDecl{{
			Name: "storage",
			From: "@env/storage",
		}},
		Lifecycles: []parser.LifecycleDecl{
			{
				Phase: "mount",
				Statements: []parser.Statement{{
					Tokens: []lexer.Token{
						{Type: lexer.VOID}, {Type: lexer.MAP_ARROW}, {Type: lexer.SIGNAL, Literal: "@restore"},
						{Type: lexer.SEMICOLON},
					},
				}},
			},
			{
				Phase: "after",
				Event: "@save",
				Statements: []parser.Statement{{
					Tokens: []lexer.Token{
						{Type: lexer.IDENT, Literal: "exportJson"}, {Type: lexer.PIPE_FWD},
						{Type: lexer.IDENT, Literal: "storage"}, {Type: lexer.DOT}, {Type: lexer.IDENT, Literal: "set"},
						{Type: lexer.IDENT, Literal: "key"}, {Type: lexer.ASSIGN_IN},
						{Type: lexer.STRING, Literal: "finance_ledger_export"},
						{Type: lexer.IDENT, Literal: "value"}, {Type: lexer.ASSIGN_IN},
						{Type: lexer.IDENT, Literal: "exportJson"},
						{Type: lexer.SEMICOLON},
					},
				}},
			},
		},
	}
	lifecycles := buildContractLifecycles(
		[]ModuleRef{{Path: "src/FinancePersistence.nova"}},
		map[string]parser.File{"src/FinancePersistence.nova": file},
		map[string]bool{"exportJson": true},
	)
	if len(lifecycles) != 2 {
		t.Fatalf("lifecycles = %d, want 2", len(lifecycles))
	}
	var mount, after contract.Lifecycle
	for _, lifecycle := range lifecycles {
		switch lifecycle.Phase {
		case "mount":
			mount = lifecycle
		case "after":
			after = lifecycle
		}
	}
	if mount.Owner != "FinancePersistence" || mount.Phase != "mount" {
		t.Fatalf("mount lifecycle = %+v", mount)
	}
	if mount.Steps[0].Emit == nil || mount.Steps[0].Emit.Name != "@restore" {
		t.Fatalf("mount emit = %+v", mount.Steps[0])
	}
	external := after.Steps[0].External
	if external == nil || external.EffectID != "@env/storage#set" {
		t.Fatalf("external step = %+v", after.Steps[0])
	}
	if external.Input["key"] != `"finance_ledger_export"` {
		t.Fatalf("external key expr = %q", external.Input["key"])
	}
	if external.Input["value"] != "state.exportJson" {
		t.Fatalf("external value expr = %q", external.Input["value"])
	}
}

func TestBuildContractLifecyclesLowersExternalCompletionEvents(t *testing.T) {
	file := parser.File{
		ExternalImports: []parser.ExternalImportDecl{{
			Name: "storage",
			From: "@env/storage",
		}},
		Lifecycles: []parser.LifecycleDecl{{
			Phase: "after",
			Event: "@save",
			Statements: []parser.Statement{{
				Tokens: []lexer.Token{
					{Type: lexer.IDENT, Literal: "payload"}, {Type: lexer.PIPE_FWD},
					{Type: lexer.IDENT, Literal: "storage"}, {Type: lexer.DOT}, {Type: lexer.IDENT, Literal: "set"},
					{Type: lexer.IDENT, Literal: "key"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.STRING, Literal: "k"},
					{Type: lexer.IDENT, Literal: "value"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.IDENT, Literal: "payload"},
					{Type: lexer.IDENT, Literal: "onSuccess"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.SIGNAL, Literal: "@saved"},
					{Type: lexer.IDENT, Literal: "onFailure"}, {Type: lexer.ASSIGN_IN}, {Type: lexer.SIGNAL, Literal: "@save_failed"},
					{Type: lexer.SEMICOLON},
				},
			}},
		}},
	}
	lifecycles := buildContractLifecycles(
		[]ModuleRef{{Path: "src/Persist.nova"}},
		map[string]parser.File{"src/Persist.nova": file},
		map[string]bool{"payload": true},
	)
	if len(lifecycles) != 1 {
		t.Fatalf("lifecycles = %d, want 1", len(lifecycles))
	}
	external := lifecycles[0].Steps[0].External
	if external == nil {
		t.Fatal("expected external step")
	}
	if external.OnSuccess != "@saved" || external.OnFailure != "@save_failed" {
		t.Fatalf("completion events = %+v", external)
	}
	if _, ok := external.Input["onSuccess"]; ok {
		t.Fatalf("onSuccess should be removed from input map: %+v", external.Input)
	}
}
