package expr

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/lexer"
)

type exprParser struct {
	tokens []lexer.Token
	pos    int
	reg    *Registry
	states map[string]bool
	params map[string]bool
}

func Parse(tokens []lexer.Token, reg *Registry, stateNames map[string]bool, paramNames map[string]bool) (Node, error) {
	tokens = trimTokens(tokens)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty expression")
	}
	p := &exprParser{
		tokens: tokens,
		reg:    reg,
		states: stateNames,
		params: paramNames,
	}
	node, err := p.parsePipe()
	if err != nil {
		return nil, err
	}
	if !p.done() {
		return nil, fmt.Errorf("unexpected trailing tokens at %q", p.peek().Literal)
	}
	return node, nil
}

func (p *exprParser) parsePipe() (Node, error) {
	left, err := p.parseTernary()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.PIPE_FWD) {
		call, err := p.parsePipeCall()
		if err != nil {
			return nil, err
		}
		call.Args = append([]Node{left}, call.Args...)
		left = call
	}
	return left, nil
}

func (p *exprParser) parsePipeCall() (Call, error) {
	if p.done() {
		return Call{}, fmt.Errorf("pipeline missing function")
	}
	tok := p.peek()
	if tok.Type != lexer.IDENT {
		return Call{}, fmt.Errorf("pipeline expects function name, got %q", tok.Literal)
	}
	name := tok.Literal
	fn, ok := p.reg.Lookup(name)
	if !ok {
		return Call{}, fmt.Errorf("unknown function %s in pipeline", name)
	}
	p.pos++
	args, err := p.parseCallArgs(fn.Params)
	if err != nil {
		return Call{}, err
	}
	return Call{Name: name, Args: args}, nil
}

func (p *exprParser) parseTernary() (Node, error) {
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if !p.match(lexer.QUESTION) {
		return node, nil
	}
	thenBranch, err := p.parseTernary()
	if err != nil {
		return nil, err
	}
	if !p.match(lexer.COLON) {
		return nil, fmt.Errorf("ternary missing ':'")
	}
	elseBranch, err := p.parseTernary()
	if err != nil {
		return nil, err
	}
	return Ternary{Cond: node, Then: thenBranch, Else: elseBranch}, nil
}

func (p *exprParser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.OR) {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: "||", Left: left, Right: right}
	}
	return left, nil
}

func (p *exprParser) parseAnd() (Node, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.AND) {
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: "&&", Left: left, Right: right}
	}
	return left, nil
}

