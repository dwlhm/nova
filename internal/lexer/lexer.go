package lexer

import "strings"

var keywords = map[string]TokenType{
	"from":       FROM,
	"as":         AS,
	"use":        USE,
	"is":         IS,
	"not":        NOT,
	"true":       TRUE,
	"false":      FALSE,
	"void":       VOID,
	"null":       NULL,
	"type":       TYPE,
	"state":      STATE,
	"event":      EVENT,
	"capability": CAPABILITY,
	"external":   EXTERNAL,
	"operation":  OPERATION,
	"input":      INPUT,
	"output":     OUTPUT,
	"props":      PROPS,
	"emits":      EMITS,
	"returns":    RETURNS,
	"target":     TARGET,
	"mount":      MOUNT,
	"dispose":    DISPOSE,
	"before":     BEFORE,
	"after":      AFTER,
	"error":      ERROR,
	"string":     TYPE_STRING,
	"number":     TYPE_NUMBER,
	"boolean":    TYPE_BOOLEAN,
	"unknown":    TYPE_UNKNOWN,
	"Type":       BLOCK_TYPE,
	"Props":      BLOCK_PROPS,
	"State":      BLOCK_STATE,
	"Driver":     BLOCK_DRIVER,
}

var tagKeywords = map[string]TokenType{
	"<import":    TAG_IMPORT,
	"<contract":  TAG_CONTRACT,
	"<lifecycle": TAG_LIFECYCLE,
	"<template":  TAG_TEMPLATE,
	"<func":      TAG_FUNC,
}

type scanner struct {
	input string
	pos   int
	prev  TokenType
}

// Tokenize converts Nova source into a flat token stream.
//
// It is intentionally side-effect free: all scanner state is internal to this
// call, and callers receive a new token slice for the provided input.
func Tokenize(input string) []Token {
	s := scanner{input: input, prev: EOF}
	tokens := make([]Token, 0)

	for {
		next, advanced := nextToken(s)
		tokens = append(tokens, next)
		if next.Type == EOF {
			return tokens
		}
		s = advanced
	}
}

func nextToken(s scanner) (Token, scanner) {
	s = skipWhitespace(s)
	if s.pos >= len(s.input) {
		return Token{Type: EOF, Literal: ""}, s
	}

	if tag, ok := readTagKeyword(s); ok {
		return advance(s, tag, string(tag))
	}

	if tok, ok := readCompoundOperator(s); ok {
		return advance(s, tok.Type, tok.Literal)
	}

	ch := s.input[s.pos]
	if ch == '/' && peek(s, 1) == '/' {
		return readComment(s)
	}
	if ch == '"' {
		return readString(s)
	}
	if ch == '@' {
		return readSignal(s)
	}
	if isIdentifierStart(ch) {
		return readIdentifier(s)
	}
	if isDigit(ch) || (ch == '-' && isDigit(peek(s, 1)) && canStartNegativeNumber(s.prev)) {
		return readNumber(s)
	}

	tok := singleCharToken(ch)
	if tok == ILLEGAL {
		return advance(s, ILLEGAL, string(ch))
	}

	return advance(s, tok, string(ch))
}

func skipWhitespace(s scanner) scanner {
	for s.pos < len(s.input) {
		switch s.input[s.pos] {
		case ' ', '\t', '\n', '\r':
			s.pos++
		default:
			return s
		}
	}
	return s
}

func readTagKeyword(s scanner) (TokenType, bool) {
	for lit, typ := range tagKeywords {
		if strings.HasPrefix(s.input[s.pos:], lit) && hasBoundaryAfter(s.input, s.pos+len(lit)) {
			return typ, true
		}
	}
	return "", false
}

func readCompoundOperator(s scanner) (Token, bool) {
	operators := []struct {
		lit string
		typ TokenType
	}{
		{"...", SPREAD},
		{"::", SCOPE},
		{"<-", ASSIGN_IN},
		{"->", MAP_ARROW},
		{"/|", PIPE_END},
		{"|>", PIPE_FWD},
		{"?|", GATE_OPEN},
		{":|", GATE_SEP},
		{"==", EQ},
		{"!=", NOT_EQ},
		{"<=", LTE},
		{">=", GTE},
		{"&&", AND},
		{"||", OR},
	}

	for _, op := range operators {
		if strings.HasPrefix(s.input[s.pos:], op.lit) {
			return Token{Type: op.typ, Literal: op.lit}, true
		}
	}
	return Token{}, false
}

