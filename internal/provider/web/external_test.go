package web

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/provider/build"
)

func requireNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("node is required in CI: %v", err)
		}
		t.Skip("node not available")
	}
}

func TestWebExternalAdaptersPreferProjectOverrideContent(t *testing.T) {
	override := `export function register(NovaExternal) {
  NovaExternal.define("@env/storage", {
    async set() { return null; }
  });
}
// CUSTOM_STORAGE_ADAPTER_MARKER`
	bundle := externalAdapters([]build.ResolvedExternalOperation{
		{CapabilitySource: "@env/storage", Operation: "set", Implementation: build.Implementation{Path: "platform/web/storage.web.js"}},
	}, map[string]string{"platform/web/storage.web.js": override})
	if !bundle.Enabled {
		t.Fatal("expected external adapter bundle")
	}
	if !strings.Contains(bundle.Content, "CUSTOM_STORAGE_ADAPTER_MARKER") {
		t.Fatalf("bundle should include project override content")
	}
}

func TestNovaExternalInvokeContract(t *testing.T) {
	requireNode(t)

	bundle := externalAdapters([]build.ResolvedExternalOperation{
		{CapabilitySource: "@env/storage", Operation: "set", Implementation: build.Implementation{Path: "platform/web/storage.web.js"}},
	}, nil)
	if !bundle.Enabled {
		t.Fatal("expected external adapter bundle")
	}

	script := "global.window = global;\n" + bundle.Content + `
(async () => {
  NovaExternal.checkPermission({ permissions: ["storage.write"], externalOperations: [{ id: "@env/storage#set", permissions: ["storage.write"], output: "void" }] }, "@env/storage#set");
  try {
    NovaExternal.checkPermission({ permissions: [], externalOperations: [{ id: "@env/storage#set", permissions: ["storage.write"], output: "void" }] }, "@env/storage#set");
    throw new Error("expected permission denial");
  } catch (error) {
    if (!String(error.message).includes("permission denied")) throw error;
  }
  NovaExternal.define("@test/mock", {
    async ok() { return null; },
    async fail() { throw new Error("boom"); },
    async bad() { return Symbol("x"); }
  });
  const ok = await NovaExternal.invoke("@test/mock#ok", {});
  if (ok !== null) throw new Error("expected null success output");
  try {
    await NovaExternal.invoke("@test/mock#fail", {});
    throw new Error("expected adapter failure");
  } catch (error) {
    if (!String(error.message).includes("boom")) throw error;
  }
  try {
    await NovaExternal.invoke("@test/mock#bad", {});
    throw new Error("expected non-serializable rejection");
  } catch (error) {
    if (!String(error.message).includes("non-serializable")) throw error;
  }
  try {
    await NovaExternal.invoke("@missing/source#op", {});
    throw new Error("expected missing adapter failure");
  } catch (error) {
    if (!String(error.message).includes("missing external adapter")) throw error;
  }
})();
`

	cmd := exec.Command("node", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node contract test failed: %v\n%s", err, output)
	}
}
