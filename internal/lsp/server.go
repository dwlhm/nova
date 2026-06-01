package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	novaformat "github.com/dwlhm/nova/internal/core/format"
	"github.com/dwlhm/nova/internal/core/lexer"
)

const (
	textDocumentSyncKindFull = 1
	rpcMethodNotFound        = -32601
	rpcInvalidParams         = -32602
)

type Server struct {
	reader    *bufio.Reader
	writer    io.Writer
	writeLock sync.Mutex
	documents map[string]document
	shutdown  bool
}

func Run(ctx context.Context, reader io.Reader, writer io.Writer) error {
	server := &Server{
		reader:    bufio.NewReader(reader),
		writer:    writer,
		documents: make(map[string]document),
	}
	return server.run(ctx)
}

func (s *Server) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		message, err := readRPCMessage(s.reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		exit, err := s.handle(message)
		if err != nil {
			return err
		}
		if exit {
			return nil
		}
	}
}

func (s *Server) handle(message rpcMessage) (bool, error) {
	switch message.Method {
	case "initialize":
		return false, s.respond(message.ID, initializeResult())
	case "initialized", "$/cancelRequest", "workspace/didChangeConfiguration":
		return false, nil
	case "shutdown":
		s.shutdown = true
		return false, s.respond(message.ID, nil)
	case "exit":
		return true, nil
	case "textDocument/didOpen":
		return false, s.didOpen(message.Params)
	case "textDocument/didChange":
		return false, s.didChange(message.Params)
	case "textDocument/didSave":
		return false, s.didSave(message.Params)
	case "textDocument/didClose":
		return false, s.didClose(message.Params)
	case "textDocument/formatting":
		return false, s.documentFormatting(message)
	case "textDocument/hover":
		return false, s.hover(message)
	case "textDocument/definition":
		return false, s.definition(message)
	case "textDocument/references":
		return false, s.references(message)
	case "textDocument/rename":
		return false, s.rename(message)
	case "textDocument/prepareRename":
		return false, s.prepareRename(message)
	case "textDocument/documentSymbol":
		return false, s.documentSymbol(message)
	case "textDocument/completion":
		return false, s.completion(message)
	default:
		if len(message.ID) == 0 {
			return false, nil
		}
		return false, s.respondError(message.ID, rpcMethodNotFound, "method not found")
	}
}

func initializeResult() map[string]any {
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    textDocumentSyncKindFull,
				"save": map[string]any{
					"includeText": false,
				},
			},
			"documentFormattingProvider": true,
			"hoverProvider":              true,
			"definitionProvider":         true,
			"referencesProvider":         true,
			"renameProvider": map[string]any{
				"prepareProvider": true,
			},
			"documentSymbolProvider": true,
			"completionProvider": map[string]any{
				"triggerCharacters": []string{"<", "@", ":", "."},
			},
			"workspace": map[string]any{
				"workspaceFolders": map[string]any{
					"supported":           true,
					"changeNotifications": false,
				},
			},
		},
		"serverInfo": map[string]any{
			"name":    "Nova LSP",
			"version": "0.1.0",
		},
	}
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type versionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version,omitempty"`
}

type didOpenTextDocumentParams struct {
	TextDocument struct {
		URI        string `json:"uri"`
		LanguageID string `json:"languageId"`
		Version    int    `json:"version"`
		Text       string `json:"text"`
	} `json:"textDocument"`
}

type didChangeTextDocumentParams struct {
	TextDocument   versionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []textDocumentContentChange     `json:"contentChanges"`
}

type textDocumentContentChange struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

type didSaveTextDocumentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type didCloseTextDocumentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type textDocumentPositionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type referenceParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      struct {
		IncludeDeclaration bool `json:"includeDeclaration"`
	} `json:"context"`
}

type renameParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

func (s *Server) didOpen(params json.RawMessage) error {
	var decoded didOpenTextDocumentParams
	if err := json.Unmarshal(params, &decoded); err != nil {
		return err
	}
	s.updateDocument(decoded.TextDocument.URI, decoded.TextDocument.Text, decoded.TextDocument.Version)
	return nil
}

func (s *Server) didChange(params json.RawMessage) error {
	var decoded didChangeTextDocumentParams
	if err := json.Unmarshal(params, &decoded); err != nil {
		return err
	}
	doc, ok := s.documents[decoded.TextDocument.URI]
	if !ok {
		doc = document{URI: decoded.TextDocument.URI}
	}
	text := doc.Text
	for _, change := range decoded.ContentChanges {
		text = applyContentChange(text, change)
	}
	s.updateDocument(decoded.TextDocument.URI, text, decoded.TextDocument.Version)
	return nil
}

func (s *Server) didSave(params json.RawMessage) error {
	var decoded didSaveTextDocumentParams
	if err := json.Unmarshal(params, &decoded); err != nil {
		return err
	}
	if doc, ok := s.documents[decoded.TextDocument.URI]; ok {
		s.publishDiagnostics(doc)
	}
	return nil
}

