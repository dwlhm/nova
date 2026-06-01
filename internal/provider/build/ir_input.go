package build

import (
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/provider/target"
)

type (
	TargetManifest     = target.PlanManifest
	ExternalCapability = target.ExternalCapability
	ExternalOperation  = target.ExternalOperation
	Field              = target.Field
	Implementation     = target.Implementation
)

func WebTargetManifest() TargetManifest {
	return target.WebPlanManifest()
}

func AndroidTargetManifest() TargetManifest {
	return target.AndroidPlanManifest()
}

func TargetManifestFor(profile string) (TargetManifest, bool) {
	return target.PlanManifestFor(profile)
}

func IRLowerInput(plan BuildPlan, sources []SourceFile) ir.LowerInput {
	externals := make([]ir.ResolvedExternal, 0, len(plan.ExternalOperations))
	for _, operation := range plan.ExternalOperations {
		externals = append(externals, ir.ResolvedExternal{
			RequestingFile:   operation.RequestingFile,
			CapabilitySource: operation.CapabilitySource,
			CapabilityName:   operation.CapabilityName,
			Operation:        operation.Operation,
			Output:           operation.Output,
			Permissions:      clonePermissions(operation.Permissions),
			Implementation:   operation.Implementation.Path,
		})
	}
	irSources := make([]ir.SourceFile, 0, len(sources))
	for _, source := range sources {
		irSources = append(irSources, ir.SourceFile{Path: source.Path, File: source.File})
	}
	modules := make([]ir.ModuleRef, 0, len(plan.Modules))
	for _, module := range plan.Modules {
		modules = append(modules, ir.ModuleRef{Path: module.Path})
	}
	return ir.LowerInput{
		Profile:     plan.Target,
		Entry:       plan.Entry,
		Modules:     modules,
		Template:    ir.TemplateRef{SourceFile: plan.Template.SourceFile, Index: plan.Template.Index},
		Sources:     irSources,
		Permissions: clonePermissions(plan.Permissions),
		Externals:   externals,
	}
}

func IRDiagnostics(diagnostics []ir.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, Diagnostic{Code: diagnostic.Code, Message: diagnostic.Message})
	}
	return out
}
