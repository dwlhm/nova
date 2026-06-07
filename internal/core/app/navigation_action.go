package app

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/scheduler"
)

func ParseNavigationAction(value scheduler.DataValue) (NavigationAction, error) {
	data, ok := value.(map[string]any)
	if !ok {
		return NavigationAction{}, fmt.Errorf("navigation action must be a record")
	}
	kind, ok := data["kind"].(string)
	if !ok || kind == "" {
		return NavigationAction{}, fmt.Errorf("navigation action requires kind")
	}
	action := NavigationAction{Kind: NavigationActionKind(kind)}
	if routeValue, ok := data["route"]; ok && routeValue != nil {
		route, err := ParseRoute(routeValue)
		if err != nil {
			return NavigationAction{}, err
		}
		action.Route = &route
	}
	switch action.Kind {
	case NavigatePush, NavigateReplace:
		if action.Route == nil {
			return NavigationAction{}, fmt.Errorf("%s navigation requires route", action.Kind)
		}
	case NavigateBack:
	default:
		return NavigationAction{}, fmt.Errorf("unsupported navigation action %s", action.Kind)
	}
	return action, nil
}

func ParseRoute(value scheduler.DataValue) (Route, error) {
	switch typed := value.(type) {
	case string:
		path := typed
		if path == "" {
			path = "/"
		}
		if path[0] != '/' {
			return Route{}, fmt.Errorf("route path must start with /")
		}
		return Route{Path: path}, nil
	case map[string]any:
		return parseRouteMap(typed)
	case map[string]scheduler.DataValue:
		converted := make(map[string]any, len(typed))
		for key, item := range typed {
			converted[key] = item
		}
		return parseRouteMap(converted)
	default:
		return Route{}, fmt.Errorf("route payload must be a record or path string")
	}
}

func parseRouteMap(data map[string]any) (Route, error) {
	route := Route{Params: map[string]scheduler.DataValue{}, Query: map[string]scheduler.DataValue{}}
	rawPath, ok := data["path"].(string)
	if !ok || rawPath == "" {
		return Route{}, fmt.Errorf("route path must start with /")
	}
	route.Path = rawPath
	if params, ok := data["params"].(map[string]any); ok {
		for key, value := range params {
			route.Params[key] = value
		}
	}
	if query, ok := data["query"].(map[string]any); ok {
		for key, value := range query {
			route.Query[key] = value
		}
	}
	if fragment, ok := data["fragment"].(string); ok {
		route.Fragment = fragment
	}
	return route, ValidateRoute(route)
}

func ActionFromEventPayload(payload scheduler.DataValue) (NavigationAction, error) {
	if payload == nil {
		return NavigationAction{}, fmt.Errorf("navigation action payload is required")
	}
	if envelope, ok := payload.(map[string]any); ok {
		if actionValue, ok := envelope["action"]; ok {
			return ParseNavigationAction(actionValue)
		}
	}
	return ParseNavigationAction(payload)
}

func RouteDataFromState(value scheduler.DataValue) (Route, error) {
	return ParseRoute(value)
}
