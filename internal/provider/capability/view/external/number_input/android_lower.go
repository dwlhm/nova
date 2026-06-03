package numberinput

import (
	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	irandroid "github.com/dwlhm/nova/internal/provider/capability/view/ir/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func lowerAndroid(ctx external.Context, node contract.Node, path []int) (irandroid.Node, []shared.Diagnostic) {
	lowered := irandroid.LowerNode(node, path)
	lowered.Number = true
	return lowered, nil
}
