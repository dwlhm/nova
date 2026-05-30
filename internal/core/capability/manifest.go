package capability

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/parser"
)

const UnknownEventArity = -1

type CapabilityRef string

type SymbolRef struct {
	Module CapabilityRef
	Name   string
}

func Symbol(module CapabilityRef, name string) SymbolRef {
	return SymbolRef{Module: module, Name: name}
}

type ImportKind string

const (
	ImportCapability ImportKind = "capability"
	ImportState      ImportKind = "state"
	ImportEvent      ImportKind = "event"
	ImportExternal   ImportKind = "external"
)

type Manifest struct {
	Ref                CapabilityRef
	Imports            []ImportRef
	States             []StateRef
	Events             []EventRef
	Types              []TypeRef
	Funcs              []FuncRef
	Templates          []TemplateRef
	Lifecycles         []LifecycleRef
	ExternalOperations []ExternalOperationRef
	TargetConstraints  []TargetConstraint
}

type ImportRef struct {
	Kind          ImportKind
	Module        CapabilityRef
	Name          string
	Alias         string
	EffectiveName string
}

type StateRef struct {
	Ref      SymbolRef
	Contract string
	Type     string
}

type EventSource string

const (
	EventSourceTransition EventSource = "transition"
	EventSourceCapability EventSource = "capability"
)

type EventRef struct {
	Ref    SymbolRef
	Source EventSource
	Arity  int
}

type TypeRef struct {
	Ref    SymbolRef
	Opaque bool
	Alias  string
}

type FuncRef struct {
	Ref SymbolRef
}

type TemplateRef struct {
	Ref    SymbolRef
	Target string
}

type LifecycleRef struct {
	Owner CapabilityRef
	Phase string
	Event string
}

type ExternalOperationRef struct {
	Ref        SymbolRef
	Capability string
	Operation  string
	Inputs     []FieldRef
	Output     string
}

type FieldRef struct {
	Name     string
	Optional bool
	Type     string
}

type TargetConstraint struct {
	Target string
}

type ModuleGraph map[CapabilityRef][]CapabilityRef

type GraphDiagnostic struct {
	Message string
	Cycle   []CapabilityRef
}

func BuildManifest(module string, file parser.File) Manifest {
	ref := CapabilityRef(module)
	manifest := Manifest{Ref: ref}

	manifest.Imports = append(manifest.Imports, buildNormalImports(file.Imports)...)
	manifest.Imports = append(manifest.Imports, buildExternalImports(file.ExternalImports)...)
	manifest.Types = buildTypeRefs(ref, file.ContractTypes)
	manifest.States = buildStateRefs(ref, file.ContractStates)
	manifest.Events = buildEventRefs(ref, file)
	manifest.Funcs = buildFuncRefs(ref, file.Funcs)
	manifest.Templates = buildTemplateRefs(ref, file.Templates)
	manifest.Lifecycles = buildLifecycleRefs(ref, file.Lifecycles)
	manifest.ExternalOperations = buildExternalOperationRefs(file.ExternalImports)
	manifest.TargetConstraints = buildTargetConstraints(file.Templates)

	return manifest
}

func EffectiveImportName(item parser.ImportItem) string {
	if item.Alias != "" {
		return item.Alias
	}
	return item.Name
}

func BuildModuleGraph(manifests []Manifest) ModuleGraph {
	graph := make(ModuleGraph, len(manifests))
	for _, manifest := range manifests {
		graph[manifest.Ref] = importModules(manifest.Imports)
	}
	return cloneModuleGraph(graph)
}

func ValidateAcyclicModuleGraph(manifests []Manifest) []GraphDiagnostic {
	cycles := FindModuleGraphCycles(BuildModuleGraph(manifests))
	diagnostics := make([]GraphDiagnostic, 0, len(cycles))
	for _, cycle := range cycles {
		diagnostics = append(diagnostics, GraphDiagnostic{
			Message: fmt.Sprintf("runtime module graph contains cycle %s", joinCycle(cycle)),
			Cycle:   cloneRefs(cycle),
		})
	}
	return diagnostics
}

