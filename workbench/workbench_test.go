package workbench_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/workbench"
)

// recordingT captures Errorf calls so the tests can assert what RunStories reported.
type recordingT struct {
	messages []string
}

func (parseR *recordingT) Helper() {}
func (parseR *recordingT) Errorf(parseFormat string, parseArgs ...any) {
	parseR.messages = append(parseR.messages, fmt.Sprintf(parseFormat, parseArgs...))
}

// TestRunStoriesPassesHealthyStories proves well-formed stories mount with no reported
// failures — the gallery doubling as a green smoke suite.
func TestRunStoriesPassesHealthyStories(parseT *testing.T) {
	parseRecorder := &recordingT{}
	workbench.RunStories(parseRecorder,
		workbench.Story{Name: "button", Render: func() ui.Node {
			return html.Button(html.Props{}, html.Text("Click me"))
		}},
		workbench.Story{Name: "card", Render: func() ui.Node {
			return html.Div(html.Props{Class: "card"}, html.Text("Body"))
		}},
	)
	if len(parseRecorder.messages) != 0 {
		parseT.Fatalf("expected healthy stories to pass, got failures: %v", parseRecorder.messages)
	}
}

// TestRunStoriesReportsPanic proves a story that panics is caught and reported, not allowed
// to crash the whole run.
func TestRunStoriesReportsPanic(parseT *testing.T) {
	parseRecorder := &recordingT{}
	workbench.RunStories(parseRecorder,
		workbench.Story{Name: "boom", Render: func() ui.Node { panic("kaboom") }},
		workbench.Story{Name: "after", Render: func() ui.Node { return html.Text("still runs") }},
	)
	if len(parseRecorder.messages) != 1 || !strings.Contains(parseRecorder.messages[0], "boom") {
		parseT.Fatalf("expected one panic report for boom, got %v", parseRecorder.messages)
	}
}

// TestRunStoriesReportsNilAndMissingRender proves nil renders and missing Render funcs are
// reported.
func TestRunStoriesReportsNilAndMissingRender(parseT *testing.T) {
	parseRecorder := &recordingT{}
	workbench.RunStories(parseRecorder,
		workbench.Story{Name: "nil", Render: func() ui.Node { return nil }},
		workbench.Story{Name: "missing"},
	)
	if len(parseRecorder.messages) != 2 {
		parseT.Fatalf("expected two reports (nil + missing), got %v", parseRecorder.messages)
	}
}
