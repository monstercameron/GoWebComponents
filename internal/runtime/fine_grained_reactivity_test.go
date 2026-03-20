package runtime

import (
	"strings"
	"testing"
)

func runScheduledTimeouts(scheduler *testScheduler) {
	for len(scheduler.timeouts) > 0 {
		callback := scheduler.timeouts[0]
		scheduler.timeouts = scheduler.timeouts[1:]
		callback()
	}
}

func TestScheduleGranularUpdateForFiber_DoesNotDirtyAncestors(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	grandparent := &Fiber{typeOf: "grandparent"}
	parent := &Fiber{typeOf: "parent", parent: grandparent}
	child := &Fiber{typeOf: ReactiveTextNodeType, parent: parent, fineGrained: true}

	rt.ScheduleGranularUpdateForFiber(child)

	if !child.dirty || !child.needsUpdate {
		t.Fatal("expected child to be marked for granular update")
	}
	if parent.dirty || parent.needsUpdate {
		t.Fatal("expected parent to remain clean during granular update")
	}
	if grandparent.dirty || grandparent.needsUpdate {
		t.Fatal("expected grandparent to remain clean during granular update")
	}
}

func TestReactiveTextAtomUpdate_DoesNotRerenderOwnerComponent(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	app := func() *Element {
		renderCount++
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]interface{}{
				reactiveTextAtomIDProp: "count",
				reactiveTextGetterProp: func() string {
					value, _ := rt.atomRegistry.GetAtom("count")
					if count, ok := value.(int); ok {
						return string(rune('0' + count))
					}
					return "?"
				},
			}),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	if renderCount != 1 {
		t.Fatalf("expected one initial owner render, got %d", renderCount)
	}
	parent := container.(*testDOMNode)
	if len(parent.children) != 1 {
		t.Fatalf("expected root container child, got %d", len(parent.children))
	}
	host := parent.children[0].(*testDOMNode)
	if len(host.children) != 1 {
		t.Fatalf("expected host child text node, got %d", len(host.children))
	}
	textNode := host.children[0].(*testDOMNode)
	if textNode.text != "0" {
		t.Fatalf("expected initial reactive text 0, got %q", textNode.text)
	}

	if err := rt.SetAtomValue("count", 1); err != nil {
		t.Fatalf("unexpected atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if renderCount != 1 {
		t.Fatalf("expected owner component render count to stay at 1, got %d", renderCount)
	}
	if textNode.text != "1" {
		t.Fatalf("expected reactive text node to update to 1, got %q", textNode.text)
	}

	if err := rt.SetAtomValue("count", 2); err != nil {
		t.Fatalf("unexpected second atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if renderCount != 1 {
		t.Fatalf("expected owner component render count to still be 1 after second update, got %d", renderCount)
	}
	if textNode.text != "2" {
		t.Fatalf("expected reactive text node to update to 2 on second update, got %q", textNode.text)
	}
}

func TestReactiveTextAtomUpdate_PreservesSiblingDomSubtrees(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("count", 0)

	appRenderCount := 0
	staticRenderCount := 0

	staticPanel := func() *Element {
		staticRenderCount++
		return CreateElement("section", nil,
			CreateElement("strong", nil, "stable"),
		)
	}

	app := func() *Element {
		appRenderCount++
		return CreateElement("main", nil,
			CreateElement("div", map[string]interface{}{"id": "static"},
				CreateElement(staticPanel, nil),
			),
			CreateElement("div", map[string]interface{}{"id": "hot"},
				CreateElement("span", nil,
					CreateElement(ReactiveTextNodeType, map[string]interface{}{
						reactiveTextAtomIDProp: "count",
						reactiveTextGetterProp: func() string {
							value, _ := rt.atomRegistry.GetAtom("count")
							if count, ok := value.(int); ok {
								return string(rune('0' + count))
							}
							return "?"
						},
					}),
				),
			),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	root := container.(*testDOMNode)
	mainNode := root.children[0].(*testDOMNode)
	staticWrapBefore := mainNode.children[0].(*testDOMNode)
	staticSectionBefore := staticWrapBefore.children[0].(*testDOMNode)
	staticStrongBefore := staticSectionBefore.children[0].(*testDOMNode)
	hotWrapBefore := mainNode.children[1].(*testDOMNode)
	hotSpanBefore := hotWrapBefore.children[0].(*testDOMNode)
	hotTextBefore := hotSpanBefore.children[0].(*testDOMNode)

	if err := rt.SetAtomValue("count", 4); err != nil {
		t.Fatalf("unexpected atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if appRenderCount != 1 {
		t.Fatalf("expected app component to stay at one render, got %d", appRenderCount)
	}
	if staticRenderCount != 1 {
		t.Fatalf("expected unrelated sibling component to stay at one render, got %d", staticRenderCount)
	}

	mainAfter := root.children[0].(*testDOMNode)
	staticWrapAfter := mainAfter.children[0].(*testDOMNode)
	staticSectionAfter := staticWrapAfter.children[0].(*testDOMNode)
	staticStrongAfter := staticSectionAfter.children[0].(*testDOMNode)
	hotWrapAfter := mainAfter.children[1].(*testDOMNode)
	hotSpanAfter := hotWrapAfter.children[0].(*testDOMNode)
	hotTextAfter := hotSpanAfter.children[0].(*testDOMNode)

	if staticWrapAfter != staticWrapBefore || staticSectionAfter != staticSectionBefore || staticStrongAfter != staticStrongBefore {
		t.Fatal("expected unrelated static subtree DOM identity to remain stable during fine-grained update")
	}
	if hotWrapAfter != hotWrapBefore || hotSpanAfter != hotSpanBefore || hotTextAfter != hotTextBefore {
		t.Fatal("expected fine-grained update to mutate the existing hot DOM subtree instead of replacing it")
	}
	if staticStrongAfter.children[0].(*testDOMNode).text != "stable" {
		t.Fatalf("expected static subtree text to remain stable, got %q", staticStrongAfter.children[0].(*testDOMNode).text)
	}
	if hotTextAfter.text != "4" {
		t.Fatalf("expected hot text subtree to update in place to 4, got %q", hotTextAfter.text)
	}
	if rt.profiling.scheduledGranularMarks == 0 {
		t.Fatal("expected fine-grained update to record a granular mark")
	}
	if rt.profiling.fineGrainedCommits == 0 {
		t.Fatal("expected fine-grained update to record a granular commit")
	}
}

func TestReactiveTextMultipleRegions_UpdateOnlyTargetDomTree(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("left", 1)
	rt.atomRegistry.InitAtom("right", 8)

	renderCount := 0
	app := func() *Element {
		renderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]interface{}{"id": "left"},
				CreateElement(ReactiveTextNodeType, map[string]interface{}{
					reactiveTextAtomIDProp: "left",
					reactiveTextGetterProp: func() string {
						value, _ := rt.atomRegistry.GetAtom("left")
						if count, ok := value.(int); ok {
							return string(rune('0' + count))
						}
						return "?"
					},
				}),
			),
			CreateElement("div", map[string]interface{}{"id": "right"},
				CreateElement(ReactiveTextNodeType, map[string]interface{}{
					reactiveTextAtomIDProp: "right",
					reactiveTextGetterProp: func() string {
						value, _ := rt.atomRegistry.GetAtom("right")
						if count, ok := value.(int); ok {
							return string(rune('0' + count))
						}
						return "?"
					},
				}),
			),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	root := container.(*testDOMNode)
	section := root.children[0].(*testDOMNode)
	leftWrapBefore := section.children[0].(*testDOMNode)
	leftTextBefore := leftWrapBefore.children[0].(*testDOMNode)
	rightWrapBefore := section.children[1].(*testDOMNode)
	rightTextBefore := rightWrapBefore.children[0].(*testDOMNode)

	if err := rt.SetAtomValue("left", 2); err != nil {
		t.Fatalf("unexpected left atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if renderCount != 1 {
		t.Fatalf("expected owning component to avoid rerender on left-region update, got %d renders", renderCount)
	}
	leftWrapAfterLeft := section.children[0].(*testDOMNode)
	leftTextAfterLeft := leftWrapAfterLeft.children[0].(*testDOMNode)
	rightWrapAfterLeft := section.children[1].(*testDOMNode)
	rightTextAfterLeft := rightWrapAfterLeft.children[0].(*testDOMNode)
	if leftWrapAfterLeft != leftWrapBefore || leftTextAfterLeft != leftTextBefore {
		t.Fatal("expected left reactive region to update in place")
	}
	if rightWrapAfterLeft != rightWrapBefore || rightTextAfterLeft != rightTextBefore {
		t.Fatal("expected right region DOM tree to remain untouched when left region updates")
	}
	if leftTextAfterLeft.text != "2" || rightTextAfterLeft.text != "8" {
		t.Fatalf("expected left/right texts to be 2 and 8, got %q and %q", leftTextAfterLeft.text, rightTextAfterLeft.text)
	}

	if err := rt.SetAtomValue("right", 9); err != nil {
		t.Fatalf("unexpected right atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	leftWrapAfterRight := section.children[0].(*testDOMNode)
	leftTextAfterRight := leftWrapAfterRight.children[0].(*testDOMNode)
	rightWrapAfterRight := section.children[1].(*testDOMNode)
	rightTextAfterRight := rightWrapAfterRight.children[0].(*testDOMNode)
	if leftWrapAfterRight != leftWrapBefore || leftTextAfterRight != leftTextBefore {
		t.Fatal("expected left region DOM tree to remain untouched when right region updates")
	}
	if rightWrapAfterRight != rightWrapBefore || rightTextAfterRight != rightTextBefore {
		t.Fatal("expected right reactive region to update in place")
	}
	if leftTextAfterRight.text != "2" || rightTextAfterRight.text != "9" {
		t.Fatalf("expected left/right texts to be 2 and 9, got %q and %q", leftTextAfterRight.text, rightTextAfterRight.text)
	}
}

func TestReactiveTextCollision_HookRerenderWinsAndKeepsTreeCoherent(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	var setLabel func(interface{})
	app := func() *Element {
		renderCount++
		label, set := GoUseState(rt, "ready")
		setLabel = set
		return CreateElement("section", nil,
			CreateElement("h1", nil, label()),
			CreateElement("span", nil,
				CreateElement(ReactiveTextNodeType, map[string]interface{}{
					reactiveTextAtomIDProp: "count",
					reactiveTextGetterProp: func() string {
						value, _ := rt.atomRegistry.GetAtom("count")
						if count, ok := value.(int); ok {
							return string(rune('0' + count))
						}
						return "?"
					},
				}),
			),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	if setLabel == nil {
		t.Fatal("expected owner component state setter to be captured")
	}
	initialCommitCount := rt.profiling.commitCount

	if err := rt.SetAtomValue("count", 5); err != nil {
		t.Fatalf("unexpected fine-grained atom update error: %v", err)
	}
	setLabel("updated")
	runScheduledTimeouts(scheduler)

	if renderCount != 2 {
		t.Fatalf("expected exactly one owner rerender during mixed hook/fine-grained collision, got %d renders", renderCount)
	}
	root := container.(*testDOMNode)
	section := root.children[0].(*testDOMNode)
	headline := section.children[0].(*testDOMNode)
	headlineText := headline.children[0].(*testDOMNode)
	hotSpan := section.children[1].(*testDOMNode)
	hotText := hotSpan.children[0].(*testDOMNode)
	if headlineText.text != "updated" {
		t.Fatalf("expected hook-driven label update to commit coherently, got %q", headlineText.text)
	}
	if hotText.text != "5" {
		t.Fatalf("expected fine-grained text to match the latest atom value during collision, got %q", hotText.text)
	}
	if rt.profiling.commitCount != initialCommitCount+1 {
		t.Fatalf("expected mixed update window to coalesce into one follow-up commit, got %d commits from %d", rt.profiling.commitCount, initialCommitCount)
	}
}

func TestReactiveTextTransitionUpdate_DefersUntilTimeout(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	app := func() *Element {
		renderCount++
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]interface{}{
				reactiveTextAtomIDProp: "count",
				reactiveTextGetterProp: func() string {
					value, _ := rt.atomRegistry.GetAtom("count")
					if count, ok := value.(int); ok {
						return string(rune('0' + count))
					}
					return "?"
				},
			}),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	root := container.(*testDOMNode)
	host := root.children[0].(*testDOMNode)
	textNode := host.children[0].(*testDOMNode)
	initialCommitCount := rt.profiling.commitCount
	initialGranularMarks := rt.profiling.scheduledGranularMarks

	rt.StartTransition(func() {
		rt.ScheduleTransition(func() {
			rt.atomRegistry.setAtomAndNotify("count", 3, rt.ScheduleSubscribedFiberUpdate)
		})
	})

	if textNode.text != "0" {
		t.Fatalf("expected deferred fine-grained transition update to keep text at 0 before timeout, got %q", textNode.text)
	}
	if renderCount != 1 {
		t.Fatalf("expected owner component to stay at one render before transition timeout, got %d", renderCount)
	}
	if rt.profiling.commitCount != initialCommitCount {
		t.Fatalf("expected no new commit before transition timeout, got %d from %d", rt.profiling.commitCount, initialCommitCount)
	}
	if rt.profiling.scheduledGranularMarks != initialGranularMarks {
		t.Fatalf("expected no granular marks before transition timeout, got %d from %d", rt.profiling.scheduledGranularMarks, initialGranularMarks)
	}
	if len(scheduler.timeouts) == 0 {
		t.Fatal("expected deferred transition timeout for fine-grained update")
	}

	runScheduledTimeouts(scheduler)

	if textNode.text != "3" {
		t.Fatalf("expected deferred fine-grained update to apply after timeout, got %q", textNode.text)
	}
	if renderCount != 1 {
		t.Fatalf("expected owner component to avoid rerender after deferred fine-grained update, got %d", renderCount)
	}
	if rt.profiling.commitCount <= initialCommitCount {
		t.Fatal("expected deferred fine-grained update to produce a follow-up commit")
	}
	if rt.profiling.scheduledGranularMarks <= initialGranularMarks {
		t.Fatal("expected deferred fine-grained update to record a granular mark after timeout")
	}
}

func TestHydratedReactiveText_UpdateAfterResumeStaysNarrow(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("p")
	serverText := adapter.CreateTextNode("0")
	adapter.AppendChild(serverNode, serverText)
	adapter.AppendChild(container, serverNode)
	rt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	app := func() *Element {
		renderCount++
		return CreateElement("p", nil,
			CreateElement(ReactiveTextNodeType, map[string]interface{}{
				reactiveTextAtomIDProp: "count",
				reactiveTextGetterProp: func() string {
					value, _ := rt.atomRegistry.GetAtom("count")
					if count, ok := value.(int); ok {
						return string(rune('0' + count))
					}
					return "?"
				},
			}),
		)
	}

	rt.Hydrate(CreateElement(app, nil), container)
	runHydrationWork(t, scheduler)

	children := adapter.GetChildren(container)
	if len(children) != 1 || !children[0].Equals(serverNode) {
		t.Fatal("expected hydrated fine-grained host node to reuse the existing server DOM node")
	}
	textChildren := adapter.GetChildren(serverNode)
	if len(textChildren) != 1 || !textChildren[0].Equals(serverText) {
		t.Fatal("expected hydrated fine-grained text node to reuse the existing server text node")
	}
	initialCommitCount := rt.profiling.commitCount

	if err := rt.SetAtomValue("count", 4); err != nil {
		t.Fatalf("unexpected post-hydration atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if renderCount != 1 {
		t.Fatalf("expected hydrated owner component to avoid rerender on fine-grained post-resume update, got %d", renderCount)
	}
	if got := serverText.(*testDOMNode).text; got != "4" {
		t.Fatalf("expected hydrated reactive text node to update in place to 4, got %q", got)
	}
	if rt.profiling.commitCount <= initialCommitCount {
		t.Fatal("expected post-hydration fine-grained update to produce a follow-up commit")
	}
}

func TestReactiveTextSourceSwap_AvoidsStaleReadsAndOldSourceUpdates(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("primary", 0)
	rt.atomRegistry.InitAtom("secondary", 9)

	app := func(props map[string]interface{}) *Element {
		atomID, _ := props["source"].(string)
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]interface{}{
				reactiveTextAtomIDProp: atomID,
				reactiveTextGetterProp: func() string {
					value, _ := rt.atomRegistry.GetAtom(atomID)
					if count, ok := value.(int); ok {
						return string(rune('0' + count))
					}
					return "?"
				},
			}),
		)
	}

	rt.Render(CreateElement(app, map[string]interface{}{"source": "primary"}), container)
	runScheduledTimeouts(scheduler)

	if rt.atomRegistry.GetSubscriberCount("primary") != 1 {
		t.Fatalf("expected primary atom to have 1 subscriber after initial render, got %d", rt.atomRegistry.GetSubscriberCount("primary"))
	}
	if rt.atomRegistry.GetSubscriberCount("secondary") != 0 {
		t.Fatalf("expected secondary atom to have 0 subscribers before source swap, got %d", rt.atomRegistry.GetSubscriberCount("secondary"))
	}

	rt.Render(CreateElement(app, map[string]interface{}{"source": "secondary"}), container)
	runScheduledTimeouts(scheduler)

	if rt.atomRegistry.GetSubscriberCount("primary") != 0 {
		t.Fatalf("expected primary atom subscription to be released after source swap, got %d", rt.atomRegistry.GetSubscriberCount("primary"))
	}
	if rt.atomRegistry.GetSubscriberCount("secondary") != 1 {
		t.Fatalf("expected secondary atom to have 1 subscriber after source swap, got %d", rt.atomRegistry.GetSubscriberCount("secondary"))
	}

	parent := container.(*testDOMNode)
	host := parent.children[0].(*testDOMNode)
	textNode := host.children[0].(*testDOMNode)
	if textNode.text != "9" {
		t.Fatalf("expected swapped reactive text node to show secondary value 9, got %q", textNode.text)
	}

	if err := rt.SetAtomValue("primary", 4); err != nil {
		t.Fatalf("unexpected primary atom update error after source swap: %v", err)
	}
	runScheduledTimeouts(scheduler)
	if textNode.text != "9" {
		t.Fatalf("expected old source update to be ignored after source swap, got %q", textNode.text)
	}

	if err := rt.SetAtomValue("secondary", 7); err != nil {
		t.Fatalf("unexpected secondary atom update error after source swap: %v", err)
	}
	runScheduledTimeouts(scheduler)
	if textNode.text != "7" {
		t.Fatalf("expected new source update to drive reactive text after source swap, got %q", textNode.text)
	}
}

func TestReactiveTextDerivedCycleFailsSafely(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}

	if err := rt.RegisterDerivedAtom("derived-a", []string{"derived-b"}, func() interface{} {
		value, _ := rt.GetAtomValue("derived-b")
		return value
	}); err != nil {
		t.Fatalf("unexpected error registering first derived atom: %v", err)
	}
	if err := rt.RegisterDerivedAtom("derived-b", []string{"derived-a"}, func() interface{} {
		value, _ := rt.GetAtomValue("derived-a")
		return value
	}); err == nil {
		t.Fatal("expected derived cycle registration to fail")
	}

	subscriber := &Fiber{typeOf: ReactiveTextNodeType, fineGrained: true}
	rt.atomRegistry.Subscribe("derived-b", subscriber)

	if subscriber.needsUpdate || subscriber.dirty {
		t.Fatal("expected rejected derived cycle to avoid notifying subscribed fine-grained fibers")
	}

	diagnostics := GetDiagnostics()
	if len(diagnostics) == 0 {
		t.Fatal("expected derived cycle to be reported as a diagnostic")
	}
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "derived atom cycle detected") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected derived cycle diagnostic, got %+v", diagnostics)
	}
}

func TestReactiveTextUnmount_CleansUpDeletedSubtreeSubscriptions(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("count", 0)

	app := func(props map[string]interface{}) *Element {
		children := []interface{}{}
		showReactive, _ := props["showReactive"].(bool)
		if showReactive {
			children = append(children, CreateElement("span", nil,
				CreateElement(ReactiveTextNodeType, map[string]interface{}{
					reactiveTextAtomIDProp: "count",
					reactiveTextGetterProp: func() string {
						value, _ := rt.atomRegistry.GetAtom("count")
						if count, ok := value.(int); ok {
							return string(rune('0' + count))
						}
						return "?"
					},
				}),
			))
		} else {
			children = append(children, CreateElement("span", nil, "gone"))
		}
		return CreateElement("div", nil, children...)
	}

	rt.Render(CreateElement(app, map[string]interface{}{"showReactive": true}), container)
	runScheduledTimeouts(scheduler)

	if rt.atomRegistry.GetSubscriberCount("count") != 1 {
		t.Fatalf("expected count atom to have 1 fine-grained subscriber before unmount, got %d", rt.atomRegistry.GetSubscriberCount("count"))
	}

	rt.Render(CreateElement(app, map[string]interface{}{"showReactive": false}), container)
	runScheduledTimeouts(scheduler)

	if rt.atomRegistry.GetSubscriberCount("count") != 0 {
		t.Fatalf("expected deleted fine-grained subtree to release count subscription, got %d", rt.atomRegistry.GetSubscriberCount("count"))
	}

	if err := rt.SetAtomValue("count", 3); err != nil {
		t.Fatalf("unexpected atom update error after reactive subtree unmount: %v", err)
	}
	runScheduledTimeouts(scheduler)

	parent := container.(*testDOMNode)
	host := parent.children[0].(*testDOMNode)
	fallback := host.children[0].(*testDOMNode)
	textNode := fallback.children[0].(*testDOMNode)
	if textNode.text != "gone" {
		t.Fatalf("expected fallback subtree to remain stable after old subscription source updated, got %q", textNode.text)
	}
}

func TestReactiveTextDerivedProjection_UnchangedValueSkipsFineGrainedCommit(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("count", 1)

	if err := rt.RegisterDerivedAtom("parity", []string{"count"}, func() interface{} {
		value, _ := rt.GetAtomValue("count")
		if value.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	}); err != nil {
		t.Fatalf("unexpected derived parity registration error: %v", err)
	}

	appRenderCount := 0
	app := func() *Element {
		appRenderCount++
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]interface{}{
				reactiveTextAtomIDProp: "parity",
				reactiveTextGetterProp: func() string {
					value, _ := rt.GetAtomValue("parity")
					if label, ok := value.(string); ok {
						return label
					}
					return "?"
				},
			}),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	root := container.(*testDOMNode)
	host := root.children[0].(*testDOMNode)
	textNode := host.children[0].(*testDOMNode)
	if textNode.text != "odd" {
		t.Fatalf("expected initial projected text odd, got %q", textNode.text)
	}
	initialCommitCount := rt.profiling.commitCount
	initialGranularMarks := rt.profiling.scheduledGranularMarks
	initialFineGrainedCommits := rt.profiling.fineGrainedCommits

	if err := rt.SetAtomValue("count", 3); err != nil {
		t.Fatalf("unexpected unchanged-projection atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if appRenderCount != 1 {
		t.Fatalf("expected owner component to stay at one render when projection is unchanged, got %d", appRenderCount)
	}
	if textNode.text != "odd" {
		t.Fatalf("expected projected text to remain odd when projection is unchanged, got %q", textNode.text)
	}
	if rt.profiling.commitCount != initialCommitCount {
		t.Fatalf("expected unchanged projected value to avoid an extra commit, got %d commits from %d", rt.profiling.commitCount, initialCommitCount)
	}
	if rt.profiling.scheduledGranularMarks != initialGranularMarks {
		t.Fatalf("expected unchanged projected value to avoid granular dirty marks, got %d from %d", rt.profiling.scheduledGranularMarks, initialGranularMarks)
	}
	if rt.profiling.fineGrainedCommits != initialFineGrainedCommits {
		t.Fatalf("expected unchanged projected value to avoid fine-grained commits, got %d from %d", rt.profiling.fineGrainedCommits, initialFineGrainedCommits)
	}

	if err := rt.SetAtomValue("count", 4); err != nil {
		t.Fatalf("unexpected changed-projection atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if textNode.text != "even" {
		t.Fatalf("expected projected text to update to even when projection changes, got %q", textNode.text)
	}
	if rt.profiling.commitCount <= initialCommitCount {
		t.Fatal("expected changed projected value to produce a follow-up commit")
	}
	if rt.profiling.scheduledGranularMarks <= initialGranularMarks {
		t.Fatal("expected changed projected value to produce a granular mark")
	}
	if rt.profiling.fineGrainedCommits <= initialFineGrainedCommits {
		t.Fatal("expected changed projected value to produce a fine-grained commit")
	}
}

func TestReactiveRegionHostPropertyUpdate_DoesNotRerenderOwnerComponent(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("input-value", "draft")

	ownerRenderCount := 0
	staticRenderCount := 0
	staticPanel := func() *Element {
		staticRenderCount++
		return CreateElement("aside", nil, "stable")
	}
	app := func() *Element {
		ownerRenderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]interface{}{"id": "reactive-shell"},
				CreateElement(ReactiveRegionNodeType, map[string]interface{}{
					reactiveRegionSourceIDsProp: []string{"input-value"},
					reactiveRegionRenderProp: func() *Element {
						value, _ := rt.GetAtomValue("input-value")
						current, _ := value.(string)
						return CreateElement("input", map[string]interface{}{"id": "live-input", "value": current})
					},
				}),
			),
			CreateElement(staticPanel, nil),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	root := container.(*testDOMNode)
	section := root.children[0].(*testDOMNode)
	shell := section.children[0].(*testDOMNode)
	inputBefore := shell.children[0].(*testDOMNode)
	staticBefore := section.children[1].(*testDOMNode)
	if got := inputBefore.properties["value"]; got != "draft" {
		t.Fatalf("expected initial input property draft, got %#v", got)
	}

	if err := rt.SetAtomValue("input-value", "published"); err != nil {
		t.Fatalf("unexpected input atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if ownerRenderCount != 1 {
		t.Fatalf("expected owner component render count to stay at 1, got %d", ownerRenderCount)
	}
	if staticRenderCount != 1 {
		t.Fatalf("expected sibling component render count to stay at 1, got %d", staticRenderCount)
	}
	shellAfter := section.children[0].(*testDOMNode)
	inputAfter := shellAfter.children[0].(*testDOMNode)
	staticAfter := section.children[1].(*testDOMNode)
	if shellAfter != shell {
		t.Fatal("expected reactive shell anchor to stay stable during region property update")
	}
	if inputAfter != inputBefore {
		t.Fatal("expected reactive region to update the existing host node in place")
	}
	if staticAfter != staticBefore {
		t.Fatal("expected sibling subtree to remain untouched during region property update")
	}
	if got := inputAfter.properties["value"]; got != "published" {
		t.Fatalf("expected input property published after fine-grained update, got %#v", got)
	}
	if rt.profiling.scheduledGranularMarks == 0 {
		t.Fatal("expected region property update to record a granular mark")
	}
}

func TestReactiveRegionAnchoredHostSubtreeUpdate_DoesNotRerenderOwnerComponent(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	rt.atomRegistry.InitAtom("expanded", false)

	ownerRenderCount := 0
	app := func() *Element {
		ownerRenderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]interface{}{"id": "region-anchor"},
				CreateElement(ReactiveRegionNodeType, map[string]interface{}{
					reactiveRegionSourceIDsProp: []string{"expanded"},
					reactiveRegionRenderProp: func() *Element {
						value, _ := rt.GetAtomValue("expanded")
						expanded, _ := value.(bool)
						children := []interface{}{
							CreateElement("span", map[string]interface{}{"id": "status"}, "closed"),
						}
						if expanded {
							children = []interface{}{
								CreateElement("span", map[string]interface{}{"id": "status"}, "open"),
								CreateElement("button", map[string]interface{}{"id": "action", "disabled": true}, "archive"),
							}
						}
						return CreateElement("FRAGMENT", nil, children...)
					},
				}),
			),
		)
	}

	rt.Render(CreateElement(app, nil), container)
	runScheduledTimeouts(scheduler)

	root := container.(*testDOMNode)
	section := root.children[0].(*testDOMNode)
	anchor := section.children[0].(*testDOMNode)
	statusBefore := anchor.children[0].(*testDOMNode)
	statusTextBefore := statusBefore.children[0].(*testDOMNode)
	if statusTextBefore.text != "closed" {
		t.Fatalf("expected initial region status closed, got %q", statusTextBefore.text)
	}

	if err := rt.SetAtomValue("expanded", true); err != nil {
		t.Fatalf("unexpected region subtree atom update error: %v", err)
	}
	runScheduledTimeouts(scheduler)

	if ownerRenderCount != 1 {
		t.Fatalf("expected owner component render count to stay at 1, got %d", ownerRenderCount)
	}
	anchorAfter := section.children[0].(*testDOMNode)
	if anchorAfter != anchor {
		t.Fatal("expected anchored region host node to remain stable during subtree update")
	}
	if len(anchorAfter.children) != 2 {
		t.Fatalf("expected anchored region to grow to two host children, got %d", len(anchorAfter.children))
	}
	statusAfter := anchorAfter.children[0].(*testDOMNode)
	actionAfter := anchorAfter.children[1].(*testDOMNode)
	statusTextAfter := statusAfter.children[0].(*testDOMNode)
	actionTextAfter := actionAfter.children[0].(*testDOMNode)
	if statusAfter != statusBefore {
		t.Fatal("expected existing anchored child host node to be updated in place")
	}
	if statusTextAfter.text != "open" {
		t.Fatalf("expected anchored status text to update to open, got %q", statusTextAfter.text)
	}
	if actionTextAfter.text != "archive" {
		t.Fatalf("expected anchored region to append action child, got %q", actionTextAfter.text)
	}
	if got := actionAfter.properties["disabled"]; got != true {
		t.Fatalf("expected appended action node property disabled=true, got %#v", got)
	}
}
