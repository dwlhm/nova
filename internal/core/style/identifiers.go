package style

import "fmt"

var validPseudos = map[string]bool{
	"hover":    true,
	"focus":    true,
	"active":   true,
	"disabled": true,
	"checked":  true,
}

func validateClassIdent(name string) bool {
	if name == "" {
		return false
	}
	for index, ch := range name {
		if index == 0 {
			if ch < 'a' || ch > 'z' {
				return false
			}
			continue
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return false
	}
	return true
}

func validateTokenIdent(name string) bool {
	if name == "" {
		return false
	}
	segment := 0
	for _, ch := range name {
		if ch == '.' {
			if segment == 0 {
				return false
			}
			segment = 0
			continue
		}
		if segment == 0 {
			if ch < 'a' || ch > 'z' {
				return false
			}
		} else if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
			return false
		}
		segment++
	}
	return segment > 0
}

func validatePseudo(pseudo string) bool {
	return validPseudos[pseudo]
}

func diagnosticInvalidIdent(code, kind, name string) Diagnostic {
	return Diagnostic{
		Code:    code,
		Message: fmt.Sprintf("invalid %s name %q", kind, name),
	}
}
