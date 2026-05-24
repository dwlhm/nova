package build

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/security"
)

type Selection string

const (
	SelectionExactTarget Selection = "exact_target"
	SelectionPolymorphic Selection = "polymorphic"
)

type SourceFile struct {
	Path string
	File parser.File
}

type TargetManifest struct {
	ID                   string
	Families             []string
	ExternalCapabilities []ExternalCapability
	PermissionMappings   security.PermissionMap
}

type ExternalCapability struct {
	Source     string
	Operations []ExternalOperation
}

type ExternalOperation struct {
	Name            string
	Inputs          []Field
	Output          string
	Permissions     []security.Permission
	Implementations []Implementation
}

type Field struct {
	Name     string
	Optional bool
	Type     string
}

type Implementation struct {
	Path    string
	Target  string
	Family  string
	Common  bool
	Default bool
}

type ResolutionInput struct {
	Project        project.Manifest
	Target         string
	Sources        []SourceFile
	TargetManifest TargetManifest
	PackageGraph   packages.ResolvedGraph
}

type ResolutionResult struct {
	Plan        BuildPlan
	Diagnostics []Diagnostic
}

type BuildPlan struct {
	Target             string
	Entry              string
	Modules            []ModuleRef
	Template           TemplateRef
	ExternalOperations []ResolvedExternalOperation
	Permissions        []security.Permission
	Renderer           RendererPlan
	Artifact           ArtifactMetadata
}

type ModuleRef struct {
	Path string
}

type TemplateRef struct {
	SourceFile string
	Target     string
	Selection  Selection
	Index      int
}

type ResolvedExternalOperation struct {
	RequestingFile   string
	CapabilitySource string
	CapabilityName   string
	Operation        string
	Permissions      []security.Permission
	Implementation   Implementation
}

type ArtifactMetadata struct {
	Target      string
	Entry       string
	Modules     []string
	Permissions []security.Permission
}

type RendererPlan struct {
	UnknownKind string
	Primitives  []RendererPrimitive
	Extensions  []RendererExtension
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

type RendererExtension struct {
	Package        string
	Version        string
	AdapterPath    string
	AdapterContent string
	Primitives     []string
}

type Diagnostic struct {
	Code           string
	Message        string
	Target         string
	RequestingFile string
	ImportSource   string
	Considered     []string
}

func Resolve(input ResolutionInput) ResolutionResult {
	target := resolveTarget(input)
	entry := normalizePath(input.Project.Project.Entry)
	sourceMap := buildSourceMap(input.Sources)

	plan := BuildPlan{Target: target, Entry: entry}
	diagnostics := make([]Diagnostic, 0)
	modules, moduleDiagnostics := resolveModuleGraph(entry, sourceMap)
	diagnostics = append(diagnostics, moduleDiagnostics...)
	plan.Modules = moduleRefs(modules)

	if entryFile, ok := sourceMap[entry]; ok {
		template, templateDiagnostics := selectTemplate(entry, target, entryFile.Templates)
		plan.Template = template
		diagnostics = append(diagnostics, templateDiagnostics...)
	}

	external, externalDiagnostics := resolveExternalOperations(target, input.TargetManifest, modules, sourceMap)
	plan.ExternalOperations = external
	diagnostics = append(diagnostics, externalDiagnostics...)
	plan.Permissions = collectPermissions(external)

	securityDiagnostics := auditPermissions(input.Project.Permissions, input.TargetManifest.PermissionMappings, external)
	diagnostics = append(diagnostics, securityDiagnostics...)

	renderer, rendererDiagnostics := resolveRendererPlan(input.Project.Renderer, input.PackageGraph, target)
	diagnostics = append(diagnostics, rendererDiagnostics...)
	plan.Renderer = renderer

	plan.Artifact = ArtifactMetadata{
		Target:      target,
		Entry:       entry,
		Modules:     modulePaths(modules),
		Permissions: clonePermissions(plan.Permissions),
	}
	return ResolutionResult{Plan: plan, Diagnostics: diagnostics}
}

func resolveTarget(input ResolutionInput) string {
	if input.Target != "" {
		return input.Target
	}
	return input.TargetManifest.ID
}

func resolveModuleGraph(entry string, sources map[string]parser.File) ([]string, []Diagnostic) {
	visited := make(map[string]bool)
	modules := make([]string, 0)
	diagnostics := make([]Diagnostic, 0)

	var visit func(string)
	visit = func(modulePath string) {
		modulePath = normalizePath(modulePath)
		if visited[modulePath] {
			return
		}
		visited[modulePath] = true

		file, ok := sources[modulePath]
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:           "NVA-TARGET-001",
				RequestingFile: modulePath,
				Message:        fmt.Sprintf("source module %s was not found", modulePath),
			})
			return
		}
		modules = append(modules, modulePath)
		for _, decl := range file.Imports {
			if !isLocalImport(decl.From) {
				continue
			}
			visit(normalizePath(path.Join(path.Dir(modulePath), decl.From)))
		}
	}

	visit(entry)
	return modules, diagnostics
}

