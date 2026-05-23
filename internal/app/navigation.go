package app

import (
	"fmt"

	"github.com/dwlhm/nova/internal/scheduler"
)

type NavigationStack struct {
	Current Route
	Entries []Route
}

func NewNavigationStack(route Route) NavigationStack {
	return NavigationStack{Current: route, Entries: []Route{route}}
}

type NavigationActionKind string

const (
	NavigatePush    NavigationActionKind = "push"
	NavigateReplace NavigationActionKind = "replace"
	NavigateBack    NavigationActionKind = "back"
)

type NavigationAction struct {
	Kind  NavigationActionKind
	Route *Route
}

func ApplyNavigation(stack NavigationStack, action NavigationAction) (NavigationStack, scheduler.EventToEmit, error) {
	switch action.Kind {
	case NavigatePush:
		if action.Route == nil {
			return stack, scheduler.EventToEmit{}, fmt.Errorf("push navigation requires route")
		}
		return pushRoute(stack, *action.Route)
	case NavigateReplace:
		if action.Route == nil {
			return stack, scheduler.EventToEmit{}, fmt.Errorf("replace navigation requires route")
		}
		return replaceRoute(stack, *action.Route)
	case NavigateBack:
		return popRoute(stack)
	default:
		return stack, scheduler.EventToEmit{}, fmt.Errorf("unsupported navigation action %s", action.Kind)
	}
}

func pushRoute(stack NavigationStack, route Route) (NavigationStack, scheduler.EventToEmit, error) {
	if err := ValidateRoute(route); err != nil {
		return stack, scheduler.EventToEmit{}, err
	}
	next := NavigationStack{
		Current: route,
		Entries: append(cloneRoutes(stack.Entries), route),
	}
	return next, routeChanged(route), nil
}

func replaceRoute(stack NavigationStack, route Route) (NavigationStack, scheduler.EventToEmit, error) {
	if err := ValidateRoute(route); err != nil {
		return stack, scheduler.EventToEmit{}, err
	}
	entries := cloneRoutes(stack.Entries)
	if len(entries) == 0 {
		entries = []Route{route}
	} else {
		entries[len(entries)-1] = route
	}
	return NavigationStack{Current: route, Entries: entries}, routeChanged(route), nil
}

func popRoute(stack NavigationStack) (NavigationStack, scheduler.EventToEmit, error) {
	if len(stack.Entries) <= 1 {
		return stack, scheduler.EventToEmit{}, fmt.Errorf("back navigation rejected by route state")
	}
	entries := cloneRoutes(stack.Entries[:len(stack.Entries)-1])
	route := entries[len(entries)-1]
	return NavigationStack{Current: route, Entries: entries}, routeChanged(route), nil
}

func routeChanged(route Route) scheduler.EventToEmit {
	return scheduler.Emit("@nova/navigation", EventRouteChanged, route.Data())
}

func cloneRoutes(routes []Route) []Route {
	out := make([]Route, len(routes))
	for i, route := range routes {
		out[i] = route
		out[i].Params = cloneDataMap(route.Params)
		out[i].Query = cloneDataMap(route.Query)
	}
	return out
}
