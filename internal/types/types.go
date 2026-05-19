package types

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
)

type Kind string

const (
	KindPrimitive Kind = "primitive"
	KindUnknown   Kind = "unknown"
	KindVoid      Kind = "void"
	KindNull      Kind = "null"
	KindLiteral   Kind = "literal"
	KindArray     Kind = "array"
	KindRecord    Kind = "record"
	KindUnion     Kind = "union"
	KindOpaque    Kind = "opaque"
	KindNamed     Kind = "named"
	KindInvalid   Kind = "invalid"
)

type LiteralKind string

const (
	LiteralString  LiteralKind = "string"
	LiteralNumber  LiteralKind = "number"
	LiteralBoolean LiteralKind = "boolean"
	LiteralNull    LiteralKind = "null"
)

type Type struct {
	Kind        Kind
	Name        string
	Element     *Type
	Options     []Type
	Fields      []Field
	LiteralKind LiteralKind
	Literal     any
}

type Field struct {
	Name     string
	Optional bool
	Type     Type
}

type Environment struct {
	Definitions                   map[string]Type
	AllowUnknownNamedSerializable bool
}

type Diagnostic struct {
	Message string
	Token   lexer.Token
}

type Scope map[string]Type

type OpaqueValue struct {
	Type  string
	Value any
}

func EmptyEnvironment() Environment {
	return Environment{
		Definitions:                   make(map[string]Type),
		AllowUnknownNamedSerializable: true,
	}
}

func BuildEnvironment(file parser.File) (Environment, []Diagnostic) {
	env := EmptyEnvironment()
	diagnostics := make([]Diagnostic, 0)

	for _, decl := range file.ContractTypes {
		if decl.Name == "" {
			continue
		}
		env.Definitions[decl.Name] = Type{Kind: KindNamed, Name: decl.Name}
	}

	for _, decl := range file.ContractTypes {
		if decl.Name == "" {
			continue
		}
		switch {
		case decl.Opaque:
			env.Definitions[decl.Name] = Type{Kind: KindOpaque, Name: decl.Name}
		case len(decl.Fields) > 0:
			fields := make([]Field, 0, len(decl.Fields))
			for _, field := range decl.Fields {
				fieldType, fieldDiagnostics := ParseRef(env, field.Type)
				diagnostics = append(diagnostics, fieldDiagnostics...)
				fields = append(fields, Field{
					Name:     field.Name,
					Optional: field.Optional,
					Type:     fieldType,
				})
			}
			env.Definitions[decl.Name] = Type{Kind: KindRecord, Name: decl.Name, Fields: fields}
		default:
			alias, aliasDiagnostics := ParseRef(env, decl.Alias)
			diagnostics = append(diagnostics, aliasDiagnostics...)
			env.Definitions[decl.Name] = alias
		}
	}

	return env, diagnostics
}

func ParseRef(env Environment, ref parser.TypeRef) (Type, []Diagnostic) {
	tokens := ref.Tokens
	if len(tokens) == 0 && strings.TrimSpace(ref.Text) != "" {
		tokens = tokenizeTypeText(ref.Text)
	}
	return ParseTokens(env, tokens)
}

func ParseText(text string) (Type, []Diagnostic) {
	return ParseTokens(EmptyEnvironment(), tokenizeTypeText(text))
}

func ParseTokens(env Environment, tokens []lexer.Token) (Type, []Diagnostic) {
	p := typeParser{tokens: trimEOF(tokens), env: env}
	typ := p.parseUnion()
	if typ.Kind == "" {
		typ = Type{Kind: KindInvalid}
	}
	for !p.done() {
		if p.peek().Type != lexer.COMMENT {
			p.errorf(p.peek(), "unexpected token in type expression %s", p.peek().Literal)
		}
		p.advance()
	}
	return typ, p.diagnostics
}

func ValidateValueText(typeRef string, value any) error {
	typ, diagnostics := ParseText(typeRef)
	if len(diagnostics) > 0 {
		return fmt.Errorf("invalid type %s: %s", typeRef, diagnostics[0].Message)
	}
	return EmptyEnvironment().ValidateValue(typ, value)
}

