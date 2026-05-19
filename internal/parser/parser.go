package parser

import (
	"fmt"

	"github.com/dwlhm/nova/internal/lexer"
)

type parser struct {
	tokens      []lexer.Token
	pos         int
	diagnostics []Diagnostic
}

func Parse(tokens []lexer.Token) (File, []Diagnostic) {
	p := parser{tokens: tokens}
	file := File{}

	for !p.at(lexer.EOF) {
		switch p.peek().Type {
		case lexer.COMMENT:
			file.Comments = append(file.Comments, Comment{Text: p.advance().Literal})
		case lexer.TAG_IMPORT:
			p.parseImportLike(&file)
		case lexer.TAG_CONTRACT:
			p.parseContractLike(&file)
		case lexer.TAG_FUNC:
			file.Funcs = append(file.Funcs, p.parseFunc())
		case lexer.TAG_TEMPLATE:
			file.Templates = append(file.Templates, p.parseTemplate())
		case lexer.TAG_LIFECYCLE:
			file.Lifecycles = append(file.Lifecycles, p.parseLifecycle())
		default:
			p.errorf(p.peek(), "unexpected top-level token %s", p.peek().Type)
			p.advance()
		}
	}

	return file, p.diagnostics
}

func (p *parser) parseImportLike(file *File) {
	p.expect(lexer.TAG_IMPORT)
	if p.match(lexer.EXTERNAL) {
		file.ExternalImports = append(file.ExternalImports, p.parseExternalImportAfterKeyword())
		return
	}

	decl := ImportDecl{Kind: ImportCapability}
	if p.match(lexer.STATE) {
		decl.Kind = ImportState
	} else if p.match(lexer.EVENT) {
		decl.Kind = ImportEvent
	}

	for !p.at(lexer.EOF) && !p.at(lexer.FROM) && !p.at(lexer.PIPE_END) {
		start := p.pos
		item := ImportItem{}
		if decl.Kind == ImportEvent {
			item.Name = p.expectLiteral(lexer.SIGNAL, "event import")
		} else {
			item.Name = p.expectLiteral(lexer.IDENT, "import name")
		}
		if p.match(lexer.AS) {
			item.Alias = p.parseImportAlias(decl.Kind)
		}
		item.Tokens = cloneTokens(p.tokens[start:p.pos])
		if item.Name != "" {
			decl.Items = append(decl.Items, item)
		}
		if !p.match(lexer.COMMA) {
			break
		}
	}

	p.expect(lexer.FROM)
	decl.From = p.expectLiteral(lexer.STRING, "import source")
	p.expect(lexer.PIPE_END)
	file.Imports = append(file.Imports, decl)
}

func (p *parser) parseImportAlias(kind ImportKind) string {
	if kind == ImportEvent {
		if p.at(lexer.SIGNAL) {
			return p.advance().Literal
		}
		p.errorf(p.peek(), "expected event import alias")
		p.advance()
		return ""
	}
	return p.expectLiteral(lexer.IDENT, "import alias")
}

func (p *parser) parseExternalImportAfterKeyword() ExternalImportDecl {
	decl := ExternalImportDecl{}
	decl.Name = p.expectLiteral(lexer.IDENT, "external import name")
	p.expect(lexer.FROM)
	decl.From = p.expectLiteral(lexer.STRING, "external import source")
	p.expect(lexer.GT)
	decl.Tokens = p.collectTopLevelBlockBody()
	decl.Operations = p.parseExternalOperations(decl.Tokens)
	return decl
}

func (p *parser) parseContractLike(file *File) {
	p.expect(lexer.TAG_CONTRACT)
	switch p.peek().Type {
	case lexer.TYPE:
		file.ContractTypes = append(file.ContractTypes, p.parseContractTypeAfterKeyword())
	case lexer.STATE:
		file.ContractStates = append(file.ContractStates, p.parseContractStateAfterKeyword())
	case lexer.CAPABILITY:
		file.ContractCapabilities = append(file.ContractCapabilities, p.parseContractCapabilityAfterKeyword())
	default:
		p.errorf(p.peek(), "expected contract kind type, state, or capability")
		p.collectTopLevelBlockBody()
	}
}

func (p *parser) parseContractTypeAfterKeyword() ContractTypeDecl {
	p.expect(lexer.TYPE)
	decl := ContractTypeDecl{Name: p.expectLiteral(lexer.IDENT, "type contract name")}
	if p.match(lexer.PIPE_END) {
		decl.Opaque = true
		return decl
	}

	p.expect(lexer.GT)
	decl.Tokens = p.collectTopLevelBlockBody()
	decl.Fields, decl.Alias = p.parseContractTypeBody(decl.Tokens)
	return decl
}

