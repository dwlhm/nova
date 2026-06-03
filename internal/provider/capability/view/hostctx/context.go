package hostctx

import (
	androidcodegen "github.com/dwlhm/nova/internal/provider/capability/view/codegen/android"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

// Context carries target-neutral and target-specific state for capability pipelines.
// Registry instances are owned by internal host bootstrap, not stored here.
type Context struct {
	Input shared.GenerateInput

	// AndroidNodes is populated after primitive node emitters register.
	AndroidNodes *androidcodegen.NodeRegistry

	Android *AndroidContext
}

type AndroidContext struct {
	Config     androidtarget.Config
	SourceRoot string
}

func New(input shared.GenerateInput) Context {
	return Context{Input: input}
}
