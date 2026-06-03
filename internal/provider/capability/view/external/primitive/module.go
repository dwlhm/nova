package primitive

import (
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	androidregistry "github.com/dwlhm/nova/internal/provider/capability/view/registry/android"
)

// Android builds a primitive module with default lower and a shared node renderer.
func Android(meta external.Metadata, kind string, render NodeRender) external.Module {
	return AndroidKinds(meta, []string{kind}, render)
}

// AndroidKinds registers multiple Nova kinds that share the same lower/render path.
func AndroidKinds(meta external.Metadata, kinds []string, render NodeRender) external.Module {
	return external.Module{
		Metadata: meta,
		AndroidLower: func(registry *androidregistry.LowerRegistry) {
			for _, kind := range kinds {
				RegisterAndroidLower(registry, kind)
			}
		},
		AndroidEmit: func(registry *androidregistry.EmitRegistry) {
			for _, kind := range kinds {
				RegisterAndroidEmit(registry, kind, render)
			}
		},
	}
}

// AndroidCustom builds a primitive module with custom lower and emit functions.
func AndroidCustom(
	meta external.Metadata,
	kind string,
	lower androidregistry.NodeLowerFunc,
	emit androidregistry.NodeEmitFunc,
) external.Module {
	return external.Module{
		Metadata: meta,
		AndroidLower: func(registry *androidregistry.LowerRegistry) {
			registry.Register(kind, lower)
		},
		AndroidEmit: func(registry *androidregistry.EmitRegistry) {
			registry.RegisterNode(kind, emit)
		},
	}
}
