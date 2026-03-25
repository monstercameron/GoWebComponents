package mockdom

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type foreignNode struct{}

func (foreignNode) IsNull() bool                         { return false }
func (foreignNode) Equals(other runtime.DOMNode) bool   { return false }

func TestMockDOMAdapterNodeLifecycleAndOperations(t *testing.T) {
	adapter := NewMockDOMAdapter()
	parent := adapter.CreateElement("div")
	first := adapter.CreateElement("span")
	second := adapter.CreateTextNode("hello")

	adapter.SetAttribute(parent, "class", "surface")
	adapter.SetProperty(parent, "tabIndex", 3)
	if got := adapter.GetProperty(parent, "className"); got != "surface" {
		t.Fatalf("expected className property to mirror class attr, got %#v", got)
	}
	if got := adapter.GetProperty(parent, "tabIndex"); got != 3 {
		t.Fatalf("expected custom property value 3, got %#v", got)
	}
	if got := adapter.GetProperty(second, "nodeType"); got != 3 {
		t.Fatalf("expected text node type 3, got %#v", got)
	}
	if got := adapter.GetProperty(parent, "nodeType"); got != 1 {
		t.Fatalf("expected element node type 1, got %#v", got)
	}

	adapter.AppendChild(parent, first)
	adapter.AppendChild(parent, second)
	children := adapter.GetChildren(parent)
	if len(children) != 2 {
		t.Fatalf("expected 2 children after append, got %d", len(children))
	}
	if got := adapter.GetFirstChild(parent); got == nil || got.(*MockDOMNode).Tag != "span" {
		t.Fatalf("expected first child to be span, got %#v", got)
	}
	if got := adapter.GetNextSibling(first); got == nil || got.(*MockDOMNode).Tag != "#text" {
		t.Fatalf("expected next sibling to be text node, got %#v", got)
	}
	if got := adapter.GetParent(second); got == nil || got.(*MockDOMNode).Tag != "div" {
		t.Fatalf("expected text parent to be div, got %#v", got)
	}

	inserted := adapter.CreateElement("strong")
	adapter.InsertBefore(parent, inserted, second)
	if got := adapter.GetChildren(parent)[1].(*MockDOMNode).Tag; got != "strong" {
		t.Fatalf("expected inserted child before text node, got %q", got)
	}

	replacement := adapter.CreateElement("em")
	adapter.ReplaceChild(parent, replacement, inserted)
	if got := adapter.GetChildren(parent)[1].(*MockDOMNode).Tag; got != "em" {
		t.Fatalf("expected replacement child at index 1, got %q", got)
	}
	adapter.RemoveChild(parent, replacement)
	if len(adapter.GetChildren(parent)) != 2 {
		t.Fatalf("expected 2 children after remove, got %d", len(adapter.GetChildren(parent)))
	}

	adapter.SetStyle(parent, "display", "grid")
	adapter.SetStyles(parent, map[string]string{"gap": "8px"})
	adapter.SetTextContent(second, "updated")
	if got := adapter.GetProperty(second, "textContent"); got != "updated" {
		t.Fatalf("expected updated text content, got %#v", got)
	}

	adapter.SetInnerHTML(parent, "<div>mock</div>")
	if got := adapter.GetNode(parent.(*MockDOMNode).ID).InnerHTML; got != "<div>mock</div>" {
		t.Fatalf("expected inner HTML to be set, got %q", got)
	}
	adapter.SetInnerHTML(parent, "")
	if got := len(adapter.GetChildren(parent)); got != 0 {
		t.Fatalf("expected children to be cleared when inner HTML emptied, got %d", got)
	}

	if wrapped := adapter.WrapFunction("noop"); wrapped != "noop" {
		t.Fatalf("expected WrapFunction to pass through values, got %#v", wrapped)
	}
	if err := adapter.AssertOperation(0, "createElement"); err != nil {
		t.Fatalf("expected first operation type createElement, got %v", err)
	}
	if err := adapter.AssertOperation(999, "missing"); err == nil {
		t.Fatal("expected AssertOperation out-of-bounds to fail")
	}
	if err := adapter.AssertOperation(0, "wrong"); err == nil {
		t.Fatal("expected AssertOperation type mismatch to fail")
	}
	if len(adapter.GetOperations()) == 0 {
		t.Fatal("expected operation log to be recorded")
	}
	adapter.ClearOperations()
	if got := len(adapter.GetOperations()); got != 0 {
		t.Fatalf("expected operations to clear, got %d", got)
	}
}

