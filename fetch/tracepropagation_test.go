package fetch

import "testing"

func TestShouldInjectTraceparentDecision(parseT *testing.T) {
	const parsePage = "https://app.example.com"
	parseCases := []struct {
		name    string
		headers map[string]any
		url     string
		page    string
		allow   []string
		want    bool
	}{
		{"relative path", nil, "/api/users", parsePage, nil, true},
		{"relative no page origin", nil, "api/users", "", nil, true},
		{"same origin absolute", nil, "https://app.example.com/api", parsePage, nil, true},
		{"same origin case", nil, "HTTPS://APP.example.com/api", parsePage, nil, true},
		{"cross origin default", nil, "https://api.other.com/x", parsePage, nil, false},
		{"cross origin allowlisted", nil, "https://api.other.com/x", parsePage, []string{"https://api.other.com"}, true},
		{"allowlist trailing slash", nil, "https://api.other.com/x", parsePage, []string{"https://API.other.com/"}, true},
		{"allowlist other origin", nil, "https://evil.com/x", parsePage, []string{"https://api.other.com"}, false},
		{"scheme differs", nil, "http://app.example.com/api", parsePage, nil, false},
		{"scheme relative same", nil, "//app.example.com/api", parsePage, nil, true},
		{"scheme relative cross", nil, "//api.other.com/api", parsePage, nil, false},
		{"caller header lower", map[string]any{"traceparent": "x"}, "/api", parsePage, nil, false},
		{"caller header mixed", map[string]any{"TraceParent": "x"}, "/api", parsePage, nil, false},
		{"unrelated header", map[string]any{"Accept": "json"}, "/api", parsePage, nil, true},
		{"data url", nil, "data:text/plain,hi", parsePage, nil, false},
		{"unparseable", nil, "http://[::1", parsePage, nil, false},
	}
	for _, parseCase := range parseCases {
		if parseGot := shouldInjectTraceparent(parseCase.headers, parseCase.url, parseCase.page, parseCase.allow); parseGot != parseCase.want {
			parseT.Errorf("%s: got %v, want %v", parseCase.name, parseGot, parseCase.want)
		}
	}
}
