package style

import "testing"

func TestResolvedStylePaddingMatchesCSSBoxModel(t *testing.T) {
	resolved := NewResolvedStyle(map[string]string{"padding": "4 0 0 0"})
	box, ok := resolved.Padding()
	if !ok || box.Top != 4 || box.Left != 0 {
		t.Fatalf("box = %+v ok=%v", box, ok)
	}
	padding := AndroidPaddingArray(box)
	if padding[0] != 0 || padding[1] != 4 {
		t.Fatalf("android padding = %+v", padding)
	}
}

func TestResolvedStyleMarginMatchesCSSBoxModel(t *testing.T) {
	resolved := NewResolvedStyle(map[string]string{"margin": "8 12"})
	box, ok := resolved.Margin()
	if !ok || box.Top != 8 || box.Left != 12 {
		t.Fatalf("box = %+v ok=%v", box, ok)
	}
}

func TestResolvedStyleBorderShorthand(t *testing.T) {
	resolved := NewResolvedStyle(map[string]string{"border": "2 #93c5fd"})
	width, ok := resolved.BorderWidth()
	if !ok || width != 2 {
		t.Fatalf("width = %d ok=%v", width, ok)
	}
	literal, ok := resolved.BorderColorLiteral()
	if !ok || literal != "#93c5fd" {
		t.Fatalf("literal = %q ok=%v", literal, ok)
	}
}
