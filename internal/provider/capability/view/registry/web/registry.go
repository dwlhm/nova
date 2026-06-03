package webregistry

import (
	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/provider/capability/view/hostctx"
	irweb "github.com/dwlhm/nova/internal/provider/capability/view/ir/web"
	"github.com/dwlhm/nova/internal/provider/shared"
)

type NodeLowerFunc func(ctx hostctx.Context, node contract.Node, path []int) (irweb.Node, []shared.Diagnostic)

type ArtifactEmitFunc func(ctx hostctx.Context) ([]shared.File, []shared.Diagnostic)

type LowerRegistry struct {
	byKind map[string]NodeLowerFunc
}

type EmitRegistry struct {
	artifacts []artifactEmitter
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
	return &EmitRegistry{}
}

func (registry *LowerRegistry) Register(kind string, lower NodeLowerFunc) {
	if kind == "" || lower == nil {
		return
	}
	registry.byKind[kind] = lower
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
