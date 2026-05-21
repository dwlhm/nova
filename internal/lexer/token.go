package lexer

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Offset  int
	Line    int
	Column  int
	Length  int
}

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	// Literals
	IDENT  TokenType = "IDENT"
	STRING TokenType = "STRING"
	NUMBER TokenType = "NUMBER"
	SIGNAL TokenType = "SIGNAL"

	// Top-level tags
	TAG_IMPORT    TokenType = "<import"
	TAG_CONTRACT  TokenType = "<contract"
	TAG_LIFECYCLE TokenType = "<lifecycle"
	TAG_TEMPLATE  TokenType = "<template"
	TAG_FUNC      TokenType = "<func"

	COMMENT TokenType = "//"

	// Keywords
	FROM       TokenType = "from"
	AS         TokenType = "as"
	USE        TokenType = "use"
	IS         TokenType = "is"
	NOT        TokenType = "not"
	TRUE       TokenType = "true"
	FALSE      TokenType = "false"
	VOID       TokenType = "void"
	NULL       TokenType = "null"
	TYPE       TokenType = "type"
	STATE      TokenType = "state"
	EVENT      TokenType = "event"
	CAPABILITY TokenType = "capability"
	EXTERNAL   TokenType = "external"
	OPERATION  TokenType = "operation"
	INPUT      TokenType = "input"
	OUTPUT     TokenType = "output"
	PROPS      TokenType = "props"
	EMITS      TokenType = "emits"
	RETURNS    TokenType = "returns"
	TARGET     TokenType = "target"
	MOUNT      TokenType = "mount"
	DISPOSE    TokenType = "dispose"
	BEFORE     TokenType = "before"
	AFTER      TokenType = "after"
	ERROR      TokenType = "error"

	// Types
	TYPE_STRING  TokenType = "string"
	TYPE_NUMBER  TokenType = "number"
	TYPE_BOOLEAN TokenType = "boolean"
	TYPE_UNKNOWN TokenType = "unknown"

	// Legacy block classifiers. Kept as tokens so old files produce clear
	// parser diagnostics instead of generic identifiers.
	BLOCK_TYPE   TokenType = "Type"
	BLOCK_PROPS  TokenType = "Props"
	BLOCK_STATE  TokenType = "State"
	BLOCK_DRIVER TokenType = "Driver"

	// Compound Operators
	ASSIGN_IN TokenType = "<-" // input
	MAP_ARROW TokenType = "->" // output
	PIPE_END  TokenType = "/|"
	PIPE_FWD  TokenType = "|>"
	GATE_OPEN TokenType = "?|"
	GATE_SEP  TokenType = ":|"

	SCOPE     TokenType = "::"
	EQ        TokenType = "=="
	NOT_EQ    TokenType = "!="
	LTE       TokenType = "<="
	GTE       TokenType = ">="
	AND       TokenType = "&&"
	OR        TokenType = "||"
	SPREAD    TokenType = "..."
	LT        TokenType = "<"
	GT        TokenType = ">"
	PLUS      TokenType = "+"
	MINUS     TokenType = "-"
	ASTERISK  TokenType = "*"
	SLASH     TokenType = "/"
	BANG      TokenType = "!"
	COMMA     TokenType = ","
	PIPE      TokenType = "|"
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	LBRACKET  TokenType = "["
	RBRACKET  TokenType = "]"
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"
	COLON     TokenType = ":"
	SEMICOLON TokenType = ";"
	DOT       TokenType = "."
	QUESTION  TokenType = "?"
)
