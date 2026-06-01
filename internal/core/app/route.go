package app

import (
	"fmt"
	"strings"

	"github.com/dwlhm/nova/internal/core/scheduler"
	novatypes "github.com/dwlhm/nova/internal/core/types"
)

type Route struct {
	Path     string
	Params   map[string]scheduler.DataValue
	Query    map[string]scheduler.DataValue
	Fragment string
}

func (r Route) Data() map[string]scheduler.DataValue {
	data := map[string]scheduler.DataValue{"path": r.Path}
	if len(r.Params) > 0 {
		data["params"] = cloneDataMap(r.Params)
	}
	if len(r.Query) > 0 {
		data["query"] = cloneDataMap(r.Query)
	}
	if r.Fragment != "" {
		data["fragment"] = r.Fragment
	}
	return data
}

func ValidateRoute(route Route) error {
	if route.Path == "" || !strings.HasPrefix(route.Path, "/") {
		return fmt.Errorf("route path must start with /")
	}
	if !novatypes.IsSerializable(route.Data()) {
		return fmt.Errorf("route payload must be serializable Nova data")
	}
	return nil
}

func cloneDataMap(input map[string]scheduler.DataValue) map[string]scheduler.DataValue {
	if input == nil {
		return nil
	}
	out := make(map[string]scheduler.DataValue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
