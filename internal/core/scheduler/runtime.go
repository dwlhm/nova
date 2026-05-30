package scheduler

import "fmt"

func NewRuntime(cells []StateCell, lifecycles []LifecycleHandler, configs ...Config) Runtime {
	config := Config{}
	if len(configs) > 0 {
		config = configs[0]
	}
	config = normalizeConfig(config)
	return Runtime{
		cells:        copyCells(cells),
		lifecycles:   copyLifecycles(lifecycles),
		queue:        nil,
		nextSequence: 1,
		config:       config,
	}
}

func (r Runtime) Queue() []EventEnvelope {
	return copyQueue(r.queue)
}

func (r Runtime) State(key StateKey) (DataValue, bool) {
	for _, cell := range r.cells {
		if cellKey(cell) == key {
			return cell.Value, true
		}
	}
	return nil, false
}

func (r Runtime) Cells() []StateCell {
	return copyCells(r.cells)
}

func (r Runtime) NextSequence() LogicalSequence {
	return r.nextSequence
}

func (r Runtime) LastSequence() LogicalSequence {
	if r.nextSequence <= 1 {
		return 0
	}
	return r.nextSequence - 1
}

func NewRuntimeFromState(cells []StateCell, lifecycles []LifecycleHandler, values map[StateKey]DataValue, nextSequence LogicalSequence, configs ...Config) (Runtime, []SchedulerError) {
	runtime := NewRuntime(cells, lifecycles, configs...)
	if nextSequence < 1 {
		nextSequence = 1
	}
	runtime.nextSequence = nextSequence

	pending := make(map[StateKey]DataValue, len(values))
	for key, value := range values {
		pending[key] = value
	}

	validate := stateValueValidator(runtime.config)
	errs := make([]SchedulerError, 0)
	for i, cell := range runtime.cells {
		key := cellKey(cell)
		value, ok := pending[key]
		if !ok {
			continue
		}
		delete(pending, key)
		if err := validate(cell.Type, value); err != nil {
			stateKey := key
			errs = append(errs, SchedulerError{
				Source:  cell.Owner,
				Phase:   SchedulerPhaseTransition,
				State:   &stateKey,
				Message: err.Error(),
				Cause:   err,
			})
			continue
		}
		runtime.cells[i].Value = value
	}

	for key := range pending {
		stateKey := key
		errs = append(errs, SchedulerError{
			Source:  key.Owner,
			Phase:   SchedulerPhaseTransition,
			State:   &stateKey,
			Message: fmt.Sprintf("restored state %s.%s is not present in scheduler schema", key.Owner, key.Name),
		})
	}

	if len(errs) > 0 {
		return NewRuntime(cells, lifecycles, configs...), errs
	}
	return runtime, nil
}
