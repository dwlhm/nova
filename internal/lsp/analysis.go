package lsp

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/validator"
)

const (
	lspSeverityError       = 1
	lspSymbolKindClass     = 5
	lspSymbolKindFunction  = 12
	lspSymbolKindVariable  = 13
	lspSymbolKindString    = 15
	lspSymbolKindEvent     = 24
	lspCompletionFunction  = 3
	lspCompletionVariable  = 6
	lspCompletionKeyword   = 14
	lspCompletionEvent     = 23
	markupKindMarkdown     = "markdown"
	diagnosticSource       = "nova"
	parserDiagnosticCode   = "NVA-PARSE-001"
	semanticDiagnosticCode = "NVA-SEMANTIC-001"
)

type document struct {
	URI         string
	Text        string
	Version     int
	Tokens      []lexer.Token
	File        parser.File
	Diagnostics []Diagnostic
	Symbols     []symbol
}

type symbol struct {
	Name   string
	Key    string
	Kind   string
	Detail string
	Token  lexer.Token
	Range  Range
}

type tokenMatch struct {
	Token lexer.Token
	Range Range
}

type completionMode string

const (
	completionDefault     completionMode = "default"
	completionTagName     completionMode = "tag_name"
	completionAttribute   completionMode = "attribute"
	completionEventTarget completionMode = "event_target"
	completionExpression  completionMode = "expression"
)

type completionContext struct {
	Mode     completionMode
	NodeName string
}

func analyzeDocument(uri string, text string, version int) document {
	tokens := lexer.Tokenize(text)
	diagnostics := make([]Diagnostic, 0)
	for _, tok := range tokens {
		if tok.Type == lexer.ILLEGAL {
			diagnostics = append(diagnostics, lspDiagnostic(text, parserDiagnosticCode, fmt.Sprintf("illegal token %q", tok.Literal), tok))
		}
	}

	file, parserDiagnostics := parser.Parse(tokens)
	for _, item := range parserDiagnostics {
		diagnostics = append(diagnostics, lspDiagnostic(text, parserDiagnosticCode, item.Message, item.Token))
	}
	if len(parserDiagnostics) == 0 {
		for _, item := range validator.Validate(file) {
			diagnostics = append(diagnostics, lspDiagnostic(text, semanticDiagnosticCode, item.Message, item.Token))
		}
	}

	sort.SliceStable(diagnostics, func(i, j int) bool {
		left := diagnostics[i]
		right := diagnostics[j]
		if left.Range.Start.Line != right.Range.Start.Line {
			return left.Range.Start.Line < right.Range.Start.Line
		}
		if left.Range.Start.Character != right.Range.Start.Character {
			return left.Range.Start.Character < right.Range.Start.Character
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		return left.Message < right.Message
	})

	return document{
		URI:         uri,
		Text:        text,
		Version:     version,
		Tokens:      tokens,
		File:        file,
		Diagnostics: diagnostics,
		Symbols:     collectSymbols(text, tokens),
	}
}

func lspDiagnostic(text string, code string, message string, token lexer.Token) Diagnostic {
	return Diagnostic{
		Range:    tokenRange(text, token),
		Severity: lspSeverityError,
		Code:     code,
		Source:   diagnosticSource,
		Message:  message,
	}
}

func collectSymbols(text string, tokens []lexer.Token) []symbol {
	symbols := make([]symbol, 0)
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].Type {
		case lexer.TAG_IMPORT:
			symbols = collectImportSymbols(text, tokens, i, symbols)
		case lexer.TAG_CONTRACT:
			symbols = collectContractSymbols(text, tokens, i, symbols)
		case lexer.TAG_FUNC:
			if name, ok := nextToken(tokens, i+1, lexer.IDENT); ok {
				symbols = appendSymbol(text, symbols, name, "func", "func "+name.Literal)
			}
		case lexer.TAG_TEMPLATE:
			if target, ok := templateTargetToken(tokens, i); ok {
				symbols = appendSymbol(text, symbols, target, "template target", "template target "+target.Literal)
			}
		}
	}
	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].Token.Offset != symbols[j].Token.Offset {
			return symbols[i].Token.Offset < symbols[j].Token.Offset
		}
		return symbols[i].Name < symbols[j].Name
	})
	return symbols
}