func (env Environment) ValidateValue(typ Type, value any) error {
	if env.matches(typ, value) {
		return nil
	}
	return fmt.Errorf("expected %s", Format(typ))
}

func (env Environment) Assignable(source Type, target Type) bool {
	source = env.resolve(source)
	target = env.resolve(target)

	if target.Kind == KindUnknown {
		return true
	}
	if source.Kind == KindInvalid || target.Kind == KindInvalid {
		return false
	}
	if target.Kind == KindUnion {
		for _, option := range target.Options {
			if env.Assignable(source, option) {
				return true
			}
		}
		return false
	}
	if source.Kind == KindUnion {
		for _, option := range source.Options {
			if !env.Assignable(option, target) {
				return false
			}
		}
		return true
	}
	if source.Kind == KindLiteral {
		switch source.LiteralKind {
		case LiteralString:
			return target.Kind == KindPrimitive && target.Name == "string" ||
				target.Kind == KindLiteral && target.LiteralKind == LiteralString && target.Literal == source.Literal
		case LiteralNumber:
			return target.Kind == KindPrimitive && target.Name == "number" ||
				target.Kind == KindLiteral && target.LiteralKind == LiteralNumber && target.Literal == source.Literal
		case LiteralBoolean:
			return target.Kind == KindPrimitive && target.Name == "boolean" ||
				target.Kind == KindLiteral && target.LiteralKind == LiteralBoolean && target.Literal == source.Literal
		case LiteralNull:
			return target.Kind == KindNull
		}
	}
	if source.Kind == KindNull {
		return target.Kind == KindNull
	}
	if source.Kind != target.Kind {
		return false
	}
	switch source.Kind {
	case KindPrimitive, KindOpaque, KindNamed:
		return source.Name == target.Name
	case KindUnknown, KindVoid, KindNull:
		return true
	case KindArray:
		return source.Element != nil && target.Element != nil && env.Assignable(*source.Element, *target.Element)
	case KindRecord:
		return env.recordAssignable(source, target)
	case KindLiteral:
		return source.LiteralKind == target.LiteralKind && source.Literal == target.Literal
	default:
		return false
	}
}

func (env Environment) InferExpression(scope Scope, tokens []lexer.Token) (Type, bool) {
	tokens = trimNoise(tokens)
	tokens = trimWrappedParens(tokens)
	if len(tokens) == 0 {
		return Type{}, false
	}
	if len(tokens) == 1 {
		return env.inferSingle(scope, tokens[0])
	}
	if fieldType, ok := env.inferFieldAccess(scope, tokens); ok {
		return fieldType, true
	}
	if typ, ok := env.inferInfix(scope, tokens, lexer.PLUS); ok {
		return typ, true
	}
	for _, op := range []lexer.TokenType{lexer.MINUS, lexer.ASTERISK, lexer.SLASH} {
		if typ, ok := env.inferInfix(scope, tokens, op); ok {
			return typ, true
		}
	}
	return Type{}, false
}

func Format(typ Type) string {
	switch typ.Kind {
	case KindPrimitive, KindOpaque, KindNamed:
		return typ.Name
	case KindUnknown:
		return "unknown"
	case KindVoid:
		return "void"
	case KindNull:
		return "null"
	case KindLiteral:
		switch typ.LiteralKind {
		case LiteralString:
			return strconv.Quote(fmt.Sprint(typ.Literal))
		case LiteralNumber, LiteralBoolean:
			return fmt.Sprint(typ.Literal)
		case LiteralNull:
			return "null"
		default:
			return fmt.Sprint(typ.Literal)
		}
	case KindArray:
		if typ.Element == nil {
			return "invalid[]"
		}
		return Format(*typ.Element) + "[]"
	case KindUnion:
		parts := make([]string, 0, len(typ.Options))
		for _, option := range typ.Options {
			parts = append(parts, Format(option))
		}
		return strings.Join(parts, " | ")
	case KindRecord:
		if typ.Name != "" {
			return typ.Name
		}
		return "record"
	default:
		return "invalid"
	}
}