func FindModuleGraphCycles(graph ModuleGraph) [][]CapabilityRef {
	state := make(map[CapabilityRef]int, len(graph))
	stack := make([]CapabilityRef, 0, len(graph))
	stackIndex := make(map[CapabilityRef]int, len(graph))
	seen := make(map[string]bool)
	cycles := make([][]CapabilityRef, 0)

	var visit func(CapabilityRef)
	visit = func(node CapabilityRef) {
		state[node] = 1
		stackIndex[node] = len(stack)
		stack = append(stack, node)

		for _, dep := range graph[node] {
			if _, ok := graph[dep]; !ok {
				continue
			}
			switch state[dep] {
			case 0:
				visit(dep)
			case 1:
				cycle := append(cloneRefs(stack[stackIndex[dep]:]), dep)
				key := canonicalCycleKey(cycle)
				if !seen[key] {
					seen[key] = true
					cycles = append(cycles, cycle)
				}
			}
		}

		stack = stack[:len(stack)-1]
		delete(stackIndex, node)
		state[node] = 2
	}

	for _, node := range moduleGraphNodes(graph) {
		if state[node] == 0 {
			visit(node)
		}
	}
	return cycles
}

func buildNormalImports(decls []parser.ImportDecl) []ImportRef {
	imports := make([]ImportRef, 0)
	for _, decl := range decls {
		for _, item := range decl.Items {
			imports = append(imports, ImportRef{
				Kind:          importKind(decl.Kind),
				Module:        CapabilityRef(decl.From),
				Name:          item.Name,
				Alias:         item.Alias,
				EffectiveName: EffectiveImportName(item),
			})
		}
	}
	return imports
}

func buildExternalImports(decls []parser.ExternalImportDecl) []ImportRef {
	imports := make([]ImportRef, 0, len(decls))
	for _, decl := range decls {
		imports = append(imports, ImportRef{
			Kind:          ImportExternal,
			Module:        CapabilityRef(decl.From),
			Name:          decl.Name,
			EffectiveName: decl.Name,
		})
	}
	return imports
}

func buildTypeRefs(module CapabilityRef, decls []parser.ContractTypeDecl) []TypeRef {
	refs := make([]TypeRef, 0, len(decls))
	for _, decl := range decls {
		refs = append(refs, TypeRef{
			Ref:    Symbol(module, decl.Name),
			Opaque: decl.Opaque,
			Alias:  decl.Alias.Text,
		})
	}
	return refs
}

func buildStateRefs(module CapabilityRef, decls []parser.ContractStateDecl) []StateRef {
	refs := make([]StateRef, 0)
	for _, contract := range decls {
		for _, state := range contract.States {
			refs = append(refs, StateRef{
				Ref:      Symbol(module, state.Name),
				Contract: contract.Name,
				Type:     state.Type.Text,
			})
		}
	}
	return refs
}

func buildEventRefs(module CapabilityRef, file parser.File) []EventRef {
	refs := make([]EventRef, 0)
	for _, contract := range file.ContractStates {
		for _, state := range contract.States {
			for _, transition := range state.Transitions {
				refs = append(refs, EventRef{
					Ref:    Symbol(module, transition.Event.Name),
					Source: EventSourceTransition,
					Arity:  len(transition.Event.Params),
				})
			}
		}
	}
	for _, contract := range file.ContractCapabilities {
		for _, emit := range contract.Emits {
			refs = append(refs, EventRef{
				Ref:    Symbol(module, emit.Event),
				Source: EventSourceCapability,
				Arity:  emitArity(emit),
			})
		}
	}
	return refs
}

func buildFuncRefs(module CapabilityRef, decls []parser.FuncDecl) []FuncRef {
	refs := make([]FuncRef, 0, len(decls))
	for _, decl := range decls {
		refs = append(refs, FuncRef{Ref: Symbol(module, decl.Name)})
	}
	return refs
}

