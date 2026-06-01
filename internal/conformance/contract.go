package conformance

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
)

type ExpectedAppContract struct {
	Version     int      `json:"v"`
	Target      string   `json:"target"`
	Entry       string   `json:"entry"`
	Permissions []string `json:"permissions,omitempty"`
	StateCount  int      `json:"stateCount,omitempty"`
	Bindings    int      `json:"bindings,omitempty"`
	Effects     int      `json:"effects,omitempty"`
	Forbidden   []string `json:"forbidden,omitempty"`
}

func compareExpectedAppContract(expected *ExpectedAppContract, actual contract.App) []Diagnostic {
	if expected == nil {
		return nil
	}
	diagnostics := make([]Diagnostic, 0)
	if expected.Version != 0 && actual.V != expected.Version {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-040",
			fmt.Sprintf("contract v mismatch: expected %d, actual %d", expected.Version, actual.V),
		))
	}
	if expected.Target != "" && actual.Target != expected.Target {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-041",
			fmt.Sprintf("contract target mismatch: expected %q, actual %q", expected.Target, actual.Target),
		))
	}
	if expected.Entry != "" && actual.Entry != expected.Entry {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-042",
			fmt.Sprintf("contract entry mismatch: expected %q, actual %q", expected.Entry, actual.Entry),
		))
	}
	if expected.Permissions != nil && !reflect.DeepEqual(expected.Permissions, actual.Permissions) {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-043",
			fmt.Sprintf("contract permissions mismatch: expected %v, actual %v", expected.Permissions, actual.Permissions),
		))
	}
	if expected.StateCount != 0 && len(actual.Model.States) != expected.StateCount {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-044",
			fmt.Sprintf("contract state count mismatch: expected %d, actual %d", expected.StateCount, len(actual.Model.States)),
		))
	}
	if expected.Bindings != 0 && len(actual.View.Bindings) != expected.Bindings {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-045",
			fmt.Sprintf("contract binding count mismatch: expected %d, actual %d", expected.Bindings, len(actual.View.Bindings)),
		))
	}
	if expected.Effects != 0 && len(actual.Effects) != expected.Effects {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-046",
			fmt.Sprintf("contract effect count mismatch: expected %d, actual %d", expected.Effects, len(actual.Effects)),
		))
	}
	if len(expected.Forbidden) > 0 {
		raw, err := json.Marshal(actual)
		if err != nil {
			diagnostics = append(diagnostics, fixtureDiagnostic("NVA-CONFORMANCE-047", "marshal contract: "+err.Error()))
			return diagnostics
		}
		encoded := string(raw)
		for _, token := range expected.Forbidden {
			if strings.Contains(encoded, token) {
				diagnostics = append(diagnostics, fixtureDiagnostic(
					"NVA-CONFORMANCE-047",
					fmt.Sprintf("contract must not contain %q", token),
				))
			}
		}
	}
	return diagnostics
}