func IsSerializable(value any) bool {
	if value == nil {
		return true
	}
	if opaque, ok := value.(OpaqueValue); ok {
		return IsSerializable(opaque.Value)
	}
	return isSerializableValue(reflect.ValueOf(value))
}

func (env Environment) matches(typ Type, value any) bool {
	typ = env.resolve(typ)
	switch typ.Kind {
	case KindInvalid:
		return false
	case KindUnknown:
		return IsSerializable(value)
	case KindVoid, KindNull:
		return value == nil
	case KindLiteral:
		return literalMatches(typ, value)
	case KindPrimitive:
		return primitiveMatches(typ.Name, value)
	case KindArray:
		return typ.Element != nil && env.arrayMatches(*typ.Element, value)
	case KindRecord:
		return env.recordMatches(typ, value)
	case KindUnion:
		for _, option := range typ.Options {
			if env.matches(option, value) {
				return true
			}
		}
		return false
	case KindOpaque:
		opaque, ok := value.(OpaqueValue)
		return ok && opaque.Type == typ.Name && IsSerializable(opaque.Value)
	case KindNamed:
		return env.AllowUnknownNamedSerializable && IsSerializable(value)
	default:
		return false
	}
}

func (env Environment) resolve(typ Type) Type {
	if typ.Kind != KindNamed {
		return typ
	}
	resolved, ok := env.Definitions[typ.Name]
	if !ok {
		return typ
	}
	if resolved.Kind == KindNamed && resolved.Name == typ.Name {
		return typ
	}
	return env.resolve(resolved)
}

func (env Environment) recordAssignable(source Type, target Type) bool {
	sourceFields := make(map[string]Field, len(source.Fields))
	for _, field := range source.Fields {
		sourceFields[field.Name] = field
	}
	for _, targetField := range target.Fields {
		sourceField, ok := sourceFields[targetField.Name]
		if !ok {
			if targetField.Optional {
				continue
			}
			return false
		}
		if !env.Assignable(sourceField.Type, targetField.Type) {
			return false
		}
	}
	return true
}

func (env Environment) inferSingle(scope Scope, tok lexer.Token) (Type, bool) {
	switch tok.Type {
	case lexer.STRING:
		return Type{Kind: KindLiteral, LiteralKind: LiteralString, Literal: tok.Literal}, true
	case lexer.NUMBER:
		return Type{Kind: KindLiteral, LiteralKind: LiteralNumber, Literal: tok.Literal}, true
	case lexer.TRUE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: true}, true
	case lexer.FALSE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: false}, true
	case lexer.NULL:
		return Type{Kind: KindNull}, true
	case lexer.VOID:
		return Type{Kind: KindVoid}, true
	case lexer.IDENT:
		typ, ok := scope[tok.Literal]
		return typ, ok
	default:
		return Type{}, false
	}
}

func (env Environment) inferFieldAccess(scope Scope, tokens []lexer.Token) (Type, bool) {
	if len(tokens) != 3 || tokens[0].Type != lexer.IDENT || tokens[1].Type != lexer.DOT || tokens[2].Type != lexer.IDENT {
		return Type{}, false
	}
	base, ok := scope[tokens[0].Literal]
	if !ok {
		return Type{}, false
	}
	base = env.resolve(base)
	if base.Kind != KindRecord {
		return Type{}, false
	}
	for _, field := range base.Fields {
		if field.Name == tokens[2].Literal {
			return field.Type, true
		}
	}
	return Type{}, false
}

func (env Environment) inferInfix(scope Scope, tokens []lexer.Token, op lexer.TokenType) (Type, bool) {
	parts := splitTopLevel(tokens, op)
	if len(parts) < 2 {
		return Type{}, false
	}
	allNumber := true
	allString := op == lexer.PLUS
	for _, part := range parts {
		typ, ok := env.InferExpression(scope, part)
		if !ok {
			return Type{}, false
		}
		if !isNumberLike(env.resolve(typ)) {
			allNumber = false
		}
		if !isStringLike(env.resolve(typ)) {
			allString = false
		}
	}
	if allNumber {
		return Type{Kind: KindPrimitive, Name: "number"}, true
	}
	if allString {
		return Type{Kind: KindPrimitive, Name: "string"}, true
	}
	return Type{}, false
}

