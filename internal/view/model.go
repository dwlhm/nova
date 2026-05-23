package view

import (
	"github.com/dwlhm/nova/internal/lexer"
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
	Pages       []PageRef
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

type PageRef struct {
	NodePath []int
	Path     Binding
}

type Diagnostic struct {
	Message string
	Token   lexer.Token
}
