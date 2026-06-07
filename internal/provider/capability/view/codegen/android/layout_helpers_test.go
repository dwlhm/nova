package androidcodegen

import (
	"strings"
	"testing"
)

func TestAndroidJavaBlockContainerKind(t *testing.T) {
	for _, kind := range []string{"surface", "page", "column", "row", "scroll"} {
		if !androidJavaBlockContainerKind(kind) {
			t.Fatalf("kind %q should fill horizontal", kind)
		}
	}
	if androidJavaBlockContainerKind("button") {
		t.Fatalf("button should not be treated as block container")
	}
}

func TestAndroidJavaFillHorizontalPreservesMargins(t *testing.T) {
	code := androidJavaFillHorizontal("        ", "node_0")
	for _, part := range []string{
		"params.width = ViewGroup.LayoutParams.MATCH_PARENT",
		"setLayoutParams(params)",
		"ViewGroup.MarginLayoutParams",
	} {
		if !strings.Contains(code, part) {
			t.Fatalf("code = %q, missing %q", code, part)
		}
	}
}