func selectTemplate(entry string, target string, templates []parser.TemplateDecl) (TemplateRef, []Diagnostic) {
	for i, template := range templates {
		if template.Target == target {
			return TemplateRef{
				SourceFile: entry,
				Target:     template.Target,
				Selection:  SelectionExactTarget,
				Index:      i,
			}, nil
		}
	}
	for i, template := range templates {
		if template.Target == "" {
			return TemplateRef{
				SourceFile: entry,
				Selection:  SelectionPolymorphic,
				Index:      i,
			}, nil
		}
	}
	return TemplateRef{}, []Diagnostic{{
		Code:           "NVA-TEMPLATE-001",
		Target:         target,
		RequestingFile: entry,
		Message:        fmt.Sprintf("no template for target %s requested by %s", target, entry),
	}}
}

func resolveExternalOperations(target string, targetManifest TargetManifest, modules []string, sources map[string]parser.File) ([]ResolvedExternalOperation, []Diagnostic) {
	capabilities := externalCapabilityMap(targetManifest.ExternalCapabilities)
	resolved := make([]ResolvedExternalOperation, 0)
	diagnostics := make([]Diagnostic, 0)

	for _, modulePath := range modules {
		file := sources[modulePath]
		for _, external := range file.ExternalImports {
			capability, ok := capabilities[external.From]
			if !ok {
				diagnostics = append(diagnostics, Diagnostic{
					Code:           "NVA-TARGET-003",
					Target:         target,
					RequestingFile: modulePath,
					ImportSource:   external.From,
					Message:        fmt.Sprintf("%s has no target capability manifest entry requested by %s", external.From, modulePath),
				})
				continue
			}
			operations := externalOperationMap(capability.Operations)
			for _, declaredOperation := range external.Operations {
				targetOperation, ok := operations[declaredOperation.Name]
				if !ok {
					diagnostics = append(diagnostics, Diagnostic{
						Code:           "NVA-TARGET-005",
						Target:         target,
						RequestingFile: modulePath,
						ImportSource:   external.From,
						Message:        fmt.Sprintf("%s.%s is not provided by target %s requested by %s", external.From, declaredOperation.Name, target, modulePath),
					})
					continue
				}
				diagnostics = append(diagnostics, validateExternalSignature(target, modulePath, external.From, declaredOperation, targetOperation)...)
				implementation, considered, ok := selectImplementation(target, targetManifest.Families, targetOperation.Implementations)
				if !ok {
					diagnostics = append(diagnostics, Diagnostic{
						Code:           "NVA-TARGET-004",
						Target:         target,
						RequestingFile: modulePath,
						ImportSource:   external.From,
						Considered:     considered,
						Message: fmt.Sprintf(
							"%s.%s has no %s implementation; requested by %s; target: %s; considered: %s",
							external.From,
							declaredOperation.Name,
							target,
							modulePath,
							target,
							strings.Join(considered, ", "),
						),
					})
					continue
				}
				resolved = append(resolved, ResolvedExternalOperation{
					RequestingFile:   modulePath,
					CapabilitySource: external.From,
					CapabilityName:   external.Name,
					Operation:        declaredOperation.Name,
					Permissions:      clonePermissions(targetOperation.Permissions),
					Implementation:   implementation,
				})
			}
		}
	}

	return resolved, diagnostics
}

