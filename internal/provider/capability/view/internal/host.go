package internal

import (
	"sort"

	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/runtime"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

type Host struct {
	modules []external.Module
}

func NewDefaultHost() Host {
	return Host{modules: defaultModules()}
}

func (host Host) CapabilityIDs() []string {
	ids := make([]string, 0, len(host.modules))
	for _, module := range host.modules {
		ids = append(ids, module.Metadata.ID)
	}
	sort.Strings(ids)
	return ids
}

func (host Host) ModulesForTarget(target string) []external.Module {
	out := make([]external.Module, 0, len(host.modules))
	for _, module := range host.modules {
		if module.Metadata.SupportsTarget(target) {
			out = append(out, module)
		}
	}
	return out
}

func (host Host) Generate(input shared.GenerateInput) ([]shared.File, []shared.Diagnostic) {
	ctx := external.NewContext(input)
	target := input.Plan.Target
	active := host.ModulesForTarget(target)
	regs := bootstrap(&ctx, active, target)

	if target == "android" {
		config, diagnostics := androidtarget.ParseConfig(input.Project)
		if len(diagnostics) > 0 {
			return nil, diagnostics
		}
		ctx.Android = &external.AndroidContext{
			Config:     config,
			SourceRoot: runtime.AndroidSourceRoot(config.Namespace),
		}
	}

	out := make([]shared.File, 0)
	diagnostics := make([]shared.Diagnostic, 0)

	switch target {
	case "android":
		if regs.androidEmit != nil {
			files, artifactDiagnostics := regs.androidEmit.Artifacts(ctx)
			diagnostics = append(diagnostics, artifactDiagnostics...)
			out = append(out, files...)
		}
	case "web":
		if regs.webEmit != nil {
			files, artifactDiagnostics := regs.webEmit.Artifacts(ctx)
			diagnostics = append(diagnostics, artifactDiagnostics...)
			out = append(out, files...)
		}
	}

	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out, nil
}