func (s *Server) didClose(params json.RawMessage) error {
	var decoded didCloseTextDocumentParams
	if err := json.Unmarshal(params, &decoded); err != nil {
		return err
	}
	delete(s.documents, decoded.TextDocument.URI)
	return s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri":         decoded.TextDocument.URI,
		"diagnostics": []Diagnostic{},
	})
}

func (s *Server) updateDocument(uri string, text string, version int) {
	doc := analyzeDocument(uri, text, version)
	s.documents[uri] = doc
	s.publishDiagnostics(doc)
}

func (s *Server) publishDiagnostics(doc document) {
	_ = s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri":         doc.URI,
		"version":     doc.Version,
		"diagnostics": doc.Diagnostics,
	})
}

func (s *Server) documentFormatting(message rpcMessage) error {
	var params struct {
		TextDocument textDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		return s.respondError(message.ID, rpcInvalidParams, err.Error())
	}
	doc, ok := s.documents[params.TextDocument.URI]
	if !ok {
		return s.respond(message.ID, []TextEdit{})
	}
	formatted := novaformat.Nova(doc.Text)
	if formatted == doc.Text {
		return s.respond(message.ID, []TextEdit{})
	}
	return s.respond(message.ID, []TextEdit{{
		Range:   fullDocumentRange(doc.Text),
		NewText: formatted,
	}})
}

func (s *Server) hover(message rpcMessage) error {
	params, doc, ok := s.positionRequest(message)
	if !ok {
		return nil
	}
	sym, ok := symbolAtPosition(doc, params.Position)
	if !ok {
		tok, tokenOK := tokenAtPosition(doc, params.Position)
		if !tokenOK {
			return s.respond(message.ID, nil)
		}
		index := builtins()
		if primitive, ok := index.primitive(tok.Literal); ok {
			return s.respond(message.ID, Hover{
				Contents: MarkupContent{Kind: markupKindMarkdown, Value: primitiveHover(primitive)},
				Range:    rangePtr(tokenRange(doc.Text, tok)),
			})
		}
		if nodeName, ok := nodeNameForToken(doc, tok); ok {
			if primitive, prop, event, ok := index.attribute(nodeName, tok.Literal); ok {
				return s.respond(message.ID, Hover{
					Contents: MarkupContent{Kind: markupKindMarkdown, Value: attributeHover(primitive, prop, event)},
					Range:    rangePtr(tokenRange(doc.Text, tok)),
				})
			}
		}
		return s.respond(message.ID, Hover{
			Contents: MarkupContent{Kind: markupKindMarkdown, Value: "`" + tok.Literal + "`"},
			Range:    rangePtr(tokenRange(doc.Text, tok)),
		})
	}
	return s.respond(message.ID, Hover{
		Contents: MarkupContent{Kind: markupKindMarkdown, Value: hoverText(sym)},
		Range:    rangePtr(sym.Range),
	})
}

func (s *Server) definition(message rpcMessage) error {
	params, doc, ok := s.positionRequest(message)
	if !ok {
		return nil
	}
	tok, ok := tokenAtPosition(doc, params.Position)
	if !ok {
		return s.respond(message.ID, nil)
	}
	sym, ok := doc.firstSymbol(tokenKey(tok))
	if !ok {
		return s.respond(message.ID, nil)
	}
	return s.respond(message.ID, Location{URI: doc.URI, Range: sym.Range})
}

func (s *Server) references(message rpcMessage) error {
	var params referenceParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		return s.respondError(message.ID, rpcInvalidParams, err.Error())
	}
	doc, ok := s.documents[params.TextDocument.URI]
	if !ok {
		return s.respond(message.ID, []Location{})
	}
	tok, ok := tokenAtPosition(doc, params.Position)
	if !ok {
		return s.respond(message.ID, []Location{})
	}
	matches := doc.references(tokenKey(tok))
	locations := make([]Location, 0, len(matches))
	for _, match := range matches {
		if !params.Context.IncludeDeclaration {
			if sym, ok := doc.symbolForToken(match.Token); ok && sym.Token.Offset == match.Token.Offset {
				continue
			}
		}
		locations = append(locations, Location{URI: doc.URI, Range: match.Range})
	}
	return s.respond(message.ID, locations)
}

func (s *Server) prepareRename(message rpcMessage) error {
	params, doc, ok := s.positionRequest(message)
	if !ok {
		return nil
	}
	tok, ok := tokenAtPosition(doc, params.Position)
	if !ok || !isSymbolToken(tok) {
		return s.respond(message.ID, nil)
	}
	return s.respond(message.ID, map[string]any{
		"range":       tokenRange(doc.Text, tok),
		"placeholder": tok.Literal,
	})
}

