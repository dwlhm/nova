package numberinput

import (
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/primitive"
)

var Module = primitive.AndroidCustom(
	external.PrimitiveMetadata("number_input", []string{"number_input"}, 40),
	"number_input",
	lowerAndroid,
	emitAndroid,
)
