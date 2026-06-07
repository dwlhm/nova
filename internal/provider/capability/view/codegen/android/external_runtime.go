package androidcodegen

import externaljava "github.com/dwlhm/nova/runtime/nova-external-java"
import androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"

func JavaExternal(config androidtarget.Config) string {
	return externaljava.Source(config.Namespace)
}
