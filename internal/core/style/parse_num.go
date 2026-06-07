package style

import "strconv"

func parseIntString(value string) (int, error) {
	return strconv.Atoi(value)
}

func parseFloatString(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}
