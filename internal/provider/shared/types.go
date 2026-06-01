package shared

import (
	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/build"
)

const (
	LanguageVersion  = "0.1.0"
	ABIVersion       = "0.1.0"
	SchedulerVersion = "0.1.0"
	ViewIRVersion    = "0.1.0"
	RuntimeVersion   = "0.1.0"
)

type File struct {
	Path    string
	Content string
}

type StyleAsset struct {
	SourcePath string
	Content    string
	Scope      StyleScope
}

type StyleScope string

const (
	StyleScopeGlobal StyleScope = "global"
	StyleScopeApp    StyleScope = "app"
)

type Diagnostic = diagnostic.Diagnostic

type GenerateInput struct {
	Project                 project.Manifest
	Bundle                  ir.Bundle
	Plan                    build.BuildPlan
	TargetManifest          build.TargetManifest
	StyleAssets             []StyleAsset
	ExternalAdapterContents map[string]string
}

type ManifestVersions struct {
	LanguageVersion  string
	SchedulerVersion string
	RuntimeVersion   string
}

func DefaultManifestVersions() ManifestVersions {
	return ManifestVersions{
		LanguageVersion:  LanguageVersion,
		SchedulerVersion: SchedulerVersion,
		RuntimeVersion:   RuntimeVersion,
	}
}
