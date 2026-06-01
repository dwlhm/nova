package build

import (
	"fmt"
	"sort"

	"github.com/dwlhm/nova/internal/core/view"
)

func ValidateViewRenderer(plan BuildPlan, viewIR view.IR) []Diagnostic {
	known := make(map[string]bool, len(plan.Renderer.Primitives))
	known["#text"] = true
	for _, primitive := range plan.Renderer.Primitives {
		known[primitive.Kind] = true
	}
	diagnostics := make([]Diagnostic, 0)
	var visit func([]view.Node)
	visit = func(nodes []view.Node) {
		for _, node := range nodes {
			if !known[node.Kind] {
				diagnostics = append(diagnostics, unknownKindDiagnostic(node.Kind, plan.Renderer.UnknownKind, plan.Target))
			}
			visit(node.Children)
		}
	}
	visit(viewIR.Nodes)
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return diagnostics
}

func HasBlockingDiagnostics(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != "NVA-RENDER-002" {
			return true
		}
	}
	return false
}

func unknownKindDiagnostic(kind string, policy string, profile string) Diagnostic {
	if policy == "warn" || policy == "passthrough_web" && profile == "web" {
		return Diagnostic{
			Code:    "NVA-RENDER-002",
			Message: fmt.Sprintf("unknown view kind %s is allowed by renderer.unknown_kind policy for target %s", kind, profile),
			Target:  profile,
		}
	}
	return Diagnostic{
		Code:    "NVA-RENDER-001",
		Message: fmt.Sprintf("unknown view kind %s for target %s", kind, profile),
		Target:  profile,
	}
}