func (s *Server) rename(message rpcMessage) error {
	var params renameParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		return s.respondError(message.ID, rpcInvalidParams, err.Error())
	}
	doc, ok := s.documents[params.TextDocument.URI]
	if !ok {
		return s.respond(message.ID, WorkspaceEdit{Changes: map[string][]TextEdit{}})
	}
	tok, ok := tokenAtPosition(doc, params.Position)
	if !ok || !isSymbolToken(tok) {
		return s.respondError(message.ID, rpcInvalidParams, "position is not renameable")
	}
	if err := validateRename(tok, params.NewName); err != nil {
		return s.respondError(message.ID, rpcInvalidParams, err.Error())
	}
	matches := doc.references(tokenKey(tok))
	edits := make([]TextEdit, 0, len(matches))
	for _, match := range matches {
		edits = append(edits, TextEdit{Range: match.Range, NewText: params.NewName})
	}
	return s.respond(message.ID, WorkspaceEdit{Changes: map[string][]TextEdit{doc.URI: edits}})
}

func (s *Server) documentSymbol(message rpcMessage) error {
	var params struct {
		TextDocument textDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		return s.respondError(message.ID, rpcInvalidParams, err.Error())
	}
	doc, ok := s.documents[params.TextDocument.URI]
	if !ok {
		return s.respond(message.ID, []SymbolInformation{})
	}
	symbols := make([]SymbolInformation, 0, len(doc.Symbols))
	for _, sym := range doc.Symbols {
		symbols = append(symbols, SymbolInformation{
			Name:     sym.Name,
			Kind:     symbolKind(sym.Kind),
			Location: Location{URI: doc.URI, Range: sym.Range},
		})
	}
	return s.respond(message.ID, symbols)
}

func (s *Server) completion(message rpcMessage) error {
	params, doc, ok := s.positionRequest(message)
	if !ok {
		return nil
	}
	return s.respond(message.ID, completionItems(doc, params.Position))
}

func (s *Server) positionRequest(message rpcMessage) (textDocumentPositionParams, document, bool) {
	var params textDocumentPositionParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		_ = s.respondError(message.ID, rpcInvalidParams, err.Error())
		return params, document{}, false
	}
	doc, ok := s.documents[params.TextDocument.URI]
	if !ok {
		_ = s.respond(message.ID, nil)
		return params, document{}, false
	}
	return params, doc, true
}

func (s *Server) respond(id json.RawMessage, result any) error {
	if len(id) == 0 {
		return nil
	}
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return writeRPCMessage(s.writer, rpcResponse{JSONRPC: jsonRPCVersion, ID: id, Result: result})
}

func (s *Server) respondError(id json.RawMessage, code int, message string) error {
	if len(id) == 0 {
		return nil
	}
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return writeRPCMessage(s.writer, rpcErrorResponse{JSONRPC: jsonRPCVersion, ID: id, Error: rpcError{Code: code, Message: message}})
}

func (s *Server) notify(method string, params any) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return writeRPCMessage(s.writer, rpcNotification{JSONRPC: jsonRPCVersion, Method: method, Params: params})
}

func applyContentChange(text string, change textDocumentContentChange) string {
	if change.Range == nil {
		return change.Text
	}
	start := offsetAt(text, change.Range.Start)
	end := offsetAt(text, change.Range.End)
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(text) {
		end = len(text)
	}
	return text[:start] + change.Text + text[end:]
}

func hoverText(sym symbol) string {
	if sym.Detail == "" {
		return "`" + sym.Name + "`"
	}
	return "```nova\n" + sym.Detail + "\n```"
}

func rangePtr(value Range) *Range {
	return &value
}

func validateRename(tok lexer.Token, newName string) error {
	if tok.Type == lexer.SIGNAL {
		if !strings.HasPrefix(newName, "@") || len(newName) == 1 || !isIdentifierStart(newName[1]) {
			return fmt.Errorf("event names must start with @ followed by an identifier")
		}
		for i := 2; i < len(newName); i++ {
			if !isIdentifierPart(newName[i]) {
				return fmt.Errorf("event names must contain only identifier characters")
			}
		}
		return nil
	}
	if newName == "" || strings.HasPrefix(newName, "@") || !isIdentifierStart(newName[0]) {
		return fmt.Errorf("symbol names must be valid Nova identifiers")
	}
	for i := 1; i < len(newName); i++ {
		if !isIdentifierPart(newName[i]) {
			return fmt.Errorf("symbol names must contain only identifier characters")
		}
	}
	return nil
}

func keywordCompletions() []CompletionItem {
	labels := []string{
		"<import", "<contract", "<func", "<template", "<lifecycle",
		"state", "event", "capability", "external", "operation",
		"props", "emits", "input", "output", "returns", "target",
		"mount", "dispose", "before", "after", "error",
		"string", "number", "boolean", "unknown", "void", "true", "false", "null",
	}
	items := make([]CompletionItem, 0, len(labels))
	for _, label := range labels {
		items = append(items, CompletionItem{Label: label, Kind: lspCompletionKeyword})
	}
	return items
}

func isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentifierPart(ch byte) bool {
	return isIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}
