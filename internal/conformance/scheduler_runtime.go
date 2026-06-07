package conformance

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/app"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/scheduler"
	"github.com/dwlhm/nova/internal/provider/build"
)

func buildSchedulerRuntime(plan build.BuildPlan, sources []build.SourceFile) (scheduler.Runtime, []Diagnostic) {
	sourceMap := make(map[string]parser.File, len(sources))
	parserFiles := make([]parser.File, 0, len(sources))
	for _, source := range sources {
		sourceMap[source.Path] = source.File
		parserFiles = append(parserFiles, source.File)
	}
	stateNames := collectExpressionStateNames(parserFiles)
	exprRegistry := buildExpressionRegistry(parserFiles)

	cells := make([]scheduler.StateCell, 0)
	var navigationContext *navigationFixtureContext
	for _, module := range plan.Modules {
		file, ok := sourceMap[module.Path]
		if !ok {
			continue
		}
		for _, contract := range file.ContractStates {
			for _, state := range contract.States {
				initial, err := evaluateExpressionWithRegistry(state.Initial, exprRegistry, stateNames, nil, scheduler.Snapshot{}, scheduler.EventEnvelope{})
				if err != nil {
					return scheduler.Runtime{}, []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-030", fmt.Sprintf("evaluate initial state %s.%s: %s", contract.Name, state.Name, err.Error()))}
				}
				transitions := make([]scheduler.TransitionRule, 0, len(state.Transitions))
				for _, transition := range state.Transitions {
					paramNames := eventParamNames(transition.Event)
					expr := transition.Expr
					eventName := scheduler.SchedulerEvent(transition.Event.Name)
					transitions = append(transitions, scheduler.On(eventName, func(snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
						return evaluateExpressionWithRegistry(expr, exprRegistry, stateNames, paramNames, snapshot, event)
					}))
				}
				if navigationContext == nil && state.Name == "route" && !hasParserTransition(state.Transitions, "@navigate") {
					route, err := app.RouteDataFromState(initial)
					if err != nil {
						route = app.Route{Path: "/"}
					}
					navigationContext = &navigationFixtureContext{stack: app.NewNavigationStack(route)}
				}
				transitions = appendFrameworkNavigationTransitions(
					scheduler.CapabilityRef(contract.Name),
					state,
					initial,
					transitions,
					navigationContext,
				)
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
	return scheduler.NewRuntime(cells, buildSchedulerLifecycles(plan, sourceMap, stateNames, exprRegistry)), nil
}
