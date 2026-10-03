package runtime

import "testing"

// TestSerializeStyleMapKebabCasesKeys pins the camelCase to kebab-case
// conversion SSR needs because virtualization and other callers use DOM
// style property names.
func TestSerializeStyleMapKebabCasesKeys(parseT *testing.T) {
	parseCases := []struct {
		parseName string
		parseIn   map[string]string
		parseWant string
	}{
		{"camel", map[string]string{"boxSizing": "border-box", "overflowY": "auto"}, "box-sizing:border-box;overflow-y:auto"},
		{"custom property untouched", map[string]string{"--myVar": "1"}, "--myVar:1"},
		{"webkit", map[string]string{"WebkitTransition": "none"}, "-webkit-transition:none"},
		{"ms", map[string]string{"msTransform": "none"}, "-ms-transform:none"},
		{"already kebab", map[string]string{"background-color": "red"}, "background-color:red"},
	}
	for _, parseCase := range parseCases {
		if parseGot := serializeStyleMap(parseCase.parseIn); parseGot != parseCase.parseWant {
			parseT.Errorf("%s: got %q want %q", parseCase.parseName, parseGot, parseCase.parseWant)
		}
	}
}

// TestSerializeStyleMapDropsInjection verifies values that would smuggle extra
// declarations, rules or style-element terminators are dropped.
func TestSerializeStyleMapDropsInjection(parseT *testing.T) {
	parseBad := []string{"red;position:fixed", "red}body{x:y", "red</style><script>", "url(\"a", "red/*", "a{b"}
	for _, parseValue := range parseBad {
		parseGot := serializeStyleMap(map[string]string{"color": parseValue, "margin": "0"})
		if parseGot != "margin:0" {
			parseT.Errorf("value %q: got %q want only margin:0", parseValue, parseGot)
		}
	}
	if parseGot := serializeStyleMap(map[string]string{"color;x": "red"}); parseGot != "" {
		parseT.Errorf("bad key kept: %q", parseGot)
	}
	parseOK := serializeStyleMap(map[string]string{"backgroundImage": "url(\"data:image/png;base64,AAA\")", "fontFamily": "'a;b', serif"})
	if parseOK != "background-image:url(\"data:image/png;base64,AAA\");font-family:'a;b', serif" {
		parseT.Errorf("legit values dropped: %q", parseOK)
	}
}