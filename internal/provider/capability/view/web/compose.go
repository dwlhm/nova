package web

import (
	viewinternal "github.com/dwlhm/nova/internal/provider/capability/view/internal"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func Compose(input shared.GenerateInput) ([]shared.File, []shared.Diagnostic) {
	return viewinternal.NewDefaultHost().Generate(input)
}
