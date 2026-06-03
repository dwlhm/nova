package style

// Scope matches the single scope header in a .nova-style file.
type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeApp    Scope = "app"
)

// Sheet is the parsed portable style document (format v1).
type Sheet struct {
	SourcePath string
	Scope      Scope
	Tokens     map[string]string
	Classes    map[string]ClassRule
	States     []StateRule
}

type ClassRule struct {
	Properties map[string]string
}

type StateRule struct {
	Class      string
	Pseudo     string
	Properties map[string]string
}

// ImportKind matches parser style import kinds.
type ImportKind string

const (
	ImportStyle      ImportKind = "style"
	ImportStylesheet ImportKind = "stylesheet"
)

// ImportRef is a resolved style dependency from the module graph.
type ImportRef struct {
	Kind           ImportKind
	Path           string
	RequestingFile string
}
