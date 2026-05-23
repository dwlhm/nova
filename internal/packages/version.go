package packages

import (
	"strconv"
	"strings"
)

func versionSatisfies(version string, constraint string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}
	parts := strings.Fields(constraint)
	if len(parts) == 0 {
		return true
	}
	if len(parts) == 1 && !strings.HasPrefix(parts[0], ">") && !strings.HasPrefix(parts[0], "<") && !strings.HasPrefix(parts[0], "=") {
		return version == parts[0]
	}
	for _, part := range parts {
		if !versionSatisfiesPart(version, part) {
			return false
		}
	}
	return true
}

func versionSatisfiesPart(version string, part string) bool {
	switch {
	case strings.HasPrefix(part, ">="):
		return compareVersion(version, strings.TrimPrefix(part, ">=")) >= 0
	case strings.HasPrefix(part, ">"):
		return compareVersion(version, strings.TrimPrefix(part, ">")) > 0
	case strings.HasPrefix(part, "<="):
		return compareVersion(version, strings.TrimPrefix(part, "<=")) <= 0
	case strings.HasPrefix(part, "<"):
		return compareVersion(version, strings.TrimPrefix(part, "<")) < 0
	case strings.HasPrefix(part, "="):
		return version == strings.TrimPrefix(part, "=")
	default:
		return version == part
	}
}

func compareVersion(left string, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for i := 0; i < 3; i++ {
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	return 0
}

func versionParts(version string) [3]int {
	version = strings.TrimPrefix(version, "v")
	raw := strings.Split(version, ".")
	parts := [3]int{}
	for i := 0; i < len(raw) && i < len(parts); i++ {
		value, _ := strconv.Atoi(strings.TrimLeftFunc(raw[i], func(r rune) bool {
			return r < '0' || r > '9'
		}))
		parts[i] = value
	}
	return parts
}
