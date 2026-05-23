package scheduler

import "reflect"

func planAndCommit(cells []StateCell, snapshot Snapshot, event EventEnvelope, validate StateValueValidator) ([]StateCell, StateCommit, []SchedulerError) {
	if validate == nil {
		validate = DefaultStateValueValidator
	}
	commit := StateCommit{
		Sequence:  event.Sequence,
		Event:     event.Name,
		Committed: true,
	}
	type candidate struct {
		cell  StateCell
		key   StateKey
		value DataValue
	}
	candidates := make([]candidate, 0)
	errs := make([]SchedulerError, 0)

	for _, cell := range cells {
		for _, transition := range cell.Transitions {
			if transition.Event != event.Name {
				continue
			}
			if transition.Apply == nil {
				continue
			}
			value, err := transition.Apply(snapshot, event)
			if err != nil {
				key := cellKey(cell)
				errs = append(errs, SchedulerError{
					Source:  cell.Owner,
					Phase:   SchedulerPhaseTransition,
					Event:   event.Name,
					State:   &key,
					Message: err.Error(),
					Cause:   err,
				})
				continue
			}
			candidates = append(candidates, candidate{cell: cell, key: cellKey(cell), value: value})
		}
	}

	if len(errs) > 0 {
		commit.Committed = false
		return copyCells(cells), commit, errs
	}
	for _, candidate := range candidates {
		if err := validate(candidate.cell.Type, candidate.value); err != nil {
			key := candidate.key
			errs = append(errs, SchedulerError{
				Source:  candidate.cell.Owner,
				Phase:   SchedulerPhaseTransition,
				Event:   event.Name,
				State:   &key,
				Message: err.Error(),
				Cause:   err,
			})
		}
	}
	if len(errs) > 0 {
		commit.Committed = false
		return copyCells(cells), commit, errs
	}

	next := copyCells(cells)
	for _, candidate := range candidates {
		for i, cell := range next {
			if cellKey(cell) != candidate.key {
				continue
			}
			before := cell.Value
			if reflect.DeepEqual(before, candidate.value) {
				continue
			}
			next[i].Value = candidate.value
			commit.Changes = append(commit.Changes, StateChange{
				Key:    candidate.key,
				Before: before,
				After:  candidate.value,
			})
			commit.Invalidations = append(commit.Invalidations, candidate.key)
			break
		}
	}

	return next, commit, nil
}
