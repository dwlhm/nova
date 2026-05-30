package scheduler

import (
	"errors"
	"testing"
)

func TestEnqueueAssignsLogicalSequenceWithoutMutatingRuntime(t *testing.T) {
	runtime := NewRuntime(nil, nil)

	withFirst, first, err := Enqueue(runtime, "counter", "@increment", nil)
	if err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	withSecond, second, err := Enqueue(withFirst, "counter", "@decrement", nil)
	if err != nil {
		t.Fatalf("enqueue second: %v", err)
	}

	if len(runtime.Queue()) != 0 {
		t.Fatalf("original runtime queue mutated: %+v", runtime.Queue())
	}
	if first.Sequence != 1 || second.Sequence != 2 {
		t.Fatalf("sequences = %d, %d; want 1, 2", first.Sequence, second.Sequence)
	}
	assertQueueEvents(t, withSecond.Queue(), "@increment", "@decrement")
}

func TestEnqueueRejectsEventsWithoutSchedulerPrefix(t *testing.T) {
	_, _, err := Enqueue(NewRuntime(nil, nil), "counter", "increment", nil)
	if !errors.Is(err, ErrInvalidEventName) {
		t.Fatalf("error = %v, want ErrInvalidEventName", err)
	}
}

func TestAtomicCommitEvaluatesEveryTransitionFromTheSameSnapshot(t *testing.T) {
	runtime := NewRuntime([]StateCell{
		NewStateCell("pair", "left", "number", 1, On("@swap", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
			return snapshot.MustValue(Key("pair", "right")), nil
		})),
		NewStateCell("pair", "right", "number", 2, On("@swap", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
			return snapshot.MustValue(Key("pair", "left")), nil
		})),
	}, nil)
	runtime, _, err := Enqueue(runtime, "pair", "@swap", nil)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	next, result, ok := Step(runtime)
	if !ok {
		t.Fatal("expected a scheduler step")
	}

	assertState(t, next, "pair", "left", 2)
	assertState(t, next, "pair", "right", 1)
	if !result.Commit.Committed {
		t.Fatalf("commit was not marked committed: %+v", result.Commit)
	}
	assertInvalidations(t, result.Commit.Invalidations, Key("pair", "left"), Key("pair", "right"))
}

func TestBeforeAndAfterLifecycleUseCorrectSnapshotsAndEmitWithoutReentrancy(t *testing.T) {
	runtime := NewRuntime(
		[]StateCell{
			NewStateCell("counter", "count", "number", 0, On("@increment", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
				return snapshot.MustValue(Key("counter", "count")).(int) + 1, nil
			})),
		},
		[]LifecycleHandler{
			Before("counter", "@increment", func(ctx LifecycleContext) (LifecycleOutput, error) {
				return LifecycleOutput{
					Emit: []EventToEmit{
						Emit("counter", "@seen_before", ctx.Snapshot.MustValue(Key("counter", "count"))),
					},
				}, nil
			}),
			After("counter", "@increment", func(ctx LifecycleContext) (LifecycleOutput, error) {
				return LifecycleOutput{
					Emit: []EventToEmit{
						Emit("counter", "@seen_after", ctx.Snapshot.MustValue(Key("counter", "count"))),
					},
				}, nil
			}),
		},
	)
	runtime, _, err := Enqueue(runtime, "counter", "@increment", nil)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	next, result, ok := Step(runtime)
	if !ok {
		t.Fatal("expected a scheduler step")
	}

	assertState(t, next, "counter", "count", 1)
	assertQueueEvents(t, next.Queue(), "@seen_before", "@seen_after")
	if got := next.Queue()[0].Payload; got != 0 {
		t.Fatalf("before lifecycle saw payload %v, want pre-commit count 0", got)
	}
	if got := next.Queue()[1].Payload; got != 1 {
		t.Fatalf("after lifecycle saw payload %v, want committed count 1", got)
	}
	if len(result.Emitted) != 2 || result.Emitted[0].Sequence != 2 || result.Emitted[1].Sequence != 3 {
		t.Fatalf("emitted events = %+v, want sequences 2 and 3", result.Emitted)
	}
}

