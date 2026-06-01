package shared

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/provider/build"
)

func MustJSON(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(encoded) + "\n"
}

func ClonePermissions(permissions []security.Permission) []security.Permission {
	out := make([]security.Permission, len(permissions))
	copy(out, permissions)
	return out
}

func QuoteCodeString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func EscapeHTML(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return replacer.Replace(value)
}

func EscapeGradleComment(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}

func ErrorDiagnostic(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: diagnostic.SeverityError, Message: message}
}

func ExternalOperationNames(operations []build.ResolvedExternalOperation) []string {
	names := make([]string, 0, len(operations))
	for _, operation := range operations {
		names = append(names, operation.CapabilitySource+"."+operation.Operation)
	}
	sort.Strings(names)
	return names
}

func CloneIntPath(values []int) []int {
	out := make([]int, len(values))
	copy(out, values)
	return out
}
