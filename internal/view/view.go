package view

import (
	"context"
	"fmt"
	"strings"

	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/scheduler"
)

type IR struct {
	Target   string
	Nodes    []Node
	Metadata DependencyMetadata
}

type Node struct {
	Kind     string
	Target   string
	Props    map[string]Binding
	Events   map[string]EventRoute
	Children []Node
	Key      *Binding
}

type Binding struct {
	Tokens []lexer.Token
	Text   string
}

type EventRoute struct {
	Slot  string
	Event scheduler.SchedulerEvent
	Args  []Binding
}

type DependencyMetadata struct {
	Bindings    []BindingRef
	EventRoutes []EventRouteRef
}

type BindingRef struct {
	NodePath []int
	Prop     string
	States   []string
}

type EventRouteRef struct {
	NodePath []int
	Slot     string
	Event    scheduler.SchedulerEvent
	Arity    int
}

type Diagnostic struct {
	Message string
	Token   lexer.Token
}

type RendererPort interface {
	Mount(context.Context, IR) error
	Update(context.Context, scheduler.StateCommit, DependencyMetadata) error
	Dispose(context.Context) error
}

func Project(template parser.TemplateDecl, stateNames map[string]bool) (IR, []Diagnostic) {
	p := viewParser{tokens: template.Tokens}
	nodes := p.parseNodes(false)
	ir := IR{
		Target: template.Target,
		Nodes:  nodes,
	}
	ir.Metadata = BuildMetadata(ir.Nodes, stateNames)
	return ir, p.diagnostics
}

func BuildMetadata(nodes []Node, stateNames map[string]bool) DependencyMetadata {
	metadata := DependencyMetadata{}
	for i, node := range nodes {
		metadata = appendNodeMetadata(metadata, node, []int{i}, stateNames)
	}
	return metadata
}

func appendNodeMetadata(metadata DependencyMetadata, node Node, path []int, stateNames map[string]bool) DependencyMetadata {
	if node.Key != nil {
		metadata = appendBindingMetadata(metadata, path, "key", *node.Key, stateNames)
	}
	for prop, binding := range node.Props {
		metadata = appendBindingMetadata(metadata, path, prop, binding, stateNames)
	}
	for slot, route := range node.Events {
		metadata.EventRoutes = append(metadata.EventRoutes, EventRouteRef{
			NodePath: clonePath(path),
			Slot:     slot,
			Event:    route.Event,
			Arity:    len(route.Args),
		})
		for i, arg := range route.Args {
			metadata = appendBindingMetadata(metadata, path, slot+"#arg"+string(rune('0'+i)), arg, stateNames)
		}
	}
	for i, child := range node.Children {
		childPath := append(clonePath(path), i)
		metadata = appendNodeMetadata(metadata, child, childPath, stateNames)
	}
	return metadata
}

func appendBindingMetadata(metadata DependencyMetadata, path []int, prop string, binding Binding, stateNames map[string]bool) DependencyMetadata {
	states := bindingStates(binding.Tokens, stateNames)
	if len(states) == 0 {
		return metadata
	}
	metadata.Bindings = append(metadata.Bindings, BindingRef{
		NodePath: clonePath(path),
		Prop:     prop,
		States:   states,
	})
	return metadata
}

type viewParser struct {
	tokens      []lexer.Token
	pos         int
	diagnostics []Diagnostic
}

func (p *viewParser) parseNodes(stopAtPipeEnd bool) []Node {
	nodes := make([]Node, 0)
	for !p.done() {
		if p.match(lexer.COMMENT) {
			continue
		}
		if p.at(lexer.PIPE_END) {
			if stopAtPipeEnd {
				p.advance()
			}
			return nodes
		}
		if p.at(lexer.LT) {
			nodes = append(nodes, p.parseNode())
			continue
		}
		text := p.collectText()
		if len(text) > 0 {
			nodes = append(nodes, Node{
				Kind:  "#text",
				Props: map[string]Binding{"value": binding(text)},
			})
		}
	}
	return nodes
}

func (p *viewParser) parseNode() Node {
	p.expect(lexer.LT)
	name := p.expectName("view node")
	node := Node{
		Kind:   name.Literal,
		Props:  make(map[string]Binding),
		Events: make(map[string]EventRoute),
	}

	for !p.done() && !p.at(lexer.GT) && !p.at(lexer.PIPE_END) {
		if p.match(lexer.COMMENT) {
			continue
		}
		key := p.advance()
		if !isAttributeName(key.Type) {
			p.errorf(key, "expected view attribute or event route")
			continue
		}
		switch {
		case p.match(lexer.ASSIGN_IN):
			value := binding(p.collectAttributeExpression())
			if key.Literal == "key" {
				node.Key = &value
			} else {
				node.Props[key.Literal] = value
			}
		case p.match(lexer.MAP_ARROW):
			route := p.parseEventRoute(key.Literal)
			node.Events[key.Literal] = route
		default:
			p.errorf(key, "expected <- binding or -> event route")
		}
	}

	if p.match(lexer.PIPE_END) {
		return node
	}
	p.expect(lexer.GT)
	node.Children = p.parseNodes(true)
	return node
}