func collectImportSymbols(text string, tokens []lexer.Token, start int, symbols []symbol) []symbol {
	pos := start + 1
	kind := "capability import"
	if pos < len(tokens) {
		switch tokens[pos].Type {
		case lexer.EXTERNAL:
			if name, ok := nextToken(tokens, pos+1, lexer.IDENT); ok {
				symbols = appendSymbol(text, symbols, name, "external import", "external "+name.Literal)
			}
			end := findToken(tokens, pos, lexer.PIPE_END)
			for i := pos; i < end; i++ {
				if tokens[i].Type == lexer.OPERATION {
					if operation, ok := nextToken(tokens, i+1, lexer.IDENT); ok {
						symbols = appendSymbol(text, symbols, operation, "external operation", "operation "+operation.Literal)
					}
				}
			}
			return symbols
		case lexer.STATE:
			kind = "state import"
			pos++
		case lexer.EVENT:
			kind = "event import"
			pos++
		}
	}

	for i := pos; i < len(tokens) && tokens[i].Type != lexer.FROM && tokens[i].Type != lexer.PIPE_END; i++ {
		switch tokens[i].Type {
		case lexer.IDENT, lexer.SIGNAL:
			detail := kind + " " + tokens[i].Literal
			symbols = appendSymbol(text, symbols, tokens[i], kind, detail)
		}
	}
	return symbols
}

func collectContractSymbols(text string, tokens []lexer.Token, start int, symbols []symbol) []symbol {
	if start+2 >= len(tokens) {
		return symbols
	}
	contractKind := tokens[start+1]
	name := tokens[start+2]
	if name.Type == lexer.IDENT {
		symbols = appendSymbol(text, symbols, name, contractKindLabel(contractKind.Type), contractKindLabel(contractKind.Type)+" "+name.Literal)
	}

	bodyStart := findToken(tokens, start, lexer.GT)
	if bodyStart >= len(tokens) {
		return symbols
	}
	bodyStart++
	bodyEnd := findToken(tokens, bodyStart, lexer.PIPE_END)

	switch contractKind.Type {
	case lexer.STATE:
		for i := bodyStart; i < bodyEnd; i++ {
			if stateDeclAt(tokens, i, bodyEnd) {
				typeText := stateTypeText(tokens, i, bodyEnd)
				detail := "state " + tokens[i].Literal
				if typeText != "" {
					detail += ": " + typeText
				}
				symbols = appendSymbol(text, symbols, tokens[i], "state", detail)
			}
			if eventPatternAt(tokens, i, bodyEnd) {
				symbols = appendSymbol(text, symbols, tokens[i], "scheduler event", "event "+tokens[i].Literal)
			}
		}
	case lexer.CAPABILITY:
		for i := bodyStart; i < bodyEnd; i++ {
			if tokens[i].Type == lexer.SIGNAL && nextSignificantType(tokens, i+1) == lexer.COLON {
				symbols = appendSymbol(text, symbols, tokens[i], "scheduler event", "event "+tokens[i].Literal)
			}
		}
	}
	return symbols
}

func appendSymbol(text string, symbols []symbol, tok lexer.Token, kind string, detail string) []symbol {
	if !isSymbolToken(tok) {
		return symbols
	}
	return append(symbols, symbol{
		Name:   tok.Literal,
		Key:    tokenKey(tok),
		Kind:   kind,
		Detail: detail,
		Token:  tok,
		Range:  tokenRange(text, tok),
	})
}

func contractKindLabel(typ lexer.TokenType) string {
	switch typ {
	case lexer.TYPE:
		return "contract type"
	case lexer.STATE:
		return "state contract"
	case lexer.CAPABILITY:
		return "contract capability"
	default:
		return "contract"
	}
}

func stateDeclAt(tokens []lexer.Token, pos int, end int) bool {
	if pos >= end || tokens[pos].Type != lexer.IDENT {
		return false
	}
	colon := nextSignificantIndex(tokens, pos+1)
	if colon >= end || tokens[colon].Type != lexer.COLON {
		return false
	}
	depth := 0
	for i := colon + 1; i < end; i++ {
		tok := tokens[i]
		if depth == 0 {
			switch tok.Type {
			case lexer.ASSIGN_IN:
				return true
			case lexer.SEMICOLON, lexer.RBRACE:
				return false
			}
		}
		depth = declarationDepth(depth, tok.Type)
	}
	return false
}

