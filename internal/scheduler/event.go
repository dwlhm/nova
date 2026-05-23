package scheduler

import "fmt"

type StateKey struct {
	Owner CapabilityRef
	Name  StateName
}

func Key(owner CapabilityRef, name StateName) StateKey {
	return StateKey{Owner: owner, Name: name}
}

type EventEnvelope struct {
	Sequence LogicalSequence
	Source   CapabilityRef
	Name     SchedulerEvent
	Payload  DataValue
}

type EventToEmit struct {
	Source  CapabilityRef
	Name    SchedulerEvent
	Payload DataValue
}

func Emit(source CapabilityRef, name SchedulerEvent, payload DataValue) EventToEmit {
	return EventToEmit{Source: source, Name: name, Payload: payload}
}

type Snapshot struct {
	values map[StateKey]DataValue
}

func (s Snapshot) Value(key StateKey) (DataValue, bool) {
	value, ok := s.values[key]
	return value, ok
}

func (s Snapshot) MustValue(key StateKey) DataValue {
	value, ok := s.Value(key)
	if !ok {
		panic(fmt.Sprintf("missing scheduler state %s.%s", key.Owner, key.Name))
	}
	return value
}

func (s Snapshot) Values() map[StateKey]DataValue {
	out := make(map[StateKey]DataValue, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out
}
