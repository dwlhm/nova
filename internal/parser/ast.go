package parser

import "github.com/dwlhm/nova/internal/lexer"

type File struct {
	Imports              []ImportDecl
	ExternalImports      []ExternalImportDecl
	ContractTypes        []ContractTypeDecl
	ContractStates       []ContractStateDecl
	ContractCapabilities []ContractCapabilityDecl
	Funcs                []FuncDecl
	Lifecycles           []LifecycleDecl
	Templates            []TemplateDecl
	Comments             []Comment
}

type ImportDecl struct {
	Kind  ImportKind
	Items []ImportItem
	From  string
}

type ImportKind string

const (
	ImportCapability ImportKind = "capability"
	ImportState      ImportKind = "state"
	ImportEvent      ImportKind = "event"
)

type ImportItem struct {
	Name   string
	Alias  string
	Tokens []lexer.Token
}

type ExternalImportDecl struct {
	Name       string
	From       string
	Operations []ExternalOperationDecl
	Tokens     []lexer.Token
}

type ExternalOperationDecl struct {
	Name   string
	Inputs []FieldDecl
	Output TypeRef
	Tokens []lexer.Token
}

type ContractTypeDecl struct {
	Name   string
	Opaque bool
	Fields []FieldDecl
	Alias  TypeRef
	Tokens []lexer.Token
}

type ContractStateDecl struct {
	Name   string
	States []StateDecl
	Tokens []lexer.Token
}

type StateDecl struct {
	Name        string
	Type        TypeRef
	Initial     []lexer.Token
	Transitions []TransitionRule
	Tokens      []lexer.Token
}

type TransitionRule struct {
	Event  EventPattern
	Expr   []lexer.Token
	Tokens []lexer.Token
}

type ContractCapabilityDecl struct {
	Name   string
	Props  []FieldDecl
	Emits  []EmitDecl
	Tokens []lexer.Token
}

type EmitDecl struct {
	Event  string
	Type   TypeRef
	Tokens []lexer.Token
}

type FuncDecl struct {
	Name   string
	Params []FieldDecl
	Return TypeRef
	Body   []lexer.Token
}

type TemplateDecl struct {
	Target string
	Tokens []lexer.Token
}

type LifecycleDecl struct {
	Phase      string
	Event      string
	Statements []Statement
	Tokens     []lexer.Token
}

type Statement struct {
	Tokens []lexer.Token
}

type EventPattern struct {
	Name   string
	Params []FieldDecl
	Tokens []lexer.Token
}

type FieldDecl struct {
	Name     string
	Optional bool
	Type     TypeRef
	Tokens   []lexer.Token
}

type TypeRef struct {
	Tokens []lexer.Token
	Text   string
}

type Comment struct {
	Text string
}

type Diagnostic struct {
	Message string
	Token   lexer.Token
}