func TestMockDOMAdapterHandlesNonMockNodesAndNodeEquality(t *testing.T) {
	adapter := NewMockDOMAdapter()
	unknown := foreignNode{}

	adapter.SetAttribute(unknown, "class", "x")
	adapter.RemoveAttribute(unknown, "class")
	adapter.SetProperty(unknown, "k", "v")
	adapter.AppendChild(unknown, unknown)
	adapter.RemoveChild(unknown, unknown)
	adapter.InsertBefore(unknown, unknown, unknown)
	adapter.ReplaceChild(unknown, unknown, unknown)
	adapter.SetStyle(unknown, "display", "none")
	adapter.SetStyles(unknown, map[string]string{"a": "b"})
	adapter.SetInnerHTML(unknown, "<p/>")
	adapter.SetTextContent(unknown, "t")
	if got := adapter.GetProperty(unknown, "missing"); got != nil {
		t.Fatalf("expected GetProperty for foreign node to return nil, got %#v", got)
	}
	if got := adapter.GetParent(unknown); got != nil {
		t.Fatalf("expected foreign parent lookup to return nil, got %#v", got)
	}
	if got := adapter.GetChildren(unknown); got != nil {
		t.Fatalf("expected foreign children lookup to return nil, got %#v", got)
	}
	if got := adapter.GetFirstChild(unknown); got != nil {
		t.Fatalf("expected foreign first child lookup to return nil, got %#v", got)
	}
	if got := adapter.GetNextSibling(unknown); got != nil {
		t.Fatalf("expected foreign sibling lookup to return nil, got %#v", got)
	}

	var nilNode *MockDOMNode
	if !nilNode.IsNull() {
		t.Fatal("expected nil node to report IsNull=true")
	}
	left := &MockDOMNode{ID: 7}
	right := &MockDOMNode{ID: 7}
	if !left.Equals(right) {
		t.Fatal("expected nodes with same id to be equal")
	}
	if left.Equals(unknown) {
		t.Fatal("expected non-mock node equality to return false")
	}
	if left.Equals(nil) {
		t.Fatal("expected non-nil node to be unequal to nil")
	}
}

func TestMockSchedulerSyncAndAsyncModes(t *testing.T) {
	syncScheduler := NewMockScheduler(true)
	idleRuns := 0
	timeoutRuns := 0
	syncScheduler.RequestIdleCallback(func(deadline runtime.Deadline) {
		idleRuns++
		if deadline.TimeRemaining() <= 0 || deadline.DidTimeout() {
			t.Fatalf("expected synchronous idle callback to receive positive non-timeout deadline")
		}
	})
	syncScheduler.SetTimeout(func() { timeoutRuns++ }, 10)
	if idleRuns != 1 || timeoutRuns != 1 {
		t.Fatalf("expected synchronous scheduler to execute callbacks immediately, idle=%d timeout=%d", idleRuns, timeoutRuns)
	}

	asyncScheduler := NewMockScheduler(false)
	asyncIdle := 0
	asyncTimeout := 0
	asyncScheduler.RequestIdleCallback(func(runtime.Deadline) { asyncIdle++ })
	asyncScheduler.SetTimeout(func() { asyncTimeout++ }, 10)
	if asyncScheduler.GetPendingCount() != 1 || asyncScheduler.GetPendingTimeoutCount() != 1 {
		t.Fatalf("expected queued async callbacks, idle=%d timeout=%d", asyncScheduler.GetPendingCount(), asyncScheduler.GetPendingTimeoutCount())
	}
	asyncScheduler.FlushIdleCallbacks()
	if asyncIdle != 1 || asyncScheduler.GetPendingCount() != 0 {
		t.Fatalf("expected idle callbacks to flush once, runs=%d pending=%d", asyncIdle, asyncScheduler.GetPendingCount())
	}
	asyncScheduler.FlushTimeouts()
	if asyncTimeout != 1 || asyncScheduler.GetPendingTimeoutCount() != 0 {
		t.Fatalf("expected timeout callbacks to flush once, runs=%d pending=%d", asyncTimeout, asyncScheduler.GetPendingTimeoutCount())
	}

	asyncScheduler.RequestIdleCallback(func(runtime.Deadline) { asyncIdle++ })
	asyncScheduler.SetTimeout(func() { asyncTimeout++ }, 10)
	asyncScheduler.FlushAll()
	if asyncIdle != 2 || asyncTimeout != 2 {
		t.Fatalf("expected FlushAll to drain all callbacks, idle=%d timeout=%d", asyncIdle, asyncTimeout)
	}
}

func TestMockDeadlineExposesValues(t *testing.T) {
	deadline := &MockDeadline{timeRemaining: 9.5, didTimeout: true}
	if got := deadline.TimeRemaining(); got != 9.5 {
		t.Fatalf("expected TimeRemaining 9.5, got %v", got)
	}
	if !deadline.DidTimeout() {
		t.Fatal("expected DidTimeout to return true")
	}
}
