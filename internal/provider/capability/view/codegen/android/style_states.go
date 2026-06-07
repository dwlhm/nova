package androidcodegen

import (
	"github.com/dwlhm/nova/internal/core/style"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func androidJavaStyleApplication(target string, nodeKind string, base style.ResolvedStyle, states map[string]style.ResolvedStyle, indent string) string {
	if base.Empty() && len(states) == 0 {
		return ""
	}
	return indent + "NovaStyle.applyWithStates(" + target + ", " + shared.QuoteCodeString(nodeKind) + ", " +
		androidJavaPropertyMapLiteral(base.Properties) + ", " +
		androidJavaStatesMapLiteral(states) + ", this::dp);\n"
}
