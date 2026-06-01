package conformance

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/scheduler"
)

func evaluateExpression(tokens []lexer.Token, stateNames map[string]bool, paramNames map[string]bool, snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
	tokens = trimExpressionTokens(tokens)
	if len(tokens) == 0 {
		return nil, nil
	}
	if record, ok := evaluateRecordExpression(tokens, stateNames, paramNames, snapshot, event); ok {
		return record, nil
	}
	return evaluateExpressionTokens(tokens, stateNames, paramNames, snapshot, event)
}

func evaluateExpressionTokens(tokens []lexer.Token, stateNames map[string]bool, paramNames map[string]bool, snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	if len(tokens) == 1 {
		return evaluateAtom(tokens[0], stateNames, paramNames, snapshot, event)
	}

	parts := splitBinaryTokens(tokens, lexer.PLUS)
	if parts != nil {
		left, err := evaluateExpressionTokens(parts[0], stateNames, paramNames, snapshot, event)
		if err != nil {
			return nil, err
		}
		right, err := evaluateExpressionTokens(parts[1], stateNames, paramNames, snapshot, event)
		if err != nil {
			return nil, err
		}
		if leftNumber, ok := asNumber(left); ok {
			if rightNumber, ok := asNumber(right); ok {
				return leftNumber + rightNumber, nil
			}
		}
		return fmt.Sprint(left) + fmt.Sprint(right), nil
	}

	parts = splitBinaryTokens(tokens, lexer.MINUS)
	if parts != nil {
		left, err := evaluateExpressionTokens(parts[0], stateNames, paramNames, snapshot, event)
		if err != nil {
			return nil, err
		}
		right, err := evaluateExpressionTokens(parts[1], stateNames, paramNames, snapshot, event)
		if err != nil {
			return nil, err
		}
		leftNumber, okLeft := asNumber(left)
		rightNumber, okRight := asNumber(right)
		if !okLeft || !okRight {
			return nil, fmt.Errorf("subtraction requires numeric operands")
		}
		return leftNumber - rightNumber, nil
	}

	literals := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		literals = append(literals, tok.Literal)
	}
	return nil, fmt.Errorf("unsupported expression %q", strings.Join(literals, " "))
}

func evaluateAtom(tok lexer.Token, stateNames map[string]bool, paramNames map[string]bool, snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
	switch tok.Type {
	case lexer.STRING:
		return tok.Literal, nil
	case lexer.NUMBER:
		value, err := strconv.ParseFloat(tok.Literal, 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	case lexer.TRUE:
		return true, nil
	case lexer.FALSE:
		return false, nil
	case lexer.NULL, lexer.VOID:
		return nil, nil
	case lexer.IDENT:
		switch {
		case stateNames[tok.Literal]:
			value, ok := snapshotValue(snapshot, tok.Literal)
			if !ok {
				return nil, fmt.Errorf("missing state %s", tok.Literal)
			}
			return value, nil
		case paramNames[tok.Literal]:
			value, ok := payloadValue(event, tok.Literal)
			if !ok {
				return nil, fmt.Errorf("missing event payload field %s", tok.Literal)
			}
			return value, nil
		default:
			return tok.Literal, nil
		}
	default:
		return tok.Literal, nil
	}
}

func evaluateRecordExpression(tokens []lexer.Token, stateNames map[string]bool, paramNames map[string]bool, snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (map[string]scheduler.DataValue, bool) {
	fields, ok := parseRecordExpressionFields(tokens)
	if !ok {
		return nil, false
	}
	out := make(map[string]scheduler.DataValue, len(fields))
	for _, field := range fields {
		value, err := evaluateExpression(field.Value, stateNames, paramNames, snapshot, event)
		if err != nil {
			return nil, false
		}
		out[field.Name.Literal] = value
	}
	return out, true
}

func snapshotValue(snapshot scheduler.Snapshot, name string) (scheduler.DataValue, bool) {
	for key, value := range snapshot.Values() {
		if string(key.Name) == name {
			return value, true
		}
	}
	return nil, false
}

func payloadValue(event scheduler.EventEnvelope, name string) (scheduler.DataValue, bool) {
	switch payload := event.Payload.(type) {
	case map[string]scheduler.DataValue:
		value, ok := payload[name]
		return value, ok
	case map[string]any:
		value, ok := payload[name]
		return value, ok
	default:
		return nil, false
	}
}

func splitBinaryTokens(tokens []lexer.Token, operator lexer.TokenType) [][]lexer.Token {
	for index := len(tokens) - 1; index >= 0; index-- {
		if tokens[index].Type != operator {
			continue
		}
		left := trimExpressionTokens(tokens[:index])
		right := trimExpressionTokens(tokens[index+1:])
		if len(left) == 0 || len(right) == 0 {
			continue
		}
		return [][]lexer.Token{left, right}
	}
	return nil
}

func asNumber(value scheduler.DataValue) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	default:
		return 0, false
	}
}

