package scheduler

import "errors"

type LogicalSequence uint64
type CapabilityRef string
type SchedulerEvent string
type StateName string
type DataValue any

type LifecyclePhase string

const (
	PhaseMount   LifecyclePhase = "mount"
	PhaseDispose LifecyclePhase = "dispose"
	PhaseBefore  LifecyclePhase = "before"
	PhaseAfter   LifecyclePhase = "after"
	PhaseError   LifecyclePhase = "error"
)

type SchedulerPhase string

const (
	SchedulerPhaseEnqueue           SchedulerPhase = "enqueue"
	SchedulerPhaseTransition        SchedulerPhase = "transition"
	SchedulerPhaseLifecycleMount    SchedulerPhase = "lifecycle_mount"
	SchedulerPhaseLifecycleDispose  SchedulerPhase = "lifecycle_dispose"
	SchedulerPhaseLifecycleBefore   SchedulerPhase = "lifecycle_before"
	SchedulerPhaseLifecycleAfter    SchedulerPhase = "lifecycle_after"
	SchedulerPhaseLifecycleError    SchedulerPhase = "lifecycle_error"
	SchedulerPhaseExternalOperation SchedulerPhase = "external_operation"
)

var (
	ErrInvalidEventName  = errors.New("scheduler event name must start with @")
	ErrInvalidPayload    = errors.New("event payload must be a serializable Nova data value")
	ErrInvalidStateValue = errors.New("state transition produced value outside state type")
	ErrQueueOverloaded   = errors.New("scheduler queue overloaded")
)
