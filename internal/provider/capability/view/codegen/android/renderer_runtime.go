package androidcodegen

import rendererjava "github.com/dwlhm/nova/runtime/nova-renderer-java"
import androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"

func JavaRenderer(config androidtarget.Config) string {
	return rendererjava.Source(config.Namespace)
}