func trimExpressionTokens(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && (tokens[start].Type == lexer.EOF || tokens[start].Type == lexer.COMMENT || tokens[start].Type == lexer.SEMICOLON) {
		start++
	}
	end := len(tokens)
	for end > start && (tokens[end-1].Type == lexer.EOF || tokens[end-1].Type == lexer.COMMENT || tokens[end-1].Type == lexer.SEMICOLON) {
		end--
	}
	return tokens[start:end]
}

type recordExpressionField struct {
	Name  lexer.Token
	Value []lexer.Token
}

func parseRecordExpressionFields(tokens []lexer.Token) ([]recordExpressionField, bool) {
	if len(tokens) < 2 || tokens[0].Type != lexer.LBRACE || tokens[len(tokens)-1].Type != lexer.RBRACE {
		return nil, false
	}

	body := tokens[1 : len(tokens)-1]
	fields := make([]recordExpressionField, 0)
	pos := 0
	for pos < len(body) {
		for pos < len(body) && isRecordExpressionSeparator(body[pos].Type) {
			pos++
		}
		if pos >= len(body) {
			break
		}

		name := body[pos]
		if !isRecordExpressionName(name.Type) {
			return nil, false
		}
		pos++
		if pos >= len(body) || body[pos].Type != lexer.ASSIGN_IN {
			return nil, false
		}
		pos++

		start := pos
		depth := 0
		for pos < len(body) {
			tok := body[pos]
			if depth == 0 && isRecordExpressionSeparator(tok.Type) {
				break
			}
			depth = expressionDepth(depth, tok.Type)
			pos++
		}
		value := trimExpressionTokens(body[start:pos])
		if len(value) == 0 {
			return nil, false
		}
		fields = append(fields, recordExpressionField{Name: name, Value: value})
	}
	return fields, len(fields) > 0
}

func isRecordExpressionSeparator(typ lexer.TokenType) bool {
	return typ == lexer.SEMICOLON || typ == lexer.COMMA || typ == lexer.COMMENT
}

func isRecordExpressionName(typ lexer.TokenType) bool {
	switch typ {
	case lexer.IDENT, lexer.TYPE, lexer.STATE, lexer.EVENT, lexer.CAPABILITY, lexer.EXTERNAL,
		lexer.OPERATION, lexer.INPUT, lexer.OUTPUT, lexer.PROPS, lexer.EMITS, lexer.RETURNS,
		lexer.TARGET, lexer.MOUNT, lexer.DISPOSE, lexer.BEFORE, lexer.AFTER, lexer.ERROR:
		return true
	default:
		return false
	}
}

func expressionDepth(depth int, typ lexer.TokenType) int {
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

func collectExpressionStateNames(sources []parser.File) map[string]bool {
	names := make(map[string]bool)
	for _, file := range sources {
		for _, contract := range file.ContractStates {
			for _, state := range contract.States {
				names[state.Name] = true
			}
		}
	}
	return names
}

func eventParamNames(pattern parser.EventPattern) map[string]bool {
	out := make(map[string]bool, len(pattern.Params))
	for _, param := range pattern.Params {
		out[param.Name] = true
	}
	return out
}
