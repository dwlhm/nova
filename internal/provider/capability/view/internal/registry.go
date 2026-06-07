package internal

import (
	"github.com/dwlhm/nova/internal/provider/capability/view/external"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/button"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/column"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/number_input"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/page"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/row"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/runtime"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/scroll"
	selectprimitive "github.com/dwlhm/nova/internal/provider/capability/view/external/select"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/stack"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/style"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/surface"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/text"
	"github.com/dwlhm/nova/internal/provider/capability/view/external/text_input"
)

func defaultModules() []external.Module {
	// Static compile-time registry: one sandboxed module per folder.
	return []external.Module{
		runtime.Module,
		style.Module,
		text.Module,
		button.Module,
		page.Module,
		row.Module,
		scroll.Module,
		stack.Module,
		surface.Module,
		column.Module,
		textinput.Module,
		numberinput.Module,
		selectprimitive.Module,
	}
}
