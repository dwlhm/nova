package routing

import (
	"net/url"
	"path"
	"strings"
)

type Pattern struct {
	Path string
}

type MatchResult struct {
	Matched  bool
	Pattern  string
	Path     string
	Params   map[string]string
	Score    int
	Fallback bool
}

type PatternDescription struct {
	Pattern  string   `json:"pattern"`
	Params   []string `json:"params"`
	Score    int      `json:"score"`
	Fallback bool     `json:"fallback"`
}

func NormalizePath(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return "/"
	}
	if parsed, err := url.Parse(text); err == nil && parsed.IsAbs() {
		text = parsed.EscapedPath()
	}
	if cut := strings.IndexAny(text, "?#"); cut >= 0 {
		text = text[:cut]
	}
	if !strings.HasPrefix(text, "/") {
		text = "/" + text
	}
	cleaned := path.Clean(strings.ReplaceAll(text, "\\", "/"))
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	if cleaned == "/" {
		return "/"
	}
	return strings.Join(decodedSegments(cleaned), "/")
}

func Match(pattern string, value string) MatchResult {
	normalizedPattern := normalizePattern(pattern)
	normalizedPath := NormalizePath(value)
	params := make(map[string]string)

	if isFallbackPattern(normalizedPattern) {
		return MatchResult{Matched: true, Pattern: normalizedPattern, Path: normalizedPath, Params: params, Score: 0, Fallback: true}
	}

	patternSegments := routeSegments(normalizedPattern)
	pathSegments := routeSegments(normalizedPath)
	score := 0

	for i, segment := range patternSegments {
		if isSegmentWildcard(segment) {
			if i == len(patternSegments)-1 {
				return MatchResult{Matched: true, Pattern: normalizedPattern, Path: normalizedPath, Params: params, Score: score + 10, Fallback: false}
			}
			return MatchResult{Pattern: normalizedPattern, Path: normalizedPath, Params: params}
		}
		if i >= len(pathSegments) {
			return MatchResult{Pattern: normalizedPattern, Path: normalizedPath, Params: params}
		}
		if name, ok := dynamicSegmentName(segment); ok {
			params[name] = pathSegments[i]
			score += 50
			continue
		}
		if segment != pathSegments[i] {
			return MatchResult{Pattern: normalizedPattern, Path: normalizedPath, Params: params}
		}
		score += 100
	}
	if len(patternSegments) != len(pathSegments) {
		return MatchResult{Pattern: normalizedPattern, Path: normalizedPath, Params: params}
	}
	return MatchResult{Matched: true, Pattern: normalizedPattern, Path: normalizedPath, Params: params, Score: score + 1000, Fallback: false}
}

func BestMatch(patterns []Pattern, value string) MatchResult {
	best := MatchResult{}
	for _, pattern := range patterns {
		match := Match(pattern.Path, value)
		if !match.Matched {
			continue
		}
		if !best.Matched || match.Score > best.Score {
			best = match
		}
	}
	return best
}

func DescribePattern(pattern string) PatternDescription {
	normalized := normalizePattern(pattern)
	description := PatternDescription{
		Pattern:  normalized,
		Params:   dynamicSegmentNames(normalized),
		Score:    Match(normalized, representativePath(normalized)).Score,
		Fallback: isFallbackPattern(normalized),
	}
	if description.Fallback {
		description.Score = 0
	}
	return description
}

func normalizePattern(pattern string) string {
	text := strings.TrimSpace(pattern)
	if isFallbackPattern(text) {
		return "*"
	}
	if strings.HasSuffix(text, "/*") {
		prefix := strings.TrimSuffix(text, "/*")
		return NormalizePath(prefix) + "/*"
	}
	return NormalizePath(text)
}

func isFallbackPattern(pattern string) bool {
	return pattern == "*" || pattern == "/*"
}

func routeSegments(value string) []string {
	normalized := NormalizePath(value)
	if normalized == "/" {
		return nil
	}
	return strings.Split(strings.Trim(normalized, "/"), "/")
}

func decodedSegments(value string) []string {
	parts := strings.Split(value, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		out = append(out, decoded)
	}
	if len(out) == 0 {
		return []string{""}
	}
	out[0] = "/" + out[0]
	return out
}

func isSegmentWildcard(segment string) bool {
	return segment == "*"
}

func dynamicSegmentName(segment string) (string, bool) {
	if strings.HasPrefix(segment, ":") && len(segment) > 1 {
		return segment[1:], true
	}
	if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") && len(segment) > 2 {
		return strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}"), true
	}
	return "", false
}

func dynamicSegmentNames(pattern string) []string {
	seen := make(map[string]bool)
	names := make([]string, 0)
	for _, segment := range routeSegments(pattern) {
		name, ok := dynamicSegmentName(segment)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

func representativePath(pattern string) string {
	if isFallbackPattern(pattern) {
		return "/"
	}
	segments := routeSegments(pattern)
	for i, segment := range segments {
		if _, ok := dynamicSegmentName(segment); ok {
			segments[i] = "value"
		}
		if isSegmentWildcard(segment) {
			segments[i] = "rest"
		}
	}
	return "/" + strings.Join(segments, "/")
}
