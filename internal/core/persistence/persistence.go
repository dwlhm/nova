package persistence

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/scheduler"
)

type SnapshotSource string

const (
	SnapshotSourceRuntime           SnapshotSource = "runtime"
	SnapshotSourceSSR               SnapshotSource = "ssr"
	SnapshotSourceStorage           SnapshotSource = "storage"
	SnapshotSourceAndroidSavedState SnapshotSource = "android_saved_state"
	SnapshotSourceDevtools          SnapshotSource = "devtools"
)

type HydrationSnapshot struct {
	ABIVersion    string
	Target        string
	AppInstanceID string
	Sequence      scheduler.LogicalSequence
	StateCells    []StateCellSnapshot
	Route         map[string]scheduler.DataValue
	Metadata      SnapshotMetadata
}

type StateCellSnapshot struct {
	Owner   scheduler.CapabilityRef
	Name    scheduler.StateName
	Type    string
	Value   scheduler.DataValue
	Version string
}

type SnapshotMetadata struct {
	CreatedAt        string
	Source           SnapshotSource
	LanguageVersion  string
	SchedulerVersion string
	ViewIRVersion    string
	ProjectVersion   string
	Redactions       []string
}

type CaptureOptions struct {
	ABIVersion    string
	Target        string
	AppInstanceID string
	Route         map[string]scheduler.DataValue
	Metadata      SnapshotMetadata
	IncludeState  func(scheduler.StateKey) bool
}

type RestorePolicy struct {
	ABIVersion               string
	Target                   string
	LanguageVersion          string
	SchedulerVersion         string
	ViewIRVersion            string
	AllowPartialStateRestore bool
}

type Diagnostic = diagnostic.Diagnostic

func Capture(runtime scheduler.Runtime, options CaptureOptions) HydrationSnapshot {
	cells := runtime.Cells()
	stateCells := make([]StateCellSnapshot, 0, len(cells))
	for _, cell := range cells {
		key := scheduler.Key(cell.Owner, cell.Name)
		if options.IncludeState != nil && !options.IncludeState(key) {
			continue
		}
		stateCells = append(stateCells, StateCellSnapshot{
			Owner: cell.Owner,
			Name:  cell.Name,
			Type:  cell.Type,
			Value: cell.Value,
		})
	}
	return HydrationSnapshot{
		ABIVersion:    options.ABIVersion,
		Target:        options.Target,
		AppInstanceID: options.AppInstanceID,
		Sequence:      runtime.LastSequence(),
		StateCells:    stateCells,
		Route:         cloneDataMap(options.Route),
		Metadata:      options.Metadata,
	}
}

func Restore(cells []scheduler.StateCell, lifecycles []scheduler.LifecycleHandler, snapshot HydrationSnapshot, policy RestorePolicy, configs ...scheduler.Config) (scheduler.Runtime, []Diagnostic) {
	diagnostics := validateSnapshotEnvelope(snapshot, policy)
	schema := stateSchema(cells)
	values := make(map[scheduler.StateKey]scheduler.DataValue, len(snapshot.StateCells))

	for _, cell := range snapshot.StateCells {
		key := scheduler.Key(cell.Owner, cell.Name)
		schemaCell, ok := schema[key]
		if !ok {
			diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-020", fmt.Sprintf("snapshot state %s.%s is not present in scheduler schema", cell.Owner, cell.Name)))
			continue
		}
		if cell.Type != "" && schemaCell.Type != "" && cell.Type != schemaCell.Type {
			diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-021", fmt.Sprintf("snapshot state %s.%s type %s does not match scheduler type %s", cell.Owner, cell.Name, cell.Type, schemaCell.Type)))
			continue
		}
		if err := scheduler.DefaultStateValueValidator(schemaCell.Type, cell.Value); err != nil {
			diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-019", fmt.Sprintf("snapshot state %s.%s value is invalid: %s", cell.Owner, cell.Name, err.Error())))
			continue
		}
		values[key] = cell.Value
	}

	if !policy.AllowPartialStateRestore {
		for key := range schema {
			if _, ok := values[key]; ok {
				continue
			}
			diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-022", fmt.Sprintf("snapshot is missing required state %s.%s", key.Owner, key.Name)))
		}
	}

	if len(diagnostics) > 0 {
		return scheduler.NewRuntime(cells, lifecycles, configs...), diagnostics
	}

	nextSequence := snapshot.Sequence + 1
	if nextSequence < 1 {
		nextSequence = 1
	}
	runtime, errors := scheduler.NewRuntimeFromState(cells, lifecycles, values, nextSequence, configs...)
	for _, err := range errors {
		diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-023", err.Message))
	}
	return runtime, diagnostics
}

func validateSnapshotEnvelope(snapshot HydrationSnapshot, policy RestorePolicy) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if policy.ABIVersion != "" && snapshot.ABIVersion != policy.ABIVersion {
		diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-024", fmt.Sprintf("snapshot ABI %s does not match runtime ABI %s", snapshot.ABIVersion, policy.ABIVersion)))
	}
	if policy.Target != "" && snapshot.Target != policy.Target {
		diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-025", fmt.Sprintf("snapshot target %s does not match runtime target %s", snapshot.Target, policy.Target)))
	}
	if policy.LanguageVersion != "" && snapshot.Metadata.LanguageVersion != policy.LanguageVersion {
		diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-026", fmt.Sprintf("snapshot language version %s does not match runtime language version %s", snapshot.Metadata.LanguageVersion, policy.LanguageVersion)))
	}
	if policy.SchedulerVersion != "" && snapshot.Metadata.SchedulerVersion != policy.SchedulerVersion {
		diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-027", fmt.Sprintf("snapshot scheduler version %s does not match runtime scheduler version %s", snapshot.Metadata.SchedulerVersion, policy.SchedulerVersion)))
	}
	if policy.ViewIRVersion != "" && snapshot.Metadata.ViewIRVersion != policy.ViewIRVersion {
		diagnostics = append(diagnostics, runtimeDiagnostic("NVA-RUNTIME-028", fmt.Sprintf("snapshot ViewIR version %s does not match runtime ViewIR version %s", snapshot.Metadata.ViewIRVersion, policy.ViewIRVersion)))
	}
	return diagnostics
}

func stateSchema(cells []scheduler.StateCell) map[scheduler.StateKey]scheduler.StateCell {
	out := make(map[scheduler.StateKey]scheduler.StateCell, len(cells))
	for _, cell := range cells {
		out[scheduler.Key(cell.Owner, cell.Name)] = cell
	}
	return out
}

func runtimeDiagnostic(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: diagnostic.SeverityError, Message: message}
}

func cloneDataMap(input map[string]scheduler.DataValue) map[string]scheduler.DataValue {
	if input == nil {
		return nil
	}
	out := make(map[string]scheduler.DataValue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
