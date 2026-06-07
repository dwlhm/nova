package validator

import (
	"path/filepath"
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func TestValidateWithCapabilityImportsMergesTypes(t *testing.T) {
	typesFile := parser.File{
		ContractTypes: []parser.ContractTypeDecl{{
			Name: "LedgerV1",
			Fields: []parser.FieldDecl{{
				Name: "netTotal",
				Type: parser.TypeRef{Text: "number"},
			}},
		}},
	}
	storeFile := parser.File{
		Imports: []parser.ImportDecl{{
			Kind: parser.ImportCapability,
			From: "./FinancePure.nova",
		}},
		ContractStates: []parser.ContractStateDecl{{
			Name: "FinanceLedger",
			States: []parser.StateDecl{{
				Name:    "ledger",
				Type:    parser.TypeRef{Text: "LedgerV1"},
				Initial: lexer.Tokenize("emptyLedger()")[:3],
				Transitions: []parser.TransitionRule{{
					Event: parser.EventPattern{Name: "@save"},
					Expr:  lexer.Tokenize("ledger")[:1],
				}},
			}},
		}},
		Funcs: []parser.FuncDecl{{
			Name:   "emptyLedger",
			Return: parser.TypeRef{Text: "LedgerV1"},
			Body: lexer.Tokenize(`{
				netTotal <- 0;
			}`)[:7],
		}},
	}
	pureFile := parser.File{
		ContractTypes: typesFile.ContractTypes,
		Funcs:         storeFile.Funcs,
	}
	modules := map[string]parser.File{
		filepath.ToSlash("src/FinanceStore.nova"): storeFile,
		filepath.ToSlash("src/FinancePure.nova"):  pureFile,
	}
	imports := CapabilityImportsForModule("src/FinanceStore.nova", storeFile, modules)
	if len(imports) != 1 {
		t.Fatalf("imports = %d, want 1", len(imports))
	}
	diags := ValidateWithImports(storeFile, imports)
	if len(diags) > 0 {
		t.Fatalf("diagnostics = %+v", diags)
	}
}
