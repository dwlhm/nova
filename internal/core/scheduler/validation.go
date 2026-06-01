package scheduler

import (
	"fmt"
	"strings"

	novatypes "github.com/dwlhm/nova/internal/core/types"
)

func isSchedulerEvent(name SchedulerEvent) bool {
	return strings.HasPrefix(string(name), "@") && len(name) > 1
}

func normalizeConfig(config Config) Config {
	if config.ValidateStateValue == nil {
		config.ValidateStateValue = DefaultStateValueValidator
	}
	return config
}

func stateValueValidator(config Config) StateValueValidator {
	if config.ValidateStateValue != nil {
		return config.ValidateStateValue
	}
	return DefaultStateValueValidator
}

func DefaultStateValueValidator(typeRef string, value DataValue) error {
	typeRef = strings.TrimSpace(typeRef)
	if typeRef == "" || typeRef == "unknown" {
		return nil
	}
	if err := novatypes.ValidateValueText(typeRef, value); err == nil {
		return nil
	}
	return fmt.Errorf("%w: expected %s", ErrInvalidStateValue, typeRef)
}

func matchesStateType(typeRef string, value DataValue) bool {
	for _, candidate := range strings.Split(typeRef, "|") {
		if matchesSingleStateType(strings.TrimSpace(candidate), value) {
			return true
		}
	}
	return false
}
