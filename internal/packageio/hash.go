package packageio

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/provider/standard"
)

func assignContentHashes(root string, manifests []packages.Manifest) []packages.Manifest {
	out := make([]packages.Manifest, len(manifests))
	for i, manifest := range manifests {
		packageDir, _ := packageDirectory(root, manifest.Name)
		manifest.ContentHash = manifestContentHash(root, packageDir, manifest)
		out[i] = manifest
	}
	return out
}

func manifestContentHash(root string, packageDir string, manifest packages.Manifest) string {
	parts := make([]string, 0)
	if packageDir != "" {
		manifestPath := filepath.Join(root, filepath.FromSlash(packageDir), "nova.package.toml")
		if content, err := os.ReadFile(manifestPath); err == nil {
			parts = append(parts, filepath.ToSlash(filepath.Join(packageDir, "nova.package.toml"))+"\n"+string(content))
		}
	}

	exportNames := sortedStringKeys(manifest.Exports)
	for _, exportName := range exportNames {
		relPath := manifest.Exports[exportName]
		content := readPackageSource(root, packageDir, relPath)
		parts = append(parts, relPath+"\n"+content)
	}

	targetNames := sortedStringKeysFromTargets(manifest.Targets)
	for _, targetID := range targetNames {
		adapter := manifest.Targets[targetID]
		content := adapter.Content
		if content == "" {
			content = readPackageSource(root, packageDir, adapter.Adapter)
		}
		if content == "" {
			if bundled, ok := standard.WebPlatformAdapter(adapter.Adapter); ok {
				content = bundled
			} else if bundled, ok := standard.AndroidPlatformAdapterSource(adapter.Adapter); ok {
				content = bundled
			}
		}
		parts = append(parts, targetID+":"+adapter.Adapter+"\n"+content)
	}

	if len(parts) == 0 {
		parts = append(parts, manifest.Name+"@"+manifest.Version)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n---\n")))
	return hex.EncodeToString(sum[:])
}

func readPackageSource(root string, packageDir string, relPath string) string {
	if strings.TrimSpace(relPath) == "" {
		return ""
	}
	if packageDir != "" {
		if content, err := readPackageAdapter(root, packageDir, relPath); err == nil {
			return content
		}
	}
	if content, err := readProjectAdapter(root, relPath); err == nil {
		return content
	}
	return ""
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringKeysFromTargets(values map[string]packages.TargetAdapter) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
