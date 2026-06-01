package ir

import (
	"github.com/dwlhm/nova/internal/core/capability"
	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/core/view"
)

type SourceFile struct {
	Path string
	File parser.File
}

type ModuleRef struct {
	Path string
}

type TemplateRef struct {
	SourceFile string
	Index      int
}

type ResolvedExternal struct {
	RequestingFile   string
	CapabilitySource string
	CapabilityName   string
	Operation        string
	Output           string
	Permissions      []security.Permission
	Implementation   string
}

type LowerInput struct {
	Profile     string
	Entry       string
	Modules     []ModuleRef
	Template    TemplateRef
	Sources     []SourceFile
	Permissions []security.Permission
	Externals   []ResolvedExternal
}

type Bundle struct {
	App                 contract.App
	ViewIR              view.IR
	Modules             []string
	CapabilityManifests []capability.Manifest
	Routes              []RouteModel
}

type RouteModel struct {
	Pattern  string   `json:"pattern"`
	NodePath []int    `json:"nodePath"`
	Params   []string `json:"params"`
	Score    int      `json:"score"`
	Fallback bool     `json:"fallback"`
}

type Diagnostic struct {
	Code    string
	Message string
}