func validateExternalSignature(target string, requestingFile string, source string, declared parser.ExternalOperationDecl, targetOperation ExternalOperation) []Diagnostic {
	if targetOperation.Output == "" && len(targetOperation.Inputs) == 0 {
		return nil
	}

	diagnostics := make([]Diagnostic, 0)
	targetInputs := fieldMap(targetOperation.Inputs)
	declaredInputs := parserFieldMap(declared.Inputs)
	for _, input := range declared.Inputs {
		targetInput, ok := targetInputs[input.Name]
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:           "NVA-TARGET-006",
				Target:         target,
				RequestingFile: requestingFile,
				ImportSource:   source,
				Message:        fmt.Sprintf("external operation %s.%s missing input %s requested by %s", source, declared.Name, input.Name, requestingFile),
			})
			continue
		}
		if input.Type.Text != "" && targetInput.Type != "" && input.Type.Text != targetInput.Type {
			diagnostics = append(diagnostics, Diagnostic{
				Code:           "NVA-TARGET-006",
				Target:         target,
				RequestingFile: requestingFile,
				ImportSource:   source,
				Message:        fmt.Sprintf("external operation %s.%s input %s type %s does not match declared %s", source, declared.Name, input.Name, targetInput.Type, input.Type.Text),
			})
		}
	}
	for _, input := range targetOperation.Inputs {
		if input.Optional {
			continue
		}
		if _, ok := declaredInputs[input.Name]; ok {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code:           "NVA-TARGET-006",
			Target:         target,
			RequestingFile: requestingFile,
			ImportSource:   source,
			Message:        fmt.Sprintf("external operation %s.%s requires undeclared input %s requested by %s", source, declared.Name, input.Name, requestingFile),
		})
	}
	if targetOperation.Output != "" && declared.Output.Text != "" && targetOperation.Output != declared.Output.Text {
		diagnostics = append(diagnostics, Diagnostic{
			Code:           "NVA-TARGET-006",
			Target:         target,
			RequestingFile: requestingFile,
			ImportSource:   source,
			Message:        fmt.Sprintf("external operation %s.%s output type %s does not match declared %s", source, declared.Name, targetOperation.Output, declared.Output.Text),
		})
	}
	return diagnostics
}

func selectImplementation(target string, families []string, implementations []Implementation) (Implementation, []string, bool) {
	ordered := sortImplementations(implementations)
	considered := implementationPaths(ordered)

	for _, implementation := range ordered {
		if implementation.Target == target {
			return implementation, considered, true
		}
	}
	for _, family := range families {
		for _, implementation := range ordered {
			if implementation.Family == family {
				return implementation, considered, true
			}
		}
	}
	for _, implementation := range ordered {
		if implementation.Common {
			return implementation, considered, true
		}
	}
	for _, implementation := range ordered {
		if implementation.Default {
			return implementation, considered, true
		}
	}
	return Implementation{}, considered, false
}

