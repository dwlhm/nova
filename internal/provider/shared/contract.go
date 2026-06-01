package shared

import (
	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/provider/target"
)

func BuildAppContract(bundle ir.Bundle) contract.App {
	return bundle.App
}

func RuntimeContract(targetID string) (target.RuntimeContract, bool) {
	switch targetID {
	case "web":
		return target.WebContract(), true
	case "android":
		return target.AndroidContract(), true
	default:
		return target.RuntimeContract{}, false
	}
}
