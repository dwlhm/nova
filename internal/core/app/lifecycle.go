package app

import (
	"fmt"
	"reflect"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/scheduler"
	novatypes "github.com/dwlhm/nova/internal/core/types"
)

type LifecycleEvent struct {
	Name    scheduler.SchedulerEvent
	Payload scheduler.DataValue
}

type LifecycleEnqueueResult struct {
	Enqueued    []scheduler.EventEnvelope
	Diagnostics []diagnostic.Diagnostic
}

func CoalesceLifecycleEvents(events []LifecycleEvent) []LifecycleEvent {
	out := make([]LifecycleEvent, 0, len(events))
	for _, event := range events {
		if len(out) > 0 && out[len(out)-1].Name == event.Name && reflect.DeepEqual(out[len(out)-1].Payload, event.Payload) {
			continue
		}
		out = append(out, event)
	}
	return out
}

func EnqueueLifecycleEvents(runtime scheduler.Runtime, source scheduler.CapabilityRef, active map[scheduler.SchedulerEvent]bool, events []LifecycleEvent) (scheduler.Runtime, LifecycleEnqueueResult) {
	next := runtime
	result := LifecycleEnqueueResult{}
	for _, event := range CoalesceLifecycleEvents(events) {
		if !novatypes.IsSerializable(event.Payload) {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:     "NVA-EVENT-019",
				Severity: diagnostic.SeverityError,
				Message:  fmt.Sprintf("app lifecycle event %s payload must be Nova data", event.Name),
			})
			continue
		}
		if !active[event.Name] {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:     "NVA-EVENT-018",
				Severity: diagnostic.SeverityWarning,
				Message:  fmt.Sprintf("app lifecycle event %s is not active for this capability", event.Name),
			})
			continue
		}
		var envelope scheduler.EventEnvelope
		var err error
		next, envelope, err = scheduler.Enqueue(next, source, event.Name, event.Payload)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:     "NVA-RUNTIME-018",
				Severity: diagnostic.SeverityError,
				Message:  err.Error(),
			})
			continue
		}
		result.Enqueued = append(result.Enqueued, envelope)
	}
	return next, result
}
