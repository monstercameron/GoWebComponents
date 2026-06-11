package runtime

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// TestWriteSSRPropsMatchesLegacySerializer verifies the streaming attribute
// writer produces byte-identical output to the legacy serializeProps path
// across randomized prop bags: names (valid and malicious), value types
// (string, bool, map, int), skip-classes (children/key/handlers/internal),
// and escaping-sensitive content.
func TestWriteSSRPropsMatchesLegacySerializer(parseT *testing.T) {
	parseRng := rand.New(rand.NewSource(42))

	parseNames := []string{
		"class", "id", "data-x", "title", "href", "aria-label", "style",
		"children", "key", "onClick", "onchange", "__gwc_prop__:value",
		"x onmouseover=alert(1)", "<bad>", "ok-name", "className", "htmlFor",
	}
	parseValues := []interface{}{
		"plain", `quote " and <angle> & amp`, "", true, false, 42, 3.14,
		map[string]string{"color": "red", "width": "10px"},
		nil,
		" line sep",
	}

	for parseTrial := 0; parseTrial < 500; parseTrial++ {
		parseProps := map[string]interface{}{}
		parseCount := parseRng.Intn(8)
		for parseIdx := 0; parseIdx < parseCount; parseIdx++ {
			parseName := parseNames[parseRng.Intn(len(parseNames))]
			parseProps[parseName] = parseValues[parseRng.Intn(len(parseValues))]
		}

		// Legacy path: serializeProps slices joined with leading spaces.
		var parseLegacy strings.Builder
		for _, parseAttr := range serializeProps(parseProps) {
			parseLegacy.WriteByte(' ')
			parseLegacy.WriteString(parseAttr)
		}

		// Streaming path.
		var parseStreaming strings.Builder
		writeSSRProps(&parseStreaming, parseProps)

		if parseLegacy.String() != parseStreaming.String() {
			parseT.Fatalf("trial %d: serializer divergence for props %#v\nlegacy:    %q\nstreaming: %q",
				parseTrial, parseProps, parseLegacy.String(), parseStreaming.String())
		}
	}
}

// TestRenderToStringDeterministic verifies SSR output is byte-identical across
// repeated renders of the same tree (map-iteration order must not leak).
func TestRenderToStringDeterministic(parseT *testing.T) {
	parseTree := buildSSRBenchmarkTree(25)
	parseFirst, parseErr := RenderToString(parseTree)
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	for parseIdx := 0; parseIdx < 20; parseIdx++ {
		parseAgain, parseErr := RenderToString(parseTree)
		if parseErr != nil {
			parseT.Fatalf("render %d: %v", parseIdx, parseErr)
		}
		if parseAgain != parseFirst {
			parseT.Fatalf("render %d diverged from first render:\nfirst: %s\nagain: %s",
				parseIdx, parseFirst[:min(200, len(parseFirst))], parseAgain[:min(200, len(parseAgain))])
		}
	}
	if !strings.Contains(parseFirst, `class="row"`) || !strings.Contains(parseFirst, "value-24") {
		parseT.Fatalf("rendered output missing expected content: %s", parseFirst[:min(300, len(parseFirst))])
	}
	_ = fmt.Sprintf
}
