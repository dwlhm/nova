package internal

import (
	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/capability/view/hostctx"
	androidregistry "github.com/dwlhm/nova/internal/provider/capability/view/registry/android"
	webregistry "github.com/dwlhm/nova/internal/provider/capability/view/registry/web"
)

type registries struct {
	androidLower *androidregistry.LowerRegistry
	androidEmit  *androidregistry.EmitRegistry
	webLower     *webregistry.LowerRegistry
	webEmit      *webregistry.EmitRegistry
}

func bootstrap(ctx *hostctx.Context, modules []external.Module, target string) registries {
	regs := registries{
		androidLower: androidregistry.NewLowerRegistry(),
		androidEmit:  androidregistry.NewEmitRegistry(),
		webLower:     webregistry.NewLowerRegistry(),
		webEmit:      webregistry.NewEmitRegistry(),
	}

	for _, module := range modules {
		if !module.Metadata.SupportsTarget(target) {
			continue
		}
		if module.AndroidLower != nil {
			module.AndroidLower(regs.androidLower)
		}
		if module.WebLower != nil {
			module.WebLower(regs.webLower)
		}
		if module.AndroidEmit != nil {
			module.AndroidEmit(regs.androidEmit)
		}
		if module.WebEmit != nil {
			module.WebEmit(regs.webEmit)
		}
	}

	if target == "android" && regs.androidEmit != nil {
		ctx.AndroidNodes = regs.androidEmit.NodeRegistry()
	}

	return regs
}

// BootstrapAndroidNodes is kept for compatibility with tests that build node registries directly.
func BootstrapAndroidNodes(registry *androidcodegen.NodeRegistry) {
	ctx := hostctx.Context{}
	regs := bootstrap(&ctx, defaultModules(), "android")
	if regs.androidEmit != nil {
		regs.androidEmit.NodeRegistry().MergeInto(registry)
	}
}
