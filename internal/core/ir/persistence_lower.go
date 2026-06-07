package ir

import (
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
)

func buildHydrationManifest(lifecycles []contract.Lifecycle) *contract.HydrationManifest {
	bootstrap := bootstrapEventFromMount(lifecycles)
	if bootstrap == "" {
		return nil
	}

	afterByEvent := make(map[string]contract.Lifecycle, len(lifecycles))
	for _, lifecycle := range lifecycles {
		if lifecycle.Phase != "after" || lifecycle.Event == "" {
			continue
		}
		afterByEvent[lifecycle.Event] = lifecycle
	}

	current := bootstrap
	loads := make([]contract.HydrationLoadStep, 0, 8)
	skip := map[string]bool{bootstrap: true}

	for guard := 0; guard < 64; guard++ {
		lifecycle, ok := afterByEvent[current]
		if !ok || len(lifecycle.Steps) != 1 {
			return nil
		}
		step := lifecycle.Steps[0]
		if step.Emit != nil {
			if len(loads) == 0 {
				return nil
			}
			skip[current] = true
			return &contract.HydrationManifest{
				BootstrapEvent:  bootstrap,
				Loads:           loads,
				TerminalEvent:   step.Emit.Name,
				SkipAfterEvents: sortedEventNames(skip),
			}
		}
		if step.External == nil || !strings.HasSuffix(step.External.EffectID, "#load") {
			return nil
		}
		if step.External.OnSuccess == "" {
			return nil
		}
		loads = append(loads, contract.HydrationLoadStep{
			TriggerEvent: current,
			EffectID:     step.External.EffectID,
			Input:        cloneStringMap(step.External.Input),
			SuccessEvent: step.External.OnSuccess,
			FailureEvent: step.External.OnFailure,
		})
		skip[current] = true
		skip[step.External.OnSuccess] = true
		current = step.External.OnSuccess
	}
	return nil
}

func bootstrapEventFromMount(lifecycles []contract.Lifecycle) string {
	for _, lifecycle := range lifecycles {
		if lifecycle.Phase != "mount" || len(lifecycle.Steps) != 1 || lifecycle.Steps[0].Emit == nil {
			continue
		}
		return lifecycle.Steps[0].Emit.Name
	}
	return ""
}

func sortedEventNames(events map[string]bool) []string {
	out := make([]string, 0, len(events))
	for event := range events {
		out = append(out, event)
	}
	sort.Strings(out)
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
