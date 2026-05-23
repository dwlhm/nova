package scheduler

type TransitionFunc func(Snapshot, EventEnvelope) (DataValue, error)

type TransitionRule struct {
	Event SchedulerEvent
	Apply TransitionFunc
}

func On(event SchedulerEvent, apply TransitionFunc) TransitionRule {
	return TransitionRule{Event: event, Apply: apply}
}

type StateCell struct {
	Owner       CapabilityRef
	Name        StateName
	Type        string
	Value       DataValue
	Transitions []TransitionRule
}

func NewStateCell(owner CapabilityRef, name StateName, typ string, value DataValue, transitions ...TransitionRule) StateCell {
	return StateCell{
		Owner:       owner,
		Name:        name,
		Type:        typ,
		Value:       value,
		Transitions: copyTransitions(transitions),
	}
}