func (p *parser) parseContractStateAfterKeyword() ContractStateDecl {
	p.expect(lexer.STATE)
	decl := ContractStateDecl{Name: p.expectLiteral(lexer.IDENT, "state contract name")}
	p.expect(lexer.GT)
	decl.Tokens = p.collectTopLevelBlockBody()
	decl.States = p.parseStateDecls(decl.Tokens)
	return decl
}

func (p *parser) parseContractCapabilityAfterKeyword() ContractCapabilityDecl {
	p.expect(lexer.CAPABILITY)
	decl := ContractCapabilityDecl{Name: p.expectLiteral(lexer.IDENT, "capability contract name")}
	p.expect(lexer.GT)
	decl.Tokens = p.collectTopLevelBlockBody()

	body := parser{tokens: decl.Tokens}
	for !body.done() {
		switch body.peek().Type {
		case lexer.COMMENT:
			body.advance()
		case lexer.PROPS:
			body.advance()
			body.expect(lexer.LBRACE)
			decl.Props = append(decl.Props, body.parseFieldDeclsUntil(lexer.RBRACE)...)
			body.expect(lexer.RBRACE)
		case lexer.EMITS:
			body.advance()
			body.expect(lexer.LBRACE)
			decl.Emits = append(decl.Emits, body.parseEmitDeclsUntil(lexer.RBRACE)...)
			body.expect(lexer.RBRACE)
		default:
			body.errorf(body.peek(), "expected capability section props or emits")
			body.advance()
		}
	}
	p.diagnostics = append(p.diagnostics, body.diagnostics...)
	return decl
}

func (p *parser) parseFunc() FuncDecl {
	p.expect(lexer.TAG_FUNC)
	decl := FuncDecl{Name: p.expectLiteral(lexer.IDENT, "function name")}
	decl.Params = p.parseParamDeclsUntil(lexer.RETURNS, lexer.GT)
	if p.match(lexer.RETURNS) {
		decl.Return = p.parseTypeUntil(lexer.GT)
	}
	p.expect(lexer.GT)
	decl.Body = p.collectTopLevelBlockBody()
	return decl
}

func (p *parser) parseTemplate() TemplateDecl {
	p.expect(lexer.TAG_TEMPLATE)
	decl := TemplateDecl{}
	if p.match(lexer.TARGET) {
		p.expect(lexer.ASSIGN_IN)
		decl.Target = p.expectLiteral(lexer.IDENT, "template target")
	}
	p.expect(lexer.GT)
	decl.Tokens = p.collectTopLevelBlockBody()
	return decl
}

func (p *parser) parseLifecycle() LifecycleDecl {
	p.expect(lexer.TAG_LIFECYCLE)
	decl := LifecycleDecl{}
	phase := p.peek()
	if isLifecyclePhase(phase.Type) {
		decl.Phase = p.advance().Literal
	} else {
		p.errorf(phase, "expected lifecycle phase")
		p.advance()
	}

	if phase.Type == lexer.BEFORE || phase.Type == lexer.AFTER {
		if p.at(lexer.SIGNAL) {
			decl.Event = p.advance().Literal
		} else {
			p.errorf(p.peek(), "expected scheduler event")
			p.advance()
		}
	}

	p.expect(lexer.GT)
	decl.Tokens = p.collectTopLevelBlockBody()
	decl.Statements = splitStatements(decl.Tokens)
	return decl
}

func (p *parser) parseExternalOperations(tokens []lexer.Token) []ExternalOperationDecl {
	body := parser{tokens: tokens}
	operations := make([]ExternalOperationDecl, 0)

	for !body.done() {
		if body.match(lexer.COMMENT) {
			continue
		}
		if !body.match(lexer.OPERATION) {
			body.errorf(body.peek(), "expected external operation")
			body.advance()
			continue
		}

		start := body.pos - 1
		op := ExternalOperationDecl{Name: body.expectLiteral(lexer.IDENT, "operation name")}
		body.expect(lexer.LBRACE)
		for !body.done() && !body.at(lexer.RBRACE) {
			switch body.peek().Type {
			case lexer.INPUT:
				body.advance()
				body.expect(lexer.LBRACE)
				op.Inputs = body.parseFieldDeclsUntil(lexer.RBRACE)
				body.expect(lexer.RBRACE)
			case lexer.OUTPUT:
				body.advance()
				op.Output = body.parseTypeUntil(lexer.SEMICOLON)
				body.expect(lexer.SEMICOLON)
			case lexer.COMMENT:
				body.advance()
			default:
				body.errorf(body.peek(), "expected input or output section")
				body.advance()
			}
		}
		body.expect(lexer.RBRACE)
		op.Tokens = cloneTokens(tokens[start:body.pos])
		operations = append(operations, op)
	}

	p.diagnostics = append(p.diagnostics, body.diagnostics...)
	return operations
}

