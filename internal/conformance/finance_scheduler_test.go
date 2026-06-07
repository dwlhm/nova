package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dwlhm/nova/internal/packageio"
	"github.com/dwlhm/nova/internal/provider/build"
)

func TestFinanceSchedulerTraceMatchesFixture(t *testing.T) {
	root := filepath.Join("..", "..", "tests", "conformance", "web", "finance")
	spec, diagnostics := loadFixtureSpec(root)
	if len(diagnostics) > 0 {
		t.Fatalf("load spec: %+v", diagnostics)
	}
	if spec.Expected.Scheduler == nil {
		t.Fatal("finance fixture must declare scheduler expectations")
	}

	manifest, diagnostics := readFixtureManifest(root)
	if len(diagnostics) > 0 {
		t.Fatalf("manifest: %+v", diagnostics)
	}
	sources, diagnostics := readFixtureSources(root, manifest.Project.Entry)
	if len(diagnostics) > 0 {
		t.Fatalf("sources: %+v", diagnostics)
	}
	packageGraph, diagnostics := packageio.ResolveProjectGraph(root, "web", manifest, packageio.ResolveOptions{Production: false})
	if len(diagnostics) > 0 {
		t.Fatalf("packages: %+v", diagnostics)
	}
	resolution := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        sources,
		TargetManifest: mustTargetManifest(t, "web"),
		PackageGraph:   packageGraph,
	})
	if len(resolution.Diagnostics) > 0 {
		t.Fatalf("resolve: %+v", resolution.Diagnostics)
	}

	schedulerDiagnostics, _, ok := runSchedulerFixture(resolution.Plan, sources, manifest.Permissions, spec.Expected.Scheduler)
	if !ok || len(schedulerDiagnostics) > 0 {
		t.Fatalf("scheduler fixture: ok=%v diagnostics=%+v", ok, schedulerDiagnostics)
	}
}

func TestFinanceConformanceSourcesMatchExample(t *testing.T) {
	exampleRoot := filepath.Join("..", "..", "examples", "finance", "src")
	fixtureRoot := filepath.Join("..", "..", "tests", "conformance", "web", "finance", "src")
	for _, name := range []string{
		"App.nova",
		"FinancePure.nova",
		"FinanceStore.nova",
		"FinancePersistence.nova",
		"Finance.css",
		"Finance.nova-style",
	} {
		exampleBytes, err := os.ReadFile(filepath.Join(exampleRoot, name))
		if err != nil {
			t.Fatalf("read example %s: %v", name, err)
		}
		fixtureBytes, err := os.ReadFile(filepath.Join(fixtureRoot, name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		if string(exampleBytes) != string(fixtureBytes) {
			t.Fatalf("finance conformance source drift: %s differs from examples/finance", name)
		}
	}
}
