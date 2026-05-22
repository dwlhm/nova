package conformance

import (
	"fmt"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/scheduler"
)

func buildSchedulerRuntime(plan build.BuildPlan, sources []build.SourceFile) (scheduler.Runtime, []Diagnostic) {
	sourceMap := make(map[string]parser.File, len(sources))
	parserFiles := make([]parser.File, 0, len(sources))
	for _, source := range sources {
		sourceMap[source.Path] = source.File
		parserFiles = append(parserFiles, source.File)
	}
	stateNames := collectExpressionStateNames(parserFiles)

	cells := make([]scheduler.StateCell, 0)
	for _, module := range plan.Modules {
		file, ok := sourceMap[module.Path]
		if !ok {
			continue
		}
		for _, contract := range file.ContractStates {
			for _, state := range contract.States {
				initial, err := evaluateExpression(state.Initial, stateNames, nil, scheduler.Snapshot{}, scheduler.EventEnvelope{})
				if err != nil {
					return scheduler.Runtime{}, []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-030", fmt.Sprintf("evaluate initial state %s.%s: %s", contract.Name, state.Name, err.Error()))}
				}
				transitions := make([]scheduler.TransitionRule, 0, len(state.Transitions))
				for _, transition := range state.Transitions {
					paramNames := eventParamNames(transition.Event)
					expr := transition.Expr
					eventName := scheduler.SchedulerEvent(transition.Event.Name)
					transitions = append(transitions, scheduler.On(eventName, func(snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
						return evaluateExpression(expr, stateNames, paramNames, snapshot, event)
					}))
				}
				cells = append(cells, scheduler.NewStateCell(
					scheduler.CapabilityRef(contract.Name),
					scheduler.StateName(state.Name),
					state.Type.Text,
					initial,
					transitions...,
				))
			}
		}
	}
	return scheduler.NewRuntime(cells, nil), nil
}
