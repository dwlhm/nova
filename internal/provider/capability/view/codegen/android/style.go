package androidcodegen

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/style"
)

type androidStyleSheet struct {
	Classes map[string]androidStyleRule
	States  map[string]map[string]androidStyleRule
}

type androidStyleRule struct {
	Properties map[string]string
}

type androidResolvedStyle struct {
	Properties map[string]string
}

func newAndroidStyleSheet(bundle style.Bundle) androidStyleSheet {
	sheet := androidStyleSheet{
		Classes: make(map[string]androidStyleRule),
		States:  make(map[string]map[string]androidStyleRule),
	}
	for _, doc := range bundle.Sheets {
		for className, rule := range doc.Classes {
			current := sheet.Classes[className]
			if current.Properties == nil {
				current.Properties = make(map[string]string)
			}
			for name, value := range rule.Properties {
				current.Properties[name] = value
			}
			sheet.Classes[className] = current
		}
		for _, state := range doc.States {
			classStates := sheet.States[state.Class]
			if classStates == nil {
				classStates = make(map[string]androidStyleRule)
				sheet.States[state.Class] = classStates
			}
			current := classStates[state.Pseudo]
			if current.Properties == nil {
				current.Properties = make(map[string]string)
			}
			for name, value := range state.Properties {
				current.Properties[name] = value
			}
			classStates[state.Pseudo] = current
		}
	}
	return sheet
}

func (sheet androidStyleSheet) StyleForClassList(classList string) androidResolvedStyle {
	out := androidResolvedStyle{Properties: make(map[string]string)}
	for _, className := range strings.Fields(classList) {
		rule, ok := sheet.Classes[className]
		if !ok {
			continue
		}
		for name, value := range rule.Properties {
			out.Properties[name] = value
		}
	}
	return out
}

func (sheet androidStyleSheet) StatesForClassList(classList string) map[string]androidResolvedStyle {
	out := make(map[string]androidResolvedStyle)
	for _, className := range strings.Fields(classList) {
		classStates, ok := sheet.States[className]
		if !ok {
			continue
		}
		for pseudo, rule := range classStates {
			current := out[pseudo]
			if current.Properties == nil {
				current.Properties = make(map[string]string)
			}
			for name, value := range rule.Properties {
				current.Properties[name] = value
			}
			out[pseudo] = current
		}
	}
	return out
}

func (style androidResolvedStyle) Empty() bool {
	return len(style.Properties) == 0
}

func (style androidResolvedStyle) Value(names ...string) (string, bool) {
	for _, name := range names {
		value := strings.TrimSpace(style.Properties[name])
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func mergeAndroidStyle(base, override androidResolvedStyle) androidResolvedStyle {
	out := androidResolvedStyle{Properties: make(map[string]string)}
	for name, value := range base.Properties {
		out.Properties[name] = value
	}
	for name, value := range override.Properties {
		out.Properties[name] = value
	}
	return out
}
