package style

import (
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func emitAndroidArtifacts(ctx external.Context) ([]shared.File, []shared.Diagnostic) {
	if ctx.Android == nil {
		return nil, nil
	}
	return []shared.File{{
		Path:    "build/android/app/src/main/res/values/styles.xml",
		Content: androidtarget.StylesXML(ctx.Android.Config),
	}}, nil
}
