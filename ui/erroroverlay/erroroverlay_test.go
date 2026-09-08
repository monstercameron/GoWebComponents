package erroroverlay_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui/erroroverlay"
)

// TestErrorOverlayRendersMessageHintAndStack renders the overlay headlessly and asserts the
// title, message, actionable hint, and stack frames all appear — the Elm-grade error surface.
func TestErrorOverlayRendersMessageHintAndStack(parseT *testing.T) {
	parseText := render(parseT, erroroverlay.ErrorOverlay(erroroverlay.Props{
		Title:   "Render error",
		Message: "nil pointer in UserCard",
		Hint:    "guard props.User before reading props.User.Name",
		Stack:   []string{"UserCard (user.go:42)", "App (app.go:10)"},
	}))

	for _, parseWant := range []string{
		"Render error", "nil pointer in UserCard", "Try:",
		"guard props.User", "UserCard (user.go:42)", "App (app.go:10)",
	} {
		if !strings.Contains(parseText, parseWant) {
			parseT.Fatalf("overlay DOM missing %q; got %q", parseWant, parseText)
		}
	}
}

// TestErrorOverlayDefaultsTitleAndOmitsOptional proves a bare error renders with a default
// title and no hint/stack sections.
func TestErrorOverlayDefaultsTitleAndOmitsOptional(parseT *testing.T) {
	parseText := render(parseT, erroroverlay.ErrorOverlay(erroroverlay.FromError(errors.New("boom"))))
	if !strings.Contains(parseText, "Error") || !strings.Contains(parseText, "boom") {
		parseT.Fatalf("expected default title + message, got %q", parseText)
	}
	if strings.Contains(parseText, "Try:") {
		parseT.Fatalf("no hint was supplied, the Try line should be absent: %q", parseText)
	}
}

// TestFromErrorNil proves FromError tolerates a nil error.
func TestFromErrorNil(parseT *testing.T) {
	if parseProps := erroroverlay.FromError(nil); parseProps.Message != "" {
		parseT.Fatalf("nil error should yield empty props, got %+v", parseProps)
	}
}

func render(parseT *testing.T, parseNode *runtime.Element) string {
	parseT.Helper()
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, parseNode); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	return collectText(parseAdapter, parseRoot)
}

func collectText(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode) string {
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return ""
	}
	parseText := parseMock.TextContent
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseText += collectText(parseAdapter, parseChild)
	}
	return parseText
}
