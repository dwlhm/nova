package types

import (
	"testing"

	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
)

func TestEnvironmentValidatesRecordUnionArrayAndOpaqueValues(t *testing.T) {
	input := `<contract type UserId /|
<contract type Status>
  "idle" | "ready" | "error";
/|
<contract type User>
  id: UserId;
  name: string;
  email?: string;
/|`

	file, parserDiagnostics := parser.Parse(lexer.Tokenize(input))
	if len(parserDiagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %v", parserDiagnostics)
	}
	env, diagnostics := BuildEnvironment(file)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected type diagnostics: %v", diagnostics)
	}

	userType, _ := ParseText("User")
	err := env.ValidateValue(userType, map[string]any{
		"id":   OpaqueValue{Type: "UserId", Value: "u1"},
		"name": "Ada",
	})
	if err != nil {
		t.Fatalf("validate user: %v", err)
	}

	statusType, _ := ParseTokens(env, file.ContractTypes[1].Alias.Tokens)
	if err := env.ValidateValue(statusType, "ready"); err != nil {
		t.Fatalf("validate status: %v", err)
	}
	if err := env.ValidateValue(statusType, "missing"); err == nil {
		t.Fatal("expected invalid status value")
	}
}

func TestAssignableAndInferExpressionArePureDataRelations(t *testing.T) {
	env := EmptyEnvironment()
	numberType, _ := ParseText("number")
	stringType, _ := ParseText("string")
	nullableNumber, _ := ParseText("number|null")

	numberLiteral, ok := env.InferExpression(nil, lexer.Tokenize("1")[:1])
	if !ok {
		t.Fatal("expected to infer number literal")
	}
	if !env.Assignable(numberLiteral, numberType) {
		t.Fatal("number literal should be assignable to number")
	}

	stringLiteral, ok := env.InferExpression(nil, lexer.Tokenize(`"1"`)[:1])
	if !ok {
		t.Fatal("expected to infer string literal")
	}
	if env.Assignable(stringLiteral, numberType) {
		t.Fatal("string literal should not be assignable to number")
	}
	if !env.Assignable(Type{Kind: KindUnknown}, numberType) {
		t.Fatal("unknown should be assignable across module boundaries")
	}

	nullType, ok := env.InferExpression(nil, lexer.Tokenize("null")[:1])
	if !ok || !env.Assignable(nullType, nullableNumber) {
		t.Fatal("null should be assignable to nullable union")
	}

	scope := Scope{"name": stringType, "count": numberType}
	inferred, ok := env.InferExpression(scope, lexer.Tokenize("count + 1")[:3])
	if !ok || Format(inferred) != "number" {
		t.Fatalf("inferred = %s, ok=%v; want number", Format(inferred), ok)
	}
}

func TestInferExpressionBuildsAssignableRecordLiteral(t *testing.T) {
	input := `<contract type Route>
  path: string;
  title?: string;
/|`
	file, parserDiagnostics := parser.Parse(lexer.Tokenize(input))
	if len(parserDiagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %v", parserDiagnostics)
	}
	env, diagnostics := BuildEnvironment(file)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected type diagnostics: %v", diagnostics)
	}

	routeType := env.Definitions["Route"]
	tokens := lexer.Tokenize(`{ path <- "/settings"; }`)
	inferred, ok := env.InferExpression(nil, tokens[:len(tokens)-1])
	if !ok {
		t.Fatal("expected record literal inference")
	}
	if !env.Assignable(inferred, routeType) {
		t.Fatalf("inferred = %s, want assignable Route", Format(inferred))
	}
}
