package artifact

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/provider/android"
	"github.com/dwlhm/nova/internal/provider/shared"
	"github.com/dwlhm/nova/internal/provider/web"
)

type (
	File          = shared.File
	Diagnostic    = shared.Diagnostic
	GenerateInput = shared.GenerateInput
)

func Generate(input GenerateInput) ([]File, []Diagnostic) {
	switch input.Plan.Target {
	case "web":
		return web.Generate(input)
	case "android":
		return android.Generate(input)
	default:
		return nil, []Diagnostic{shared.ErrorDiagnostic("NVA-TARGET-019", fmt.Sprintf("unsupported build target %s", input.Plan.Target))}
	}
}

func BuildAppContract(bundle ir.Bundle) contract.App {
	return shared.BuildAppContract(bundle)
}
