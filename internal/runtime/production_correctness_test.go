package runtime

import (
	"fmt"
	"testing"
)

func TestProductionCorrectness_ComposedAppFlowWithBoundaryPortalAndHydration(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseOverlay := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#overlay-root"] = parseOverlay

	parseServerRoot := parseAdapter.CreateElement("div")
	parseServerShell := parseAdapter.CreateElement("section")
	parseAdapter.SetAttribute(parseServerShell, "id", "shell")
	parseServerStatus := parseAdapter.CreateElement("p")
	parseAdapter.SetAttribute(parseServerStatus, "id", "draft-status")
	parseAdapter.AppendChild(parseServerStatus, parseAdapter.CreateTextNode("draft-0"))
	parseAdapter.AppendChild(parseServerShell, parseServerStatus)
	parseAdapter.AppendChild(parseServerRoot, parseServerShell)
	parseAdapter.AppendChild(parseContainer, parseServerRoot)

	parseBoundary := NewErrorBoundaryType()
	var (
		setOpen  func(interface{})
		setDraft func(interface{})
		setCrash func(interface{})
	)

	parseOverlayView := func() *Element {
		parseDraft, _ := GoUseAtom(parseRt, "draft", "draft-0")
		return CreateElement("aside", map[string]interface{}{"id": "overlay-draft"}, "overlay:"+parseDraft())
	}
	parseRiskyPanel := func() *Element {
		parseCrash, setCrashState := GoUseState(parseRt, false)
		setCrash = setCrashState
		if parseCrash() {
			panic("panel boom")
		}
		return CreateElement("button", map[string]interface{}{"id": "crash-button"}, "stable panel")
	}
	parseApp := func() *Element {
		parseOpen, setOpenState := GoUseState(parseRt, false)
		parseDraft2, setDraftState := GoUseAtom(parseRt, "draft", "draft-0")
		setOpen = setOpenState
		setDraft = setDraftState

		parseChildren := []interface{}{
			CreateElement("section", map[string]interface{}{"id": "shell"},
				CreateElement("p", map[string]interface{}{"id": "draft-status"}, parseDraft2()),
				CreateElement(parseBoundary, map[string]interface{}{
					"errorFallback": func(parseErr error, reset func()) *Element {
						return CreateElement("button", map[string]interface{}{
							"id": "boundary-reset",
							"onclick": func() {
								if setCrash != nil {
									setCrash(false)
								}
								reset()
							},
						}, "recover:"+parseErr.Error())
					},
				}, CreateElement(parseRiskyPanel, nil)),
			),
		}
		if parseOpen() {
			parseChildren = append(parseChildren, CreateElement(PortalNodeType, map[string]interface{}{
				"portalTargetSelector": "#overlay-root",
			}, CreateElement(parseOverlayView, nil)))
		}
		return CreateElement("div", nil, parseChildren...)
	}

	parseRt.Hydrate(CreateElement(parseApp, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if parseGot := findNodeByID(parseContainer, "draft-status"); parseGot == nil || nodeTextContent(parseGot) != "draft-0" {
		parseT.Fatalf("expected hydrated shell draft state, got %q", nodeTextContent(parseGot))
	}

	setOpen(true)
	drainScheduledTimeouts(parseT, parseScheduler, 64)
	if parseGot2 := findNodeByID(parseOverlay, "overlay-draft"); parseGot2 == nil || nodeTextContent(parseGot2) != "overlay:draft-0" {
		parseT.Fatalf("expected portal overlay to render with hydrated draft state, got %q", nodeTextContent(parseGot2))
	}

	setDraft("draft-1")
	drainScheduledTimeouts(parseT, parseScheduler, 64)
	if parseGot3 := findNodeByID(parseContainer, "draft-status"); parseGot3 == nil || nodeTextContent(parseGot3) != "draft-1" {
		parseT.Fatalf("expected shell to reflect shared atom update, got %q", nodeTextContent(parseGot3))
	}
	if parseGot4 := findNodeByID(parseOverlay, "overlay-draft"); parseGot4 == nil || nodeTextContent(parseGot4) != "overlay:draft-1" {
		parseT.Fatalf("expected overlay to reflect shared atom update, got %q", nodeTextContent(parseGot4))
	}

	setCrash(true)
	drainScheduledTimeouts(parseT, parseScheduler, 64)
	if parseGot5 := findNodeByID(parseContainer, "boundary-reset"); parseGot5 == nil || nodeTextContent(parseGot5) != "recover:panel boom" {
		parseT.Fatalf("expected boundary fallback after composed failure, got %q", nodeTextContent(parseGot5))
	}
	if parseGot6 := findNodeByID(parseContainer, "draft-status"); parseGot6 == nil || nodeTextContent(parseGot6) != "draft-1" {
		parseT.Fatalf("expected surrounding shell to survive boundary recovery, got %q", nodeTextContent(parseGot6))
	}

	resetButton := findNodeByID(parseContainer, "boundary-reset")
	if resetButton == nil {
		parseT.Fatal("expected boundary reset button")
	}
	invokeClick(parseT, resetButton)
	drainScheduledTimeouts(parseT, parseScheduler, 64)
	if parseGot7 := findNodeByID(parseContainer, "crash-button"); parseGot7 == nil || nodeTextContent(parseGot7) != "stable panel" {
		parseT.Fatalf("expected reset to restore risky panel, got %q", nodeTextContent(parseGot7))
	}

	if len(parseScheduler.timeouts) != 0 {
		parseT.Fatalf("expected composed flow to settle all scheduled work, found %d callbacks", len(parseScheduler.timeouts))
	}
}

func TestProductionCorrectness_MountUnmountChurnReleasesSubscribersAndRunsCleanup(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	var (
		setVisible         func(interface{})
		parseCleanupCount  int
		parseExpectedClean int
	)

	parseChild := func() *Element {
		parseValue, _ := GoUseAtom(parseRt, "session-theme", "light")
		GoUseEffect(func() func() {
			return func() {
				parseCleanupCount++
			}
		}, parseValue())
		return CreateElement("p", map[string]interface{}{"id": "child"}, parseValue())
	}
	parseApp := func() *Element {
		parseVisible, setVisibleState := GoUseState(parseRt, true)
		setVisible = setVisibleState
		if parseVisible() {
			return CreateElement("section", nil, CreateElement(parseChild, nil))
		}
		return CreateElement("section", nil, CreateElement("p", map[string]interface{}{"id": "empty"}, "hidden"))
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 64)

	for parseIteration := 0; parseIteration < 40; parseIteration++ {
		setVisible(false)
		drainScheduledTimeouts(parseT, parseScheduler, 64)
		parseExpectedClean++
		if parseGot := parseRt.atomRegistry.GetSubscriberCount("session-theme"); parseGot != 0 {
			parseT.Fatalf("iteration %d: expected no atom subscribers while hidden, got %d", parseIteration, parseGot)
		}

		setVisible(true)
		drainScheduledTimeouts(parseT, parseScheduler, 64)
		if parseGot2 := parseRt.atomRegistry.GetSubscriberCount("session-theme"); parseGot2 != 1 {
			parseT.Fatalf("iteration %d: expected one atom subscriber after remount, got %d", parseIteration, parseGot2)
		}
	}

	setVisible(false)
	drainScheduledTimeouts(parseT, parseScheduler, 64)
	parseExpectedClean++
	if parseGot3 := parseRt.atomRegistry.GetSubscriberCount("session-theme"); parseGot3 != 0 {
		parseT.Fatalf("expected no atom subscribers after final hide, got %d", parseGot3)
	}
	if parseCleanupCount != parseExpectedClean {
		parseT.Fatalf("expected %d cleanup calls after churn, got %d", parseExpectedClean, parseCleanupCount)
	}
	if len(parseScheduler.timeouts) != 0 {
		parseT.Fatalf("expected churn coverage to settle all scheduled work, found %d callbacks", len(parseScheduler.timeouts))
	}
}

func TestProductionCorrectness_OverlappingUrgentAndTransitionUpdatesSettleConsistently(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	var parseBurst func()
	parseApp := func() *Element {
		parseCount, setCount := GoUseState(parseRt, 0)
		parseShared, setShared := GoUseAtom(parseRt, "shared-burst", 0)
		parseBurst = func() {
			parseRt.StartTransition(func() {
				setCount(func(parsePrevious int) int { return parsePrevious + 1 })
				setShared(func(parsePrevious2 int) int { return parsePrevious2 + 1 })
			})
			setCount(func(parsePrevious3 int) int { return parsePrevious3 + 10 })
			setShared(func(parsePrevious4 int) int { return parsePrevious4 + 10 })
			parseRt.StartTransition(func() {
				setCount(func(parsePrevious5 int) int { return parsePrevious5 + 100 })
				setShared(func(parsePrevious6 int) int { return parsePrevious6 + 100 })
			})
		}

		return CreateElement("p", map[string]interface{}{"id": "status"}, fmt.Sprintf("%d/%d", parseCount(), parseShared()))
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 64)

	if parseBurst == nil {
		parseT.Fatal("expected burst handler to be installed")
	}
	for parseI := 0; parseI < 20; parseI++ {
		parseBurst()
	}
	drainScheduledTimeouts(parseT, parseScheduler, 512)

	parseStatus := findNodeByID(parseContainer, "status")
	if parseStatus == nil {
		parseT.Fatal("expected status node after burst updates")
	}
	const expected = "2220/2220"
	if parseGot := nodeTextContent(parseStatus); parseGot != expected {
		parseT.Fatalf("expected overlapping updates to settle to %q, got %q", expected, parseGot)
	}
	if parsePending, _ := parseRt.GetAtomValue(transitionPendingAtomID); parsePending != false {
		parseT.Fatalf("expected transition pending atom to clear after burst, got %#v", parsePending)
	}
	if len(parseScheduler.timeouts) != 0 {
		parseT.Fatalf("expected no leftover scheduled callbacks after burst, found %d", len(parseScheduler.timeouts))
	}
}
