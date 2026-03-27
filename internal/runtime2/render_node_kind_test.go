package runtime2

import "testing"

func TestParseRenderNodeKindSupportedKindsDecode(t *testing.T) {
	parseCases := []struct {
		parseName string
		parseRaw  uint8
		parseWant RenderNodeKind
	}{
		{
			parseName: "text",
			parseRaw:  1,
			parseWant: RenderNodeKindText,
		},
		{
			parseName: "host element",
			parseRaw:  2,
			parseWant: RenderNodeKindHostElement,
		},
		{
			parseName: "fragment",
			parseRaw:  3,
			parseWant: RenderNodeKindFragment,
		},
	}

	for _, parseCase := range parseCases {
		parseCase := parseCase
		t.Run(parseCase.parseName, func(t *testing.T) {
			parseGot, parseErr := ParseRenderNodeKind(parseCase.parseRaw)
			if parseErr != nil {
				t.Fatalf("ParseRenderNodeKind(%d) error = %v", parseCase.parseRaw, parseErr)
			}
			if parseGot != parseCase.parseWant {
				t.Fatalf("ParseRenderNodeKind(%d) = %v, want %v", parseCase.parseRaw, parseGot, parseCase.parseWant)
			}
		})
	}
}

func TestParseRenderNodeKindUnknownKindFails(t *testing.T) {
	_, parseErr := ParseRenderNodeKind(99)
	if parseErr == nil {
		t.Fatal("ParseRenderNodeKind(99) error = nil, want error")
	}
}

func TestParseRenderNodeKindZeroKindFails(t *testing.T) {
	_, parseErr := ParseRenderNodeKind(0)
	if parseErr == nil {
		t.Fatal("ParseRenderNodeKind(0) error = nil, want error")
	}
}