func stateTypeText(tokens []lexer.Token, pos int, end int) string {
	colon := nextSignificantIndex(tokens, pos+1)
	if colon >= end {
		return ""
	}
	parts := make([]string, 0)
	for i := colon + 1; i < end && tokens[i].Type != lexer.ASSIGN_IN; i++ {
		if tokens[i].Type == lexer.COMMENT {
			continue
		}
		parts = append(parts, tokens[i].Literal)
	}
	return strings.Join(parts, "")
}

func eventPatternAt(tokens []lexer.Token, pos int, end int) bool {
	if pos >= end || tokens[pos].Type != lexer.SIGNAL {
		return false
	}
	next := nextSignificantIndex(tokens, pos+1)
	if next >= end {
		return false
	}
	if tokens[next].Type == lexer.MAP_ARROW {
		return true
	}
	if tokens[next].Type != lexer.LPAREN {
		return false
	}
	depth := 1
	for i := next + 1; i < end; i++ {
		depth = declarationDepth(depth, tokens[i].Type)
		if depth == 0 {
			return nextSignificantType(tokens, i+1) == lexer.MAP_ARROW
		}
	}
	return false
}

func declarationDepth(depth int, typ lexer.TokenType) int {
	switch typ {
	case lexer.LPAREN, lexer.LBRACKET, lexer.LBRACE:
		return depth + 1
	case lexer.RPAREN, lexer.RBRACKET, lexer.RBRACE:
		if depth > 0 {
			return depth - 1
		}
	}
	return depth
}

func templateTargetToken(tokens []lexer.Token, start int) (lexer.Token, bool) {
	for i := start + 1; i < len(tokens) && tokens[i].Type != lexer.GT && tokens[i].Type != lexer.PIPE_END; i++ {
		if tokens[i].Type == lexer.TARGET && nextSignificantType(tokens, i+1) == lexer.ASSIGN_IN {
			assign := nextSignificantIndex(tokens, i+1)
			target := nextSignificantIndex(tokens, assign+1)
			if target < len(tokens) && tokens[target].Type == lexer.IDENT {
				return tokens[target], true
			}
		}
	}
	return lexer.Token{}, false
}

func nextToken(tokens []lexer.Token, start int, typ lexer.TokenType) (lexer.Token, bool) {
	pos := nextSignificantIndex(tokens, start)
	if pos < len(tokens) && tokens[pos].Type == typ {
		return tokens[pos], true
	}
	return lexer.Token{}, false
}

func findToken(tokens []lexer.Token, start int, typ lexer.TokenType) int {
	for i := start; i < len(tokens); i++ {
		if tokens[i].Type == typ || tokens[i].Type == lexer.EOF {
			return i
		}
	}
	return len(tokens)
}

func nextSignificantIndex(tokens []lexer.Token, start int) int {
	for start < len(tokens) && tokens[start].Type == lexer.COMMENT {
		start++
	}
	return start
}

func nextSignificantType(tokens []lexer.Token, start int) lexer.TokenType {
	pos := nextSignificantIndex(tokens, start)
	if pos >= len(tokens) {
		return lexer.EOF
	}
	return tokens[pos].Type
}

func completionContextAt(doc document, position Position) completionContext {
	offset := offsetAt(doc.Text, position)
	context := completionContext{Mode: completionDefault}
	insideTag := false
	nodeName := ""
	previous := lexer.Token{Type: lexer.EOF}

	for i := 0; i < len(doc.Tokens); i++ {
		tok := doc.Tokens[i]
		if tok.Offset >= offset {
			break
		}
		switch tok.Type {
		case lexer.LT:
			insideTag = true
			nodeName = ""
		case lexer.GT, lexer.PIPE_END:
			insideTag = false
			nodeName = ""
		case lexer.COMMENT, lexer.EOF:
			continue
		default:
			if insideTag && nodeName == "" && tok.Type == lexer.IDENT {
				nodeName = tok.Literal
			}
		}
		previous = tok
	}

	if !insideTag {
		if previous.Type == lexer.MAP_ARROW {
			context.Mode = completionEventTarget
		} else if previous.Type == lexer.ASSIGN_IN {
			context.Mode = completionExpression
		}
		return context
	}

	context.NodeName = nodeName
	switch previous.Type {
	case lexer.LT:
		context.Mode = completionTagName
	case lexer.MAP_ARROW:
		context.Mode = completionEventTarget
	case lexer.ASSIGN_IN, lexer.LPAREN, lexer.COMMA:
		context.Mode = completionExpression
	default:
		context.Mode = completionAttribute
	}
	if nodeName == "" {
		context.Mode = completionTagName
	}
	return context
}

