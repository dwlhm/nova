package security

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/scheduler"
	novatypes "github.com/dwlhm/nova/internal/core/types"
)

func ValidateHostEvent(env novatypes.Environment, contracts []EventContract, event HostEvent) []Diagnostic {
	contract, ok := findEventContract(contracts, event.Name)
	if !ok {
		return []Diagnostic{{
			Code:    "NVA-SEC-010",
			Message: fmt.Sprintf("undeclared scheduler event %s", event.Name),
		}}
	}

	diagnostics := make([]Diagnostic, 0)
	if len(contract.Emitters) > 0 && !canEmit(contract.Emitters, event.Source) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-SEC-011",
			Message: fmt.Sprintf("%s cannot emit scheduler event %s", event.Source, event.Name),
		})
	}
	if !novatypes.IsSerializable(event.Payload) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-SEC-012",
			Message: fmt.Sprintf("event %s payload must be serializable Nova data", event.Name),
		})
		return diagnostics
	}

	return append(diagnostics, validateEventPayload(env, contract, event)...)
}

func validateEventPayload(env novatypes.Environment, contract EventContract, event HostEvent) []Diagnostic {
	payloadType := contract.PayloadType
	if payloadType == "" || payloadType == "unknown" {
		return nil
	}
	if payloadType == "void" {
		if event.Payload == nil {
			return nil
		}
		return []Diagnostic{{
			Code:    "NVA-SEC-013",
			Message: fmt.Sprintf("event %s payload must match void", event.Name),
		}}
	}

	typ, typeDiagnostics := novatypes.ParseRef(env, parser.TypeRef{Text: payloadType})
	if len(typeDiagnostics) > 0 {
		return []Diagnostic{{
			Code:    "NVA-SEC-014",
			Message: fmt.Sprintf("event %s has invalid payload type %s: %s", event.Name, payloadType, typeDiagnostics[0].Message),
		}}
	}
	if err := env.ValidateValue(typ, event.Payload); err != nil {
		return []Diagnostic{{
			Code:    "NVA-SEC-015",
			Message: fmt.Sprintf("event %s payload must match %s", event.Name, payloadType),
		}}
	}
	return nil
}

func findEventContract(contracts []EventContract, name scheduler.SchedulerEvent) (EventContract, bool) {
	for _, contract := range contracts {
		if contract.Name == name {
			return contract, true
		}
	}
	return EventContract{}, false
}

func canEmit(emitters []scheduler.CapabilityRef, source scheduler.CapabilityRef) bool {
	for _, emitter := range emitters {
		if emitter == source {
			return true
		}
	}
	return false
}
