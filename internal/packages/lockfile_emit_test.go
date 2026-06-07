package packages

import (
	"strings"
	"testing"
)

func TestFormatLockfileRoundTrip(t *testing.T) {
	input := Lockfile{Entries: []LockEntry{{
		Name:        "@acme/charts",
		Version:     "1.0.0",
		Source:      "packages/acme/charts",
		ContentHash: "abc123",
		TargetAdapters: map[string]string{
			"web": "platform/web/register.web.js",
		},
	}}}
	formatted := FormatLockfile(input)
	parsed, diagnostics := ParseLockfile(formatted)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if len(parsed.Entries) != 1 {
		t.Fatalf("entries = %+v", parsed.Entries)
	}
	entry := parsed.Entries[0]
	if entry.Name != "@acme/charts" || entry.Version != "1.0.0" || entry.ContentHash != "abc123" {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.TargetAdapters["web"] != "platform/web/register.web.js" {
		t.Fatalf("adapters = %+v", entry.TargetAdapters)
	}
	if !strings.Contains(formatted, "content_hash = \"abc123\"") {
		t.Fatalf("formatted lock missing content hash:\n%s", formatted)
	}
}

func TestBuildLockfileUsesResolvedGraphManifests(t *testing.T) {
	manifests := []Manifest{{
		Name:        "@acme/charts",
		Version:     "1.0.0",
		ContentHash: "hash-1",
		Targets: map[string]TargetAdapter{
			"web": {Adapter: "platform/web/register.web.js", Content: "register"},
		},
	}}
	graph := ResolvedGraph{Packages: []ResolvedPackage{{
		Name:    "@acme/charts",
		Version: "1.0.0",
	}}}
	lockfile := BuildLockfile(graph, manifests, "/tmp/project")
	if len(lockfile.Entries) != 1 {
		t.Fatalf("entries = %+v", lockfile.Entries)
	}
	entry := lockfile.Entries[0]
	if entry.ContentHash != "hash-1" {
		t.Fatalf("content hash = %q", entry.ContentHash)
	}
	if entry.TargetAdapters["web"] != "platform/web/register.web.js" {
		t.Fatalf("adapters = %+v", entry.TargetAdapters)
	}
}
