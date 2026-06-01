package persistence

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/scheduler"
)

func TestRestoreSnapshotValidatesStateAndContinuesLogicalSequence(t *testing.T) {
	key := scheduler.Key("Counter", "count")
	cells := []scheduler.StateCell{
		scheduler.NewStateCell("Counter", "count", "number", 0, scheduler.On("@increment", func(snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
			return snapshot.MustValue(key).(int) + 1, nil
		})),
	}

	runtime := scheduler.NewRuntime(cells, nil)
	runtime, _, _ = scheduler.Enqueue(runtime, "host", "@increment", nil)
	runtime, _, _ = scheduler.Step(runtime)

	snapshot := Capture(runtime, CaptureOptions{
		ABIVersion:    "0.1.0",
		Target:        "web",
		AppInstanceID: "app-1",
		Metadata: SnapshotMetadata{
			Source:           SnapshotSourceRuntime,
			LanguageVersion:  "0.1.0",
			SchedulerVersion: "0.1.0",
			ViewIRVersion:    "0.1.0",
		},
	})

	restored, diagnostics := Restore(cells, nil, snapshot, RestorePolicy{
		ABIVersion:       "0.1.0",
		Target:           "web",
		LanguageVersion:  "0.1.0",
		SchedulerVersion: "0.1.0",
		ViewIRVersion:    "0.1.0",
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if value, _ := restored.State(key); value != 1 {
		t.Fatalf("restored count = %v, want 1", value)
	}

	restored, event, err := scheduler.Enqueue(restored, "host", "@increment", nil)
	if err != nil {
		t.Fatalf("enqueue after restore: %v", err)
	}
	if event.Sequence != 2 {
		t.Fatalf("restored event sequence = %d, want 2", event.Sequence)
	}
	restored, _, _ = scheduler.Step(restored)
	if value, _ := restored.State(key); value != 2 {
		t.Fatalf("count after restored increment = %v, want 2", value)
	}
}

func TestRestoreRejectsInvalidSnapshotValuesAndUnknownCells(t *testing.T) {
	cells := []scheduler.StateCell{scheduler.NewStateCell("Counter", "count", "number", 0)}
	snapshot := HydrationSnapshot{
		ABIVersion:    "0.1.0",
		Target:        "web",
		AppInstanceID: "app-1",
		Sequence:      4,
		Metadata: SnapshotMetadata{
			Source:           SnapshotSourceStorage,
			LanguageVersion:  "0.1.0",
			SchedulerVersion: "0.1.0",
			ViewIRVersion:    "0.1.0",
		},
		StateCells: []StateCellSnapshot{
			{Owner: "Counter", Name: "count", Type: "number", Value: make(chan int)},
			{Owner: "Missing", Name: "ghost", Type: "unknown", Value: "stale"},
		},
	}

	_, diagnostics := Restore(cells, nil, snapshot, RestorePolicy{
		ABIVersion:       "0.1.0",
		Target:           "web",
		LanguageVersion:  "0.1.0",
		SchedulerVersion: "0.1.0",
		ViewIRVersion:    "0.1.0",
	})

	assertPersistenceDiagnostic(t, diagnostics, "NVA-RUNTIME-019")
	assertPersistenceDiagnostic(t, diagnostics, "NVA-RUNTIME-020")
}

func assertPersistenceDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Code == want || strings.Contains(diagnostic.Message, want) || strings.Contains(diagnostic.Message, "snapshot") {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %+v", want, diagnostics)
}