func nodeNameForToken(doc document, target lexer.Token) (string, bool) {
	insideTag := false
	nodeName := ""
	for _, tok := range doc.Tokens {
		if tok.Offset > target.Offset {
			break
		}
		switch tok.Type {
		case lexer.LT:
			insideTag = true
			nodeName = ""
		case lexer.GT, lexer.PIPE_END:
			if tok.Offset < target.Offset {
				insideTag = false
				nodeName = ""
			}
		default:
			if insideTag && nodeName == "" && tok.Type == lexer.IDENT {
				nodeName = tok.Literal
			}
		}
	}
	return nodeName, insideTag && nodeName != ""
}

func tokenAtPosition(doc document, position Position) (lexer.Token, bool) {
	offset := offsetAt(doc.Text, position)
	for _, tok := range doc.Tokens {
		if !isSymbolToken(tok) && tok.Type != lexer.IDENT && tok.Type != lexer.SIGNAL {
			continue
		}
		end := tok.Offset + tok.Length
		if tok.Length == 0 {
			end = tok.Offset + len(tok.Literal)
		}
		if offset >= tok.Offset && offset <= end {
			return tok, true
		}
	}
	return lexer.Token{}, false
}

func completionItems(doc document, position Position) []CompletionItem {
	index := builtins()
	context := completionContextAt(doc, position)
	items := make([]CompletionItem, 0)

	switch context.Mode {
	case completionTagName:
		items = append(items, index.nodeCompletions()...)
		items = append(items, capabilitySymbolCompletions(doc)...)
	case completionAttribute:
		items = append(items, index.attributeCompletions(context.NodeName)...)
	case completionEventTarget:
		items = append(items, eventSymbolCompletions(doc)...)
	case completionExpression:
		items = append(items, expressionSymbolCompletions(doc)...)
		items = append(items, literalCompletions()...)
	default:
		items = append(items, keywordCompletions()...)
		items = append(items, index.nodeCompletions()...)
		items = append(items, allSymbolCompletions(doc)...)
	}

	return stableCompletionItems(items)
}

func allSymbolCompletions(doc document) []CompletionItem {
	items := make([]CompletionItem, 0, len(doc.Symbols))
	for _, sym := range doc.Symbols {
		items = append(items, symbolCompletion(sym, "30_"))
	}
	return items
}

func eventSymbolCompletions(doc document) []CompletionItem {
	items := make([]CompletionItem, 0)
	for _, sym := range doc.Symbols {
		if sym.Kind == "scheduler event" || sym.Kind == "event import" {
			items = append(items, symbolCompletion(sym, "01_"))
		}
	}
	return items
}

func expressionSymbolCompletions(doc document) []CompletionItem {
	items := make([]CompletionItem, 0)
	for _, sym := range doc.Symbols {
		switch sym.Kind {
		case "state", "state import", "func", "external import":
			items = append(items, symbolCompletion(sym, "01_"))
		}
	}
	return items
}

func capabilitySymbolCompletions(doc document) []CompletionItem {
	items := make([]CompletionItem, 0)
	for _, sym := range doc.Symbols {
		switch sym.Kind {
		case "contract capability", "capability import":
			items = append(items, symbolCompletion(sym, "21_"))
		}
	}
	return items
}

func symbolCompletion(sym symbol, sortPrefix string) CompletionItem {
	return CompletionItem{
		Label:      sym.Name,
		Kind:       completionKind(sym.Kind),
		Detail:     sym.Detail,
		InsertText: sym.Name,
		SortText:   sortPrefix + sym.Name,
	}
}

func literalCompletions() []CompletionItem {
	return []CompletionItem{
		{Label: "true", Kind: lspCompletionKeyword, InsertText: "true", SortText: "40_true"},
		{Label: "false", Kind: lspCompletionKeyword, InsertText: "false", SortText: "40_false"},
		{Label: "null", Kind: lspCompletionKeyword, InsertText: "null", SortText: "40_null"},
		{Label: "void", Kind: lspCompletionKeyword, InsertText: "void", SortText: "40_void"},
	}
}

func stableCompletionItems(items []CompletionItem) []CompletionItem {
	seen := make(map[string]bool)
	out := make([]CompletionItem, 0, len(items))
	for _, item := range items {
		key := item.Label + "\x00" + item.Detail
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := out[i].SortText
		right := out[j].SortText
		if left == "" {
			left = out[i].Label
		}
		if right == "" {
			right = out[j].Label
		}
		if left != right {
			return left < right
		}
		return out[i].Label < out[j].Label
	})
	return out
}

