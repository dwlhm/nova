package cli

import (
	"github.com/dwlhm/nova/internal/packageio"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/provider/build"
)

func externalAdapterContents(root string, target string, operations []build.ResolvedExternalOperation, manifests []packages.Manifest) map[string]string {
	requests := make([]packageio.ExternalAdapterRequest, 0, len(operations))
	for _, operation := range operations {
		requests = append(requests, packageio.ExternalAdapterRequest{
			CapabilitySource: operation.CapabilitySource,
			Path:             operation.Implementation.Path,
		})
	}
	return packageio.CollectExternalAdapterContents(root, target, requests, manifests)
}
