package lsp

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestServerHandlesInitializeAndOpenDocument(t *testing.T) {
	input := frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) +
		frame(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///App.nova","languageId":"nova","version":1,"text":"<contract state Counter>\n  count: number <- \"1\";\n/|\n"}}}`) +
		frame(`{"jsonrpc":"2.0","id":2,"method":"shutdown","params":null}`) +
		frame(`{"jsonrpc":"2.0","method":"exit","params":null}`)

	var output bytes.Buffer
	if err := Run(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := output.String()
	if !strings.Contains(got, `"hoverProvider":true`) {
		t.Fatalf("initialize response missing hover capability: %s", got)
	}
	if !strings.Contains(got, `"method":"textDocument/publishDiagnostics"`) {
		t.Fatalf("didOpen did not publish diagnostics: %s", got)
	}
	if !strings.Contains(got, semanticDiagnosticCode) {
		t.Fatalf("diagnostics missing semantic code: %s", got)
	}
}

func frame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}
