package expr

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
)

// LowerJS parses tokens and lowers them to a JavaScript expression string.
func LowerJS(tokens []lexer.Token, reg *Registry, stateNames map[string]bool, paramNames map[string]bool) (string, error) {
	node, err := Parse(tokens, reg, stateNames, paramNames)
	if err != nil {
		return "", err
	}
	return lowerNodeJS(node, reg, false)
}

// LowerJSMethod lowers a pure func body for NovaExpr helper emission.
func LowerJSMethod(tokens []lexer.Token, reg *Registry, paramNames map[string]bool) (string, error) {
	node, err := Parse(tokens, reg, nil, paramNames)
	if err != nil {
		return "", err
	}
	return lowerNodeJS(node, reg, true)
}

// LowerJava parses tokens and lowers them to a Java expression string using NovaExpr helpers.
func LowerJava(tokens []lexer.Token, reg *Registry, stateNames map[string]bool, paramNames map[string]bool) (string, error) {
	node, err := Parse(tokens, reg, stateNames, paramNames)
	if err != nil {
		return "", err
	}
	return lowerNodeJava(node, reg)
}

func lowerNodeJS(node Node, reg *Registry, methodBody bool) (string, error) {
	switch typed := node.(type) {
	case Literal:
		return lowerLiteralJS(typed.Value), nil
	case Ident:
		switch typed.Kind {
		case IdentState:
			return "state." + typed.Name, nil
		case IdentParam:
			if methodBody {
				return typed.Name, nil
			}
			return "payload." + typed.Name, nil
		default:
			return quoteJS(typed.Name), nil
		}
	case Unary:
		expr, err := lowerNodeJS(typed.Expr, reg, methodBody)
		if err != nil {
			return "", err
		}
		return "(" + typed.Op + expr + ")", nil
	case Binary:
		left, err := lowerNodeJS(typed.Left, reg, methodBody)
		if err != nil {
			return "", err
		}
		right, err := lowerNodeJS(typed.Right, reg, methodBody)
		if err != nil {
			return "", err
		}
		return "(" + left + " " + typed.Op + " " + right + ")", nil
	case Ternary:
		cond, err := lowerNodeJS(typed.Cond, reg, methodBody)
		if err != nil {
			return "", err
		}
		thenExpr, err := lowerNodeJS(typed.Then, reg, methodBody)
		if err != nil {
			return "", err
		}
		elseExpr, err := lowerNodeJS(typed.Else, reg, methodBody)
		if err != nil {
			return "", err
		}
		return "(" + cond + " ? " + thenExpr + " : " + elseExpr + ")", nil
	case Call:
		args := make([]string, 0, len(typed.Args))
		for _, arg := range typed.Args {
			lowered, err := lowerNodeJS(arg, reg, methodBody)
			if err != nil {
				return "", err
			}
			args = append(args, lowered)
		}
		return "NovaExpr." + typed.Name + "(" + strings.Join(args, ", ") + ")", nil
	case FieldAccess:
		base, err := lowerNodeJS(typed.Base, reg, methodBody)
		if err != nil {
			return "", err
		}
		return "(" + base + ")." + typed.Field, nil
	case Record:
		parts := make([]string, 0, len(typed.Fields))
		for _, field := range typed.Fields {
			value, err := lowerNodeJS(field.Expr, reg, methodBody)
			if err != nil {
				return "", err
			}
			parts = append(parts, field.Name+": "+value)
		}
		return "({ " + strings.Join(parts, ", ") + " })", nil
	case List:
		parts := make([]string, 0, len(typed.Elements))
		for _, element := range typed.Elements {
			value, err := lowerNodeJS(element, reg, methodBody)
			if err != nil {
				return "", err
			}
			parts = append(parts, value)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case Pipe:
		return lowerNodeJS(typed.Call, reg, methodBody)
	default:
		return "", fmt.Errorf("unsupported expression node %T", node)
	}
}

func lowerNodeJava(node Node, reg *Registry) (string, error) {
	switch typed := node.(type) {
	case Literal:
		return lowerLiteralJava(typed.Value), nil
	case Ident:
		switch typed.Kind {
		case IdentState:
			return "state." + typed.Name, nil
		case IdentParam:
			return typed.Name, nil
		default:
			return quoteCodeString(typed.Name), nil
		}
	case Unary:
		expr, err := lowerNodeJava(typed.Expr, reg)
		if err != nil {
			return "", err
		}
		switch typed.Op {
		case "!":
			return "(!NovaRuntime.booleanValue(" + expr + "))", nil
		case "-":
			return "(-NovaRuntime.numberValue(" + expr + "))", nil
		default:
			return expr, nil
		}
	case Binary:
		left, err := lowerNodeJava(typed.Left, reg)
		if err != nil {
			return "", err
		}
		right, err := lowerNodeJava(typed.Right, reg)
		if err != nil {
			return "", err
		}
		switch typed.Op {
		case "+":
			return "NovaRuntime.add(" + left + ", " + right + ")", nil
		case "==":
			return "NovaRuntime.equalsValue(" + left + ", " + right + ")", nil
		case "!=":
			return "(!NovaRuntime.equalsValue(" + left + ", " + right + "))", nil
		case "&&":
			return "(NovaRuntime.booleanValue(" + left + ") && NovaRuntime.booleanValue(" + right + "))", nil
		case "||":
			return "(NovaRuntime.booleanValue(" + left + ") || NovaRuntime.booleanValue(" + right + "))", nil
		case "-", "*", "/":
			return "(NovaRuntime.numberValue(" + left + ") " + typed.Op + " NovaRuntime.numberValue(" + right + "))", nil
		case ">", ">=", "<", "<=":
			return "NovaRuntime.compareValues(" + left + ", " + right + ", " + quoteCodeString(typed.Op) + ")", nil
		default:
			return "", fmt.Errorf("unsupported binary operator %q", typed.Op)
		}
	case Ternary:
		cond, err := lowerNodeJava(typed.Cond, reg)
		if err != nil {
			return "", err
		}
		thenExpr, err := lowerNodeJava(typed.Then, reg)
		if err != nil {
			return "", err
		}
		elseExpr, err := lowerNodeJava(typed.Else, reg)
		if err != nil {
			return "", err
		}
		return "(NovaRuntime.booleanValue(" + cond + ") ? " + thenExpr + " : " + elseExpr + ")", nil
	case Call:
		args := make([]string, 0, len(typed.Args))
		for _, arg := range typed.Args {
			lowered, err := lowerNodeJava(arg, reg)
			if err != nil {
				return "", err
			}
			args = append(args, lowered)
		}
		return "NovaExpr." + typed.Name + "(" + strings.Join(args, ", ") + ")", nil
	case FieldAccess:
		base, err := lowerNodeJava(typed.Base, reg)
		if err != nil {
			return "", err
		}
		return "NovaExpr.field(" + base + ", " + quoteCodeString(typed.Field) + ")", nil
	case Record:
		parts := make([]string, 0, len(typed.Fields))
		for _, field := range typed.Fields {
			value, err := lowerNodeJava(field.Expr, reg)
			if err != nil {
				return "", err
			}
			parts = append(parts, "NovaRuntime.entry("+quoteCodeString(field.Name)+", "+value+")")
		}
		if len(parts) == 0 {
			return "NovaRuntime.record()", nil
		}
		return "NovaRuntime.record(" + strings.Join(parts, ", ") + ")", nil
	case List:
		parts := make([]string, 0, len(typed.Elements))
		for _, element := range typed.Elements {
			value, err := lowerNodeJava(element, reg)
			if err != nil {
				return "", err
			}
			parts = append(parts, value)
		}
		if len(parts) == 0 {
			return "NovaRuntime.list()", nil
		}
		return "NovaRuntime.list(" + strings.Join(parts, ", ") + ")", nil
	case Pipe:
		return lowerNodeJava(typed.Call, reg)
	default:
		return "", fmt.Errorf("unsupported expression node %T", node)
	}
}

func lowerLiteralJS(value any) string {
	switch typed := value.(type) {
	case string:
		return quoteJS(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return quoteJS(fmt.Sprint(typed))
	}
}

func lowerLiteralJava(value any) string {
	switch typed := value.(type) {
	case string:
		return quoteCodeString(typed)
	case bool:
		if typed {
			return "Boolean.TRUE"
		}
		return "Boolean.FALSE"
	case nil:
		return "null"
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10) + ".0"
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return quoteCodeString(fmt.Sprint(typed))
	}
}

func quoteJS(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func quoteCodeString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
