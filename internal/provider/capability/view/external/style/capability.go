package style

import "github.com/dwlhm/nova/internal/provider/capability/view/external"

var Module = external.Module{
	Metadata:    external.CrossCuttingMetadata("style", 20),
	AndroidEmit: external.RegisterAndroidArtifact("style", 20, emitAndroidArtifacts),
	WebEmit:     external.RegisterWebArtifact("style", 20, emitWebArtifacts),
}
