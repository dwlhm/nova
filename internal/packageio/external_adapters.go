package packageio

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dwlhm/nova/internal/packages"
)

type ExternalAdapterRequest struct {
	CapabilitySource string
	Path             string
}

func CollectExternalAdapterContents(root string, target string, requests []ExternalAdapterRequest, manifests []packages.Manifest) map[string]string {
	contents := make(map[string]string)
	for _, request := range requests {
		path := strings.TrimSpace(request.Path)
		if path == "" {
			continue
		}
		for _, manifest := range manifests {
			if manifest.Name != request.CapabilitySource {
				continue
			}
			if !hasExternalCapabilityPackage(manifest) {
				continue
			}
			targetAdapter, ok := manifest.Targets[target]
			if !ok || targetAdapter.Adapter != path || strings.TrimSpace(targetAdapter.Content) == "" {
				continue
			}
			contents[path] = targetAdapter.Content
		}
	}
	for _, request := range requests {
		path := strings.TrimSpace(request.Path)
		if path == "" {
			continue
		}
		content, err := readProjectAdapter(root, path)
		if err != nil {
			continue
		}
		contents[path] = content
	}
	return contents
}

func hasExternalCapabilityPackage(manifest packages.Manifest) bool {
	for _, typ := range manifest.Types {
		if typ == packages.PackageExternalCapability {
			return true
		}
	}
	return false
}

func readProjectAdapter(root string, adapterPath string) (string, error) {
	cleanPath := filepath.Clean(filepath.FromSlash(adapterPath))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
		return "", os.ErrInvalid
	}
	content, err := os.ReadFile(filepath.Join(root, cleanPath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}
