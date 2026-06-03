package style

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/view"
)

// ValidateViewClasses checks template class bindings against merged .nova-style sheets.
func ValidateViewClasses(viewIR view.IR, bundle Bundle) []Diagnostic {
	if len(bundle.Sheets) == 0 {
		return nil
	}
	known := knownClasses(bundle)
	if len(known) == 0 {
		return nil
	}

	diagnostics := make([]Diagnostic, 0)
	var walk func(nodes []view.Node)
	walk = func(nodes []view.Node) {
		for _, node := range nodes {
			if binding, ok := node.Props["class"]; ok {
				diagnostics = append(diagnostics, validateClassBinding(node.Kind, binding, known)...)
			}
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(viewIR.Nodes)

	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return diagnostics
}

func validateClassBinding(nodeKind string, binding view.Binding, known map[string]bool) []Diagnostic {
	classList, static := staticClassBinding(binding)
	if !static {
		return []Diagnostic{{
			Code:    "NVA-STYLE-007",
			Message: fmt.Sprintf("<%s> class binding is not a static string literal", nodeKind),
		}}
	}
	diagnostics := make([]Diagnostic, 0)
	for _, className := range classList {
		if known[className] {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-STYLE-008",
			Message: fmt.Sprintf("unknown class %q on <%s>", className, nodeKind),
		})
	}
	return diagnostics
}

func staticClassBinding(binding view.Binding) ([]string, bool) {
	tokens := trimBindingTokens(binding.Tokens)
	if len(tokens) != 1 || tokens[0].Type != lexer.STRING {
		return nil, false
	}
	value := strings.TrimSpace(tokens[0].Literal)
	if value == "" {
		return nil, false
	}
	return strings.Fields(value), true
}

func trimBindingTokens(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && tokens[start].Type == lexer.COMMENT {
		start++
	}
	end := len(tokens)
	for end > start && tokens[end-1].Type == lexer.COMMENT {
		end--
	}
	return tokens[start:end]
}

func knownClasses(bundle Bundle) map[string]bool {
	known := make(map[string]bool)
	for _, sheet := range bundle.Sheets {
		for name := range sheet.Classes {
			known[name] = true
		}
		for _, state := range sheet.States {
			if state.Class != "" {
				known[state.Class] = true
			}
		}
	}
	return known
}
