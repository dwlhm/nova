package view

import (
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/scheduler"
)

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
