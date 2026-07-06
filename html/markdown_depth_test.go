package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TestRenderMarkdownDeepNestingDoesNotOverflow pins that adversarially deep
// blockquote nesting is truncated instead of overflowing the recursive render
// stack (goldmark does not cap blockquote/list nesting depth). Without the depth
// guard this input crashes the process with a stack overflow.
func TestRenderMarkdownDeepNestingDoesNotOverflow(parseT *testing.T) {
	// Build "> > > ... text" nested far past maxMarkdownRenderDepth.
	parseDepth := maxMarkdownRenderDepth + 200
	parseMarkdown := strings.Repeat("> ", parseDepth) + "boom"

	// Must return (not panic/overflow) and produce some output.
	parseNodes := RenderMarkdown(parseMarkdown)
	if len(parseNodes) == 0 {
		parseT.Fatal("expected some rendered output for deeply nested markdown")
	}
}

// TestRenderMarkdownShallowNestingIsUnaffected pins that ordinary nesting well
// within the cap renders normally (the guard does not clip real documents).
func TestRenderMarkdownShallowNestingIsUnaffected(parseT *testing.T) {
	parseNodes := RenderMarkdown("> > > deeply but reasonably nested")
	if len(parseNodes) == 0 {
		parseT.Fatal("expected rendered output for shallow nesting")
	}
	parseMarkup, parseErr := ui.RenderToString(ui.Fragment(parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("render failed: %v", parseErr)
	}
	if strings.Contains(parseMarkup, "truncated") {
		parseT.Fatalf("shallow nesting must not be truncated, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "deeply but reasonably nested") {
		parseT.Fatalf("expected nested text to survive, got %q", parseMarkup)
	}
}
