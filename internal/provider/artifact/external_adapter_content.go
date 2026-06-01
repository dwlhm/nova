package artifact

import (
	"strings"

	"github.com/dwlhm/nova/internal/provider/standard"
)

func webAdapterContent(path string, overrides map[string]string) (string, bool) {
	if overrides != nil {
		if content, ok := overrides[path]; ok && strings.TrimSpace(content) != "" {
			return content, true
		}
	}
	return standard.WebPlatformAdapter(path)
}

func androidAdapterContent(path string, overrides map[string]string) (string, bool) {
	if overrides != nil {
		if content, ok := overrides[path]; ok && strings.TrimSpace(content) != "" {
			return content, true
		}
	}
	return standard.AndroidPlatformAdapterSource(path)
}
