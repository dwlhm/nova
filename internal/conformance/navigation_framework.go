package conformance

import (
	"github.com/dwlhm/nova/internal/core/app"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/scheduler"
)

type navigationFixtureContext struct {
	stack app.NavigationStack
}

func appendFrameworkNavigationTransitions(
	owner scheduler.CapabilityRef,
	state parser.StateDecl,
	initial scheduler.DataValue,
	transitions []scheduler.TransitionRule,
	nav *navigationFixtureContext,
) []scheduler.TransitionRule {
	if state.Name != "route" || hasParserTransition(state.Transitions, "@navigate") {
		return transitions
	}
	context := nav
	if context == nil {
		route, err := app.RouteDataFromState(initial)
		if err != nil {
			route = app.Route{Path: "/"}
		}
		context = &navigationFixtureContext{stack: app.NewNavigationStack(route)}
	}
	transitions = append(transitions, scheduler.On("@navigate", func(snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
		current, err := app.RouteDataFromState(snapshot.MustValue(scheduler.Key(owner, "route")))
		if err == nil {
			context.stack.Current = current
		}
		action, err := app.ActionFromEventPayload(event.Payload)
		if err != nil {
			return nil, err
		}
		nextStack, _, err := app.ApplyNavigation(context.stack, action)
		if err != nil {
			return nil, err
		}
		context.stack = nextStack
		return routeStateValue(nextStack.Current), nil
	}))
	return transitions
}

func hasParserTransition(transitions []parser.TransitionRule, event string) bool {
	for _, transition := range transitions {
		if transition.Event.Name == event {
			return true
		}
	}
	return false
}

func routeStateValue(route app.Route) scheduler.DataValue {
	data := route.Data()
	if len(data) == 1 {
		if path, ok := data["path"].(string); ok {
			return path
		}
	}
	return data
}