func (p *parser) parseContractTypeBody(tokens []lexer.Token) ([]FieldDecl, TypeRef) {
	body := parser{tokens: tokens}
	first := body.nextNonComment(0)
	if isNameToken(first.Type) && body.hasFieldShapeAt(body.indexOf(first)) {
		fields := body.parseFieldDeclsUntil(lexer.EOF)
		p.diagnostics = append(p.diagnostics, body.diagnostics...)
		return fields, TypeRef{}
	}

	aliasTokens := body.collectUntil(lexer.SEMICOLON)
	body.match(lexer.SEMICOLON)
	p.diagnostics = append(p.diagnostics, body.diagnostics...)
	return nil, typeRef(aliasTokens)
}

func (p *parser) parseStateDecls(tokens []lexer.Token) []StateDecl {
	body := parser{tokens: tokens}
	states := make([]StateDecl, 0)

	for !body.done() {
		if body.match(lexer.COMMENT) {
			continue
		}
		start := body.pos
		if !isNameToken(body.peek().Type) {
			body.errorf(body.peek(), "expected state name")
			body.advance()
			continue
		}

		state := StateDecl{Name: body.advance().Literal}
		body.expect(lexer.COLON)
		state.Type = body.parseTypeUntil(lexer.ASSIGN_IN)
		body.expect(lexer.ASSIGN_IN)
		state.Initial = body.collectStateInitial()
		if body.match(lexer.LBRACE) {
			state.Transitions = body.parseTransitionRules()
			body.expect(lexer.RBRACE)
		}
		body.expect(lexer.SEMICOLON)
		state.Tokens = cloneTokens(tokens[start:body.pos])
		states = append(states, state)
	}

	p.diagnostics = append(p.diagnostics, body.diagnostics...)
	return states
}

func (p *parser) parseTransitionRules() []TransitionRule {
	rules := make([]TransitionRule, 0)
	for !p.done() && !p.at(lexer.RBRACE) {
		if p.match(lexer.COMMENT) {
			continue
		}
		start := p.pos
		if !p.at(lexer.SIGNAL) {
			p.errorf(p.peek(), "expected scheduler event")
			p.collectUntil(lexer.SEMICOLON, lexer.RBRACE)
			p.match(lexer.SEMICOLON)
			continue
		}

		rule := TransitionRule{Event: p.parseEventPattern()}
		p.expect(lexer.MAP_ARROW)
		rule.Expr = p.collectUntil(lexer.SEMICOLON)
		p.expect(lexer.SEMICOLON)
		rule.Tokens = cloneTokens(p.tokens[start:p.pos])
		rules = append(rules, rule)
	}
	return rules
}

func (p *parser) collectStateInitial() []lexer.Token {
	start := p.pos
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 {
			if tok.Type == lexer.SEMICOLON {
				break
			}
			if tok.Type == lexer.LBRACE && p.pos > start && p.looksLikeTransitionBlock(p.pos) {
				break
			}
		}
		depth = nextDepth(depth, tok.Type)
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}

func (p *parser) looksLikeTransitionBlock(pos int) bool {
	if pos >= len(p.tokens) || p.tokens[pos].Type != lexer.LBRACE {
		return false
	}
	next := p.nextSignificant(pos + 1)
	return next.Type == lexer.SIGNAL || next.Type == lexer.RBRACE
}

func (p *parser) parseEventPattern() EventPattern {
	start := p.pos
	event := EventPattern{Name: p.expectLiteral(lexer.SIGNAL, "scheduler event")}
	if p.match(lexer.LPAREN) {
		event.Params = p.parseParamDeclsUntil(lexer.RPAREN)
		p.expect(lexer.RPAREN)
	}
	event.Tokens = cloneTokens(p.tokens[start:p.pos])
	return event
}

