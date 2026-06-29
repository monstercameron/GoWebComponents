//go:build !js || !wasm

package ui

import "testing"

// TestRenderDeferRoutesEachBlock proves the @defer state machine renders the block matching the
// current status — placeholder, loading, content (with data), and error (with the error) — and
// never invokes the others. This is the sub-block routing the audit asked for (FA6).
func TestRenderDeferRoutesEachBlock(parseT *testing.T) {
	parseCases := []struct {
		name   string
		state  DeferState[string]
		expect string
	}{
		{"placeholder", DeferState[string]{Status: DeferPlaceholder}, "placeholder"},
		{"loading", DeferState[string]{Status: DeferLoading}, "loading"},
		{"ready", DeferState[string]{Status: DeferReady, Data: "report"}, "content:report"},
		{"error", DeferState[string]{Status: DeferError, Err: errDeferTest}, "error:boom"},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseRan := ""
			parseBlocks := DeferBlocks[string]{
				Placeholder: func() Node { parseRan = "placeholder"; return Text("p") },
				Loading:     func() Node { parseRan = "loading"; return Text("l") },
				Content:     func(parseData string) Node { parseRan = "content:" + parseData; return Text("c") },
				Error:       func(parseErr error) Node { parseRan = "error:" + parseErr.Error(); return Text("e") },
			}
			parseNode := RenderDefer(parseCase.state, parseBlocks)
			if parseNode == nil {
				parseT.Fatal("expected a rendered node for an active block")
			}
			if parseRan != parseCase.expect {
				parseT.Fatalf("expected block %q to run, but ran %q", parseCase.expect, parseRan)
			}
		})
	}
}

// TestRenderDeferNilBlockIsSafe proves a missing (nil) block renderer yields nil rather than
// panicking, so callers can omit optional placeholder/loading blocks.
func TestRenderDeferNilBlockIsSafe(parseT *testing.T) {
	if parseNode := RenderDefer(DeferState[int]{Status: DeferLoading}, DeferBlocks[int]{}); parseNode != nil {
		parseT.Fatalf("a nil Loading block should render nil, got %v", parseNode)
	}
	// Content present but in placeholder state → placeholder is nil → nil, Content not called.
	parseContentCalled := false
	parseNode := RenderDefer(DeferState[int]{Status: DeferPlaceholder}, DeferBlocks[int]{
		Content: func(int) Node { parseContentCalled = true; return Text("c") },
	})
	if parseNode != nil || parseContentCalled {
		parseT.Fatal("placeholder state with no placeholder block must render nil and not call Content")
	}
}

var errDeferTest = deferTestError("boom")

type deferTestError string

func (parseE deferTestError) Error() string { return string(parseE) }
