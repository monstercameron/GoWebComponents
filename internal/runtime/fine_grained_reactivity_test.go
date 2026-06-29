package runtime

import (
	"fmt"
	"strings"
	"testing"
)

func runScheduledTimeouts(parseScheduler *testScheduler) {
	for len(parseScheduler.timeouts) > 0 {
		parseCallback := parseScheduler.timeouts[0]
		parseScheduler.timeouts = parseScheduler.timeouts[1:]
		parseCallback()
	}
}

func TestScheduleGranularUpdateForFiber_DoesNotDirtyAncestors(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]any),
		},
	}

	parseGrandparent := &Fiber{typeOf: "grandparent"}
	parseParent := &Fiber{typeOf: "parent", parent: parseGrandparent}
	parseChild := &Fiber{typeOf: ReactiveTextNodeType, parent: parseParent, fineGrained: true}

	parseRt.ScheduleGranularUpdateForFiber(parseChild)

	if !parseChild.dirty || !parseChild.needsUpdate {
		parseT.Fatal("expected child to be marked for granular update")
	}
	if parseParent.dirty || parseParent.needsUpdate {
		parseT.Fatal("expected parent to remain clean during granular update")
	}
	if parseGrandparent.dirty || parseGrandparent.needsUpdate {
		parseT.Fatal("expected grandparent to remain clean during granular update")
	}
}

func TestReactiveTextAtomUpdate_DoesNotRerenderOwnerComponent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	parseApp := func() *Element {
		renderCount++
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]any{
				reactiveTextAtomIDProp: "count",
				reactiveTextGetterProp: func() string {
					parseValue, _ := parseRt.atomRegistry.GetAtom("count")
					if parseCount, parseOk := parseValue.(int); parseOk {
						return string(rune('0' + parseCount))
					}
					return "?"
				},
			}),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if renderCount != 1 {
		parseT.Fatalf("expected one initial owner render, got %d", renderCount)
	}
	parseParent := parseContainer.(*testDOMNode)
	if len(parseParent.children) != 1 {
		parseT.Fatalf("expected root container child, got %d", len(parseParent.children))
	}
	parseHost := parseParent.children[0].(*testDOMNode)
	if len(parseHost.children) != 1 {
		parseT.Fatalf("expected host child text node, got %d", len(parseHost.children))
	}
	parseTextNode := parseHost.children[0].(*testDOMNode)
	if parseTextNode.text != "0" {
		parseT.Fatalf("expected initial reactive text 0, got %q", parseTextNode.text)
	}

	if parseErr := parseRt.SetAtomValue("count", 1); parseErr != nil {
		parseT.Fatalf("unexpected atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if renderCount != 1 {
		parseT.Fatalf("expected owner component render count to stay at 1, got %d", renderCount)
	}
	if parseTextNode.text != "1" {
		parseT.Fatalf("expected reactive text node to update to 1, got %q", parseTextNode.text)
	}

	if parseErr2 := parseRt.SetAtomValue("count", 2); parseErr2 != nil {
		parseT.Fatalf("unexpected second atom update error: %v", parseErr2)
	}
	runScheduledTimeouts(parseScheduler)

	if renderCount != 1 {
		parseT.Fatalf("expected owner component render count to still be 1 after second update, got %d", renderCount)
	}
	if parseTextNode.text != "2" {
		parseT.Fatalf("expected reactive text node to update to 2 on second update, got %q", parseTextNode.text)
	}
}

