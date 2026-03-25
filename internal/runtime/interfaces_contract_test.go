package runtime

import "testing"

func TestTestDOMNodeSatisfiesDOMNodeContract(parseT *testing.T) {
	var parseNode DOMNode = &testDOMNode{tag: "div"}
	if parseNode.IsNull() {
		parseT.Fatal("expected concrete test DOM node to be non-null")
	}
	if !parseNode.Equals(parseNode) {
		parseT.Fatal("expected node to compare equal to itself")
	}
	if parseNode.Equals(&testDOMNode{tag: "div"}) {
		parseT.Fatal("expected different node pointers to compare unequal")
	}
}

func TestTestDOMAdapterSatisfiesCoreInterfaceContracts(parseT *testing.T) {
	var parseAdapter DOMAdapter = newTestDOMAdapter()

	parseParent := parseAdapter.CreateElement("div")
	parseChild := parseAdapter.CreateTextNode("hello")
	if parseParent == nil || parseChild == nil {
		parseT.Fatal("expected mock adapter to create nodes")
	}

	parseAdapter.AppendChild(parseParent, parseChild)
	parseAdapter.SetAttribute(parseParent, "id", "root")
	parseAdapter.SetProperty(parseParent, "value", 42)
	parseAdapter.SetTextContent(parseChild, "updated")

	if parseGot := parseAdapter.GetProperty(parseParent, "value"); parseGot != 42 {
		parseT.Fatalf("expected property round-trip, got %#v", parseGot)
	}
	if parseGot2 := parseAdapter.GetFirstChild(parseParent); parseGot2 == nil || parseGot2.IsNull() {
		parseT.Fatal("expected appended child to be reachable")
	}
	if len(parseAdapter.GetChildren(parseParent)) != 1 {
		parseT.Fatalf("expected one child after append")
	}
}

func TestTestSchedulerSatisfiesSchedulerContract(parseT *testing.T) {
	parseRawScheduler := newTestScheduler()
	var parseScheduler Scheduler = parseRawScheduler

	isParseTimeoutCalled := false
	parseScheduler.SetTimeout(func() {
		isParseTimeoutCalled = true
	}, 0)
	if len(parseRawScheduler.timeouts) != 1 {
		parseT.Fatalf("expected queued timeout callback, got %d", len(parseRawScheduler.timeouts))
	}
	parseRawScheduler.timeouts[0]()
	if !isParseTimeoutCalled {
		parseT.Fatal("expected queued timeout callback to execute")
	}

	isParseIdleCalled := false
	parseScheduler.RequestIdleCallback(func(parseDeadline Deadline) {
		isParseIdleCalled = true
		if parseDeadline.TimeRemaining() <= 0 {
			parseT.Fatal("expected positive time remaining")
		}
		if parseDeadline.DidTimeout() {
			parseT.Fatal("expected synchronous mock deadline to not timeout")
		}
	})
	if len(parseRawScheduler.callbacks) != 1 {
		parseT.Fatalf("expected queued idle callback, got %d", len(parseRawScheduler.callbacks))
	}
	parseRawScheduler.callbacks[0](&testDeadline{remaining: 16, timeout: false})
	if !isParseIdleCalled {
		parseT.Fatal("expected queued idle callback to execute")
	}
}