func auditPermissions(projectPermissions project.PermissionMap, targetPermissions security.PermissionMap, external []ResolvedExternalOperation) []Diagnostic {
	operationPermissions := make([]security.OperationPermission, 0, len(external))
	externalCalls := make([]security.ExternalCall, 0, len(external))
	for _, operation := range external {
		operationPermissions = append(operationPermissions, security.OperationPermission{
			CapabilitySource: operation.CapabilitySource,
			CapabilityName:   operation.CapabilityName,
			Operation:        operation.Operation,
			Requires:         clonePermissions(operation.Permissions),
		})
		externalCalls = append(externalCalls, security.ExternalCall{
			RequestingFile:   operation.RequestingFile,
			Lifecycle:        "lifecycle",
			CapabilitySource: operation.CapabilitySource,
			CapabilityName:   operation.CapabilityName,
			Operation:        operation.Operation,
		})
	}

	diagnostics := security.AuditPermissions(security.AuditInput{
		ProjectPermissions:   convertProjectPermissions(projectPermissions),
		TargetPermissions:    targetPermissions,
		OperationPermissions: operationPermissions,
		ExternalCalls:        externalCalls,
	})
	out := make([]Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, Diagnostic{
			Code:           diagnostic.Code,
			RequestingFile: diagnostic.RequestingFile,
			Message:        diagnostic.Message,
		})
	}
	return out
}

func buildSourceMap(sources []SourceFile) map[string]parser.File {
	out := make(map[string]parser.File, len(sources))
	for _, source := range sources {
		out[normalizePath(source.Path)] = source.File
	}
	return out
}

func moduleRefs(modules []string) []ModuleRef {
	refs := make([]ModuleRef, 0, len(modules))
	for _, module := range modules {
		refs = append(refs, ModuleRef{Path: module})
	}
	return refs
}

func modulePaths(modules []string) []string {
	out := make([]string, len(modules))
	copy(out, modules)
	return out
}

func externalCapabilityMap(capabilities []ExternalCapability) map[string]ExternalCapability {
	out := make(map[string]ExternalCapability, len(capabilities))
	for _, capability := range capabilities {
		out[capability.Source] = capability
	}
	return out
}

func externalOperationMap(operations []ExternalOperation) map[string]ExternalOperation {
	out := make(map[string]ExternalOperation, len(operations))
	for _, operation := range operations {
		out[operation.Name] = operation
	}
	return out
}

func fieldMap(fields []Field) map[string]Field {
	out := make(map[string]Field, len(fields))
	for _, field := range fields {
		out[field.Name] = field
	}
	return out
}

func parserFieldMap(fields []parser.FieldDecl) map[string]parser.FieldDecl {
	out := make(map[string]parser.FieldDecl, len(fields))
	for _, field := range fields {
		out[field.Name] = field
	}
	return out
}

func collectPermissions(external []ResolvedExternalOperation) []security.Permission {
	seen := make(map[security.Permission]bool)
	permissions := make([]security.Permission, 0)
	for _, operation := range external {
		for _, permission := range operation.Permissions {
			if seen[permission] {
				continue
			}
			seen[permission] = true
			permissions = append(permissions, permission)
		}
	}
	sort.Slice(permissions, func(i, j int) bool {
		return permissions[i] < permissions[j]
	})
	return permissions
}

func convertProjectPermissions(permissions project.PermissionMap) security.PermissionMap {
	out := make(security.PermissionMap, len(permissions))
	for permission, allowed := range permissions {
		if allowed {
			out[security.Permission(permission)] = true
		}
	}
	return out
}

func sortImplementations(implementations []Implementation) []Implementation {
	out := make([]Implementation, len(implementations))
	copy(out, implementations)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func implementationPaths(implementations []Implementation) []string {
	paths := make([]string, 0, len(implementations))
	for _, implementation := range implementations {
		if implementation.Path != "" {
			paths = append(paths, implementation.Path)
		}
	}
	return paths
}

func clonePermissions(permissions []security.Permission) []security.Permission {
	out := make([]security.Permission, len(permissions))
	copy(out, permissions)
	return out
}

func isLocalImport(source string) bool {
	return strings.HasPrefix(source, ".")
}

func normalizePath(filePath string) string {
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	return path.Clean(filePath)
}
