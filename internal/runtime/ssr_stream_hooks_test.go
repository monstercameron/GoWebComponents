package runtime

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func streamToString(t *testing.T, parseEl *Element) string {
	t.Helper()
	var parseBuf strings.Builder
	if parseErr := RenderToStream(context.Background(), &parseBuf, parseEl, SSRStreamOptions{}); parseErr != nil {
		t.Fatalf("RenderToStream error: %v", parseErr)
	}
	return parseBuf.String()
}

// The streaming SSR renderer runs hooks (like the buffered path) and now threads
// context through the shell, so a streamed component's GoUseContextValue resolves
// to the nearest provider rather than the descriptor default.
func TestStreamRendersHooksAndContext(t *testing.T) {
	// useState renders its initial value in the stream shell.
	parseStateComponent := func() *Element {
		parseGet, _ := GoUseStateGlobal(5)
		return CreateElement("p", map[string]any{}, fmt.Sprint(parseGet()))
	}
	if parseGot := streamToString(t, CreateElement(parseStateComponent, map[string]any{})); parseGot != "<p>5</p>" {
		t.Errorf("stream useState = %q, want <p>5</p>", parseGot)
	}

	parseDesc := NewContextDescriptor("DEFAULT")
	parseProvider := NewContextProviderType(parseDesc)
	parseConsumer := func() *Element {
		return CreateElement("span", map[string]any{}, fmt.Sprint(GoUseContextValue(parseDesc)))
	}

	// Provider value reaches a consumer nested under a host element in the stream.
	parseProvided := CreateElementOwned(parseProvider, map[string]any{"value": "PROVIDED"},
		CreateElement("div", map[string]any{}, CreateElement(parseConsumer, map[string]any{})))
	if parseGot := streamToString(t, parseProvided); parseGot != "<div><span>PROVIDED</span></div>" {
		t.Errorf("stream useContext = %q, want <div><span>PROVIDED</span></div>", parseGot)
	}

	// Default when no provider.
	if parseGot := streamToString(t, CreateElement(parseConsumer, map[string]any{})); parseGot != "<span>DEFAULT</span>" {
		t.Errorf("stream useContext default = %q", parseGot)
	}

	// Nested provider overrides for the streamed subtree.
	parseNested := CreateElementOwned(parseProvider, map[string]any{"value": "OUTER"},
		CreateElementOwned(parseProvider, map[string]any{"value": "INNER"},
			CreateElement(parseConsumer, map[string]any{})))
	if parseGot := streamToString(t, parseNested); parseGot != "<span>INNER</span>" {
		t.Errorf("stream nested provider = %q, want <span>INNER</span>", parseGot)
	}
}