func TestReactiveTextAtomUpdate_PreservesSiblingDomSubtrees(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	parseAppRenderCount := 0
	parseStaticRenderCount := 0

	parseStaticPanel := func() *Element {
		parseStaticRenderCount++
		return CreateElement("section", nil,
			CreateElement("strong", nil, "stable"),
		)
	}

	parseApp := func() *Element {
		parseAppRenderCount++
		return CreateElement("main", nil,
			CreateElement("div", map[string]any{"id": "static"},
				CreateElement(parseStaticPanel, nil),
			),
			CreateElement("div", map[string]any{"id": "hot"},
				CreateElement("span", nil,
					CreateElement(ReactiveTextNodeType, map[string]any{
						reactiveTextAtomIDProp: "count",
						reactiveTextGetterProp: func() string {
							parseValue, _ := parseRt.atomRegistry.GetAtom("count")
							if parseCount, parseOk := parseValue.(int); parseOk {
								return string(rune('0' + parseCount))
							}
							return "?"
						},
					}),
				),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	parseMainNode := parseRoot.children[0].(*testDOMNode)
	parseStaticWrapBefore := parseMainNode.children[0].(*testDOMNode)
	parseStaticSectionBefore := parseStaticWrapBefore.children[0].(*testDOMNode)
	parseStaticStrongBefore := parseStaticSectionBefore.children[0].(*testDOMNode)
	parseHotWrapBefore := parseMainNode.children[1].(*testDOMNode)
	parseHotSpanBefore := parseHotWrapBefore.children[0].(*testDOMNode)
	parseHotTextBefore := parseHotSpanBefore.children[0].(*testDOMNode)

	if parseErr := parseRt.SetAtomValue("count", 4); parseErr != nil {
		parseT.Fatalf("unexpected atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if parseAppRenderCount != 1 {
		parseT.Fatalf("expected app component to stay at one render, got %d", parseAppRenderCount)
	}
	if parseStaticRenderCount != 1 {
		parseT.Fatalf("expected unrelated sibling component to stay at one render, got %d", parseStaticRenderCount)
	}

	parseMainAfter := parseRoot.children[0].(*testDOMNode)
	parseStaticWrapAfter := parseMainAfter.children[0].(*testDOMNode)
	parseStaticSectionAfter := parseStaticWrapAfter.children[0].(*testDOMNode)
	parseStaticStrongAfter := parseStaticSectionAfter.children[0].(*testDOMNode)
	parseHotWrapAfter := parseMainAfter.children[1].(*testDOMNode)
	parseHotSpanAfter := parseHotWrapAfter.children[0].(*testDOMNode)
	parseHotTextAfter := parseHotSpanAfter.children[0].(*testDOMNode)

	if parseStaticWrapAfter != parseStaticWrapBefore || parseStaticSectionAfter != parseStaticSectionBefore || parseStaticStrongAfter != parseStaticStrongBefore {
		parseT.Fatal("expected unrelated static subtree DOM identity to remain stable during fine-grained update")
	}
	if parseHotWrapAfter != parseHotWrapBefore || parseHotSpanAfter != parseHotSpanBefore || parseHotTextAfter != parseHotTextBefore {
		parseT.Fatal("expected fine-grained update to mutate the existing hot DOM subtree instead of replacing it")
	}
	if parseStaticStrongAfter.children[0].(*testDOMNode).text != "stable" {
		parseT.Fatalf("expected static subtree text to remain stable, got %q", parseStaticStrongAfter.children[0].(*testDOMNode).text)
	}
	if parseHotTextAfter.text != "4" {
		parseT.Fatalf("expected hot text subtree to update in place to 4, got %q", parseHotTextAfter.text)
	}
	if parseRt.profiling.scheduledGranularMarks == 0 {
		parseT.Fatal("expected fine-grained update to record a granular mark")
	}
	if parseRt.profiling.fineGrainedCommits == 0 {
		parseT.Fatal("expected fine-grained update to record a granular commit")
	}
}

func TestReactiveTextMultipleRegions_UpdateOnlyTargetDomTree(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("left", 1)
	parseRt.atomRegistry.InitAtom("right", 8)

	renderCount := 0
	parseApp := func() *Element {
		renderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "left"},
				CreateElement(ReactiveTextNodeType, map[string]any{
					reactiveTextAtomIDProp: "left",
					reactiveTextGetterProp: func() string {
						parseValue, _ := parseRt.atomRegistry.GetAtom("left")
						if parseCount, parseOk := parseValue.(int); parseOk {
							return string(rune('0' + parseCount))
						}
						return "?"
					},
				}),
			),
			CreateElement("div", map[string]any{"id": "right"},
				CreateElement(ReactiveTextNodeType, map[string]any{
					reactiveTextAtomIDProp: "right",
					reactiveTextGetterProp: func() string {
						parseValue2, _ := parseRt.atomRegistry.GetAtom("right")
						if parseCount2, parseOk2 := parseValue2.(int); parseOk2 {
							return string(rune('0' + parseCount2))
						}
						return "?"
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	parseSection := parseRoot.children[0].(*testDOMNode)
	parseLeftWrapBefore := parseSection.children[0].(*testDOMNode)
	parseLeftTextBefore := parseLeftWrapBefore.children[0].(*testDOMNode)
	parseRightWrapBefore := parseSection.children[1].(*testDOMNode)
	parseRightTextBefore := parseRightWrapBefore.children[0].(*testDOMNode)

	if parseErr := parseRt.SetAtomValue("left", 2); parseErr != nil {
		parseT.Fatalf("unexpected left atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if renderCount != 1 {
		parseT.Fatalf("expected owning component to avoid rerender on left-region update, got %d renders", renderCount)
	}
	parseLeftWrapAfterLeft := parseSection.children[0].(*testDOMNode)
	parseLeftTextAfterLeft := parseLeftWrapAfterLeft.children[0].(*testDOMNode)
	parseRightWrapAfterLeft := parseSection.children[1].(*testDOMNode)
	parseRightTextAfterLeft := parseRightWrapAfterLeft.children[0].(*testDOMNode)
	if parseLeftWrapAfterLeft != parseLeftWrapBefore || parseLeftTextAfterLeft != parseLeftTextBefore {
		parseT.Fatal("expected left reactive region to update in place")
	}
	if parseRightWrapAfterLeft != parseRightWrapBefore || parseRightTextAfterLeft != parseRightTextBefore {
		parseT.Fatal("expected right region DOM tree to remain untouched when left region updates")
	}
	if parseLeftTextAfterLeft.text != "2" || parseRightTextAfterLeft.text != "8" {
		parseT.Fatalf("expected left/right texts to be 2 and 8, got %q and %q", parseLeftTextAfterLeft.text, parseRightTextAfterLeft.text)
	}

	if parseErr2 := parseRt.SetAtomValue("right", 9); parseErr2 != nil {
		parseT.Fatalf("unexpected right atom update error: %v", parseErr2)
	}
	runScheduledTimeouts(parseScheduler)

	parseLeftWrapAfterRight := parseSection.children[0].(*testDOMNode)
	parseLeftTextAfterRight := parseLeftWrapAfterRight.children[0].(*testDOMNode)
	parseRightWrapAfterRight := parseSection.children[1].(*testDOMNode)
	parseRightTextAfterRight := parseRightWrapAfterRight.children[0].(*testDOMNode)
	if parseLeftWrapAfterRight != parseLeftWrapBefore || parseLeftTextAfterRight != parseLeftTextBefore {
		parseT.Fatal("expected left region DOM tree to remain untouched when right region updates")
	}
	if parseRightWrapAfterRight != parseRightWrapBefore || parseRightTextAfterRight != parseRightTextBefore {
		parseT.Fatal("expected right reactive region to update in place")
	}
	if parseLeftTextAfterRight.text != "2" || parseRightTextAfterRight.text != "9" {
		parseT.Fatalf("expected left/right texts to be 2 and 9, got %q and %q", parseLeftTextAfterRight.text, parseRightTextAfterRight.text)
	}
}

func TestReactiveRegionSubscriptionsRedirectFromStaleTwinAfterAncestorRerender(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 1)

	var setTick func(any)
	parseApp := func() *Element {
		parseTick, set := GoUseState(parseRt, 0)
		setTick = set
		return CreateElement("section", nil,
			CreateElement("h1", nil, textFromInt(parseTick())),
			CreateElement("div", map[string]any{"id": "region"},
				CreateElement(ReactiveRegionNodeType, map[string]any{
					reactiveRegionSourceIDsProp: []string{"count"},
					reactiveRegionRenderProp: func() *Element {
						parseValue, _ := parseRt.GetAtomValue("count")
						parseCurrent, _ := parseValue.(int)
						return CreateElement("span", map[string]any{"id": "count-value"}, textFromInt(parseCurrent))
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if setTick == nil {
		parseT.Fatal("expected ancestor state setter")
	}

	parseStaleRegionFiber := findFineGrainedFiber(parseRt.currentRoot)
	if parseStaleRegionFiber == nil || !parseStaleRegionFiber.fineGrained {
		parseT.Fatal("expected to capture current reactive region fiber")
	}

	setTick(1)
	runScheduledTimeouts(parseScheduler)

	if parseRt.isFiberInCurrentTree(parseStaleRegionFiber) {
		parseT.Fatal("expected captured region fiber to become stale after ancestor rerender")
	}

	if parseErr := parseRt.SetAtomValue("count", 2); parseErr != nil {
		parseT.Fatalf("unexpected region atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	parseSection := parseRoot.children[0].(*testDOMNode)
	parseRegionWrap := parseSection.children[1].(*testDOMNode)
	parseCountNode := parseRegionWrap.children[0].(*testDOMNode)
	parseCountText := parseCountNode.children[0].(*testDOMNode)
	if parseCountText.text != "2" {
		parseT.Fatalf("expected redirected stale subscription update to reach live region, got %q", parseCountText.text)
	}
}

func textFromInt(parseValue int) string {
	return string(rune('0' + parseValue))
}

func findFineGrainedFiber(parseRoot *Fiber) *Fiber {
	if parseRoot == nil {
		return nil
	}
	if parseRoot.fineGrained {
		return parseRoot
	}
	if parseChild := findFineGrainedFiber(parseRoot.child); parseChild != nil {
		return parseChild
	}
	return findFineGrainedFiber(parseRoot.sibling)
}

func TestReactiveTextCollision_HookRerenderWinsAndKeepsTreeCoherent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	var setLabel func(any)
	parseApp := func() *Element {
		renderCount++
		parseLabel, set := GoUseState(parseRt, "ready")
		setLabel = set
		return CreateElement("section", nil,
			CreateElement("h1", nil, parseLabel()),
			CreateElement("span", nil,
				CreateElement(ReactiveTextNodeType, map[string]any{
					reactiveTextAtomIDProp: "count",
					reactiveTextGetterProp: func() string {
						parseValue, _ := parseRt.atomRegistry.GetAtom("count")
						if parseCount, parseOk := parseValue.(int); parseOk {
							return string(rune('0' + parseCount))
						}
						return "?"
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if setLabel == nil {
		parseT.Fatal("expected owner component state setter to be captured")
	}
	parseInitialCommitCount := parseRt.profiling.commitCount

	if parseErr := parseRt.SetAtomValue("count", 5); parseErr != nil {
		parseT.Fatalf("unexpected fine-grained atom update error: %v", parseErr)
	}
	setLabel("updated")
	runScheduledTimeouts(parseScheduler)

	if renderCount != 2 {
		parseT.Fatalf("expected exactly one owner rerender during mixed hook/fine-grained collision, got %d renders", renderCount)
	}
	parseRoot := parseContainer.(*testDOMNode)
	parseSection := parseRoot.children[0].(*testDOMNode)
	parseHeadline := parseSection.children[0].(*testDOMNode)
	parseHeadlineText := parseHeadline.children[0].(*testDOMNode)
	parseHotSpan := parseSection.children[1].(*testDOMNode)
	parseHotText := parseHotSpan.children[0].(*testDOMNode)
	if parseHeadlineText.text != "updated" {
		parseT.Fatalf("expected hook-driven label update to commit coherently, got %q", parseHeadlineText.text)
	}
	if parseHotText.text != "5" {
		parseT.Fatalf("expected fine-grained text to match the latest atom value during collision, got %q", parseHotText.text)
	}
	if parseRt.profiling.commitCount != parseInitialCommitCount+1 {
		parseT.Fatalf("expected mixed update window to coalesce into one follow-up commit, got %d commits from %d", parseRt.profiling.commitCount, parseInitialCommitCount)
	}
}

func TestReactiveTextTransitionUpdate_DefersUntilTimeout(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	parseApp := func() *Element {
		renderCount++
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]any{
				reactiveTextAtomIDProp: "count",
				reactiveTextGetterProp: func() string {
					parseValue, _ := parseRt.atomRegistry.GetAtom("count")
					if parseCount, parseOk := parseValue.(int); parseOk {
						return string(rune('0' + parseCount))
					}
					return "?"
				},
			}),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	parseHost := parseRoot.children[0].(*testDOMNode)
	parseTextNode := parseHost.children[0].(*testDOMNode)
	parseInitialCommitCount := parseRt.profiling.commitCount
	parseInitialGranularMarks := parseRt.profiling.scheduledGranularMarks

	parseRt.StartTransition(func() {
		parseRt.ScheduleTransition(func() {
			parseRt.atomRegistry.setAtomAndNotify("count", 3, parseRt.ScheduleSubscribedFiberUpdate)
		})
	})

	if parseTextNode.text != "0" {
		parseT.Fatalf("expected deferred fine-grained transition update to keep text at 0 before timeout, got %q", parseTextNode.text)
	}
	if renderCount != 1 {
		parseT.Fatalf("expected owner component to stay at one render before transition timeout, got %d", renderCount)
	}
	if parseRt.profiling.commitCount != parseInitialCommitCount {
		parseT.Fatalf("expected no new commit before transition timeout, got %d from %d", parseRt.profiling.commitCount, parseInitialCommitCount)
	}
	if parseRt.profiling.scheduledGranularMarks != parseInitialGranularMarks {
		parseT.Fatalf("expected no granular marks before transition timeout, got %d from %d", parseRt.profiling.scheduledGranularMarks, parseInitialGranularMarks)
	}
	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected deferred transition timeout for fine-grained update")
	}

	runScheduledTimeouts(parseScheduler)

	if parseTextNode.text != "3" {
		parseT.Fatalf("expected deferred fine-grained update to apply after timeout, got %q", parseTextNode.text)
	}
	if renderCount != 1 {
		parseT.Fatalf("expected owner component to avoid rerender after deferred fine-grained update, got %d", renderCount)
	}
	if parseRt.profiling.commitCount <= parseInitialCommitCount {
		parseT.Fatal("expected deferred fine-grained update to produce a follow-up commit")
	}
	if parseRt.profiling.scheduledGranularMarks <= parseInitialGranularMarks {
		parseT.Fatal("expected deferred fine-grained update to record a granular mark after timeout")
	}
}

func TestHydratedReactiveText_UpdateAfterResumeStaysNarrow(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("p")
	parseServerText := parseAdapter.CreateTextNode("0")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)
	parseRt.atomRegistry.InitAtom("count", 0)

	renderCount := 0
	parseApp := func() *Element {
		renderCount++
		return CreateElement("p", nil,
			CreateElement(ReactiveTextNodeType, map[string]any{
				reactiveTextAtomIDProp: "count",
				reactiveTextGetterProp: func() string {
					parseValue, _ := parseRt.atomRegistry.GetAtom("count")
					if parseCount, parseOk := parseValue.(int); parseOk {
						return string(rune('0' + parseCount))
					}
					return "?"
				},
			}),
		)
	}

	parseRt.Hydrate(CreateElement(parseApp, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 || !parseChildren[0].Equals(parseServerNode) {
		parseT.Fatal("expected hydrated fine-grained host node to reuse the existing server DOM node")
	}
	parseTextChildren := parseAdapter.GetChildren(parseServerNode)
	if len(parseTextChildren) != 1 || !parseTextChildren[0].Equals(parseServerText) {
		parseT.Fatal("expected hydrated fine-grained text node to reuse the existing server text node")
	}
	parseInitialCommitCount := parseRt.profiling.commitCount

	if parseErr := parseRt.SetAtomValue("count", 4); parseErr != nil {
		parseT.Fatalf("unexpected post-hydration atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if renderCount != 1 {
		parseT.Fatalf("expected hydrated owner component to avoid rerender on fine-grained post-resume update, got %d", renderCount)
	}
	if parseGot := parseServerText.(*testDOMNode).text; parseGot != "4" {
		parseT.Fatalf("expected hydrated reactive text node to update in place to 4, got %q", parseGot)
	}
	if parseRt.profiling.commitCount <= parseInitialCommitCount {
		parseT.Fatal("expected post-hydration fine-grained update to produce a follow-up commit")
	}
}

func TestReactiveTextSourceSwap_AvoidsStaleReadsAndOldSourceUpdates(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("primary", 0)
	parseRt.atomRegistry.InitAtom("secondary", 9)

	parseApp := func(parseProps map[string]any) *Element {
		parseAtomID, _ := parseProps["source"].(string)
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]any{
				reactiveTextAtomIDProp: parseAtomID,
				reactiveTextGetterProp: func() string {
					parseValue, _ := parseRt.atomRegistry.GetAtom(parseAtomID)
					if parseCount, parseOk := parseValue.(int); parseOk {
						return string(rune('0' + parseCount))
					}
					return "?"
				},
			}),
		)
	}

	parseRt.Render(CreateElement(parseApp, map[string]any{"source": "primary"}), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if parseRt.atomRegistry.GetSubscriberCount("primary") != 1 {
		parseT.Fatalf("expected primary atom to have 1 subscriber after initial render, got %d", parseRt.atomRegistry.GetSubscriberCount("primary"))
	}
	if parseRt.atomRegistry.GetSubscriberCount("secondary") != 0 {
		parseT.Fatalf("expected secondary atom to have 0 subscribers before source swap, got %d", parseRt.atomRegistry.GetSubscriberCount("secondary"))
	}

	parseRt.Render(CreateElement(parseApp, map[string]any{"source": "secondary"}), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if parseRt.atomRegistry.GetSubscriberCount("primary") != 0 {
		parseT.Fatalf("expected primary atom subscription to be released after source swap, got %d", parseRt.atomRegistry.GetSubscriberCount("primary"))
	}
	if parseRt.atomRegistry.GetSubscriberCount("secondary") != 1 {
		parseT.Fatalf("expected secondary atom to have 1 subscriber after source swap, got %d", parseRt.atomRegistry.GetSubscriberCount("secondary"))
	}

	parseParent := parseContainer.(*testDOMNode)
	parseHost := parseParent.children[0].(*testDOMNode)
	parseTextNode := parseHost.children[0].(*testDOMNode)
	if parseTextNode.text != "9" {
		parseT.Fatalf("expected swapped reactive text node to show secondary value 9, got %q", parseTextNode.text)
	}

	if parseErr := parseRt.SetAtomValue("primary", 4); parseErr != nil {
		parseT.Fatalf("unexpected primary atom update error after source swap: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)
	if parseTextNode.text != "9" {
		parseT.Fatalf("expected old source update to be ignored after source swap, got %q", parseTextNode.text)
	}

	if parseErr2 := parseRt.SetAtomValue("secondary", 7); parseErr2 != nil {
		parseT.Fatalf("unexpected secondary atom update error after source swap: %v", parseErr2)
	}
	runScheduledTimeouts(parseScheduler)
	if parseTextNode.text != "7" {
		parseT.Fatalf("expected new source update to drive reactive text after source swap, got %q", parseTextNode.text)
	}
}

func TestReactiveTextDerivedCycleFailsSafely(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}

	if parseErr := parseRt.RegisterDerivedAtom("derived-a", []string{"derived-b"}, func() any {
		parseValue, _ := parseRt.GetAtomValue("derived-b")
		return parseValue
	}); parseErr != nil {
		parseT.Fatalf("unexpected error registering first derived atom: %v", parseErr)
	}
	if parseErr2 := parseRt.RegisterDerivedAtom("derived-b", []string{"derived-a"}, func() any {
		parseValue2, _ := parseRt.GetAtomValue("derived-a")
		return parseValue2
	}); parseErr2 == nil {
		parseT.Fatal("expected derived cycle registration to fail")
	}

	parseSubscriber := &Fiber{typeOf: ReactiveTextNodeType, fineGrained: true}
	parseRt.atomRegistry.Subscribe("derived-b", parseSubscriber)

	if parseSubscriber.needsUpdate || parseSubscriber.dirty {
		parseT.Fatal("expected rejected derived cycle to avoid notifying subscribed fine-grained fibers")
	}

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		parseT.Fatal("expected derived cycle to be reported as a diagnostic")
	}
	isParseFound := false
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.Contains(parseDiagnostic.Message, "derived atom cycle detected") {
			isParseFound = true
			break
		}
	}
	if !isParseFound {
		parseT.Fatalf("expected derived cycle diagnostic, got %+v", parseDiagnostics)
	}
}

func TestReactiveTextUnmount_CleansUpDeletedSubtreeSubscriptions(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	parseApp := func(parseProps map[string]any) *Element {
		parseChildren := []any{}
		parseShowReactive, _ := parseProps["showReactive"].(bool)
		if parseShowReactive {
			parseChildren = append(parseChildren, CreateElement("span", nil,
				CreateElement(ReactiveTextNodeType, map[string]any{
					reactiveTextAtomIDProp: "count",
					reactiveTextGetterProp: func() string {
						parseValue, _ := parseRt.atomRegistry.GetAtom("count")
						if parseCount, parseOk := parseValue.(int); parseOk {
							return string(rune('0' + parseCount))
						}
						return "?"
					},
				}),
			))
		} else {
			parseChildren = append(parseChildren, CreateElement("span", nil, "gone"))
		}
		return CreateElement("div", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, map[string]any{"showReactive": true}), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if parseRt.atomRegistry.GetSubscriberCount("count") != 1 {
		parseT.Fatalf("expected count atom to have 1 fine-grained subscriber before unmount, got %d", parseRt.atomRegistry.GetSubscriberCount("count"))
	}

	parseRt.Render(CreateElement(parseApp, map[string]any{"showReactive": false}), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if parseRt.atomRegistry.GetSubscriberCount("count") != 0 {
		parseT.Fatalf("expected deleted fine-grained subtree to release count subscription, got %d", parseRt.atomRegistry.GetSubscriberCount("count"))
	}

	if parseErr := parseRt.SetAtomValue("count", 3); parseErr != nil {
		parseT.Fatalf("unexpected atom update error after reactive subtree unmount: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	parseParent := parseContainer.(*testDOMNode)
	parseHost := parseParent.children[0].(*testDOMNode)
	parseFallback := parseHost.children[0].(*testDOMNode)
	parseTextNode := parseFallback.children[0].(*testDOMNode)
	if parseTextNode.text != "gone" {
		parseT.Fatalf("expected fallback subtree to remain stable after old subscription source updated, got %q", parseTextNode.text)
	}
}

func TestReactiveTextDerivedProjection_UnchangedValueSkipsFineGrainedCommit(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 1)

	if parseErr := parseRt.RegisterDerivedAtom("parity", []string{"count"}, func() any {
		parseValue, _ := parseRt.GetAtomValue("count")
		if parseValue.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	}); parseErr != nil {
		parseT.Fatalf("unexpected derived parity registration error: %v", parseErr)
	}

	parseAppRenderCount := 0
	parseApp := func() *Element {
		parseAppRenderCount++
		return CreateElement("div", nil,
			CreateElement(ReactiveTextNodeType, map[string]any{
				reactiveTextAtomIDProp: "parity",
				reactiveTextGetterProp: func() string {
					parseValue2, _ := parseRt.GetAtomValue("parity")
					if parseLabel, parseOk := parseValue2.(string); parseOk {
						return parseLabel
					}
					return "?"
				},
			}),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	parseHost := parseRoot.children[0].(*testDOMNode)
	parseTextNode := parseHost.children[0].(*testDOMNode)
	if parseTextNode.text != "odd" {
		parseT.Fatalf("expected initial projected text odd, got %q", parseTextNode.text)
	}
	parseInitialCommitCount := parseRt.profiling.commitCount
	parseInitialGranularMarks := parseRt.profiling.scheduledGranularMarks
	parseInitialFineGrainedCommits := parseRt.profiling.fineGrainedCommits

	if parseErr2 := parseRt.SetAtomValue("count", 3); parseErr2 != nil {
		parseT.Fatalf("unexpected unchanged-projection atom update error: %v", parseErr2)
	}
	runScheduledTimeouts(parseScheduler)

	if parseAppRenderCount != 1 {
		parseT.Fatalf("expected owner component to stay at one render when projection is unchanged, got %d", parseAppRenderCount)
	}
	if parseTextNode.text != "odd" {
		parseT.Fatalf("expected projected text to remain odd when projection is unchanged, got %q", parseTextNode.text)
	}
	if parseRt.profiling.commitCount != parseInitialCommitCount {
		parseT.Fatalf("expected unchanged projected value to avoid an extra commit, got %d commits from %d", parseRt.profiling.commitCount, parseInitialCommitCount)
	}
	if parseRt.profiling.scheduledGranularMarks != parseInitialGranularMarks {
		parseT.Fatalf("expected unchanged projected value to avoid granular dirty marks, got %d from %d", parseRt.profiling.scheduledGranularMarks, parseInitialGranularMarks)
	}
	if parseRt.profiling.fineGrainedCommits != parseInitialFineGrainedCommits {
		parseT.Fatalf("expected unchanged projected value to avoid fine-grained commits, got %d from %d", parseRt.profiling.fineGrainedCommits, parseInitialFineGrainedCommits)
	}

	if parseErr3 := parseRt.SetAtomValue("count", 4); parseErr3 != nil {
		parseT.Fatalf("unexpected changed-projection atom update error: %v", parseErr3)
	}
	runScheduledTimeouts(parseScheduler)

	if parseTextNode.text != "even" {
		parseT.Fatalf("expected projected text to update to even when projection changes, got %q", parseTextNode.text)
	}
	if parseRt.profiling.commitCount <= parseInitialCommitCount {
		parseT.Fatal("expected changed projected value to produce a follow-up commit")
	}
	if parseRt.profiling.scheduledGranularMarks <= parseInitialGranularMarks {
		parseT.Fatal("expected changed projected value to produce a granular mark")
	}
	if parseRt.profiling.fineGrainedCommits <= parseInitialFineGrainedCommits {
		parseT.Fatal("expected changed projected value to produce a fine-grained commit")
	}
}

func TestReactiveRegionHostPropertyUpdate_DoesNotRerenderOwnerComponent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("input-value", "draft")

	parseOwnerRenderCount := 0
	parseStaticRenderCount := 0
	parseStaticPanel := func() *Element {
		parseStaticRenderCount++
		return CreateElement("aside", nil, "stable")
	}
	parseApp := func() *Element {
		parseOwnerRenderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "reactive-shell"},
				CreateElement(ReactiveRegionNodeType, map[string]any{
					reactiveRegionSourceIDsProp: []string{"input-value"},
					reactiveRegionRenderProp: func() *Element {
						parseValue, _ := parseRt.GetAtomValue("input-value")
						parseCurrent, _ := parseValue.(string)
						return CreateElement("input", map[string]any{"id": "live-input", "value": parseCurrent})
					},
				}),
			),
			CreateElement(parseStaticPanel, nil),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	parseSection := parseRoot.children[0].(*testDOMNode)
	parseShell := parseSection.children[0].(*testDOMNode)
	parseInputBefore := parseShell.children[0].(*testDOMNode)
	parseStaticBefore := parseSection.children[1].(*testDOMNode)
	if parseGot := parseInputBefore.properties["value"]; parseGot != "draft" {
		parseT.Fatalf("expected initial input property draft, got %#v", parseGot)
	}

	if parseErr := parseRt.SetAtomValue("input-value", "published"); parseErr != nil {
		parseT.Fatalf("unexpected input atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if parseOwnerRenderCount != 1 {
		parseT.Fatalf("expected owner component render count to stay at 1, got %d", parseOwnerRenderCount)
	}
	if parseStaticRenderCount != 1 {
		parseT.Fatalf("expected sibling component render count to stay at 1, got %d", parseStaticRenderCount)
	}
	parseShellAfter := parseSection.children[0].(*testDOMNode)
	parseInputAfter := parseShellAfter.children[0].(*testDOMNode)
	parseStaticAfter := parseSection.children[1].(*testDOMNode)
	if parseShellAfter != parseShell {
		parseT.Fatal("expected reactive shell anchor to stay stable during region property update")
	}
	if parseInputAfter != parseInputBefore {
		parseT.Fatal("expected reactive region to update the existing host node in place")
	}
	if parseStaticAfter != parseStaticBefore {
		parseT.Fatal("expected sibling subtree to remain untouched during region property update")
	}
	if parseGot2 := parseInputAfter.properties["value"]; parseGot2 != "published" {
		parseT.Fatalf("expected input property published after fine-grained update, got %#v", parseGot2)
	}
	if parseRt.profiling.scheduledGranularMarks == 0 {
		parseT.Fatal("expected region property update to record a granular mark")
	}
	if parseRt.profiling.fineGrainedDescendantHostCommits == 0 {
		parseT.Fatal("expected region property update to record descendant host commits")
	}
}

func TestReactiveRegionAnchoredHostSubtreeUpdate_DoesNotRerenderOwnerComponent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("expanded", false)

	parseOwnerRenderCount := 0
	parseApp := func() *Element {
		parseOwnerRenderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "region-anchor"},
				CreateElement(ReactiveRegionNodeType, map[string]any{
					reactiveRegionSourceIDsProp: []string{"expanded"},
					reactiveRegionRenderProp: func() *Element {
						parseValue, _ := parseRt.GetAtomValue("expanded")
						parseExpanded, _ := parseValue.(bool)
						parseChildren := []any{
							CreateElement("span", map[string]any{"id": "status"}, "closed"),
						}
						if parseExpanded {
							parseChildren = []any{
								CreateElement("span", map[string]any{"id": "status"}, "open"),
								CreateElement("button", map[string]any{"id": "action", "disabled": true}, "archive"),
							}
						}
						return CreateElement("FRAGMENT", nil, parseChildren...)
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	parseSection := parseRoot.children[0].(*testDOMNode)
	parseAnchor := parseSection.children[0].(*testDOMNode)
	parseStatusBefore := parseAnchor.children[0].(*testDOMNode)
	parseStatusTextBefore := parseStatusBefore.children[0].(*testDOMNode)
	if parseStatusTextBefore.text != "closed" {
		parseT.Fatalf("expected initial region status closed, got %q", parseStatusTextBefore.text)
	}

	if parseErr := parseRt.SetAtomValue("expanded", true); parseErr != nil {
		parseT.Fatalf("unexpected region subtree atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if parseOwnerRenderCount != 1 {
		parseT.Fatalf("expected owner component render count to stay at 1, got %d", parseOwnerRenderCount)
	}
	parseAnchorAfter := parseSection.children[0].(*testDOMNode)
	if parseAnchorAfter != parseAnchor {
		parseT.Fatal("expected anchored region host node to remain stable during subtree update")
	}
	if len(parseAnchorAfter.children) != 2 {
		parseT.Fatalf("expected anchored region to grow to two host children, got %d", len(parseAnchorAfter.children))
	}
	parseStatusAfter := parseAnchorAfter.children[0].(*testDOMNode)
	parseActionAfter := parseAnchorAfter.children[1].(*testDOMNode)
	parseStatusTextAfter := parseStatusAfter.children[0].(*testDOMNode)
	parseActionTextAfter := parseActionAfter.children[0].(*testDOMNode)
	if parseStatusAfter != parseStatusBefore {
		parseT.Fatal("expected existing anchored child host node to be updated in place")
	}
	if parseStatusTextAfter.text != "open" {
		parseT.Fatalf("expected anchored status text to update to open, got %q", parseStatusTextAfter.text)
	}
	if parseActionTextAfter.text != "archive" {
		parseT.Fatalf("expected anchored region to append action child, got %q", parseActionTextAfter.text)
	}
	if parseGot := parseActionAfter.properties["disabled"]; parseGot != true {
		parseT.Fatalf("expected appended action node property disabled=true, got %#v", parseGot)
	}
	if parseRt.profiling.fineGrainedDescendantHostCommits == 0 {
		parseT.Fatal("expected anchored region subtree update to record descendant host commits")
	}
	if parseRt.profiling.fineGrainedDescendantTextCommits == 0 {
		parseT.Fatal("expected anchored region subtree update to record descendant text commits")
	}
}

// TestReactiveRegionFunctionAtomSubscriber_UpdateDoesNotRerenderOwner verifies one atom update in a region-scoped function subscriber does not rerender the owner component.
func TestReactiveRegionFunctionAtomSubscriber_UpdateDoesNotRerenderOwner(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	parseOwnerRenderCount := 0
	parseRegionChildRenderCount := 0
	parseRegionChild := func() *Element {
		parseRegionChildRenderCount++
		parseCount, _ := GoUseAtom(parseRt, "count", 0)
		return CreateElement("span", map[string]any{"id": "count-value"}, textFromInt(parseCount()))
	}
	parseApp := func() *Element {
		parseOwnerRenderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "region-anchor"},
				CreateElement(ReactiveRegionNodeType, map[string]any{
					reactiveRegionSourceIDsProp: []string{"count"},
					reactiveRegionRenderProp: func() *Element {
						return CreateElement(parseRegionChild, nil)
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if parseOwnerRenderCount != 1 {
		parseT.Fatalf("expected one initial owner render, got %d", parseOwnerRenderCount)
	}
	if parseRegionChildRenderCount != 1 {
		parseT.Fatalf("expected one initial region child render, got %d", parseRegionChildRenderCount)
	}

	if parseErr := parseRt.SetAtomValue("count", 1); parseErr != nil {
		parseT.Fatalf("unexpected count atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if parseOwnerRenderCount != 1 {
		parseT.Fatalf("expected owner render count to stay at 1 for region-scoped atom updates, got %d", parseOwnerRenderCount)
	}
	if parseRegionChildRenderCount < 2 {
		parseT.Fatalf("expected region child to rerender after atom update, got %d renders", parseRegionChildRenderCount)
	}
	parseRoot := parseContainer.(*testDOMNode)
	parseSection := parseRoot.children[0].(*testDOMNode)
	parseAnchor := parseSection.children[0].(*testDOMNode)
	parseCountSpan := parseAnchor.children[0].(*testDOMNode)
	parseCountText := parseCountSpan.children[0].(*testDOMNode)
	if parseCountText.text != "1" {
		parseT.Fatalf("expected region-scoped atom update text 1, got %q", parseCountText.text)
	}
	if parseRt.profiling.scheduledGranularMarks == 0 {
		parseT.Fatal("expected region-scoped atom update to record granular scheduling")
	}
}

// TestReactiveRegionFunctionAtomSubscriber_MultipleUpdatesDoNotRerenderOwner verifies repeated atom updates in a region-scoped function subscriber do not rerender the owner component.
func TestReactiveRegionFunctionAtomSubscriber_MultipleUpdatesDoNotRerenderOwner(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	parseOwnerRenderCount := 0
	parseRegionChildRenderCount := 0
	parseRegionChild := func() *Element {
		parseRegionChildRenderCount++
		parseCount, _ := GoUseAtom(parseRt, "count", 0)
		return CreateElement("span", map[string]any{"id": "count-value"}, textFromInt(parseCount()))
	}
	parseApp := func() *Element {
		parseOwnerRenderCount++
		return CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "region-anchor"},
				CreateElement(ReactiveRegionNodeType, map[string]any{
					reactiveRegionSourceIDsProp: []string{"count"},
					reactiveRegionRenderProp: func() *Element {
						return CreateElement(parseRegionChild, nil)
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if parseOwnerRenderCount != 1 {
		parseT.Fatalf("expected one initial owner render, got %d", parseOwnerRenderCount)
	}

	for parseNext := 1; parseNext <= 9; parseNext++ {
		if parseErr := parseRt.SetAtomValue("count", parseNext); parseErr != nil {
			parseT.Fatalf("unexpected count atom update error at %d: %v", parseNext, parseErr)
		}
		runScheduledTimeouts(parseScheduler)
	}

	if parseOwnerRenderCount != 1 {
		parseT.Fatalf("expected owner render count to remain 1 after repeated region updates, got %d", parseOwnerRenderCount)
	}
	if parseRegionChildRenderCount < 10 {
		parseT.Fatalf("expected region child render count >= 10 after repeated updates, got %d", parseRegionChildRenderCount)
	}

	parseRoot := parseContainer.(*testDOMNode)
	parseSection := parseRoot.children[0].(*testDOMNode)
	parseAnchor := parseSection.children[0].(*testDOMNode)
	parseCountSpan := parseAnchor.children[0].(*testDOMNode)
	parseCountText := parseCountSpan.children[0].(*testDOMNode)
	if parseCountText.text != "9" {
		parseT.Fatalf("expected final region-scoped atom text 9, got %q", parseCountText.text)
	}
	if parseRt.profiling.scheduledGranularMarks < 9 {
		parseT.Fatalf("expected granular scheduling marks >= 9, got %d", parseRt.profiling.scheduledGranularMarks)
	}
}

// TestReactiveRegionFunctionAtomSubscriber_RepeatedOwnerRerendersKeepSingleSubscription verifies repeated owner rerenders do not leak reactive subscriptions.
func TestReactiveRegionFunctionAtomSubscriber_RepeatedOwnerRerendersKeepSingleSubscription(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	parseOwnerRenderCount := 0
	parseRegionChildRenderCount := 0
	var parseSetTick func(any)
	parseRegionChild := func() *Element {
		parseRegionChildRenderCount++
		parseCount, _ := GoUseAtom(parseRt, "count", 0)
		return CreateElement("span", map[string]any{"id": "count-value"}, textFromInt(parseCount()))
	}
	parseApp := func() *Element {
		parseOwnerRenderCount++
		parseTick, parseSet := GoUseState(parseRt, 0)
		parseSetTick = parseSet
		_ = parseTick()
		return CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "region-anchor"},
				CreateElement(ReactiveRegionNodeType, map[string]any{
					reactiveRegionSourceIDsProp: []string{"count"},
					reactiveRegionRenderProp: func() *Element {
						return CreateElement(parseRegionChild, nil)
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if parseSetTick == nil {
		parseT.Fatal("expected owner state setter to be captured")
	}
	parseInitialSubscriberCount := parseRt.atomRegistry.GetSubscriberCount("count")
	if parseInitialSubscriberCount < 1 {
		parseT.Fatalf("expected at least one count subscriber after initial render, got %d", parseInitialSubscriberCount)
	}

	for parseIteration := range 20 {
		parseSetTick(func(parsePrevious int) int {
			return parsePrevious + 1
		})
		runScheduledTimeouts(parseScheduler)
		if parseRt.atomRegistry.GetSubscriberCount("count") != parseInitialSubscriberCount {
			parseT.Fatalf(
				"expected stable count subscriber total %d after owner rerender %d, got %d",
				parseInitialSubscriberCount,
				parseIteration+1,
				parseRt.atomRegistry.GetSubscriberCount("count"),
			)
		}
	}

	parseOwnerRenderCountBeforeAtomUpdate := parseOwnerRenderCount
	if parseErr := parseRt.SetAtomValue("count", 1); parseErr != nil {
		parseT.Fatalf("unexpected count atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)

	if parseOwnerRenderCount != parseOwnerRenderCountBeforeAtomUpdate {
		parseT.Fatalf(
			"expected owner render count to stay at %d after region atom update, got %d; subscribers=%s",
			parseOwnerRenderCountBeforeAtomUpdate,
			parseOwnerRenderCount,
			buildReactiveRegionSubscriberSnapshot(parseRt, "count"),
		)
	}
	if parseRegionChildRenderCount < 2 {
		parseT.Fatalf("expected region child render count >= 2 after atom update, got %d", parseRegionChildRenderCount)
	}
	if parseRt.atomRegistry.GetSubscriberCount("count") != parseInitialSubscriberCount {
		parseT.Fatalf(
			"expected stable count subscriber total %d after region atom update, got %d",
			parseInitialSubscriberCount,
			parseRt.atomRegistry.GetSubscriberCount("count"),
		)
	}
}

// TestReactiveRegionFunctionAtomSubscriber_RepeatedKeyedOwnerRerendersKeepSingleSubscription verifies repeated keyed owner rerenders do not leak reactive subscriptions.
func TestReactiveRegionFunctionAtomSubscriber_RepeatedKeyedOwnerRerendersKeepSingleSubscription(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.atomRegistry.InitAtom("count", 0)

	parseOwnerRenderCount := 0
	var parseSetTick func(any)
	parseRegionChild := func() *Element {
		parseCount, _ := GoUseAtom(parseRt, "count", 0)
		return CreateElement("span", map[string]any{"id": "count-value"}, textFromInt(parseCount()))
	}
	parseApp := func() *Element {
		parseOwnerRenderCount++
		parseTick, parseSet := GoUseState(parseRt, 0)
		parseSetTick = parseSet
		parseOffset := parseTick() % 2
		return CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "lane-wrap"},
				CreateElement("div", map[string]any{"key": "lane-a"},
					textFromInt(parseOffset),
				),
				CreateElement(ReactiveRegionNodeType, map[string]any{
					"key":                       "lane-region",
					reactiveRegionSourceIDsProp: []string{"count"},
					reactiveRegionRenderProp: func() *Element {
						return CreateElement(parseRegionChild, nil)
					},
				}),
			),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if parseSetTick == nil {
		parseT.Fatal("expected owner state setter to be captured")
	}
	parseInitialSubscriberCount := parseRt.atomRegistry.GetSubscriberCount("count")
	if parseInitialSubscriberCount < 1 {
		parseT.Fatalf("expected at least one count subscriber after initial keyed render, got %d", parseInitialSubscriberCount)
	}

	for parseIteration := range 20 {
		parseSetTick(func(parsePrevious int) int {
			return parsePrevious + 1
		})
		runScheduledTimeouts(parseScheduler)
		if parseRt.atomRegistry.GetSubscriberCount("count") != parseInitialSubscriberCount {
			parseT.Fatalf(
				"expected stable keyed count subscriber total %d after owner rerender %d, got %d",
				parseInitialSubscriberCount,
				parseIteration+1,
				parseRt.atomRegistry.GetSubscriberCount("count"),
			)
		}
	}

	parseOwnerRenderCountBeforeAtomUpdate := parseOwnerRenderCount
	if parseErr := parseRt.SetAtomValue("count", 1); parseErr != nil {
		parseT.Fatalf("unexpected count atom update error: %v", parseErr)
	}
	runScheduledTimeouts(parseScheduler)
	if parseOwnerRenderCount != parseOwnerRenderCountBeforeAtomUpdate {
		parseT.Fatalf(
			"expected keyed owner render count to stay at %d after region atom update, got %d; subscribers=%s",
			parseOwnerRenderCountBeforeAtomUpdate,
			parseOwnerRenderCount,
			buildReactiveRegionSubscriberSnapshot(parseRt, "count"),
		)
	}
	if parseRt.atomRegistry.GetSubscriberCount("count") != parseInitialSubscriberCount {
		parseT.Fatalf(
			"expected stable keyed count subscriber total %d after region atom update, got %d",
			parseInitialSubscriberCount,
			parseRt.atomRegistry.GetSubscriberCount("count"),
		)
	}
}

// buildReactiveRegionSubscriberSnapshot returns one concise subscription snapshot for debugging region atom updates.
func buildReactiveRegionSubscriberSnapshot(parseRt *Runtime, parseAtomID string) string {
	if parseRt == nil || parseRt.atomRegistry == nil {
		return "<nil-runtime>"
	}
	parseRt.atomRegistry.mu.RLock()
	parseSubscribers := parseRt.atomRegistry.subscriptions[parseAtomID]
	parseRt.atomRegistry.mu.RUnlock()
	if len(parseSubscribers) == 0 {
		return "<none>"
	}
	parseRows := make([]string, 0, len(parseSubscribers))
	for parseFiber := range parseSubscribers {
		parseType := fmt.Sprintf("%T", parseFiber.typeOf)
		parseParentType := "<nil>"
		if parseFiber.parent != nil {
			parseParentType = fmt.Sprintf("%T", parseFiber.parent.typeOf)
		}
		parseRows = append(
			parseRows,
			fmt.Sprintf("type=%s parent=%s fine=%t in-tree=%t", parseType, parseParentType, parseFiber.fineGrained, parseRt.isFiberInCurrentTree(parseFiber)),
		)
	}
	return strings.Join(parseRows, " | ")
}