func symbolAtPosition(doc document, position Position) (symbol, bool) {
	tok, ok := tokenAtPosition(doc, position)
	if !ok || !isSymbolToken(tok) {
		return symbol{}, false
	}
	if exact, ok := doc.symbolForToken(tok); ok {
		return exact, true
	}
	return doc.firstSymbol(tokenKey(tok))
}

func (doc document) symbolForToken(tok lexer.Token) (symbol, bool) {
	key := tokenKey(tok)
	for _, candidate := range doc.Symbols {
		if candidate.Key == key && candidate.Token.Offset == tok.Offset {
			return candidate, true
		}
	}
	return symbol{}, false
}

func (doc document) firstSymbol(key string) (symbol, bool) {
	for _, candidate := range doc.Symbols {
		if candidate.Key == key {
			return candidate, true
		}
	}
	return symbol{}, false
}

func (doc document) references(key string) []tokenMatch {
	out := make([]tokenMatch, 0)
	for _, tok := range doc.Tokens {
		if tokenKey(tok) == key {
			out = append(out, tokenMatch{Token: tok, Range: tokenRange(doc.Text, tok)})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Token.Offset < out[j].Token.Offset
	})
	return out
}

func isSymbolToken(tok lexer.Token) bool {
	return tok.Type == lexer.IDENT || tok.Type == lexer.SIGNAL
}

func tokenKey(tok lexer.Token) string {
	if tok.Type == lexer.SIGNAL {
		return "signal:" + tok.Literal
	}
	if tok.Type == lexer.IDENT {
		return "ident:" + tok.Literal
	}
	return ""
}

func tokenRange(text string, tok lexer.Token) Range {
	length := tok.Length
	if length == 0 && tok.Literal != "" {
		length = len(tok.Literal)
	}
	start := positionAt(text, tok.Offset)
	end := positionAt(text, tok.Offset+length)
	return Range{Start: start, End: end}
}

func fullDocumentRange(text string) Range {
	return Range{
		Start: Position{Line: 0, Character: 0},
		End:   positionAt(text, len(text)),
	}
}

func positionAt(text string, offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	offsets := lineOffsets(text)
	line := sort.Search(len(offsets), func(i int) bool {
		return offsets[i] > offset
	}) - 1
	if line < 0 {
		line = 0
	}
	lineStart := offsets[line]
	return Position{
		Line:      line,
		Character: utf16Len(text[lineStart:offset]),
	}
}

func offsetAt(text string, position Position) int {
	offsets := lineOffsets(text)
	if position.Line <= 0 {
		return offsetWithinLine(text, offsets[0], position.Character)
	}
	if position.Line >= len(offsets) {
		return len(text)
	}
	return offsetWithinLine(text, offsets[position.Line], position.Character)
}

func offsetWithinLine(text string, lineStart int, character int) int {
	if character <= 0 {
		return lineStart
	}
	lineEnd := lineEndOffset(text, lineStart)
	seen := 0
	for offset := lineStart; offset < lineEnd; {
		r, size := utf8.DecodeRuneInString(text[offset:lineEnd])
		width := 1
		if r > 0xFFFF {
			width = 2
		}
		if seen+width > character {
			return offset
		}
		seen += width
		offset += size
	}
	return lineEnd
}

func lineEndOffset(text string, lineStart int) int {
	for i := lineStart; i < len(text); i++ {
		if text[i] == '\n' || text[i] == '\r' {
			return i
		}
	}
	return len(text)
}

func lineOffsets(text string) []int {
	offsets := []int{0}
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				i++
			}
			offsets = append(offsets, i+1)
		case '\n':
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

func utf16Len(text string) int {
	return len(utf16.Encode([]rune(text)))
}

func symbolKind(kind string) int {
	switch kind {
	case "func", "external operation":
		return lspSymbolKindFunction
	case "scheduler event":
		return lspSymbolKindEvent
	case "contract type", "state contract", "contract capability":
		return lspSymbolKindClass
	case "template target":
		return lspSymbolKindString
	default:
		return lspSymbolKindVariable
	}
}

func completionKind(kind string) int {
	switch kind {
	case "func", "external operation":
		return lspCompletionFunction
	case "scheduler event":
		return lspCompletionEvent
	case "contract type", "state contract", "contract capability":
		return lspCompletionClass
	default:
		return lspCompletionVariable
	}
}
