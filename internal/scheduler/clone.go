package scheduler

func snapshotFromCells(cells []StateCell) Snapshot {
	values := make(map[StateKey]DataValue, len(cells))
	for _, cell := range cells {
		values[cellKey(cell)] = cell.Value
	}
	return Snapshot{values: values}
}

func cellKey(cell StateCell) StateKey {
	return Key(cell.Owner, cell.Name)
}
func appendQueue(queue []EventEnvelope, event EventEnvelope) []EventEnvelope {
	out := make([]EventEnvelope, len(queue)+1)
	copy(out, queue)
	out[len(queue)] = event
	return out
}

func copyQueue(queue []EventEnvelope) []EventEnvelope {
	out := make([]EventEnvelope, len(queue))
	copy(out, queue)
	return out
}

func copyCells(cells []StateCell) []StateCell {
	out := make([]StateCell, len(cells))
	for i, cell := range cells {
		out[i] = cell
		out[i].Transitions = copyTransitions(cell.Transitions)
	}
	return out
}

func copyTransitions(transitions []TransitionRule) []TransitionRule {
	out := make([]TransitionRule, len(transitions))
	copy(out, transitions)
	return out
}

func copyLifecycles(lifecycles []LifecycleHandler) []LifecycleHandler {
	out := make([]LifecycleHandler, len(lifecycles))
	copy(out, lifecycles)
	return out
}

func cloneExternalRequests(requests []ExternalOperationRequest) []ExternalOperationRequest {
	out := make([]ExternalOperationRequest, len(requests))
	for i, request := range requests {
		out[i] = request
		out[i].Input = cloneInput(request.Input)
	}
	return out
}

func cloneInput(input map[string]DataValue) map[string]DataValue {
	if input == nil {
		return nil
	}
	out := make(map[string]DataValue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
