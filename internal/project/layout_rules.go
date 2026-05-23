package project

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var adrNamePattern = regexp.MustCompile(`^adr_[0-9]{3}_[a-z0-9]+(?:_[a-z0-9]+)*\.md$`)

func validateSourcePath(filePath string) []Diagnostic {
	if !strings.HasPrefix(filePath, "src/") || !strings.HasSuffix(filePath, ".nova") {
		return nil
	}
	base := path.Base(filePath)
	if isPascalCaseNova(base) {
		return nil
	}
	return []Diagnostic{{
		Code:    "NVA-LAYOUT-004",
		Message: fmt.Sprintf("capability source %s should use PascalCase.nova", filePath),
		Path:    filePath,
	}}
}

func validateGeneratedPath(filePath string) []Diagnostic {
	if !strings.HasPrefix(filePath, "src/build/") {
		return nil
	}
	return []Diagnostic{{
		Code:    "NVA-LAYOUT-005",
		Message: "generated output should stay under build/",
		Path:    filePath,
	}}
}

func validateADRPath(filePath string) []Diagnostic {
	if !strings.HasPrefix(filePath, "docs/adr/") || path.Ext(filePath) != ".md" {
		return nil
	}
	if adrNamePattern.MatchString(path.Base(filePath)) {
		return nil
	}
	return []Diagnostic{{
		Code:    "NVA-LAYOUT-006",
		Message: "ADR docs should use adr_NNN_slug.md",
		Path:    filePath,
	}}
}

func isPascalCaseNova(name string) bool {
	if !strings.HasSuffix(name, ".nova") {
		return false
	}
	stem := strings.TrimSuffix(name, ".nova")
	if stem == "" || stem[0] < 'A' || stem[0] > 'Z' {
		return false
	}
	for _, ch := range stem {
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}