func (p *parser) parseFieldDeclsUntil(end lexer.TokenType) []FieldDecl {
	fields := make([]FieldDecl, 0)
	for !p.done() && !p.at(end) {
		if p.match(lexer.COMMENT) {
			continue
		}
		field := p.parseFieldDecl()
		if field.Name != "" {
			fields = append(fields, field)
		}
		p.expect(lexer.SEMICOLON)
	}
	return fields
}

func (p *parser) parseFieldDecl() FieldDecl {
	start := p.pos
	field := FieldDecl{Name: p.expectName("field name")}
	field.Optional = p.match(lexer.QUESTION)
	p.expect(lexer.COLON)
	field.Type = p.parseTypeUntil(lexer.SEMICOLON, lexer.RBRACE)
	field.Tokens = cloneTokens(p.tokens[start:p.pos])
	return field
}

func (p *parser) parseEmitDeclsUntil(end lexer.TokenType) []EmitDecl {
	emits := make([]EmitDecl, 0)
	for !p.done() && !p.at(end) {
		if p.match(lexer.COMMENT) {
			continue
		}
		start := p.pos
		emit := EmitDecl{Event: p.expectLiteral(lexer.SIGNAL, "emitted event")}
		p.expect(lexer.COLON)
		emit.Type = p.parseTypeUntil(lexer.SEMICOLON, lexer.RBRACE)
		p.expect(lexer.SEMICOLON)
		emit.Tokens = cloneTokens(p.tokens[start:p.pos])
		emits = append(emits, emit)
	}
	return emits
}

func (p *parser) parseParamDeclsUntil(end ...lexer.TokenType) []FieldDecl {
	params := make([]FieldDecl, 0)
	for !p.done() && !p.atAny(end...) {
		if p.match(lexer.COMMENT) || p.match(lexer.COMMA) {
			continue
		}
		start := p.pos
		param := FieldDecl{Name: p.expectName("parameter name")}
		p.expect(lexer.COLON)
		param.Type = p.parseTypeUntilParamBoundary(end...)
		param.Tokens = cloneTokens(p.tokens[start:p.pos])
		if param.Name != "" {
			params = append(params, param)
		}
		p.match(lexer.COMMA)
	}
	return params
}

func (p *parser) parseTypeUntilParamBoundary(end ...lexer.TokenType) TypeRef {
	start := p.pos
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 {
			if isOneOf(tok.Type, end...) || tok.Type == lexer.COMMA {
				break
			}
			if isNameToken(tok.Type) && p.peekN(1).Type == lexer.COLON && p.pos > start {
				break
			}
		}
		depth = nextDepth(depth, tok.Type)
		p.advance()
	}
	return typeRef(p.tokens[start:p.pos])
}

func (p *parser) parseTypeUntil(end ...lexer.TokenType) TypeRef {
	return typeRef(p.collectUntil(end...))
}

func (p *parser) collectTopLevelBlockBody() []lexer.Token {
	start := p.pos
	for !p.at(lexer.EOF) {
		if p.at(lexer.PIPE_END) && p.isTopLevelEnd(p.pos) {
			body := cloneTokens(p.tokens[start:p.pos])
			p.advance()
			return body
		}
		p.advance()
	}

	p.errorf(p.peek(), "expected top-level block terminator /|")
	return cloneTokens(p.tokens[start:p.pos])
}

func (p *parser) collectUntil(end ...lexer.TokenType) []lexer.Token {
	start := p.pos
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 && isOneOf(tok.Type, end...) {
			break
		}
		depth = nextDepth(depth, tok.Type)
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}

func (p *parser) isTopLevelEnd(pos int) bool {
	next := p.nextSignificant(pos + 1)
	switch next.Type {
	case lexer.EOF, lexer.COMMENT, lexer.TAG_IMPORT, lexer.TAG_CONTRACT, lexer.TAG_LIFECYCLE, lexer.TAG_TEMPLATE, lexer.TAG_FUNC:
		return true
	default:
		return false
	}
}

func (p *parser) hasFieldShapeAt(pos int) bool {
	if pos < 0 || pos >= len(p.tokens) || !isNameToken(p.tokens[pos].Type) {
		return false
	}
	pos++
	if pos < len(p.tokens) && p.tokens[pos].Type == lexer.QUESTION {
		pos++
	}
	return pos < len(p.tokens) && p.tokens[pos].Type == lexer.COLON
}

func (p *parser) indexOf(tok lexer.Token) int {
	for i, candidate := range p.tokens {
		if candidate == tok {
			return i
		}
	}
	return -1
}