func isNumberLike(typ Type) bool {
	return typ.Kind == KindPrimitive && typ.Name == "number" ||
		typ.Kind == KindLiteral && typ.LiteralKind == LiteralNumber
}

func isStringLike(typ Type) bool {
	return typ.Kind == KindPrimitive && typ.Name == "string" ||
		typ.Kind == KindLiteral && typ.LiteralKind == LiteralString
}

func (env Environment) arrayMatches(element Type, value any) bool {
	ref := unwrap(reflect.ValueOf(value))
	if !ref.IsValid() {
		return false
	}
	switch ref.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < ref.Len(); i++ {
			if !env.matches(element, ref.Index(i).Interface()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (env Environment) recordMatches(typ Type, value any) bool {
	ref := unwrap(reflect.ValueOf(value))
	if !ref.IsValid() || ref.Kind() != reflect.Map || ref.Type().Key().Kind() != reflect.String {
		return false
	}
	for _, field := range typ.Fields {
		valueRef := ref.MapIndex(reflect.ValueOf(field.Name))
		if !valueRef.IsValid() {
			if field.Optional {
				continue
			}
			return false
		}
		if !env.matches(field.Type, valueRef.Interface()) {
			return false
		}
	}
	return IsSerializable(value)
}

func primitiveMatches(name string, value any) bool {
	if value == nil {
		return false
	}
	ref := unwrap(reflect.ValueOf(value))
	if !ref.IsValid() {
		return false
	}
	switch name {
	case "string":
		return ref.Kind() == reflect.String
	case "number":
		return isNumberValue(ref)
	case "boolean":
		return ref.Kind() == reflect.Bool
	default:
		return false
	}
}

func literalMatches(typ Type, value any) bool {
	switch typ.LiteralKind {
	case LiteralString:
		ref := unwrap(reflect.ValueOf(value))
		return ref.IsValid() && ref.Kind() == reflect.String && ref.String() == typ.Literal
	case LiteralNumber:
		return fmt.Sprint(value) == typ.Literal
	case LiteralBoolean:
		ref := unwrap(reflect.ValueOf(value))
		return ref.IsValid() && ref.Kind() == reflect.Bool && ref.Bool() == typ.Literal
	case LiteralNull:
		return value == nil
	default:
		return false
	}
}

func isNumberValue(value reflect.Value) bool {
	value = unwrap(value)
	if !value.IsValid() {
		return false
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isSerializableValue(value reflect.Value) bool {
	value = unwrap(value)
	if !value.IsValid() {
		return true
	}
	if value.CanInterface() {
		if opaque, ok := value.Interface().(OpaqueValue); ok {
			return IsSerializable(opaque.Value)
		}
	}
	switch value.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if !isSerializableValue(value.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return false
		}
		for _, key := range value.MapKeys() {
			if !isSerializableValue(value.MapIndex(key)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func unwrap(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

type typeParser struct {
	tokens      []lexer.Token
	pos         int
	env         Environment
	diagnostics []Diagnostic
}

func (p *typeParser) parseUnion() Type {
	options := []Type{p.parseArray()}
	for p.match(lexer.PIPE) {
		options = append(options, p.parseArray())
	}
	if len(options) == 1 {
		return options[0]
	}
	return Type{Kind: KindUnion, Options: options}
}

func (p *typeParser) parseArray() Type {
	typ := p.parsePrimary()
	for p.match(lexer.LBRACKET) {
		p.expect(lexer.RBRACKET)
		element := typ
		typ = Type{Kind: KindArray, Element: &element}
	}
	return typ
}

func (p *typeParser) parsePrimary() Type {
	if p.done() {
		p.errorf(lexer.Token{Type: lexer.EOF}, "expected type")
		return Type{Kind: KindInvalid}
	}
	tok := p.advance()
	switch tok.Type {
	case lexer.TYPE_STRING:
		return Type{Kind: KindPrimitive, Name: "string"}
	case lexer.TYPE_NUMBER:
		return Type{Kind: KindPrimitive, Name: "number"}
	case lexer.TYPE_BOOLEAN:
		return Type{Kind: KindPrimitive, Name: "boolean"}
	case lexer.TYPE_UNKNOWN:
		return Type{Kind: KindUnknown}
	case lexer.VOID:
		return Type{Kind: KindVoid}
	case lexer.NULL:
		return Type{Kind: KindNull}
	case lexer.STRING:
		return Type{Kind: KindLiteral, LiteralKind: LiteralString, Literal: tok.Literal}
	case lexer.NUMBER:
		return Type{Kind: KindLiteral, LiteralKind: LiteralNumber, Literal: tok.Literal}
	case lexer.TRUE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: true}
	case lexer.FALSE:
		return Type{Kind: KindLiteral, LiteralKind: LiteralBoolean, Literal: false}
	case lexer.IDENT, lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL,
		lexer.OPERATION, lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS,
		lexer.TARGET, lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return Type{Kind: KindNamed, Name: tok.Literal}
	default:
		p.errorf(tok, "expected type, got %s", tok.Type)
		return Type{Kind: KindInvalid}
	}
}

func (p *typeParser) match(typ lexer.TokenType) bool {
	if p.done() || p.peek().Type != typ {
		return false
	}
	p.pos++
	return true
}

func (p *typeParser) expect(typ lexer.TokenType) {
	if p.match(typ) {
		return
	}
	p.errorf(p.peek(), "expected %s", typ)
}

func (p *typeParser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *typeParser) peek() lexer.Token {
	if p.done() {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *typeParser) advance() lexer.Token {
	tok := p.peek()
	if !p.done() {
		p.pos++
	}
	return tok
}

func (p *typeParser) errorf(tok lexer.Token, format string, args ...any) {
	p.diagnostics = append(p.diagnostics, Diagnostic{
		Message: fmt.Sprintf(format, args...),
		Token:   tok,
	})
}

func tokenizeTypeText(text string) []lexer.Token {
	tokens := lexer.Tokenize(text)
	return trimEOF(tokens)
}

func trimEOF(tokens []lexer.Token) []lexer.Token {
	out := tokens
	for len(out) > 0 && out[len(out)-1].Type == lexer.EOF {
		out = out[:len(out)-1]
	}
	return out
}

func trimNoise(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && tokens[start].Type == lexer.COMMENT {
		start++
	}
	end := len(tokens)
	for end > start && (tokens[end-1].Type == lexer.COMMENT || tokens[end-1].Type == lexer.SEMICOLON) {
		end--
	}
	return tokens[start:end]
}

func trimWrappedParens(tokens []lexer.Token) []lexer.Token {
	for len(tokens) >= 2 && tokens[0].Type == lexer.LPAREN && tokens[len(tokens)-1].Type == lexer.RPAREN && wrapsAll(tokens) {
		tokens = tokens[1 : len(tokens)-1]
	}
	return tokens
}

func wrapsAll(tokens []lexer.Token) bool {
	depth := 0
	for i, tok := range tokens {
		switch tok.Type {
		case lexer.LPAREN:
			depth++
		case lexer.RPAREN:
			depth--
			if depth == 0 && i != len(tokens)-1 {
				return false
			}
		}
	}
	return depth == 0
}

func splitTopLevel(tokens []lexer.Token, delimiter lexer.TokenType) [][]lexer.Token {
	segments := make([][]lexer.Token, 0)
	start := 0
	depth := 0
	for i, tok := range tokens {
		if depth == 0 && tok.Type == delimiter {
			segments = append(segments, trimNoise(tokens[start:i]))
			start = i + 1
			continue
		}
		depth = nextDepth(depth, tok.Type)
	}
	if start == 0 {
		return nil
	}
	segments = append(segments, trimNoise(tokens[start:]))
	return segments
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
