package runtime

import (
	"fmt"
	"testing"
)

func TestProductionCorrectness_ComposedAppFlowWithBoundaryPortalAndHydration(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	overlay := adapter.CreateElement("div")
	adapter.selectorResults["#overlay-root"] = overlay

	serverRoot := adapter.CreateElement("div")
	serverShell := adapter.CreateElement("section")
	adapter.SetAttribute(serverShell, "id", "shell")
	serverStatus := adapter.CreateElement("p")
	adapter.SetAttribute(serverStatus, "id", "draft-status")
	adapter.AppendChild(serverStatus, adapter.CreateTextNode("draft-0"))
	adapter.AppendChild(serverShell, serverStatus)
	adapter.AppendChild(serverRoot, serverShell)
	adapter.AppendChild(container, serverRoot)

	boundary := NewErrorBoundaryType()
	var (
		setOpen  func(interface{})
		setDraft func(interface{})
		setCrash func(interface{})
	)

	overlayView := func() *Element {
		draft, _ := GoUseAtom(rt, "draft", "draft-0")
		return CreateElement("aside", map[string]interface{}{"id": "overlay-draft"}, "overlay:"+draft())
	}
	riskyPanel := func() *Element {
		crash, setCrashState := GoUseState(rt, false)
		setCrash = setCrashState
		if crash() {
			panic("panel boom")
		}
		return CreateElement("button", map[string]interface{}{"id": "crash-button"}, "stable panel")
	}
	app := func() *Element {
		open, setOpenState := GoUseState(rt, false)
		draft, setDraftState := GoUseAtom(rt, "draft", "draft-0")
		setOpen = setOpenState
		setDraft = setDraftState

		children := []interface{}{
			CreateElement("section", map[string]interface{}{"id": "shell"},
				CreateElement("p", map[string]interface{}{"id": "draft-status"}, draft()),
				CreateElement(boundary, map[string]interface{}{
					"errorFallback": func(err error, reset func()) *Element {
						return CreateElement("button", map[string]interface{}{
							"id": "boundary-reset",
							"onclick": func() {
								if setCrash != nil {
									setCrash(false)
								}
								reset()
							},
						}, "recover:"+err.Error())
					},
				}, CreateElement(riskyPanel, nil)),
			),
		}
		if open() {
			children = append(children, CreateElement(PortalNodeType, map[string]interface{}{
				"portalTargetSelector": "#overlay-root",
			}, CreateElement(overlayView, nil)))
		}
		return CreateElement("div", nil, children...)
	}

	rt.Hydrate(CreateElement(app, nil), container)
	runHydrationWork(t, scheduler)

	if got := findNodeByID(container, "draft-status"); got == nil || nodeTextContent(got) != "draft-0" {
		t.Fatalf("expected hydrated shell draft state, got %q", nodeTextContent(got))
	}

	setOpen(true)
	drainScheduledTimeouts(t, scheduler, 64)
	if got := findNodeByID(overlay, "overlay-draft"); got == nil || nodeTextContent(got) != "overlay:draft-0" {
		t.Fatalf("expected portal overlay to render with hydrated draft state, got %q", nodeTextContent(got))
	}

	setDraft("draft-1")
	drainScheduledTimeouts(t, scheduler, 64)
	if got := findNodeByID(container, "draft-status"); got == nil || nodeTextContent(got) != "draft-1" {
		t.Fatalf("expected shell to reflect shared atom update, got %q", nodeTextContent(got))
	}
	if got := findNodeByID(overlay, "overlay-draft"); got == nil || nodeTextContent(got) != "overlay:draft-1" {
		t.Fatalf("expected overlay to reflect shared atom update, got %q", nodeTextContent(got))
	}

	setCrash(true)
	drainScheduledTimeouts(t, scheduler, 64)
	if got := findNodeByID(container, "boundary-reset"); got == nil || nodeTextContent(got) != "recover:panel boom" {
		t.Fatalf("expected boundary fallback after composed failure, got %q", nodeTextContent(got))
	}
	if got := findNodeByID(container, "draft-status"); got == nil || nodeTextContent(got) != "draft-1" {
		t.Fatalf("expected surrounding shell to survive boundary recovery, got %q", nodeTextContent(got))
	}

	resetButton := findNodeByID(container, "boundary-reset")
	if resetButton == nil {
		t.Fatal("expected boundary reset button")
	}
	invokeClick(t, resetButton)
	drainScheduledTimeouts(t, scheduler, 64)
	if got := findNodeByID(container, "crash-button"); got == nil || nodeTextContent(got) != "stable panel" {
		t.Fatalf("expected reset to restore risky panel, got %q", nodeTextContent(got))
	}

	if len(scheduler.timeouts) != 0 {
		t.Fatalf("expected composed flow to settle all scheduled work, found %d callbacks", len(scheduler.timeouts))
	}
}

