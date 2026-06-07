package androidcodegen

import (
	"github.com/dwlhm/nova/internal/core/style"
)

type androidStyleSheet struct {
	bundle style.Bundle
}

func newAndroidStyleSheet(bundle style.Bundle) androidStyleSheet {
	return androidStyleSheet{bundle: style.NormalizeBundle(bundle, "android")}
}

func (sheet androidStyleSheet) StyleForClassList(classList string) style.ResolvedStyle {
	return style.ResolvedStyleForClasses(sheet.bundle, classList)
}

func (sheet androidStyleSheet) StatesForClassList(classList string) map[string]style.ResolvedStyle {
	return style.ResolvedStatesForClasses(sheet.bundle, classList)
}
