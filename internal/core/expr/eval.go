package expr

import (
	"fmt"
	"math"

	"github.com/dwlhm/nova/internal/core/lexer"
)

// Context carries pure inputs for expression evaluation.
type Context struct {
	State  map[string]any
	Params map[string]any
}

// Evaluate parses and evaluates tokens with a pure func registry.
func Evaluate(tokens []lexer.Token, reg *Registry, stateNames map[string]bool, paramNames map[string]bool, ctx Context) (any, error) {
	node, err := Parse(tokens, reg, stateNames, paramNames)
	if err != nil {
		return nil, err
	}
	return EvalNode(node, reg, ctx)
}

// EvalNode evaluates an AST node without side effects.
func EvalNode(node Node, reg *Registry, ctx Context) (any, error) {
	switch typed := node.(type) {
	case Literal:
		return typed.Value, nil
	case Ident:
		switch typed.Kind {
		case IdentState:
			value, ok := ctx.State[typed.Name]
			if !ok {
				return nil, fmt.Errorf("missing state %s", typed.Name)
			}
			return value, nil
		case IdentParam:
			value, ok := ctx.Params[typed.Name]
			if !ok {
				return nil, fmt.Errorf("missing payload field %s", typed.Name)
			}
			return value, nil
		default:
			return typed.Name, nil
		}
	case Unary:
		value, err := EvalNode(typed.Expr, reg, ctx)
		if err != nil {
			return nil, err
		}
		switch typed.Op {
		case "!":
			return !truthy(value), nil
		case "-":
			number, err := asNumber(value)
			if err != nil {
				return nil, err
			}
			return -number, nil
		default:
			return nil, fmt.Errorf("unsupported unary operator %q", typed.Op)
		}
	case Binary:
		left, err := EvalNode(typed.Left, reg, ctx)
		if err != nil {
			return nil, err
		}
		right, err := EvalNode(typed.Right, reg, ctx)
		if err != nil {
			return nil, err
		}
		return evalBinary(typed.Op, left, right)
	case Ternary:
		cond, err := EvalNode(typed.Cond, reg, ctx)
		if err != nil {
			return nil, err
		}
		if truthy(cond) {
			return EvalNode(typed.Then, reg, ctx)
		}
		return EvalNode(typed.Else, reg, ctx)
	case Call:
		return evalCall(typed, reg, ctx)
	case FieldAccess:
		base, err := EvalNode(typed.Base, reg, ctx)
		if err != nil {
			return nil, err
		}
		record, ok := base.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field access on non-record value")
		}
		value, ok := record[typed.Field]
		if !ok {
			return nil, fmt.Errorf("missing field %s", typed.Field)
		}
		return value, nil
	case Record:
		out := make(map[string]any, len(typed.Fields))
		for _, field := range typed.Fields {
			value, err := EvalNode(field.Expr, reg, ctx)
			if err != nil {
				return nil, err
			}
			out[field.Name] = value
		}
		return out, nil
	case List:
		out := make([]any, 0, len(typed.Elements))
		for _, element := range typed.Elements {
			value, err := EvalNode(element, reg, ctx)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		return out, nil
	case Pipe:
		return evalCall(typed.Call, reg, ctx)
	default:
		return nil, fmt.Errorf("unsupported expression node %T", node)
	}
}

func evalCall(call Call, reg *Registry, ctx Context) (any, error) {
	fn, ok := reg.Lookup(call.Name)
	if !ok {
		return nil, fmt.Errorf("unknown function %s", call.Name)
	}
	if len(call.Args) != len(fn.Params) {
		return nil, fmt.Errorf("function %s expects %d arguments, got %d", call.Name, len(fn.Params), len(call.Args))
	}
	local := make(map[string]any, len(fn.Params))
	for index, param := range fn.Params {
		value, err := EvalNode(call.Args[index], reg, ctx)
		if err != nil {
			return nil, err
		}
		local[param] = value
	}
	callCtx := Context{State: map[string]any{}, Params: local}
	return Evaluate(fn.Body, reg, map[string]bool{}, localParamNames(local), callCtx)
}

func localParamNames(params map[string]any) map[string]bool {
	names := make(map[string]bool, len(params))
	for name := range params {
		names[name] = true
	}
	return names
}

func evalBinary(op string, left, right any) (any, error) {
	switch op {
	case "+":
		if leftNumber, err := asNumber(left); err == nil {
			if rightNumber, err := asNumber(right); err == nil {
				return leftNumber + rightNumber, nil
			}
		}
		return fmt.Sprint(left) + fmt.Sprint(right), nil
	case "-":
		leftNumber, err := asNumber(left)
		if err != nil {
			return nil, err
		}
		rightNumber, err := asNumber(right)
		if err != nil {
			return nil, err
		}
		return leftNumber - rightNumber, nil
	case "*":
		leftNumber, err := asNumber(left)
		if err != nil {
			return nil, err
		}
		rightNumber, err := asNumber(right)
		if err != nil {
			return nil, err
		}
		return leftNumber * rightNumber, nil
	case "/":
		leftNumber, err := asNumber(left)
		if err != nil {
			return nil, err
		}
		rightNumber, err := asNumber(right)
		if err != nil {
			return nil, err
		}
		return leftNumber / rightNumber, nil
	case "==":
		return valuesEqual(left, right), nil
	case "!=":
		return !valuesEqual(left, right), nil
	case ">":
		return compareNumbers(left, right, func(a, b float64) bool { return a > b })
	case ">=":
		return compareNumbers(left, right, func(a, b float64) bool { return a >= b })
	case "<":
		return compareNumbers(left, right, func(a, b float64) bool { return a < b })
	case "<=":
		return compareNumbers(left, right, func(a, b float64) bool { return a <= b })
	case "&&":
		return truthy(left) && truthy(right), nil
	case "||":
		return truthy(left) || truthy(right), nil
	default:
		return nil, fmt.Errorf("unsupported binary operator %q", op)
	}
}

func compareNumbers(left, right any, cmp func(float64, float64) bool) (bool, error) {
	leftNumber, err := asNumber(left)
	if err != nil {
		return false, err
	}
	rightNumber, err := asNumber(right)
	if err != nil {
		return false, err
	}
	return cmp(leftNumber, rightNumber), nil
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case float64:
		return typed != 0
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case string:
		return typed != ""
	default:
		return true
	}
}

func valuesEqual(left, right any) bool {
	if left == nil || right == nil {
		return left == right
	}
	leftNumber, leftErr := asNumber(left)
	rightNumber, rightErr := asNumber(right)
	if leftErr == nil && rightErr == nil {
		return leftNumber == rightNumber
	}
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func asNumber(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case bool:
		if typed {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", value)
	}
}

func SnapshotValues(snapshot map[string]any) map[string]any {
	if snapshot == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(snapshot))
	for key, value := range snapshot {
		out[key] = value
	}
	return out
}

func NormalizeNumber(value float64) any {
	if math.Mod(value, 1) == 0 {
		return value
	}
	return value
}
