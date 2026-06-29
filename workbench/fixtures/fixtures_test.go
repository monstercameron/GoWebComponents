package fixtures_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/workbench"
	"github.com/monstercameron/GoWebComponents/v4/workbench/fixtures"
)

// recordingT captures RunStories failures.
type recordingT struct{ messages []string }

func (parseR *recordingT) Helper() {}
func (parseR *recordingT) Errorf(parseFormat string, parseArgs ...any) {
	parseR.messages = append(parseR.messages, fmt.Sprintf(parseFormat, parseArgs...))
}

// TestBoundaryStoriesAllMount proves every boundary fixture mounts through the real
// reconciler without panic or render failure — the stories-as-tests guarantee for the
// async/error boundaries.
func TestBoundaryStoriesAllMount(parseT *testing.T) {
	parseStories := fixtures.BoundaryStories()
	if len(parseStories) < 4 {
		parseT.Fatalf("expected the async (pending/content/error) + error boundary fixtures, got %d", len(parseStories))
	}

	parseRecorder := &recordingT{}
	workbench.RunStories(parseRecorder, parseStories...)
	if len(parseRecorder.messages) != 0 {
		parseT.Fatalf("boundary fixtures should all mount cleanly, got failures: %v", parseRecorder.messages)
	}
}

// TestAllBoundaryStoriesMount proves every boundary surface — async/suspense, error,
// suspense-boundary, AND hydration-boundary (the two D1 added) — mounts through the real
// reconciler without panic, and that the suspense + hydration fixtures are actually present.
func TestAllBoundaryStoriesMount(parseT *testing.T) {
	parseStories := fixtures.AllStories()

	parseNames := map[string]bool{}
	for _, parseStory := range parseStories {
		parseNames[parseStory.Name] = true
	}
	for _, parseWant := range []string{
		"Suspense/fallback-while-pending",
		"HydrationBoundary/on-visible",
		"HydrationBoundary/immediate",
	} {
		if !parseNames[parseWant] {
			parseT.Fatalf("missing expected boundary fixture %q (have %v)", parseWant, parseNames)
		}
	}

	parseRecorder := &recordingT{}
	workbench.RunStories(parseRecorder, parseStories...)
	if len(parseRecorder.messages) != 0 {
		parseT.Fatalf("all boundary fixtures should mount cleanly, got failures: %v", parseRecorder.messages)
	}
}
