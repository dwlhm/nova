package style

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/view"
)

// ValidateSheet runs portable subset and semantic value checks on a parsed sheet.
func ValidateSheet(sheet Sheet) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	diagnostics = append(diagnostics, ValidatePortableSubset(sheet)...)

	for className := range sheet.Classes {
		if !validateClassIdent(className) {
			diagnostics = append(diagnostics, diagnosticInvalidIdent("NVA-STYLE-010", "class", className))
		}
	}
	for name := range sheet.Tokens {
		if !validateTokenIdent(name) {
			diagnostics = append(diagnostics, diagnosticInvalidIdent("NVA-STYLE-010", "token", name))
		}
	}
	for _, state := range sheet.States {
		if !validateClassIdent(state.Class) {
			diagnostics = append(diagnostics, diagnosticInvalidIdent("NVA-STYLE-010", "class", state.Class))
		}
		if !validatePseudo(state.Pseudo) {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-010",
				Message: fmt.Sprintf("invalid pseudo %q on state %s", state.Pseudo, state.Class),
			})
		}
		if _, ok := sheet.Classes[state.Class]; !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-010",
				Message: fmt.Sprintf("state %s %s references unknown class %q", state.Class, state.Pseudo, state.Class),
			})
		}
	}

	for className, rule := range sheet.Classes {
		diagnostics = append(diagnostics, ValidatePropertyValues("class "+className, rule.Properties)...)
	}
	for _, state := range sheet.States {
		diagnostics = append(diagnostics, ValidatePropertyValues(
			fmt.Sprintf("state %s %s", state.Class, state.Pseudo),
			state.Properties,
		)...)
	}
	return diagnostics
}

// ValidateBundle reports duplicate classes in the same scope across merged sheets.
func ValidateBundle(bundle Bundle) []Diagnostic {
	type classKey struct {
		scope Scope
		name  string
	}
	seen := make(map[classKey]string)
	diagnostics := make([]Diagnostic, 0)
	for _, sheet := range bundle.Sheets {
		scope := NormalizeScope(sheet.Scope)
		for className := range sheet.Classes {
			key := classKey{scope: scope, name: className}
			if prior, ok := seen[key]; ok {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-004",
					Message: fmt.Sprintf("duplicate class %q in scope %s (%s and %s)", className, scope, prior, sheet.SourcePath),
				})
				continue
			}
			seen[key] = sheet.SourcePath
		}
	}
	sort.Slice(diagnostics, func(i, j int) bool {
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return diagnostics
}

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
