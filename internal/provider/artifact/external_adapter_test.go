package artifact

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/provider/build"
)

func TestWebExternalAdaptersPreferProjectOverrideContent(t *testing.T) {
	override := `export function register(NovaExternal) {
  NovaExternal.define("@env/storage", {
    async set() { return null; }
  });
}
// CUSTOM_STORAGE_ADAPTER_MARKER`
	bundle := webExternalAdapters([]build.ResolvedExternalOperation{
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
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available")
	}

	bundle := webExternalAdapters([]build.ResolvedExternalOperation{
		{CapabilitySource: "@env/storage", Operation: "set", Implementation: build.Implementation{Path: "platform/web/storage.web.js"}},
	}, nil)
	if !bundle.Enabled {
		t.Fatal("expected external adapter bundle")
	}

	script := "global.window = global;\n" + bundle.Content + `
(async () => {
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