func (p *exprParser) parseEquality() (Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch {
		case p.match(lexer.EQ):
			op = "=="
		case p.match(lexer.NOT_EQ):
			op = "!="
		default:
			return left, nil
		}
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseComparison() (Node, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch {
		case p.match(lexer.GTE):
			op = ">="
		case p.match(lexer.LTE):
			op = "<="
		case p.match(lexer.GT):
			op = ">"
		case p.match(lexer.LT):
			op = "<"
		default:
			return left, nil
		}
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseAdditive() (Node, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch {
		case p.match(lexer.PLUS):
			op = "+"
		case p.match(lexer.MINUS):
			op = "-"
		default:
			return left, nil
		}
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseMultiplicative() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch {
		case p.match(lexer.ASTERISK):
			op = "*"
		case p.match(lexer.SLASH):
			op = "/"
		default:
			return left, nil
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseUnary() (Node, error) {
	if p.match(lexer.BANG) {
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return Unary{Op: "!", Expr: expr}, nil
	}
	if p.match(lexer.MINUS) {
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return Unary{Op: "-", Expr: expr}, nil
	}
	return p.parsePostfix()
}

func (p *exprParser) parsePostfix() (Node, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.DOT) {
		if p.done() || p.peek().Type != lexer.IDENT {
			return nil, fmt.Errorf("field access missing name")
		}
		field := p.peek().Literal
		p.pos++
		node = FieldAccess{Base: node, Field: field}
	}
	return node, nil
}

func (p *exprParser) parsePrimary() (Node, error) {
	if p.done() {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	if p.match(lexer.LPAREN) {
		node, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		if !p.match(lexer.RPAREN) {
			return nil, fmt.Errorf("missing ')'")
		}
		return node, nil
	}
	if record, ok := p.tryParseRecord(); ok {
		return record, nil
	}
	if list, ok := p.tryParseList(); ok {
		return list, nil
	}
	tok := p.peek()
	switch tok.Type {
	case lexer.STRING:
		p.pos++
		return Literal{Value: tok.Literal}, nil
	case lexer.NUMBER:
		p.pos++
		return parseNumberLiteral(tok.Literal)
	case lexer.TRUE:
		p.pos++
		return Literal{Value: true}, nil
	case lexer.FALSE:
		p.pos++
		return Literal{Value: false}, nil
	case lexer.NULL, lexer.VOID:
		p.pos++
		return Literal{Value: nil}, nil
	case lexer.IDENT:
		name := tok.Literal
		if fn, ok := p.reg.Lookup(name); ok && p.hasFuncCallArguments(len(fn.Params)) {
			p.pos++
			args, err := p.parseCallArgs(fn.Params)
			if err != nil {
				return nil, err
			}
			return Call{Name: name, Args: args}, nil
		}
		p.pos++
		kind := IdentLocal
		switch {
		case p.states[name]:
			kind = IdentState
		case p.params[name]:
			kind = IdentParam
		}
		node := Ident{Name: name, Kind: kind}
		return p.parsePostfixFrom(node)
	default:
		return nil, fmt.Errorf("unexpected token %q", tok.Literal)
	}
}

func (p *exprParser) hasFuncCallArguments(paramCount int) bool {
	if p.pos+1 >= len(p.tokens) {
		return false
	}
	rest := trimTokens(p.tokens[p.pos+1:])
	if len(rest) == 0 {
		return false
	}
	next := rest[0]
	if paramCount == 0 {
		return next.Type == lexer.LPAREN
	}
	return !isNoise(next.Type)
}

func (p *exprParser) parseCallArgs(paramNames []string) ([]Node, error) {
	if len(paramNames) == 0 {
		if p.match(lexer.LPAREN) {
			if !p.match(lexer.RPAREN) {
				return nil, fmt.Errorf("missing ')' after call")
			}
		}
		return nil, nil
	}
	args := make([]Node, 0, len(paramNames))
	for index := range paramNames {
		if p.done() {
			return nil, fmt.Errorf("missing argument %d for call", index+1)
		}
		var arg Node
		var err error
		if index == len(paramNames)-1 {
			arg, err = p.parsePipe()
		} else {
			arg, err = p.parsePrefixArg()
		}
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	return args, nil
}

func (p *exprParser) parsePrefixArg() (Node, error) {
	if p.done() {
		return nil, fmt.Errorf("missing argument")
	}
	if p.match(lexer.LPAREN) {
		node, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		if !p.match(lexer.RPAREN) {
			return nil, fmt.Errorf("missing ')'")
		}
		return node, nil
	}
	if record, ok := p.tryParseRecord(); ok {
		return record, nil
	}
	if list, ok := p.tryParseList(); ok {
		return list, nil
	}
	tok := p.peek()
	switch tok.Type {
	case lexer.STRING, lexer.NUMBER, lexer.TRUE, lexer.FALSE, lexer.NULL, lexer.VOID:
		return p.parsePrimary()
	case lexer.IDENT:
		name := tok.Literal
		if fn, ok := p.reg.Lookup(name); ok && p.hasFuncCallArguments(len(fn.Params)) {
			p.pos++
			return p.parseCallNode(name, fn.Params)
		}
		p.pos++
		kind := IdentLocal
		switch {
		case p.states[name]:
			kind = IdentState
		case p.params[name]:
			kind = IdentParam
		}
		node := Ident{Name: name, Kind: kind}
		return p.parsePostfixFrom(node)
	default:
		return nil, fmt.Errorf("unexpected argument token %q", tok.Literal)
	}
}

func (p *exprParser) parseCallNode(name string, paramNames []string) (Node, error) {
	args, err := p.parseCallArgs(paramNames)
	if err != nil {
		return nil, err
	}
	node := Call{Name: name, Args: args}
	return p.parsePostfixFrom(node)
}

func (p *exprParser) parsePostfixFrom(node Node) (Node, error) {
	for p.match(lexer.DOT) {
		if p.done() || p.peek().Type != lexer.IDENT {
			return nil, fmt.Errorf("field access missing name")
		}
		field := p.peek().Literal
		p.pos++
		node = FieldAccess{Base: node, Field: field}
	}
	return node, nil
}

func (p *exprParser) tryParseRecord() (Record, bool) {
	if p.done() || p.peek().Type != lexer.LBRACE {
		return Record{}, false
	}
	start := p.pos
	p.pos++
	fields := make([]RecordField, 0)
	for !p.done() && p.peek().Type != lexer.RBRACE {
		for p.match(lexer.SEMICOLON) || p.match(lexer.COMMA) {
		}
		if p.done() || p.peek().Type == lexer.RBRACE {
			break
		}
		if !isRecordFieldName(p.peek().Type) {
			p.pos = start
			return Record{}, false
		}
		name := p.peek().Literal
		p.pos++
		if !p.match(lexer.ASSIGN_IN) {
			p.pos = start
			return Record{}, false
		}
		value, err := p.parseRecordValue()
		if err != nil {
			p.pos = start
			return Record{}, false
		}
		fields = append(fields, RecordField{Name: name, Expr: value})
	}
	if !p.match(lexer.RBRACE) {
		p.pos = start
		return Record{}, false
	}
	return Record{Fields: fields}, true
}

func (p *exprParser) tryParseList() (List, bool) {
	if p.done() || p.peek().Type != lexer.LBRACKET {
		return List{}, false
	}
	start := p.pos
	p.pos++
	elements := make([]Node, 0)
	for !p.done() && p.peek().Type != lexer.RBRACKET {
		for p.match(lexer.SEMICOLON) || p.match(lexer.COMMA) {
		}
		if p.done() || p.peek().Type == lexer.RBRACKET {
			break
		}
		element, err := p.parsePipe()
		if err != nil {
			p.pos = start
			return List{}, false
		}
		elements = append(elements, element)
	}
	if !p.match(lexer.RBRACKET) {
		p.pos = start
		return List{}, false
	}
	return List{Elements: elements}, true
}

func (p *exprParser) parseRecordValue() (Node, error) {
	start := p.pos
	depth := 0
	for p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		if depth == 0 && (tok.Type == lexer.SEMICOLON || tok.Type == lexer.COMMA) {
			break
		}
		depth = recordDepth(depth, tok.Type)
		p.pos++
	}
	if start == p.pos {
		return nil, fmt.Errorf("empty record field")
	}
	chunk := trimTokens(p.tokens[start:p.pos])
	return p.parsePipeFromTokens(chunk)
}

func (p *exprParser) parsePipeFromTokens(tokens []lexer.Token) (Node, error) {
	child := &exprParser{tokens: tokens, reg: p.reg, states: p.states, params: p.params}
	return child.parsePipe()
}

func (p *exprParser) peek() lexer.Token {
	if p.done() {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *exprParser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *exprParser) match(typ lexer.TokenType) bool {
	if p.done() || p.peek().Type != typ {
		return false
	}
	p.pos++
	return true
}

func trimTokens(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && isNoise(tokens[start].Type) {
		start++
	}
	end := len(tokens)
	for end > start && isNoise(tokens[end-1].Type) {
		end--
	}
	return tokens[start:end]
}

func isNoise(typ lexer.TokenType) bool {
	return typ == lexer.EOF || typ == lexer.COMMENT || typ == lexer.SEMICOLON
}

func isRecordFieldName(typ lexer.TokenType) bool {
	switch typ {
	case lexer.IDENT, lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL,
		lexer.OPERATION, lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS,
		lexer.TARGET, lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return true
	default:
		return false
	}
}

func recordDepth(depth int, typ lexer.TokenType) int {
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

func parseNumberLiteral(literal string) (Literal, error) {
	if stringsContainsDot(literal) {
		var value float64
		_, err := fmt.Sscanf(literal, "%f", &value)
		if err != nil {
			return Literal{}, err
		}
		return Literal{Value: value}, nil
	}
	var value int64
	_, err := fmt.Sscanf(literal, "%d", &value)
	if err != nil {
		return Literal{}, err
	}
	return Literal{Value: float64(value)}, nil
}

func stringsContainsDot(value string) bool {
	for _, ch := range value {
		if ch == '.' {
			return true
		}
	}
	return false
}
