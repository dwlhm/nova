package conformance

import (
	"github.com/dwlhm/nova/internal/core/expr"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/scheduler"
)

func buildExpressionRegistry(files []parser.File) *expr.Registry {
	return expr.BuildRegistry(files)
}

func collectExpressionStateNames(files []parser.File) map[string]bool {
	names := make(map[string]bool)
	for _, file := range files {
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

func evaluateExpressionWithRegistry(
	tokens []lexer.Token,
	reg *expr.Registry,
	stateNames map[string]bool,
	paramNames map[string]bool,
	snapshot scheduler.Snapshot,
	event scheduler.EventEnvelope,
) (scheduler.DataValue, error) {
	ctx := expr.ContextFromScheduler(snapshot, event, paramNames)
	return expr.Evaluate(tokens, reg, stateNames, paramNames, ctx)
}
