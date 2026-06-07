package expr

import "github.com/dwlhm/nova/internal/core/scheduler"

// ContextFromScheduler builds a pure evaluation context from scheduler snapshot and event.
func ContextFromScheduler(snapshot scheduler.Snapshot, event scheduler.EventEnvelope, paramNames map[string]bool) Context {
	state := make(map[string]any)
	for key, value := range snapshot.Values() {
		state[string(key.Name)] = value
	}
	params := make(map[string]any)
	switch payload := event.Payload.(type) {
	case map[string]scheduler.DataValue:
		for name, value := range payload {
			if paramNames[name] {
				params[name] = value
			}
		}
	case map[string]any:
		for name, value := range payload {
			if paramNames[name] {
				params[name] = value
			}
		}
	}
	return Context{State: state, Params: params}
}
