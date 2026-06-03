package runtime

import "github.com/dwlhm/nova/internal/provider/capability/view/external"

var Module = external.Module{
	Metadata:    external.ShellMetadata("runtime", 90),
	AndroidEmit: external.RegisterAndroidArtifact("runtime", 90, emitAndroidArtifacts),
	WebEmit:     external.RegisterWebArtifact("runtime", 90, emitWebArtifacts),
}
