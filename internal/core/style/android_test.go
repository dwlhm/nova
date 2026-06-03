package style

import "testing"

func TestFilterAndroidStatesSkipsHover(t *testing.T) {
	states := []StateRule{{
		Class:      "primary-btn",
		Pseudo:     "hover",
		Properties: map[string]string{"color": "#000"},
	}, {
		Class:      "primary-btn",
		Pseudo:     "active",
		Properties: map[string]string{"background-color": "#0066cc"},
	}}
	filtered, diagnostics := FilterAndroidStates(states)
	if len(filtered) != 1 || filtered[0].Pseudo != "active" {
		t.Fatalf("filtered = %+v", filtered)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "NVA-STYLE-011" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestAndroidViewStateAttrsMapsActiveToPressed(t *testing.T) {
	attrs, ok := AndroidViewStateAttrs("active")
	if !ok || attrs != "android.R.attr.state_pressed" {
		t.Fatalf("attrs = %q ok = %v", attrs, ok)
	}
}
