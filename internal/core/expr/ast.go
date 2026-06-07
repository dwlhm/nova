package expr

// Node is a pure expression AST for Nova transition and binding expressions.
type Node interface {
	node()
}

type Literal struct {
	Value any
}

func (Literal) node() {}

type Ident struct {
	Name string
	Kind IdentKind
}

type IdentKind int

const (
	IdentLocal IdentKind = iota
	IdentState
	IdentParam
)

func (Ident) node() {}

type Unary struct {
	Op   string
	Expr Node
}

func (Unary) node() {}

type Binary struct {
	Op    string
	Left  Node
	Right Node
}

func (Binary) node() {}

type Ternary struct {
	Cond Node
	Then Node
	Else Node
}

func (Ternary) node() {}

type Call struct {
	Name string
	Args []Node
}

func (Call) node() {}

type FieldAccess struct {
	Base  Node
	Field string
}

func (FieldAccess) node() {}

type Record struct {
	Fields []RecordField
}

type RecordField struct {
	Name string
	Expr Node
}

func (Record) node() {}

type List struct {
	Elements []Node
}

func (List) node() {}

type Pipe struct {
	Left Node
	Call Call
}

func (Pipe) node() {}
