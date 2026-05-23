package packages

import (
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/security"
)

type PackageType string

const (
	PackageSource             PackageType = "source-package"
	PackageRenderer           PackageType = "renderer-package"
	PackageExternalCapability PackageType = "external-capability-package"
	PackageTargetRuntime      PackageType = "target-runtime-package"
	PackageAdapter            PackageType = "adapter-package"
	PackageTooling            PackageType = "tooling-package"
	PackageConformance        PackageType = "conformance-package"
)

type Manifest struct {
	Name         string
	Version      string
	Types        []PackageType
	Language     string
	ABI          string
	Exports      map[string]string
	Targets      map[string]TargetAdapter
	Permissions  security.PermissionMap
	Dependencies []Dependency
	ContentHash  string
}

type TargetAdapter struct {
	Adapter string
}

type Dependency struct {
	Name       string
	Constraint string
}

type LockEntry struct {
	Name           string
	Version        string
	Source         string
	ContentHash    string
	Dependencies   []Dependency
	TargetAdapters map[string]string
	Permissions    security.PermissionMap
	ABI            string
}

type Lockfile struct {
	Entries []LockEntry
}

type ResolutionInput struct {
	Roots      []Dependency
	Packages   []Manifest
	Lockfile   Lockfile
	Target     string
	Production bool
}

type ResolvedGraph struct {
	Packages          []ResolvedPackage
	PermissionSources map[security.Permission][]PermissionSource
}

type ResolvedPackage struct {
	Name          string
	Version       string
	Types         []PackageType
	TargetAdapter string
	Chain         []string
}

type PermissionSource struct {
	Package    string
	Version    string
	Permission security.Permission
}

type Diagnostic = diagnostic.Diagnostic
