package androidcodegen

import (
	"github.com/dwlhm/nova/internal/core/contract"
)

func contractHasDynamicClassBindings(bindings []contract.BindingMeta) bool {
	return ContractHasDynamicClassBindings(bindings)
}

func ContractHasDynamicClassBindings(bindings []contract.BindingMeta) bool {
	for _, binding := range bindings {
		if binding.Prop == "class" {
			return true
		}
	}
	return false
}
