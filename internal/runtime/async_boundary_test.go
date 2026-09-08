package runtime

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRenderToStringAsyncBoundarySuspensionFallback(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseChild := func() *Element {
		SuspendUntil(parseDone, "load profile")
		return CreateElement("span", nil, "ready")
	}

	parseMarkup, parseErr := RenderToString(CreateElement(AsyncBoundaryNodeType, map[string]any{
		"fallback": CreateElement("p", nil, "loading"),
	}, CreateElement(parseChild, nil)))
	if parseErr != nil {
		parseT.Fatalf("unexpected async boundary render error: %v", parseErr)
	}
	if parseMarkup != `<p>loading</p>` {
		parseT.Fatalf("expected suspension fallback markup, got %q", parseMarkup)
	}
}

func TestAsyncBoundaryRetriesAfterSuspensionResolves(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseDone := make(chan struct{})
	isReady := false

	parseChild := func() *Element {
		if !isReady {
			SuspendUntil(parseDone, "load profile")
		}
		return CreateElement("span", map[string]any{"id": "ready"}, "ready")
	}

	parseRt.Render(CreateElement(AsyncBoundaryNodeType, map[string]any{
		"fallback": CreateElement("p", map[string]any{"id": "fallback"}, "loading"),
	}, CreateElement(parseChild, nil)), parseContainer)
	flushScheduledWork(parseScheduler)

	if parseGot := nodeTextContent(parseContainer); parseGot != "loading" {
		parseT.Fatalf("expected initial fallback text, got %q", parseGot)
	}

	isReady = true
	close(parseDone)
	waitForAsyncBoundaryRetry(parseT, parseScheduler)
	flushScheduledWork(parseScheduler)

	if parseGot2 := nodeTextContent(parseContainer); parseGot2 != "ready" {
		parseT.Fatalf("expected ready text after suspension resolved, got %q", parseGot2)
	}
}

func TestSuspendUntilResolvedChannelDoesNotPanic(parseT *testing.T) {
	parseDone := make(chan struct{})
	close(parseDone)

	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseT.Fatalf("SuspendUntil panicked for resolved channel: %v", parseRecovered)
		}
	}()
	SuspendUntil(parseDone, "already ready")
}

func TestSuspensionHelpersAndStateMirroring(parseT *testing.T) {
	if parseGot := (*Suspension)(nil).Error(); parseGot != "render suspended" {
		parseT.Fatalf("nil Suspension.Error() = %q", parseGot)
	}
	if parseGot := (&Suspension{}).Error(); parseGot != "render suspended" {
		parseT.Fatalf("empty Suspension.Error() = %q", parseGot)
	}
	parseDone := make(chan struct{})
	parseSuspension := NewSuspension(parseDone, "")
	if parseSuspension.Error() != "render suspended" || parseSuspension.Done != parseDone {
		parseT.Fatalf("NewSuspension default = %#v", parseSuspension)
	}
	if parseGot := NewSuspension(parseDone, "load data").Error(); parseGot != "load data" {
		parseT.Fatalf("NewSuspension reason = %q", parseGot)
	}
	if parseGot, parseOK := AsSuspension(*parseSuspension); !parseOK || parseGot == parseSuspension || parseGot.Error() != "render suspended" {
		parseT.Fatalf("AsSuspension value = (%#v, %v)", parseGot, parseOK)
	}
	if parseGot, parseOK := AsSuspension(parseSuspension); !parseOK || parseGot != parseSuspension {
		parseT.Fatalf("AsSuspension pointer = (%#v, %v)", parseGot, parseOK)
	}
	if parseGot, parseOK := AsSuspension("nope"); parseOK || parseGot != nil {
		parseT.Fatalf("AsSuspension non-suspension = (%#v, %v)", parseGot, parseOK)
	}
	if suspensionResolved(nil) {
		parseT.Fatal("nil suspension should not be resolved")
	}
	if suspensionResolved(parseSuspension) {
		parseT.Fatal("open suspension should not be resolved")
	}
	close(parseDone)
	if !doneChannelClosed(parseDone) || !suspensionResolved(parseSuspension) {
		parseT.Fatal("closed suspension should be resolved")
	}

	parseRt := NewRuntime(Config{})
	parseAlternate := &Fiber{}
	parseBoundary := &Fiber{typeOf: AsyncBoundaryNodeType, alternate: parseAlternate}
	parseChild := &Fiber{parent: &Fiber{parent: parseBoundary}}
	if parseFound := findNearestAsyncBoundary(parseChild); parseFound != parseBoundary {
		parseT.Fatalf("findNearestAsyncBoundary() = %#v", parseFound)
	}
	if parseFound := findNearestAsyncBoundary(&Fiber{}); parseFound != nil {
		parseT.Fatalf("findNearestAsyncBoundary without boundary = %#v", parseFound)
	}
	parseRt.setAsyncBoundarySuspension(parseBoundary, parseSuspension)
	if parseBoundary.asyncSuspension != parseSuspension || parseAlternate.asyncSuspension != parseSuspension {
		parseT.Fatal("setAsyncBoundarySuspension should mirror to alternate")
	}
	if asyncBoundaryCapturedSuspension(parseBoundary) != parseSuspension {
		parseT.Fatal("asyncBoundaryCapturedSuspension should prefer current fiber")
	}
	parseBoundary.asyncSuspension = nil
	if asyncBoundaryCapturedSuspension(parseBoundary) != parseSuspension {
		parseT.Fatal("asyncBoundaryCapturedSuspension should fall back to alternate")
	}
	parseRt.clearAsyncBoundarySuspension(parseBoundary)
	if parseBoundary.asyncSuspension != nil || parseAlternate.asyncSuspension != nil {
		parseT.Fatal("clearAsyncBoundarySuspension should clear both generations")
	}
	parseRt.setAsyncBoundarySuspension(nil, parseSuspension)
	parseRt.clearAsyncBoundarySuspension(nil)
}