func (p *viewParser) parseEventRoute(slot string) EventRoute {
	event := p.expect(lexer.SIGNAL)
	route := EventRoute{Slot: slot, Event: scheduler.SchedulerEvent(event.Literal)}
	if !p.match(lexer.LPAREN) {
		return route
	}
	for !p.done() && !p.at(lexer.RPAREN) {
		arg := p.collectEventArg()
		if len(arg) > 0 {
			route.Args = append(route.Args, binding(arg))
		}
		p.match(lexer.COMMA)
	}
	p.expect(lexer.RPAREN)
	return route
}

func (p *viewParser) collectAttributeExpression() []lexer.Token {
	start := p.pos
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 {
			if tok.Type == lexer.GT || tok.Type == lexer.PIPE_END {
				break
			}
			if isAttributeName(tok.Type) && (p.peekN(1).Type == lexer.ASSIGN_IN || p.peekN(1).Type == lexer.MAP_ARROW) {
				break
			}
		}
		depth = nextDepth(depth, tok.Type)
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}

func (p *viewParser) collectEventArg() []lexer.Token {
	start := p.pos
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 && (tok.Type == lexer.COMMA || tok.Type == lexer.RPAREN) {
			break
		}
		depth = nextDepth(depth, tok.Type)
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}

func (p *viewParser) collectText() []lexer.Token {
	start := p.pos
	for !p.done() && !p.at(lexer.LT) && !p.at(lexer.PIPE_END) {
		p.advance()
	}
	return cloneTokens(p.tokens[start:p.pos])
}

func binding(tokens []lexer.Token) Binding {
	tokens = trimNoise(tokens)
	return Binding{
		Tokens: cloneTokens(tokens),
		Text:   joinTokenLiterals(tokens),
	}
}

func bindingStates(tokens []lexer.Token, stateNames map[string]bool) []string {
	seen := make(map[string]bool)
	states := make([]string, 0)
	for i, tok := range tokens {
		if tok.Type != lexer.IDENT || !stateNames[tok.Literal] {
			continue
		}
		if i > 0 && tokens[i-1].Type == lexer.DOT {
			continue
		}
		if seen[tok.Literal] {
			continue
		}
		seen[tok.Literal] = true
		states = append(states, tok.Literal)
	}
	return states
}

func (p *viewParser) match(typ lexer.TokenType) bool {
	if !p.at(typ) {
		return false
	}
	p.advance()
	return true
}

func (p *viewParser) expect(typ lexer.TokenType) lexer.Token {
	if p.at(typ) {
		return p.advance()
	}
	tok := p.peek()
	p.errorf(tok, "expected %s", typ)
	return tok
}

func (p *viewParser) expectName(label string) lexer.Token {
	if isName(p.peek().Type) {
		return p.advance()
	}
	tok := p.peek()
	p.errorf(tok, "expected %s", label)
	return tok
}

func (p *viewParser) at(typ lexer.TokenType) bool {
	return p.peek().Type == typ
}

func (p *viewParser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *viewParser) peek() lexer.Token {
	if p.done() {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *viewParser) peekN(offset int) lexer.Token {
	pos := p.pos + offset
	if pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[pos]
}

func (p *viewParser) advance() lexer.Token {
	tok := p.peek()
	if !p.done() {
		p.pos++
	}
	return tok
}

func (p *viewParser) errorf(tok lexer.Token, message string, args ...any) {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	p.diagnostics = append(p.diagnostics, Diagnostic{Message: message, Token: tok})
}

func isAttributeName(typ lexer.TokenType) bool {
	return isName(typ) || typ == lexer.SIGNAL
}

func isName(typ lexer.TokenType) bool {
	switch typ {
	case lexer.IDENT, lexer.TYPE_STRING, lexer.TYPE_NUMBER, lexer.TYPE_BOOLEAN, lexer.TYPE_UNKNOWN,
		lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL, lexer.OPERATION,
		lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS, lexer.TARGET,
		lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return true
	default:
		return false
	}
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

func joinTokenLiterals(tokens []lexer.Token) string {
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if tok.Literal != "" {
			parts = append(parts, tok.Literal)
		}
	}
	return strings.Join(parts, "")
}

func cloneTokens(tokens []lexer.Token) []lexer.Token {
	out := make([]lexer.Token, len(tokens))
	copy(out, tokens)
	return out
}

func clonePath(path []int) []int {
	out := make([]int, len(path))
	copy(out, path)
	return out
}
