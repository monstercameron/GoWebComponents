package runtime

import "testing"

func TestTestDOMNodeSatisfiesDOMNodeContract(t *testing.T) {
	var node DOMNode = &testDOMNode{tag: "div"}
	if node.IsNull() {
		t.Fatal("expected concrete test DOM node to be non-null")
	}
	if !node.Equals(node) {
		t.Fatal("expected node to compare equal to itself")
	}
	if node.Equals(&testDOMNode{tag: "div"}) {
		t.Fatal("expected different node pointers to compare unequal")
	}
}

func TestTestDOMAdapterSatisfiesCoreInterfaceContracts(t *testing.T) {
	var adapter DOMAdapter = newTestDOMAdapter()

	parent := adapter.CreateElement("div")
	child := adapter.CreateTextNode("hello")
	if parent == nil || child == nil {
		t.Fatal("expected mock adapter to create nodes")
	}

	adapter.AppendChild(parent, child)
	adapter.SetAttribute(parent, "id", "root")
	adapter.SetProperty(parent, "value", 42)
	adapter.SetTextContent(child, "updated")

	if got := adapter.GetProperty(parent, "value"); got != 42 {
		t.Fatalf("expected property round-trip, got %#v", got)
	}
	if got := adapter.GetFirstChild(parent); got == nil || got.IsNull() {
		t.Fatal("expected appended child to be reachable")
	}
	if len(adapter.GetChildren(parent)) != 1 {
		t.Fatalf("expected one child after append")
	}
}

func TestTestSchedulerSatisfiesSchedulerContract(t *testing.T) {
	rawScheduler := newTestScheduler()
	var scheduler Scheduler = rawScheduler

	timeoutCalled := false
	scheduler.SetTimeout(func() {
		timeoutCalled = true
	}, 0)
	if len(rawScheduler.timeouts) != 1 {
		t.Fatalf("expected queued timeout callback, got %d", len(rawScheduler.timeouts))
	}
	rawScheduler.timeouts[0]()
	if !timeoutCalled {
		t.Fatal("expected queued timeout callback to execute")
	}

	idleCalled := false
	scheduler.RequestIdleCallback(func(deadline Deadline) {
		idleCalled = true
		if deadline.TimeRemaining() <= 0 {
			t.Fatal("expected positive time remaining")
		}
		if deadline.DidTimeout() {
			t.Fatal("expected synchronous mock deadline to not timeout")
		}
	})
	if len(rawScheduler.callbacks) != 1 {
		t.Fatalf("expected queued idle callback, got %d", len(rawScheduler.callbacks))
	}
	rawScheduler.callbacks[0](&testDeadline{remaining: 16, timeout: false})
	if !idleCalled {
		t.Fatal("expected queued idle callback to execute")
	}
}