func TestTransitionErrorCancelsCommitAndRoutesToErrorLifecycle(t *testing.T) {
	boom := errors.New("cannot calculate next value")
	runtime := NewRuntime(
		[]StateCell{
			NewStateCell("auth", "status", "Status", "idle", On("@login_ok", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
				return "success", nil
			})),
			NewStateCell("auth", "user", "User|null", nil, On("@login_ok", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
				return nil, boom
			})),
		},
		[]LifecycleHandler{
			OnError("auth", func(ctx LifecycleContext) (LifecycleOutput, error) {
				if ctx.Error == nil {
					t.Fatal("error lifecycle received nil error")
				}
				return LifecycleOutput{Emit: []EventToEmit{
					Emit("auth", "@transition_failed", ctx.Error.Message),
				}}, nil
			}),
		},
	)
	runtime, _, err := Enqueue(runtime, "auth", "@login_ok", map[string]DataValue{"id": "u1"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	next, result, ok := Step(runtime)
	if !ok {
		t.Fatal("expected a scheduler step")
	}

	assertState(t, next, "auth", "status", "idle")
	assertState(t, next, "auth", "user", nil)
	if result.Commit.Committed {
		t.Fatalf("commit unexpectedly succeeded: %+v", result.Commit)
	}
	if len(result.Errors) != 1 || result.Errors[0].Phase != SchedulerPhaseTransition {
		t.Fatalf("errors = %+v, want one transition error", result.Errors)
	}
	assertQueueEvents(t, next.Queue(), "@transition_failed")
}

func TestTransitionTypeValidationCancelsWholeCommit(t *testing.T) {
	runtime := NewRuntime([]StateCell{
		NewStateCell("counter", "count", "number", 0, On("@bad", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
			return "not a number", nil
		})),
		NewStateCell("counter", "label", "string", "old", On("@bad", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
			return "new", nil
		})),
	}, nil)
	runtime, _, err := Enqueue(runtime, "counter", "@bad", nil)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	next, result, ok := Step(runtime)
	if !ok {
		t.Fatal("expected a scheduler step")
	}

	assertState(t, next, "counter", "count", 0)
	assertState(t, next, "counter", "label", "old")
	if result.Commit.Committed {
		t.Fatalf("commit unexpectedly succeeded: %+v", result.Commit)
	}
	if len(result.Errors) != 1 || !errors.Is(result.Errors[0].Cause, ErrInvalidStateValue) {
		t.Fatalf("errors = %+v, want invalid state value", result.Errors)
	}
}

func TestLifecycleErrorDoesNotRollbackCommittedState(t *testing.T) {
	afterErr := errors.New("storage write failed")
	runtime := NewRuntime(
		[]StateCell{
			NewStateCell("counter", "count", "number", 0, On("@increment", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
				return snapshot.MustValue(Key("counter", "count")).(int) + 1, nil
			})),
		},
		[]LifecycleHandler{
			After("counter", "@increment", func(ctx LifecycleContext) (LifecycleOutput, error) {
				return LifecycleOutput{}, afterErr
			}),
			OnError("counter", func(ctx LifecycleContext) (LifecycleOutput, error) {
				return LifecycleOutput{Emit: []EventToEmit{
					Emit("counter", "@persist_failed", ctx.Error.Message),
				}}, nil
			}),
		},
	)
	runtime, _, err := Enqueue(runtime, "counter", "@increment", nil)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	next, result, ok := Step(runtime)
	if !ok {
		t.Fatal("expected a scheduler step")
	}

	assertState(t, next, "counter", "count", 1)
	if !result.Commit.Committed {
		t.Fatalf("commit did not succeed before lifecycle error: %+v", result.Commit)
	}
	if len(result.Errors) != 1 || result.Errors[0].Phase != SchedulerPhaseLifecycleAfter {
		t.Fatalf("errors = %+v, want one after lifecycle error", result.Errors)
	}
	assertQueueEvents(t, next.Queue(), "@persist_failed")
}

func TestLifecycleOutputIsVisibleOnlyAfterSuccessfulHandler(t *testing.T) {
	runtime := NewRuntime(nil, []LifecycleHandler{
		After("counter", "@increment", func(ctx LifecycleContext) (LifecycleOutput, error) {
			return LifecycleOutput{Emit: []EventToEmit{
				Emit("counter", "@should_not_escape", nil),
			}}, errors.New("handler failed")
		}),
	})
	runtime, _, err := Enqueue(runtime, "counter", "@increment", nil)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	next, result, ok := Step(runtime)
	if !ok {
		t.Fatal("expected a scheduler step")
	}
	if len(next.Queue()) != 0 {
		t.Fatalf("failed lifecycle output should not be queued: %+v", next.Queue())
	}
	if len(result.Errors) != 1 || result.Errors[0].Phase != SchedulerPhaseLifecycleAfter {
		t.Fatalf("errors = %+v, want one after lifecycle error", result.Errors)
	}
}

