package conformance

import (
	"context"
	"errors"
	"fmt"

	"github.com/dwlhm/nova/internal/core/effect"
	"github.com/dwlhm/nova/internal/core/scheduler"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/build"
)

type externalStubMode string

const (
	externalStubSuccess       externalStubMode = "success"
	externalStubFailure       externalStubMode = "failure"
	externalStubInvalidOutput externalStubMode = "invalid_output"
)

type configurableExternalPort struct {
	mode externalStubMode
}

type conformanceExternalPort struct{}

func (conformanceExternalPort) Invoke(_ context.Context, request scheduler.ExternalOperationRequest) (scheduler.ExternalOperationResult, error) {
	switch request.Capability + "#" + request.Operation {
	case "storage#load", "storage#get":
		return scheduler.ExternalOperationResult{Output: "stub-value"}, nil
	case "storage#set", "storage#remove", "storage#clear":
		return scheduler.ExternalOperationResult{Output: nil}, nil
	default:
		return scheduler.ExternalOperationResult{Output: nil}, nil
	}
}

func (port configurableExternalPort) Invoke(ctx context.Context, request scheduler.ExternalOperationRequest) (scheduler.ExternalOperationResult, error) {
	switch port.mode {
	case externalStubFailure:
		return scheduler.ExternalOperationResult{}, errors.New("adapter offline")
	case externalStubInvalidOutput:
		if request.OutputType == "void" {
			return scheduler.ExternalOperationResult{Output: "not-void"}, nil
		}
		if request.OutputType == "number" {
			return scheduler.ExternalOperationResult{Output: "not-a-number"}, nil
		}
		return scheduler.ExternalOperationResult{Output: map[string]any{"bad": true}}, nil
	default:
		return conformanceExternalPort{}.Invoke(ctx, request)
	}
}

func collectExternalRequests(results []scheduler.StepResult) []scheduler.ExternalOperationRequest {
	requests := make([]scheduler.ExternalOperationRequest, 0)
	for _, result := range results {
		requests = append(requests, result.ExternalOperations...)
	}
	return requests
}

func drainSchedulerWithOptionalExternals(runtime scheduler.Runtime, plan build.BuildPlan, projectPermissions project.PermissionMap, runtimePermissions []security.Permission, completeExternals bool, stubMode externalStubMode) (scheduler.Runtime, Trace) {
	runtime, results := scheduler.Drain(runtime)
	trace := TraceSchedulerResults(results)
	if !completeExternals {
		return runtime, trace
	}
	requests := collectExternalRequests(results)
	if len(requests) == 0 {
		return runtime, trace
	}
	runtime = completeConformanceExternals(context.Background(), runtime, plan, grantedPermissions(projectPermissions, runtimePermissions), requests, stubMode)
	runtime, followUp := scheduler.Drain(runtime)
	return runtime, mergeTrace(trace, TraceSchedulerResults(followUp))
}

func grantedPermissions(projectPermissions project.PermissionMap, runtimePermissions []security.Permission) map[security.Permission]bool {
	if runtimePermissions != nil {
		return permissionSet(runtimePermissions)
	}
	out := make(map[security.Permission]bool)
	for permission, allowed := range projectPermissions {
		if allowed {
			out[security.Permission(permission)] = true
		}
	}
	return out
}

func completeConformanceExternals(ctx context.Context, runtime scheduler.Runtime, plan build.BuildPlan, granted map[security.Permission]bool, requests []scheduler.ExternalOperationRequest, stubMode externalStubMode) scheduler.Runtime {
	port := configurableExternalPort{mode: stubMode}
	for _, request := range requests {
		if missing := missingPermission(granted, externalPermissions(plan, request)); missing != "" {
			runtime = enqueueExternalFailure(runtime, request, "permission denied: "+missing)
			continue
		}
		next, _ := effect.CompleteExternalOperations(ctx, runtime, port, []scheduler.ExternalOperationRequest{request})
		runtime = next
	}
	return runtime
}

func enqueueExternalFailure(runtime scheduler.Runtime, request scheduler.ExternalOperationRequest, message string) scheduler.Runtime {
	if request.OnFailure == "" {
		return runtime
	}
	next, _, err := scheduler.Enqueue(runtime, request.Source, request.OnFailure, message)
	if err != nil {
		return runtime
	}
	return next
}

func permissionSet(permissions []security.Permission) map[security.Permission]bool {
	out := make(map[security.Permission]bool, len(permissions))
	for _, permission := range permissions {
		out[permission] = true
	}
	return out
}

func externalPermissions(plan build.BuildPlan, request scheduler.ExternalOperationRequest) []security.Permission {
	for _, operation := range plan.ExternalOperations {
		if operation.CapabilityName == request.Capability && operation.Operation == request.Operation {
			return operation.Permissions
		}
	}
	return nil
}

func missingPermission(granted map[security.Permission]bool, required []security.Permission) string {
	for _, permission := range required {
		if !granted[permission] {
			return string(permission)
		}
	}
	return ""
}

func parseExternalStubMode(value string) (externalStubMode, error) {
	switch externalStubMode(value) {
	case "", externalStubSuccess:
		return externalStubSuccess, nil
	case externalStubFailure, externalStubInvalidOutput:
		return externalStubMode(value), nil
	default:
		return "", fmt.Errorf("unsupported external stub mode %q", value)
	}
}
