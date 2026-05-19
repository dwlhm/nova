package diagnostic

import (
	"encoding/json"
	"sort"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type SourceSpan struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Length int    `json:"length,omitempty"`
}

type RelatedSpan struct {
	Message string     `json:"message"`
	Span    SourceSpan `json:"span"`
}

type Diagnostic struct {
	Code     string        `json:"code"`
	Severity Severity      `json:"severity"`
	Message  string        `json:"message"`
	Span     *SourceSpan   `json:"span,omitempty"`
	Related  []RelatedSpan `json:"related,omitempty"`
	Hint     string        `json:"hint,omitempty"`
	Target   string        `json:"target,omitempty"`
}

func Error(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: SeverityError, Message: message}
}

func Warning(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: SeverityWarning, Message: message}
}

func Info(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: SeverityInfo, Message: message}
}

func HasErrors(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

func StableSort(diagnostics []Diagnostic) []Diagnostic {
	out := make([]Diagnostic, len(diagnostics))
	copy(out, diagnostics)
	sort.SliceStable(out, func(i, j int) bool {
		left := diagnosticKey(out[i])
		right := diagnosticKey(out[j])
		for idx := range left {
			if left[idx] == right[idx] {
				continue
			}
			return left[idx] < right[idx]
		}
		return false
	})
	return out
}

func JSONLines(diagnostics []Diagnostic) (string, error) {
	var builder strings.Builder
	for _, diagnostic := range StableSort(diagnostics) {
		if diagnostic.Severity == "" {
			diagnostic.Severity = SeverityError
		}
		line, err := json.Marshal(diagnostic)
		if err != nil {
			return "", err
		}
		builder.Write(line)
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}

func diagnosticKey(diagnostic Diagnostic) [6]string {
	span := SourceSpan{}
	if diagnostic.Span != nil {
		span = *diagnostic.Span
	}
	return [6]string{
		span.File,
		padInt(span.Line),
		padInt(span.Column),
		diagnostic.Code,
		string(diagnostic.Severity),
		diagnostic.Message,
	}
}

func padInt(value int) string {
	if value < 0 {
		value = 0
	}
	digits := "0000000000"
	text := strconvItoa(value)
	if len(text) >= len(digits) {
		return text
	}
	return digits[:len(digits)-len(text)] + text
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for value > 0 {
		pos--
		buf[pos] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[pos:])
}
