package lexer

import (
	_ "embed"
	"testing"
)

//go:embed testdata/audiolab.nova
var audiolabExample string

func TestTokenizeADRLanguageSurface(t *testing.T) {
	input := `<import external storage from "./storage.web.js">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
<import state count from "./Counter.nova" /|
<import event @increment from "./Counter.nova" /|
<contract state Counter>
  count: number <- 0 {
    @increment -> count |> add 1;
    @set(value: number) -> value;
  };
/|
<template target <- web>
  <button on_press -> @increment>
    +
  /|
/|
`

	tests := []struct {
		typ TokenType
		lit string
	}{
		{TAG_IMPORT, "<import"},
		{EXTERNAL, "external"},
		{IDENT, "storage"},
		{FROM, "from"},
		{STRING, "./storage.web.js"},
		{GT, ">"},
		{OPERATION, "operation"},
		{IDENT, "set"},
		{LBRACE, "{"},
		{INPUT, "input"},
		{LBRACE, "{"},
		{IDENT, "key"},
		{COLON, ":"},
		{TYPE_STRING, "string"},
		{SEMICOLON, ";"},
		{IDENT, "value"},
		{COLON, ":"},
		{TYPE_UNKNOWN, "unknown"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{OUTPUT, "output"},
		{VOID, "void"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{PIPE_END, "/|"},
		{TAG_IMPORT, "<import"},
		{STATE, "state"},
		{IDENT, "count"},
		{FROM, "from"},
		{STRING, "./Counter.nova"},
		{PIPE_END, "/|"},
		{TAG_IMPORT, "<import"},
		{EVENT, "event"},
		{SIGNAL, "@increment"},
		{FROM, "from"},
		{STRING, "./Counter.nova"},
		{PIPE_END, "/|"},
		{TAG_CONTRACT, "<contract"},
		{STATE, "state"},
		{IDENT, "Counter"},
		{GT, ">"},
		{IDENT, "count"},
		{COLON, ":"},
		{TYPE_NUMBER, "number"},
		{ASSIGN_IN, "<-"},
		{NUMBER, "0"},
		{LBRACE, "{"},
		{SIGNAL, "@increment"},
		{MAP_ARROW, "->"},
		{IDENT, "count"},
		{PIPE_FWD, "|>"},
		{IDENT, "add"},
		{NUMBER, "1"},
		{SEMICOLON, ";"},
		{SIGNAL, "@set"},
		{LPAREN, "("},
		{IDENT, "value"},
		{COLON, ":"},
		{TYPE_NUMBER, "number"},
		{RPAREN, ")"},
		{MAP_ARROW, "->"},
		{IDENT, "value"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{SEMICOLON, ";"},
		{PIPE_END, "/|"},
		{TAG_TEMPLATE, "<template"},
		{TARGET, "target"},
		{ASSIGN_IN, "<-"},
		{IDENT, "web"},
		{GT, ">"},
		{LT, "<"},
		{IDENT, "button"},
		{IDENT, "on_press"},
		{MAP_ARROW, "->"},
		{SIGNAL, "@increment"},
		{GT, ">"},
		{PLUS, "+"},
		{PIPE_END, "/|"},
		{PIPE_END, "/|"},
		{EOF, ""},
	}

	assertTokens(t, input, tests)
}

func TestTokenizeADRTypesRecordsAndAccess(t *testing.T) {
	input := `{ id: UserId; email?: string; value <- user.id; status <- "idle" | "done"; ok <- true && !false; none <- null; list <- IEM[]; gain <- -0.5; }`

	tests := []struct {
		typ TokenType
		lit string
	}{
		{LBRACE, "{"},
		{IDENT, "id"},
		{COLON, ":"},
		{IDENT, "UserId"},
		{SEMICOLON, ";"},
		{IDENT, "email"},
		{QUESTION, "?"},
		{COLON, ":"},
		{TYPE_STRING, "string"},
		{SEMICOLON, ";"},
		{IDENT, "value"},
		{ASSIGN_IN, "<-"},
		{IDENT, "user"},
		{DOT, "."},
		{IDENT, "id"},
		{SEMICOLON, ";"},
		{IDENT, "status"},
		{ASSIGN_IN, "<-"},
		{STRING, "idle"},
		{PIPE, "|"},
		{STRING, "done"},
		{SEMICOLON, ";"},
		{IDENT, "ok"},
		{ASSIGN_IN, "<-"},
		{TRUE, "true"},
		{AND, "&&"},
		{BANG, "!"},
		{FALSE, "false"},
		{SEMICOLON, ";"},
		{IDENT, "none"},
		{ASSIGN_IN, "<-"},
		{NULL, "null"},
		{SEMICOLON, ";"},
		{IDENT, "list"},
		{ASSIGN_IN, "<-"},
		{IDENT, "IEM"},
		{LBRACKET, "["},
		{RBRACKET, "]"},
		{SEMICOLON, ";"},
		{IDENT, "gain"},
		{ASSIGN_IN, "<-"},
		{NUMBER, "-0.5"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	assertTokens(t, input, tests)
}

func TestTokenizeDecodesEscapedStrings(t *testing.T) {
	tokens := Tokenize(`"{\"type\":\"expense\"}" "line\nbreak"`)
	if tokens[0].Type != STRING || tokens[0].Literal != `{"type":"expense"}` {
		t.Fatalf("first string = (%s, %q), want decoded JSON text", tokens[0].Type, tokens[0].Literal)
	}
	if tokens[1].Type != STRING || tokens[1].Literal != "line\nbreak" {
		t.Fatalf("second string = (%s, %q), want decoded newline", tokens[1].Type, tokens[1].Literal)
	}
}

func TestTokenizeExampleFileHasNoIllegalTokens(t *testing.T) {
	for _, tok := range Tokenize(audiolabExample) {
		if tok.Type == ILLEGAL {
			t.Fatalf("unexpected illegal token %q", tok.Literal)
		}
	}
}

func TestTokenizeTracksSourceSpans(t *testing.T) {
	tokens := Tokenize("one\r\n  @two")
	if len(tokens) != 3 {
		t.Fatalf("tokens = %d, want 3", len(tokens))
	}
	if tokens[0].Offset != 0 || tokens[0].Line != 1 || tokens[0].Column != 1 || tokens[0].Length != 3 {
		t.Fatalf("first token span = %+v, want offset 0 line 1 column 1 length 3", tokens[0])
	}
	if tokens[1].Offset != 7 || tokens[1].Line != 2 || tokens[1].Column != 3 || tokens[1].Length != 4 {
		t.Fatalf("second token span = %+v, want offset 7 line 2 column 3 length 4", tokens[1])
	}
	if tokens[2].Offset != 11 || tokens[2].Line != 2 || tokens[2].Column != 7 || tokens[2].Length != 0 {
		t.Fatalf("EOF token span = %+v, want offset 11 line 2 column 7 length 0", tokens[2])
	}
}

func assertTokens(t *testing.T, input string, tests []struct {
	typ TokenType
	lit string
}) {
	t.Helper()

	tokens := Tokenize(input)
	if len(tokens) != len(tests) {
		t.Fatalf("token count mismatch: got %d, want %d\n%v", len(tokens), len(tests), tokens)
	}

	for i, tt := range tests {
		if tokens[i].Type != tt.typ || tokens[i].Literal != tt.lit {
			t.Fatalf("token %d mismatch: got (%s, %q), want (%s, %q)", i, tokens[i].Type, tokens[i].Literal, tt.typ, tt.lit)
		}
	}
}
