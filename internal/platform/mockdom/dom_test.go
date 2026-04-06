package mockdom

import (
	"strconv"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type foreignNode struct{}

func (foreignNode) IsNull() bool                           { return false }
func (foreignNode) Equals(parseOther runtime.DOMNode) bool { return false }

func TestMockDOMAdapterNodeLifecycleAndOperations(parseT *testing.T) {
	parseAdapter := NewMockDOMAdapter()
	parseParent := parseAdapter.CreateElement("div")
	parseFirst := parseAdapter.CreateElement("span")
	parseSecond := parseAdapter.CreateTextNode("hello")

	parseAdapter.SetAttribute(parseParent, "class", "surface")
	parseAdapter.SetProperty(parseParent, "tabIndex", 3)
	if parseGot := parseAdapter.GetProperty(parseParent, "className"); parseGot != "surface" {
		parseT.Fatalf("expected className property to mirror class attr, got %#v", parseGot)
	}
	if parseGot2 := parseAdapter.GetProperty(parseParent, "tabIndex"); parseGot2 != 3 {
		parseT.Fatalf("expected custom property value 3, got %#v", parseGot2)
	}
	if parseGot3 := parseAdapter.GetProperty(parseSecond, "nodeType"); parseGot3 != 3 {
		parseT.Fatalf("expected text node type 3, got %#v", parseGot3)
	}
	if parseGot4 := parseAdapter.GetProperty(parseParent, "nodeType"); parseGot4 != 1 {
		parseT.Fatalf("expected element node type 1, got %#v", parseGot4)
	}

	parseAdapter.AppendChild(parseParent, parseFirst)
	parseAdapter.AppendChild(parseParent, parseSecond)
	parseChildren := parseAdapter.GetChildren(parseParent)
	if len(parseChildren) != 2 {
		parseT.Fatalf("expected 2 children after append, got %d", len(parseChildren))
	}
	if parseGot5 := parseAdapter.GetFirstChild(parseParent); parseGot5 == nil || parseGot5.(*MockDOMNode).Tag != "span" {
		parseT.Fatalf("expected first child to be span, got %#v", parseGot5)
	}
	if parseGot6 := parseAdapter.GetNextSibling(parseFirst); parseGot6 == nil || parseGot6.(*MockDOMNode).Tag != "#text" {
		parseT.Fatalf("expected next sibling to be text node, got %#v", parseGot6)
	}
	if parseGot7 := parseAdapter.GetParent(parseSecond); parseGot7 == nil || parseGot7.(*MockDOMNode).Tag != "div" {
		parseT.Fatalf("expected text parent to be div, got %#v", parseGot7)
	}

	parseInserted := parseAdapter.CreateElement("strong")
	parseAdapter.InsertBefore(parseParent, parseInserted, parseSecond)
	if parseGot8 := parseAdapter.GetChildren(parseParent)[1].(*MockDOMNode).Tag; parseGot8 != "strong" {
		parseT.Fatalf("expected inserted child before text node, got %q", parseGot8)
	}

	parseReplacement := parseAdapter.CreateElement("em")
	parseAdapter.ReplaceChild(parseParent, parseReplacement, parseInserted)
	if parseGot9 := parseAdapter.GetChildren(parseParent)[1].(*MockDOMNode).Tag; parseGot9 != "em" {
		parseT.Fatalf("expected replacement child at index 1, got %q", parseGot9)
	}
	parseAdapter.RemoveChild(parseParent, parseReplacement)
	if len(parseAdapter.GetChildren(parseParent)) != 2 {
		parseT.Fatalf("expected 2 children after remove, got %d", len(parseAdapter.GetChildren(parseParent)))
	}

	parseAdapter.SetStyle(parseParent, "display", "grid")
	parseAdapter.SetStyles(parseParent, map[string]string{"gap": "8px"})
	parseAdapter.SetTextContent(parseSecond, "updated")
	if parseGot10 := parseAdapter.GetProperty(parseSecond, "textContent"); parseGot10 != "updated" {
		parseT.Fatalf("expected updated text content, got %#v", parseGot10)
	}

	parseAdapter.SetInnerHTML(parseParent, "<div>mock</div>")
	if parseGot11 := parseAdapter.GetNode(parseParent.(*MockDOMNode).ID).InnerHTML; parseGot11 != "<div>mock</div>" {
		parseT.Fatalf("expected inner HTML to be set, got %q", parseGot11)
	}
	parseAdapter.SetInnerHTML(parseParent, "")
	if parseGot12 := len(parseAdapter.GetChildren(parseParent)); parseGot12 != 0 {
		parseT.Fatalf("expected children to be cleared when inner HTML emptied, got %d", parseGot12)
	}

	if parseWrapped := parseAdapter.WrapFunction("noop"); parseWrapped != "noop" {
		parseT.Fatalf("expected WrapFunction to pass through values, got %#v", parseWrapped)
	}
	if parseErr := parseAdapter.AssertOperation(0, "createElement"); parseErr != nil {
		parseT.Fatalf("expected first operation type createElement, got %v", parseErr)
	}
	if parseErr2 := parseAdapter.AssertOperation(999, "missing"); parseErr2 == nil {
		parseT.Fatal("expected AssertOperation out-of-bounds to fail")
	}
	if parseErr3 := parseAdapter.AssertOperation(0, "wrong"); parseErr3 == nil {
		parseT.Fatal("expected AssertOperation type mismatch to fail")
	}
	if len(parseAdapter.GetOperations()) == 0 {
		parseT.Fatal("expected operation log to be recorded")
	}
	parseAdapter.ClearOperations()
	if parseGot13 := len(parseAdapter.GetOperations()); parseGot13 != 0 {
		parseT.Fatalf("expected operations to clear, got %d", parseGot13)
	}
}

func TestMockDOMAdapterHandlesNonMockNodesAndNodeEquality(parseT *testing.T) {
	parseAdapter := NewMockDOMAdapter()
	parseUnknown := foreignNode{}

	parseAdapter.SetAttribute(parseUnknown, "class", "x")
	parseAdapter.RemoveAttribute(parseUnknown, "class")
	parseAdapter.SetProperty(parseUnknown, "k", "v")
	parseAdapter.AppendChild(parseUnknown, parseUnknown)
	parseAdapter.RemoveChild(parseUnknown, parseUnknown)
	parseAdapter.InsertBefore(parseUnknown, parseUnknown, parseUnknown)
	parseAdapter.ReplaceChild(parseUnknown, parseUnknown, parseUnknown)
	parseAdapter.SetStyle(parseUnknown, "display", "none")
	parseAdapter.SetStyles(parseUnknown, map[string]string{"a": "b"})
	parseAdapter.SetInnerHTML(parseUnknown, "<p/>")
	parseAdapter.SetTextContent(parseUnknown, "t")
	if parseGot := parseAdapter.GetProperty(parseUnknown, "missing"); parseGot != nil {
		parseT.Fatalf("expected GetProperty for foreign node to return nil, got %#v", parseGot)
	}
	if parseGot2 := parseAdapter.GetParent(parseUnknown); parseGot2 != nil {
		parseT.Fatalf("expected foreign parent lookup to return nil, got %#v", parseGot2)
	}
	if parseGot3 := parseAdapter.GetChildren(parseUnknown); parseGot3 != nil {
		parseT.Fatalf("expected foreign children lookup to return nil, got %#v", parseGot3)
	}
	if parseGot4 := parseAdapter.GetFirstChild(parseUnknown); parseGot4 != nil {
		parseT.Fatalf("expected foreign first child lookup to return nil, got %#v", parseGot4)
	}
	if parseGot5 := parseAdapter.GetNextSibling(parseUnknown); parseGot5 != nil {
		parseT.Fatalf("expected foreign sibling lookup to return nil, got %#v", parseGot5)
	}

	var parseNilNode *MockDOMNode
	if !parseNilNode.IsNull() {
		parseT.Fatal("expected nil node to report IsNull=true")
	}
	parseLeft := &MockDOMNode{ID: 7}
	parseRight := &MockDOMNode{ID: 7}
	if !parseLeft.Equals(parseRight) {
		parseT.Fatal("expected nodes with same id to be equal")
	}
	if parseLeft.Equals(parseUnknown) {
		parseT.Fatal("expected non-mock node equality to return false")
	}
	if parseLeft.Equals(nil) {
		parseT.Fatal("expected non-nil node to be unequal to nil")
	}
}

func TestMockDOMAdapterConcurrentOperationCapture(parseT *testing.T) {
	parseAdapter := NewMockDOMAdapter()
	parseParent := parseAdapter.CreateElement("div")

	const getWorkerCount = 24
	var parseWaitGroup sync.WaitGroup
	parseSnapshotStop := make(chan struct{})
	parseSnapshotDone := make(chan struct{})

	go func() {
		defer close(parseSnapshotDone)
		for {
			select {
			case <-parseSnapshotStop:
				return
			default:
				_ = len(parseAdapter.GetOperations())
			}
		}
	}()

	parseWaitGroup.Add(getWorkerCount)
	for parseIndex := 0; parseIndex < getWorkerCount; parseIndex++ {
		go func(parseWorkerIndex int) {
			defer parseWaitGroup.Done()
			parseChild := parseAdapter.CreateElement("span")
			parseAdapter.SetAttribute(parseChild, "data-worker", strconv.Itoa(parseWorkerIndex))
			parseAdapter.AppendChild(parseParent, parseChild)
		}(parseIndex)
	}
	parseWaitGroup.Wait()
	close(parseSnapshotStop)
	<-parseSnapshotDone

	parseOperations := parseAdapter.GetOperations()
	if len(parseOperations) != 1+(getWorkerCount*3) {
		parseT.Fatalf("expected %d operations after concurrent writes, got %d", 1+(getWorkerCount*3), len(parseOperations))
	}
	if len(parseAdapter.GetChildren(parseParent)) != getWorkerCount {
		parseT.Fatalf("expected %d appended children after concurrent writes, got %d", getWorkerCount, len(parseAdapter.GetChildren(parseParent)))
	}
	for _, parseChildNode := range parseAdapter.GetChildren(parseParent) {
		if parseParentNode := parseAdapter.GetParent(parseChildNode); parseParentNode != parseParent {
			parseT.Fatalf("expected child parent lookup to remain stable, got %#v", parseParentNode)
		}
	}
}

func TestMockSchedulerSyncAndAsyncModes(parseT *testing.T) {
	parseSyncScheduler := NewMockScheduler(true)
	parseIdleRuns := 0
	parseTimeoutRuns := 0
	parseSyncScheduler.RequestIdleCallback(func(parseDeadline runtime.Deadline) {
		parseIdleRuns++
		if parseDeadline.TimeRemaining() <= 0 || parseDeadline.DidTimeout() {
			parseT.Fatalf("expected synchronous idle callback to receive positive non-timeout deadline")
		}
	})
	parseSyncScheduler.SetTimeout(func() { parseTimeoutRuns++ }, 10)
	if parseIdleRuns != 1 || parseTimeoutRuns != 1 {
		parseT.Fatalf("expected synchronous scheduler to execute callbacks immediately, idle=%d timeout=%d", parseIdleRuns, parseTimeoutRuns)
	}

	parseAsyncScheduler := NewMockScheduler(false)
	parseAsyncIdle := 0
	parseAsyncTimeout := 0
	parseAsyncScheduler.RequestIdleCallback(func(runtime.Deadline) { parseAsyncIdle++ })
	parseAsyncScheduler.SetTimeout(func() { parseAsyncTimeout++ }, 10)
	if parseAsyncScheduler.GetPendingCount() != 1 || parseAsyncScheduler.GetPendingTimeoutCount() != 1 {
		parseT.Fatalf("expected queued async callbacks, idle=%d timeout=%d", parseAsyncScheduler.GetPendingCount(), parseAsyncScheduler.GetPendingTimeoutCount())
	}
	parseAsyncScheduler.FlushIdleCallbacks()
	if parseAsyncIdle != 1 || parseAsyncScheduler.GetPendingCount() != 0 {
		parseT.Fatalf("expected idle callbacks to flush once, runs=%d pending=%d", parseAsyncIdle, parseAsyncScheduler.GetPendingCount())
	}
	parseAsyncScheduler.FlushTimeouts()
	if parseAsyncTimeout != 1 || parseAsyncScheduler.GetPendingTimeoutCount() != 0 {
		parseT.Fatalf("expected timeout callbacks to flush once, runs=%d pending=%d", parseAsyncTimeout, parseAsyncScheduler.GetPendingTimeoutCount())
	}

	parseAsyncScheduler.RequestIdleCallback(func(runtime.Deadline) { parseAsyncIdle++ })
	parseAsyncScheduler.SetTimeout(func() { parseAsyncTimeout++ }, 10)
	parseAsyncScheduler.FlushAll()
	if parseAsyncIdle != 2 || parseAsyncTimeout != 2 {
		parseT.Fatalf("expected FlushAll to drain all callbacks, idle=%d timeout=%d", parseAsyncIdle, parseAsyncTimeout)
	}
}

func TestMockDeadlineExposesValues(parseT *testing.T) {
	parseDeadline := &MockDeadline{timeRemaining: 9.5, didTimeout: true}
	if parseGot := parseDeadline.TimeRemaining(); parseGot != 9.5 {
		parseT.Fatalf("expected TimeRemaining 9.5, got %v", parseGot)
	}
	if !parseDeadline.DidTimeout() {
		parseT.Fatal("expected DidTimeout to return true")
	}
}
