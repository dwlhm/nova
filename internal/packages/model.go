package packages

import (
	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/security"
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
	Renderer     RendererManifest
	Targets      map[string]TargetAdapter
	Permissions  security.PermissionMap
	Dependencies []Dependency
	ContentHash  string
}

type RendererManifest struct {
	Primitives map[string]RendererPrimitive
}

type RendererPrimitive struct {
	Package       string
	Kind          string
	Description   string
	Props         []RendererField
	Events        []RendererEvent
	AllowOverride bool
	Targets       map[string]RendererTarget
}

type RendererField struct {
	Name        string
	Type        string
	Optional    bool
	Description string
}

type RendererEvent struct {
	Name        string
	Payload     string
	Description string
}

type RendererTarget struct {
	Strategy string
	Adapter  string
	Tag      string
	Delegate string
}

type TargetAdapter struct {
	Adapter string
	Content string
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
	Roots              []Dependency
	RendererExtensions []Dependency
	Packages           []Manifest
	Lockfile           Lockfile
	Target             string
	Production         bool
}

type ResolvedGraph struct {
	Packages           []ResolvedPackage
	RendererExtensions []ResolvedRendererPackage
	PermissionSources  map[security.Permission][]PermissionSource
}

type ResolvedPackage struct {
	Name          string
	Version       string
	Types         []PackageType
	TargetAdapter string
	Chain         []string
}

type ResolvedRendererPackage struct {
	Name                 string
	Version              string
	TargetAdapter        string
	TargetAdapterContent string
	Primitives           []RendererPrimitive
}

type PermissionSource struct {
	Package    string
	Version    string
	Permission security.Permission
}

type Diagnostic = diagnostic.Diagnostic
