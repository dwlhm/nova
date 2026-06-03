package external

import (
	androidregistry "github.com/dwlhm/nova/internal/provider/capability/view/registry/android"
	webregistry "github.com/dwlhm/nova/internal/provider/capability/view/registry/web"
)

// Hook functions wire a sandboxed module into target registries during host bootstrap.
// Nil hooks are skipped (no-op for that phase).

type AndroidLowerHook func(*androidregistry.LowerRegistry)
type WebLowerHook func(*webregistry.LowerRegistry)
type AndroidEmitHook func(*androidregistry.EmitRegistry)
type WebEmitHook func(*webregistry.EmitRegistry)

// Module is a pure-data capability definition: metadata plus optional register hooks.
type Module struct {
	Metadata     Metadata
	AndroidLower AndroidLowerHook
	WebLower     WebLowerHook
	AndroidEmit  AndroidEmitHook
	WebEmit      WebEmitHook
}

func RegisterAndroidArtifact(id string, order int, emit androidregistry.ArtifactEmitFunc) AndroidEmitHook {
	return func(registry *androidregistry.EmitRegistry) {
		registry.RegisterArtifact(id, order, emit)
	}
}

func RegisterWebArtifact(id string, order int, emit webregistry.ArtifactEmitFunc) WebEmitHook {
	return func(registry *webregistry.EmitRegistry) {
		registry.RegisterArtifact(id, order, emit)
	}
}
