package packages

import (
	"fmt"
	"strings"

	"github.com/dwlhm/nova/internal/core/security"
)

func ParseLockfile(input string) (Lockfile, []Diagnostic) {
	lockfile := Lockfile{Entries: make([]LockEntry, 0)}
	diagnostics := make([]Diagnostic, 0)
	section := ""
	entryIndex := -1

	for lineNumber, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(stripManifestComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			if section == "packages" {
				lockfile.Entries = append(lockfile.Entries, LockEntry{TargetAdapters: make(map[string]string)})
				entryIndex = len(lockfile.Entries) - 1
			}
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-016", fmt.Sprintf("invalid lockfile entry on line %d", lineNumber+1)))
			continue
		}
		key = parseManifestKey(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		switch section {
		case "packages":
			if entryIndex < 0 || entryIndex >= len(lockfile.Entries) {
				diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-016", fmt.Sprintf("lockfile package field %s outside [[packages]] on line %d", key, lineNumber+1)))
				continue
			}
			assignLockEntryField(&lockfile.Entries[entryIndex], key, value, &diagnostics, lineNumber+1)
		case "packages.target_adapters":
			if entryIndex < 0 || entryIndex >= len(lockfile.Entries) {
				diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-016", fmt.Sprintf("lockfile target adapter outside [[packages]] on line %d", lineNumber+1)))
				continue
			}
			lockfile.Entries[entryIndex].TargetAdapters[key] = parseManifestString(value, &diagnostics, lineNumber+1)
		default:
			if targetID, ok := strings.CutPrefix(section, "packages.target_adapters."); ok && entryIndex >= 0 {
				if lockfile.Entries[entryIndex].TargetAdapters == nil {
					lockfile.Entries[entryIndex].TargetAdapters = make(map[string]string)
				}
				lockfile.Entries[entryIndex].TargetAdapters[targetID] = parseManifestString(value, &diagnostics, lineNumber+1)
			}
		}
	}
	return lockfile, diagnostics
}

func assignLockEntryField(entry *LockEntry, key string, value string, diagnostics *[]Diagnostic, lineNumber int) {
	switch key {
	case "name":
		entry.Name = parseManifestString(value, diagnostics, lineNumber)
	case "version":
		entry.Version = parseManifestString(value, diagnostics, lineNumber)
	case "source":
		entry.Source = parseManifestString(value, diagnostics, lineNumber)
	case "content_hash":
		entry.ContentHash = parseManifestString(value, diagnostics, lineNumber)
	case "abi":
		entry.ABI = parseManifestString(value, diagnostics, lineNumber)
	case "permissions":
		enabled, ok := parseManifestBool(value)
		if !ok {
			*diagnostics = append(*diagnostics, pkgDiagnostic("NVA-PKG-016", fmt.Sprintf("lock permission %s must be true or false on line %d", key, lineNumber)))
			return
		}
		if entry.Permissions == nil {
			entry.Permissions = make(security.PermissionMap)
		}
		entry.Permissions[security.Permission(key)] = enabled
	default:
		if strings.HasPrefix(key, "permissions.") {
			perm := strings.TrimPrefix(key, "permissions.")
			enabled, ok := parseManifestBool(value)
			if !ok {
				*diagnostics = append(*diagnostics, pkgDiagnostic("NVA-PKG-016", fmt.Sprintf("lock permission %s must be true or false on line %d", perm, lineNumber)))
				return
			}
			if entry.Permissions == nil {
				entry.Permissions = make(security.PermissionMap)
			}
			entry.Permissions[security.Permission(perm)] = enabled
		}
	}
}
