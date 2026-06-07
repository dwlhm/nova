package style

import (
	"strconv"
	"strings"
)

// BoxSides is CSS top/right/bottom/left after expanding shorthand (padding, margin).
type BoxSides struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// ParseLength parses a portable length (dp/sp/px suffix optional).
func ParseLength(value string) (int, bool) {
	cleaned := strings.TrimSpace(value)
	for _, suffix := range []string{"px", "dp", "sp"} {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(cleaned), 64)
	if err != nil || number < 0 {
		return 0, false
	}
	return int(number + 0.5), true
}

// ParseBoxShorthand expands 1–4 length values using CSS box-model order:
// 1 value → all sides; 2 → vertical horizontal; 3 → top horizontal bottom; 4 → top right bottom left.
func ParseBoxShorthand(value string) (BoxSides, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 || len(fields) > 4 {
		return BoxSides{}, false
	}
	values := make([]int, 0, len(fields))
	for _, field := range fields {
		size, ok := ParseLength(field)
		if !ok {
			return BoxSides{}, false
		}
		values = append(values, size)
	}
	switch len(values) {
	case 1:
		return BoxSides{values[0], values[0], values[0], values[0]}, true
	case 2:
		return BoxSides{values[0], values[1], values[0], values[1]}, true
	case 3:
		return BoxSides{values[0], values[1], values[2], values[1]}, true
	default:
		return BoxSides{values[0], values[1], values[2], values[3]}, true
	}
}

// ParseColorField returns true when a single field is a portable color literal.
func ParseColorField(field string) bool {
	field = strings.TrimSpace(strings.Trim(field, ","))
	if field == "" {
		return false
	}
	if strings.HasPrefix(field, "#") {
		return isHexColor(field)
	}
	return isCSSNamedColor(field)
}

func isHexColor(value string) bool {
	hex := strings.TrimPrefix(value, "#")
	if len(hex) != 3 && len(hex) != 6 && len(hex) != 8 {
		return false
	}
	for _, ch := range hex {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			continue
		}
		return false
	}
	return true
}

func isCSSNamedColor(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "black", "blue", "cyan", "darkgray", "gray", "green", "lightgray", "magenta", "red", "white", "yellow", "transparent":
		return true
	default:
		return false
	}
}

func isNumericLiteral(field string) bool {
	cleaned := strings.TrimSpace(field)
	for _, suffix := range []string{"px", "dp", "sp", "em"} {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	if cleaned == "" {
		return false
	}
	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}

func isKeywordLiteral(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "normal", "bold", "bolder", "lighter",
		"none", "uppercase", "lowercase", "capitalize",
		"left", "center", "right", "start", "end",
		"flex-start", "flex-end", "stretch", "space-between", "space-around":
		return true
	default:
		return false
	}
}

func isLiteralField(field string) bool {
	if field == "" {
		return false
	}
	if strings.HasPrefix(field, "#") {
		return isHexColor(field)
	}
	if isCSSNamedColor(field) {
		return true
	}
	if isNumericLiteral(field) {
		return true
	}
	return isKeywordLiteral(field)
}

func parseBorderWidthField(field string) (int, bool) {
	return ParseLength(field)
}

func parseBorderShorthand(value string) bool {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return false
	}
	width, ok := parseBorderWidthField(fields[0])
	if !ok || width < 0 {
		return false
	}
	if len(fields) == 1 {
		return true
	}
	for _, field := range fields[1:] {
		if !ParseColorField(field) {
			return false
		}
	}
	return true
}

func parseLineHeight(value string) bool {
	cleaned := strings.TrimSpace(strings.TrimSuffix(value, "em"))
	if strings.HasSuffix(cleaned, "px") {
		return false
	}
	number, err := strconv.ParseFloat(cleaned, 64)
	return err == nil && number > 0
}

func parseFontWeight(value string) bool {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	switch cleaned {
	case "normal", "bold", "bolder", "lighter":
		return true
	}
	weight, err := strconv.Atoi(cleaned)
	return err == nil && weight >= 1 && weight <= 1000
}

func parseFontSize(value string) bool {
	_, ok := ParseLength(value)
	return ok
}

func parseTextAlign(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left", "center", "right", "start", "end":
		return true
	default:
		return false
	}
}

func parseTextTransform(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "uppercase", "lowercase", "capitalize":
		return true
	default:
		return false
	}
}

func parseAlignContent(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "center", "flex-start", "flex-end", "start", "end", "stretch", "space-between", "space-around":
		return true
	default:
		return false
	}
}
