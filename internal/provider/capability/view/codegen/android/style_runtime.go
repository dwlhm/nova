package androidcodegen

import (
	stylejava "github.com/dwlhm/nova/runtime/nova-style-java"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
)

func JavaStyle(config androidtarget.Config) string {
	return stylejava.Source(config.Namespace)
}
