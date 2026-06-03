package external

import (
	"github.com/dwlhm/nova/internal/provider/capability/view/hostctx"
	"github.com/dwlhm/nova/internal/provider/shared"
)

type Context = hostctx.Context

type AndroidContext = hostctx.AndroidContext

func NewContext(input shared.GenerateInput) Context {
	return hostctx.New(input)
}