func (p *parser) nextNonComment(pos int) lexer.Token {
	for pos < len(p.tokens) && p.tokens[pos].Type == lexer.COMMENT {
		pos++
	}
	if pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[pos]
}

func splitStatements(tokens []lexer.Token) []Statement {
	segments := splitTopLevel(tokens, lexer.SEMICOLON)
	statements := make([]Statement, 0, len(segments))
	for _, segment := range segments {
		segment = trimNoise(segment)
		if len(segment) > 0 {
			statements = append(statements, Statement{Tokens: cloneTokens(segment)})
		}
	}
	return statements
}

func splitTopLevel(tokens []lexer.Token, delimiter lexer.TokenType) [][]lexer.Token {
	segments := make([][]lexer.Token, 0)
	start := 0
	depth := 0

	for i, tok := range tokens {
		if tok.Type == delimiter && depth == 0 {
			segments = append(segments, tokens[start:i])
			start = i + 1
			continue
		}
		depth = nextDepth(depth, tok.Type)
	}

	if start <= len(tokens) {
		segments = append(segments, tokens[start:])
	}
	return segments
}

func trimNoise(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && tokens[start].Type == lexer.COMMENT {
		start++
	}

	end := len(tokens)
	for end > start && tokens[end-1].Type == lexer.COMMENT {
		end--
	}

	return tokens[start:end]
}

func typeRef(tokens []lexer.Token) TypeRef {
	return TypeRef{
		Tokens: cloneTokens(tokens),
		Text:   joinTokenLiterals(tokens),
	}
}

func joinTokenLiterals(tokens []lexer.Token) string {
	if len(tokens) == 0 {
		return ""
	}

	out := tokens[0].Literal
	for i := 1; i < len(tokens); i++ {
		out += tokens[i].Literal
	}
	return out
}

func cloneTokens(tokens []lexer.Token) []lexer.Token {
	out := make([]lexer.Token, len(tokens))
	copy(out, tokens)
	return out
}

func nextDepth(depth int, typ lexer.TokenType) int {
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

func isLifecyclePhase(typ lexer.TokenType) bool {
	switch typ {
	case lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return true
	default:
		return false
	}
}

func isNameToken(typ lexer.TokenType) bool {
	switch typ {
	case lexer.IDENT, lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL,
		lexer.OPERATION, lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS,
		lexer.TARGET, lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR,
		lexer.FROM, lexer.AS, lexer.USE, lexer.IS, lexer.NOT, lexer.VOID, lexer.NULL:
		return true
	default:
		return false
	}
}

func isOneOf(typ lexer.TokenType, types ...lexer.TokenType) bool {
	for _, candidate := range types {
		if typ == candidate {
			return true
		}
	}
	return false
}

func (p *parser) expect(typ lexer.TokenType) lexer.Token {
	if p.at(typ) {
		return p.advance()
	}

	tok := p.peek()
	p.errorf(tok, "expected %s, got %s", typ, tok.Type)
	return tok
}

func (p *parser) expectLiteral(typ lexer.TokenType, label string) string {
	tok := p.expect(typ)
	if tok.Type != typ {
		return ""
	}
	return tok.Literal
}

func (p *parser) expectName(label string) string {
	if isNameToken(p.peek().Type) {
		return p.advance().Literal
	}

	tok := p.peek()
	p.errorf(tok, "expected %s, got %s", label, tok.Type)
	return ""
}

func (p *parser) match(typ lexer.TokenType) bool {
	if !p.at(typ) {
		return false
	}
	p.advance()
	return true
}

func (p *parser) at(typ lexer.TokenType) bool {
	return p.peek().Type == typ
}

func (p *parser) atAny(types ...lexer.TokenType) bool {
	return isOneOf(p.peek().Type, types...)
}

func (p *parser) done() bool {
	return p.pos >= len(p.tokens) || p.at(lexer.EOF)
}

func (p *parser) peek() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) peekN(offset int) lexer.Token {
	pos := p.pos + offset
	if pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[pos]
}

func (p *parser) nextSignificant(pos int) lexer.Token {
	for pos < len(p.tokens) && p.tokens[pos].Type == lexer.COMMENT {
		pos++
	}
	if pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[pos]
}

func (p *parser) advance() lexer.Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *parser) errorf(tok lexer.Token, format string, args ...any) {
	p.diagnostics = append(p.diagnostics, Diagnostic{
		Message: fmt.Sprintf(format, args...),
		Token:   tok,
	})
}