func TestAsyncBoundaryFallbackAndContentHelpers(parseT *testing.T) {
	parseErr := errors.New("boom")
	parseFallbackElement := CreateElement("p", nil, "fallback")
	parseBoundary := &Fiber{props: map[string]any{
		"fallback":      parseFallbackElement,
		"errorFallback": func(error) *Element { return CreateElement("strong", nil, "error") },
	}}
	if parseChildren := asyncBoundaryFallbackChildren(nil, parseErr); len(parseChildren) != 0 {
		parseT.Fatalf("nil boundary fallback children = %#v", parseChildren)
	}
	if parseChildren := asyncBoundaryFallbackChildren(parseBoundary, parseErr); len(parseChildren) != 1 || parseChildren[0].(*Element).Type != "strong" {
		parseT.Fatalf("error fallback children = %#v", parseChildren)
	}
	if parseChildren := asyncBoundaryFallbackChildren(parseBoundary, nil); len(parseChildren) != 1 || parseChildren[0] != parseFallbackElement {
		parseT.Fatalf("plain fallback children = %#v", parseChildren)
	}
	if parseChildren := singleElementChild(nil); len(parseChildren) != 0 {
		parseT.Fatalf("singleElementChild(nil) = %#v", parseChildren)
	}

	parseContent := CreateElement("section", nil)
	if parseGot := asyncBoundaryContent(&Fiber{props: map[string]any{"content": parseContent}}); parseGot != parseContent {
		parseT.Fatalf("asyncBoundaryContent prop = %#v", parseGot)
	}
	if parseGot := asyncBoundaryContent(&Fiber{}); parseGot != nil {
		parseT.Fatalf("asyncBoundaryContent empty = %#v", parseGot)
	}
	if parseGot := safeAsyncBoundaryFallback(func(error) *Element { return parseFallbackElement }, parseErr); parseGot != parseFallbackElement {
		parseT.Fatalf("safeAsyncBoundaryFallback = %#v", parseGot)
	}
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil || !strings.Contains(parseRecovered.(error).Error(), "async boundary fallback panic") {
			parseT.Fatalf("expected wrapped fallback panic, got %#v", parseRecovered)
		}
	}()
	_ = safeAsyncBoundaryFallback(func(error) *Element { panic("bad fallback") }, parseErr)
}

func waitForAsyncBoundaryRetry(parseT *testing.T, parseScheduler *testScheduler) {
	parseT.Helper()
	parseDeadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(parseDeadline) {
		if parseScheduler.getPendingTimeoutCount() > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	parseT.Fatal("timed out waiting for async boundary retry")
}