func buildTemplateRefs(module CapabilityRef, decls []parser.TemplateDecl) []TemplateRef {
	refs := make([]TemplateRef, 0, len(decls))
	for _, decl := range decls {
		refs = append(refs, TemplateRef{
			Ref:    Symbol(module, decl.Target),
			Target: decl.Target,
		})
	}
	return refs
}

func buildLifecycleRefs(module CapabilityRef, decls []parser.LifecycleDecl) []LifecycleRef {
	refs := make([]LifecycleRef, 0, len(decls))
	for _, decl := range decls {
		refs = append(refs, LifecycleRef{
			Owner: module,
			Phase: decl.Phase,
			Event: decl.Event,
		})
	}
	return refs
}

func buildExternalOperationRefs(decls []parser.ExternalImportDecl) []ExternalOperationRef {
	refs := make([]ExternalOperationRef, 0)
	for _, external := range decls {
		for _, operation := range external.Operations {
			refs = append(refs, ExternalOperationRef{
				Ref:        Symbol(CapabilityRef(external.From), operation.Name),
				Capability: external.Name,
				Operation:  operation.Name,
				Inputs:     buildFieldRefs(operation.Inputs),
				Output:     operation.Output.Text,
			})
		}
	}
	return refs
}

func buildFieldRefs(fields []parser.FieldDecl) []FieldRef {
	refs := make([]FieldRef, 0, len(fields))
	for _, field := range fields {
		refs = append(refs, FieldRef{
			Name:     field.Name,
			Optional: field.Optional,
			Type:     field.Type.Text,
		})
	}
	return refs
}

func buildTargetConstraints(decls []parser.TemplateDecl) []TargetConstraint {
	constraints := make([]TargetConstraint, 0, len(decls))
	for _, decl := range decls {
		if decl.Target != "" {
			constraints = append(constraints, TargetConstraint{Target: decl.Target})
		}
	}
	return constraints
}

func importKind(kind parser.ImportKind) ImportKind {
	switch kind {
	case parser.ImportState:
		return ImportState
	case parser.ImportEvent:
		return ImportEvent
	default:
		return ImportCapability
	}
}

func emitArity(emit parser.EmitDecl) int {
	switch emit.Type.Text {
	case "void":
		return 0
	case "":
		return UnknownEventArity
	default:
		return 1
	}
}

func importModules(imports []ImportRef) []CapabilityRef {
	seen := make(map[CapabilityRef]bool)
	modules := make([]CapabilityRef, 0, len(imports))
	for _, imp := range imports {
		if imp.Module == "" || seen[imp.Module] {
			continue
		}
		seen[imp.Module] = true
		modules = append(modules, imp.Module)
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i] < modules[j]
	})
	return modules
}

func cloneModuleGraph(graph ModuleGraph) ModuleGraph {
	out := make(ModuleGraph, len(graph))
	for node, deps := range graph {
		out[node] = cloneRefs(deps)
	}
	return out
}

func cloneRefs(refs []CapabilityRef) []CapabilityRef {
	out := make([]CapabilityRef, len(refs))
	copy(out, refs)
	return out
}

func moduleGraphNodes(graph ModuleGraph) []CapabilityRef {
	nodes := make([]CapabilityRef, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i] < nodes[j]
	})
	return nodes
}

func canonicalCycleKey(cycle []CapabilityRef) string {
	if len(cycle) <= 1 {
		return joinCycle(cycle)
	}
	body := cycle
	if cycle[0] == cycle[len(cycle)-1] {
		body = cycle[:len(cycle)-1]
	}
	if len(body) == 0 {
		return ""
	}

	min := 0
	for i := 1; i < len(body); i++ {
		if body[i] < body[min] {
			min = i
		}
	}

	rotated := make([]CapabilityRef, 0, len(body)+1)
	for i := 0; i < len(body); i++ {
		rotated = append(rotated, body[(min+i)%len(body)])
	}
	rotated = append(rotated, rotated[0])
	return joinCycle(rotated)
}

func joinCycle(cycle []CapabilityRef) string {
	parts := make([]string, 0, len(cycle))
	for _, ref := range cycle {
		parts = append(parts, string(ref))
	}
	return strings.Join(parts, " -> ")
}
