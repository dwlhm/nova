package androidregistry

import (
	"github.com/dwlhm/nova/internal/core/contract"
	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	"github.com/dwlhm/nova/internal/provider/capability/view/hostctx"
	irandroid "github.com/dwlhm/nova/internal/provider/capability/view/ir/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

// NodeLowerFunc lowers one Nova view node into Android IR.
type NodeLowerFunc func(ctx hostctx.Context, node contract.Node, path []int) (irandroid.Node, []shared.Diagnostic)

// NodeEmitFunc renders one lowered node into Java source lines.
type NodeEmitFunc func(renderer *androidcodegen.Renderer, node irandroid.Node, parent, indent string, key, name string) string

// ArtifactEmitFunc emits standalone files owned by a shell/cross-cutting capability.
type ArtifactEmitFunc func(ctx hostctx.Context) ([]shared.File, []shared.Diagnostic)

type LowerRegistry struct {
	byKind map[string]NodeLowerFunc
}

type EmitRegistry struct {
	nodeByKind map[string]NodeEmitFunc
	artifacts  []artifactEmitter
}

type artifactEmitter struct {
	capabilityID string
	order        int
	emit         ArtifactEmitFunc
}

func NewLowerRegistry() *LowerRegistry {
	return &LowerRegistry{byKind: make(map[string]NodeLowerFunc)}
}

func NewEmitRegistry() *EmitRegistry {
	return &EmitRegistry{nodeByKind: make(map[string]NodeEmitFunc)}
}

func (registry *LowerRegistry) Register(kind string, lower NodeLowerFunc) {
	if kind == "" || lower == nil {
		return
	}
	registry.byKind[kind] = lower
}

func (registry *LowerRegistry) Lower(ctx hostctx.Context, node contract.Node, path []int) (irandroid.Node, bool, []shared.Diagnostic) {
	lower, ok := registry.byKind[node.Kind]
	if !ok || lower == nil {
		return irandroid.Node{}, false, nil
	}
	out, diagnostics := lower(ctx, node, path)
	return out, true, diagnostics
}

func (registry *EmitRegistry) RegisterNode(kind string, emit NodeEmitFunc) {
	if kind == "" || emit == nil {
		return
	}
	registry.nodeByKind[kind] = emit
}

func (registry *EmitRegistry) RegisterArtifact(capabilityID string, order int, emit ArtifactEmitFunc) {
	if capabilityID == "" || emit == nil {
		return
	}
	registry.artifacts = append(registry.artifacts, artifactEmitter{
		capabilityID: capabilityID,
		order:        order,
		emit:         emit,
	})
}

func (registry *EmitRegistry) LookupNode(kind string) (NodeEmitFunc, bool) {
	emit, ok := registry.nodeByKind[kind]
	return emit, ok
}

func (registry *EmitRegistry) NodeRegistry() *androidcodegen.NodeRegistry {
	out := androidcodegen.NewNodeRegistry()
	for kind, emit := range registry.nodeByKind {
		kind := kind
		emit := emit
		out.Register(kind, func(renderer *androidcodegen.Renderer, node contract.Node, parent, indent string, path []int, key, name string) string {
			lowered := irandroid.LowerNode(node, path)
			return emit(renderer, lowered, parent, indent, key, name)
		})
	}
	return out
}

func (registry *EmitRegistry) Artifacts(ctx hostctx.Context) ([]shared.File, []shared.Diagnostic) {
	ordered := append([]artifactEmitter(nil), registry.artifacts...)
	sortArtifactEmitters(ordered)
	out := make([]shared.File, 0)
	diagnostics := make([]shared.Diagnostic, 0)
	for _, entry := range ordered {
		files, entryDiagnostics := entry.emit(ctx)
		if len(entryDiagnostics) > 0 {
			diagnostics = append(diagnostics, entryDiagnostics...)
			continue
		}
		out = append(out, files...)
	}
	return out, diagnostics
}

func sortArtifactEmitters(items []artifactEmitter) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j-1].order > items[j].order; j-- {
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}
