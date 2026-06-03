package androidcodegen

import (
	"github.com/dwlhm/nova/internal/provider/shared"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
)

type storageHydrationStep struct {
	EffectID     string
	Input        map[string]string
	SuccessEvent string
	FailureEvent string
}

type storageHydrationChain struct {
	Steps           []storageHydrationStep
	TerminalEvent   string
	AfterSkipEvents map[string]bool
}

func detectStorageHydrationChain(lifecycles []contract.Lifecycle) (storageHydrationChain, bool) {
	afterByEvent := make(map[string]contract.Lifecycle, len(lifecycles))
	for _, lifecycle := range lifecycles {
		if lifecycle.Phase != "after" || lifecycle.Event == "" {
			continue
		}
		afterByEvent[lifecycle.Event] = lifecycle
	}

	current := "@restore"
	steps := make([]storageHydrationStep, 0, 8)
	skip := map[string]bool{"@restore": true}

	for guard := 0; guard < 64; guard++ {
		lifecycle, ok := afterByEvent[current]
		if !ok || len(lifecycle.Steps) != 1 {
			return storageHydrationChain{}, false
		}
		step := lifecycle.Steps[0]
		if step.Emit != nil {
			if len(steps) == 0 {
				return storageHydrationChain{}, false
			}
			skip[current] = true
			return storageHydrationChain{
				Steps:           steps,
				TerminalEvent:   step.Emit.Name,
				AfterSkipEvents: skip,
			}, true
		}
		if step.External == nil || !strings.HasSuffix(step.External.EffectID, "#load") {
			return storageHydrationChain{}, false
		}
		if step.External.OnSuccess == "" {
			return storageHydrationChain{}, false
		}
		steps = append(steps, storageHydrationStep{
			EffectID:     step.External.EffectID,
			Input:        step.External.Input,
			SuccessEvent: step.External.OnSuccess,
			FailureEvent: step.External.OnFailure,
		})
		skip[current] = true
		skip[step.External.OnSuccess] = true
		current = step.External.OnSuccess
	}
	return storageHydrationChain{}, false
}

func androidContractStorageHydration(chain storageHydrationChain) string {
	if len(chain.Steps) == 0 || chain.AfterSkipEvents == nil {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("    private static final Set<String> HYDRATION_SKIP_AFTER = new LinkedHashSet<>(Arrays.asList(\n")
	skipEvents := make([]string, 0, len(chain.AfterSkipEvents))
	for event := range chain.AfterSkipEvents {
		skipEvents = append(skipEvents, event)
	}
	sort.Strings(skipEvents)
	for _, event := range skipEvents {
		builder.WriteString("        " + shared.QuoteCodeString(event) + ",\n")
	}
	builder.WriteString("        \"\"\n")
	builder.WriteString("    ));\n\n")
	builder.WriteString("    private void hydratePersistedState() {\n")
	for _, step := range chain.Steps {
		builder.WriteString("        try {\n")
		builder.WriteString("            Object output = NovaExternalAdapters.invoke(this, ")
		builder.WriteString(shared.QuoteCodeString(step.EffectID) + ", ")
		builder.WriteString(androidJavaHydrateExternalInputMap(step.Input) + ");\n")
		builder.WriteString("            scheduler.commitTransition(")
		builder.WriteString(shared.QuoteCodeString(step.SuccessEvent) + ", ")
		builder.WriteString("output == null ? Collections.emptyList() : Collections.singletonList(output));\n")
		builder.WriteString("        } catch (Exception error) {\n")
		if step.FailureEvent != "" {
			builder.WriteString("            scheduler.commitTransition(")
			builder.WriteString(shared.QuoteCodeString(step.FailureEvent) + ", Collections.singletonList(error.getMessage()));\n")
		}
		builder.WriteString("            return;\n")
		builder.WriteString("        }\n")
	}
	if chain.TerminalEvent != "" {
		builder.WriteString("        scheduler.commitTransition(")
		builder.WriteString(shared.QuoteCodeString(chain.TerminalEvent) + ", Collections.emptyList());\n")
	}
	builder.WriteString("    }\n\n")
	return builder.String()
}

func androidJavaHydrateExternalInputMap(input map[string]string) string {
	if len(input) == 0 {
		return "Collections.emptyMap()"
	}
	names := make([]string, 0, len(input))
	for name := range input {
		names = append(names, name)
	}
	sort.Strings(names)
	pairs := make([]string, 0, len(names))
	for _, name := range names {
		pairs = append(pairs, "entry("+shared.QuoteCodeString(name)+", "+androidJavaEvalValueExpr(input[name])+")")
	}
	return "record(" + strings.Join(pairs, ", ") + ")"
}
