package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func TestValidateFinancePureInIsolation(t *testing.T) {
	path := filepath.Join("..", "..", "..", "examples", "finance", "src", "FinancePure.nova")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tokens := lexer.Tokenize(string(content))
	file, parseDiags := parser.Parse(tokens)
	if len(parseDiags) > 0 {
		t.Fatalf("parse: %+v", parseDiags)
	}
	semantic := Validate(file)
	if len(semantic) > 0 {
		t.Fatalf("validate: %+v", semantic)
	}
}

func TestValidateFinanceStoreInIsolation(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "finance", "src")
	storePath := filepath.Join(root, "FinanceStore.nova")
	purePath := filepath.Join(root, "FinancePure.nova")
	storeContent, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	pureContent, err := os.ReadFile(purePath)
	if err != nil {
		t.Fatal(err)
	}
	storeFile, parseDiags := parser.Parse(lexer.Tokenize(string(storeContent)))
	if len(parseDiags) > 0 {
		t.Fatalf("parse store: %+v", parseDiags)
	}
	pureFile, parseDiags := parser.Parse(lexer.Tokenize(string(pureContent)))
	if len(parseDiags) > 0 {
		t.Fatalf("parse pure: %+v", parseDiags)
	}
	modules := map[string]parser.File{
		filepath.ToSlash("src/FinanceStore.nova"): storeFile,
		filepath.ToSlash("src/FinancePure.nova"):  pureFile,
	}
	imports := CapabilityImportsForModule("src/FinanceStore.nova", storeFile, modules)
	diags := ValidateWithImports(storeFile, imports)
	if len(diags) > 0 {
		t.Fatalf("validate: %+v", diags)
	}
}