func readComment(s scanner) (Token, scanner) {
	start := s.pos + 2
	pos := start
	for pos < len(s.input) && s.input[pos] != '\n' && s.input[pos] != '\r' {
		pos++
	}

	return Token{Type: COMMENT, Literal: strings.TrimSpace(s.input[start:pos])}, scanner{
		input: s.input,
		pos:   pos,
		prev:  COMMENT,
	}
}

func readString(s scanner) (Token, scanner) {
	start := s.pos + 1
	pos := start
	for pos < len(s.input) {
		if s.input[pos] == '\\' && pos+1 < len(s.input) {
			pos += 2
			continue
		}
		if s.input[pos] == '"' {
			return Token{Type: STRING, Literal: s.input[start:pos]}, scanner{
				input: s.input,
				pos:   pos + 1,
				prev:  STRING,
			}
		}
		pos++
	}

	return Token{Type: ILLEGAL, Literal: s.input[start-1:]}, scanner{
		input: s.input,
		pos:   len(s.input),
		prev:  ILLEGAL,
	}
}

func readSignal(s scanner) (Token, scanner) {
	start := s.pos
	pos := s.pos + 1
	if pos >= len(s.input) || !isIdentifierStart(s.input[pos]) {
		return advance(s, ILLEGAL, "@")
	}

	for pos < len(s.input) && isIdentifierPart(s.input[pos]) {
		pos++
	}

	return Token{Type: SIGNAL, Literal: s.input[start:pos]}, scanner{
		input: s.input,
		pos:   pos,
		prev:  SIGNAL,
	}
}

func readIdentifier(s scanner) (Token, scanner) {
	start := s.pos
	pos := s.pos + 1
	for pos < len(s.input) && isIdentifierPart(s.input[pos]) {
		pos++
	}

	literal := s.input[start:pos]
	typ, ok := keywords[literal]
	if !ok {
		typ = IDENT
	}

	return Token{Type: typ, Literal: literal}, scanner{
		input: s.input,
		pos:   pos,
		prev:  typ,
	}
}

func readNumber(s scanner) (Token, scanner) {
	start := s.pos
	pos := s.pos
	if s.input[pos] == '-' {
		pos++
	}

	for pos < len(s.input) && isDigit(s.input[pos]) {
		pos++
	}
	if pos < len(s.input) && s.input[pos] == '.' {
		pos++
		for pos < len(s.input) && isDigit(s.input[pos]) {
			pos++
		}
	}

	return Token{Type: NUMBER, Literal: s.input[start:pos]}, scanner{
		input: s.input,
		pos:   pos,
		prev:  NUMBER,
	}
}

func advance(s scanner, typ TokenType, literal string) (Token, scanner) {
	return Token{Type: typ, Literal: literal}, scanner{
		input: s.input,
		pos:   s.pos + len(literal),
		prev:  typ,
	}
}

func singleCharToken(ch byte) TokenType {
	switch ch {
	case '<':
		return LT
	case '>':
		return GT
	case '+':
		return PLUS
	case '-':
		return MINUS
	case '*':
		return ASTERISK
	case '/':
		return SLASH
	case '!':
		return BANG
	case ',':
		return COMMA
	case '|':
		return PIPE
	case '(':
		return LPAREN
	case ')':
		return RPAREN
	case '[':
		return LBRACKET
	case ']':
		return RBRACKET
	case '{':
		return LBRACE
	case '}':
		return RBRACE
	case ':':
		return COLON
	case ';':
		return SEMICOLON
	case '.':
		return DOT
	case '?':
		return QUESTION
	default:
		return ILLEGAL
	}
}

func canStartNegativeNumber(prev TokenType) bool {
	switch prev {
	case EOF, ASSIGN_IN, MAP_ARROW, PIPE_FWD, GATE_OPEN, GATE_SEP, EQ, NOT_EQ, LTE, GTE, AND, OR, LT, GT, PLUS, MINUS, ASTERISK, SLASH, BANG, COMMA, PIPE, LPAREN, LBRACKET, LBRACE, SCOPE, COLON:
		return true
	default:
		return false
	}
}

func hasBoundaryAfter(input string, pos int) bool {
	return pos >= len(input) || !isIdentifierPart(input[pos])
}

func peek(s scanner, offset int) byte {
	pos := s.pos + offset
	if pos >= len(s.input) {
		return 0
	}
	return s.input[pos]
}

func isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentifierPart(ch byte) bool {
	return isIdentifierStart(ch) || isDigit(ch)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
