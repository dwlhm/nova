package contract_test

import (
	"encoding/json"
	"testing"

	"github.com/dwlhm/nova/internal/core/contract"
)

func TestAppContractVersion(t *testing.T) {
	payload, err := json.Marshal(contract.App{V: contract.Version})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["v"] != float64(contract.Version) {
		t.Fatalf("v = %v, want %d", decoded["v"], contract.Version)
	}
}