func TestMountAndDisposeLifecycleCanEmitEvents(t *testing.T) {
	runtime := NewRuntime(nil, []LifecycleHandler{
		Mount("app", func(ctx LifecycleContext) (LifecycleOutput, error) {
			return LifecycleOutput{Emit: []EventToEmit{
				Emit("app", "@mounted", nil),
			}}, nil
		}),
		Dispose("app", func(ctx LifecycleContext) (LifecycleOutput, error) {
			return LifecycleOutput{Emit: []EventToEmit{
				Emit("app", "@disposed", nil),
			}}, nil
		}),
	})

	afterMount, mountResult := RunLifecycle(runtime, PhaseMount, "app")
	assertQueueEvents(t, afterMount.Queue(), "@mounted")
	if len(mountResult.Emitted) != 1 || mountResult.Emitted[0].Sequence != 1 {
		t.Fatalf("mount emitted = %+v, want sequence 1", mountResult.Emitted)
	}

	afterDispose, disposeResult := RunLifecycle(afterMount, PhaseDispose, "app")
	assertQueueEvents(t, afterDispose.Queue(), "@mounted", "@disposed")
	if len(disposeResult.Emitted) != 1 || disposeResult.Emitted[0].Sequence != 2 {
		t.Fatalf("dispose emitted = %+v, want sequence 2", disposeResult.Emitted)
	}
}

func TestLifecycleCanReturnExternalOperationRequestsWithoutExecutingAdapters(t *testing.T) {
	runtime := NewRuntime(
		[]StateCell{
			NewStateCell("counter", "count", "number", 1, On("@persist", func(snapshot Snapshot, event EventEnvelope) (DataValue, error) {
				return snapshot.MustValue(Key("counter", "count")), nil
			})),
		},
		[]LifecycleHandler{
			After("counter", "@persist", func(ctx LifecycleContext) (LifecycleOutput, error) {
				return LifecycleOutput{
					External: []ExternalOperationRequest{
						ExternalOperation(
							"counter",
							"storage",
							"set",
							map[string]DataValue{"key": "count", "value": ctx.Snapshot.MustValue(Key("counter", "count"))},
							"void",
							"@persisted",
							"@persist_failed",
						),
					},
				}, nil
			}),
		},
	)
	runtime, _, err := Enqueue(runtime, "counter", "@persist", nil)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	next, result, ok := Step(runtime)
	if !ok {
		t.Fatal("expected a scheduler step")
	}
	if len(result.ExternalOperations) != 1 {
		t.Fatalf("external operations = %+v, want one request", result.ExternalOperations)
	}
	if result.ExternalOperations[0].Capability != "storage" || result.ExternalOperations[0].Input["value"] != 1 {
		t.Fatalf("unexpected external request: %+v", result.ExternalOperations[0])
	}
	if len(next.Queue()) != 0 {
		t.Fatalf("external operation should not run inside scheduler core: %+v", next.Queue())
	}
}

func TestBackpressureRejectsEventsWithoutChangingQueue(t *testing.T) {
	runtime := NewRuntime(nil, nil, Config{MaxQueue: 1})
	runtime, _, err := Enqueue(runtime, "counter", "@one", nil)
	if err != nil {
		t.Fatalf("enqueue first: %v", err)
	}

	next, _, err := Enqueue(runtime, "counter", "@two", nil)
	if !errors.Is(err, ErrQueueOverloaded) {
		t.Fatalf("error = %v, want ErrQueueOverloaded", err)
	}
	assertQueueEvents(t, runtime.Queue(), "@one")
	assertQueueEvents(t, next.Queue(), "@one")
}

func assertState(t *testing.T, runtime Runtime, owner CapabilityRef, name StateName, want DataValue) {
	t.Helper()

	got, ok := runtime.State(Key(owner, name))
	if !ok {
		t.Fatalf("missing state %s.%s", owner, name)
	}
	if got != want {
		t.Fatalf("state %s.%s = %v, want %v", owner, name, got, want)
	}
}

func assertQueueEvents(t *testing.T, queue []EventEnvelope, want ...SchedulerEvent) {
	t.Helper()

	if len(queue) != len(want) {
		t.Fatalf("queue length = %d, want %d: %+v", len(queue), len(want), queue)
	}
	for i := range want {
		if queue[i].Name != want[i] {
			t.Fatalf("queue[%d] = %s, want %s; queue=%+v", i, queue[i].Name, want[i], queue)
		}
	}
}

func assertInvalidations(t *testing.T, got []StateKey, want ...StateKey) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("invalidations = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("invalidations = %+v, want %+v", got, want)
		}
	}
}
