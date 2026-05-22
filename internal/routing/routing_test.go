package routing

import "testing"

func TestNormalizePathHandlesRealWorldURLShapes(t *testing.T) {
	cases := map[string]string{
		"":                                      "/",
		"settings":                              "/settings",
		"/settings/":                            "/settings",
		"//settings///profile?tab=security#top": "/settings/profile",
		"https://example.test/users/42?x=1":     "/users/42",
		"/docs/%E2%9C%93":                       "/docs/✓",
	}

	for input, want := range cases {
		if got := NormalizePath(input); got != want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMatchSupportsExactDynamicWildcardAndFallbackRoutes(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
		params  map[string]string
	}{
		{pattern: "/settings", path: "/settings/", want: true},
		{pattern: "/users/:id", path: "/users/42?tab=profile", want: true, params: map[string]string{"id": "42"}},
		{pattern: "/teams/{team}/members/{member}", path: "/teams/core/members/ada", want: true, params: map[string]string{"team": "core", "member": "ada"}},
		{pattern: "/docs/*", path: "/docs/reference/router", want: true},
		{pattern: "*", path: "/missing/route", want: true},
		{pattern: "/settings", path: "/settings/profile", want: false},
	}

	for _, tt := range cases {
		match := Match(tt.pattern, tt.path)
		if match.Matched != tt.want {
			t.Fatalf("Match(%q, %q).Matched = %v, want %v", tt.pattern, tt.path, match.Matched, tt.want)
		}
		for key, want := range tt.params {
			if got := match.Params[key]; got != want {
				t.Fatalf("Match(%q, %q).Params[%s] = %q, want %q", tt.pattern, tt.path, key, got, want)
			}
		}
	}
}

func TestBestMatchPrefersMostSpecificRoute(t *testing.T) {
	routes := []Pattern{
		{Path: "*"},
		{Path: "/users/*"},
		{Path: "/users/:id"},
		{Path: "/users/settings"},
	}

	match := BestMatch(routes, "/users/settings?mode=compact")
	if !match.Matched || match.Pattern != "/users/settings" {
		t.Fatalf("best exact match = %+v, want /users/settings", match)
	}

	match = BestMatch(routes, "/users/42")
	if !match.Matched || match.Pattern != "/users/:id" || match.Params["id"] != "42" {
		t.Fatalf("best dynamic match = %+v, want /users/:id with id", match)
	}

	match = BestMatch(routes, "/users/42/preferences")
	if !match.Matched || match.Pattern != "/users/*" {
		t.Fatalf("best wildcard match = %+v, want /users/*", match)
	}

	match = BestMatch(routes, "/elsewhere")
	if !match.Matched || match.Pattern != "*" || !match.Fallback {
		t.Fatalf("fallback match = %+v, want fallback", match)
	}
}

func TestDescribePatternExtractsMetadata(t *testing.T) {
	description := DescribePattern("/teams/{team}/members/:member")
	if description.Pattern != "/teams/{team}/members/:member" || description.Fallback {
		t.Fatalf("description = %+v, want route pattern", description)
	}
	if len(description.Params) != 2 || description.Params[0] != "team" || description.Params[1] != "member" {
		t.Fatalf("params = %+v, want team/member", description.Params)
	}
	if description.Score <= 0 {
		t.Fatalf("score = %d, want positive", description.Score)
	}
}
