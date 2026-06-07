package androidcodegen

import (
	runtimejava "github.com/dwlhm/nova/runtime/nova-runtime-java"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
)

func JavaRuntime(config androidtarget.Config) string {
	return runtimejava.Source(config.Namespace)
}

func JavaHydration(config androidtarget.Config) string {
	return runtimejava.HydrationSource(config.Namespace)
}