func TestProductionCorrectness_MountUnmountChurnReleasesSubscribersAndRunsCleanup(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	var (
		setVisible    func(interface{})
		cleanupCount  int
		expectedClean int
	)

	child := func() *Element {
		value, _ := GoUseAtom(rt, "session-theme", "light")
		GoUseEffect(func() func() {
			return func() {
				cleanupCount++
			}
		}, value())
		return CreateElement("p", map[string]interface{}{"id": "child"}, value())
	}
	app := func() *Element {
		visible, setVisibleState := GoUseState(rt, true)
		setVisible = setVisibleState
		if visible() {
			return CreateElement("section", nil, CreateElement(child, nil))
		}
		return CreateElement("section", nil, CreateElement("p", map[string]interface{}{"id": "empty"}, "hidden"))
	}

	rt.Render(CreateElement(app, nil), container)
	drainScheduledTimeouts(t, scheduler, 64)

	for iteration := 0; iteration < 40; iteration++ {
		setVisible(false)
		drainScheduledTimeouts(t, scheduler, 64)
		expectedClean++
		if got := rt.atomRegistry.GetSubscriberCount("session-theme"); got != 0 {
			t.Fatalf("iteration %d: expected no atom subscribers while hidden, got %d", iteration, got)
		}

		setVisible(true)
		drainScheduledTimeouts(t, scheduler, 64)
		if got := rt.atomRegistry.GetSubscriberCount("session-theme"); got != 1 {
			t.Fatalf("iteration %d: expected one atom subscriber after remount, got %d", iteration, got)
		}
	}

	setVisible(false)
	drainScheduledTimeouts(t, scheduler, 64)
	expectedClean++
	if got := rt.atomRegistry.GetSubscriberCount("session-theme"); got != 0 {
		t.Fatalf("expected no atom subscribers after final hide, got %d", got)
	}
	if cleanupCount != expectedClean {
		t.Fatalf("expected %d cleanup calls after churn, got %d", expectedClean, cleanupCount)
	}
	if len(scheduler.timeouts) != 0 {
		t.Fatalf("expected churn coverage to settle all scheduled work, found %d callbacks", len(scheduler.timeouts))
	}
}

func TestProductionCorrectness_OverlappingUrgentAndTransitionUpdatesSettleConsistently(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	var burst func()
	app := func() *Element {
		count, setCount := GoUseState(rt, 0)
		shared, setShared := GoUseAtom(rt, "shared-burst", 0)
		burst = func() {
			rt.StartTransition(func() {
				setCount(func(previous int) int { return previous + 1 })
				setShared(func(previous int) int { return previous + 1 })
			})
			setCount(func(previous int) int { return previous + 10 })
			setShared(func(previous int) int { return previous + 10 })
			rt.StartTransition(func() {
				setCount(func(previous int) int { return previous + 100 })
				setShared(func(previous int) int { return previous + 100 })
			})
		}

		return CreateElement("p", map[string]interface{}{"id": "status"}, fmt.Sprintf("%d/%d", count(), shared()))
	}

	rt.Render(CreateElement(app, nil), container)
	drainScheduledTimeouts(t, scheduler, 64)

	if burst == nil {
		t.Fatal("expected burst handler to be installed")
	}
	for i := 0; i < 20; i++ {
		burst()
	}
	drainScheduledTimeouts(t, scheduler, 512)

	status := findNodeByID(container, "status")
	if status == nil {
		t.Fatal("expected status node after burst updates")
	}
	const expected = "2220/2220"
	if got := nodeTextContent(status); got != expected {
		t.Fatalf("expected overlapping updates to settle to %q, got %q", expected, got)
	}
	if pending, _ := rt.GetAtomValue(transitionPendingAtomID); pending != false {
		t.Fatalf("expected transition pending atom to clear after burst, got %#v", pending)
	}
	if len(scheduler.timeouts) != 0 {
		t.Fatalf("expected no leftover scheduled callbacks after burst, found %d", len(scheduler.timeouts))
	}
}
