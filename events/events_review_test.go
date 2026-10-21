package events

import (
	"sync"
	"sync/atomic"
	"testing"
)

// TestUnsubscribeDuringDispatchSkipsRemovedSubscriber pins that a subscriber removed by an earlier
// handler of the SAME publish does not receive that publish. Delivering to a handler after its
// unsubscribe ran means a component that unmounted (cleanup ran) still gets a callback.
func TestUnsubscribeDuringDispatchSkipsRemovedSubscriber(parseT *testing.T) {
	const parseTopic = "review.unsub-during-dispatch"
	var parseSecondCalls atomic.Int32
	var parseUnsubSecond func()

	parseUnsubFirst := Subscribe[int](parseTopic, func(int) { parseUnsubSecond() })
	defer parseUnsubFirst()
	parseUnsubSecond = Subscribe[int](parseTopic, func(int) { parseSecondCalls.Add(1) })
	defer parseUnsubSecond()

	Publish(parseTopic, 1)
	if parseSecondCalls.Load() != 0 {
		parseT.Fatalf("removed subscriber was still invoked %d time(s)", parseSecondCalls.Load())
	}
}

// TestSubscribeDuringDispatchIsNotDeliveredCurrentPublish pins exactly-once semantics for a
// handler added mid-dispatch: it must see future publishes only, never the in-flight one.
func TestSubscribeDuringDispatchIsNotDeliveredCurrentPublish(parseT *testing.T) {
	const parseTopic = "review.sub-during-dispatch"
	var parseLateCalls atomic.Int32
	var parseOnce sync.Once
	var parseUnsubLate func()

	parseUnsubFirst := Subscribe[int](parseTopic, func(int) {
		parseOnce.Do(func() {
			parseUnsubLate = Subscribe[int](parseTopic, func(int) { parseLateCalls.Add(1) })
		})
	})
	defer parseUnsubFirst()

	Publish(parseTopic, 1)
	if parseLateCalls.Load() != 0 {
		parseT.Fatalf("late subscriber saw the in-flight publish (%d calls)", parseLateCalls.Load())
	}
	Publish(parseTopic, 2)
	if parseLateCalls.Load() != 1 {
		parseT.Fatalf("late subscriber calls = %d, want 1", parseLateCalls.Load())
	}
	parseUnsubLate()
}

// TestPanickingHandlerDoesNotSkipSiblingsAndFanOutIsExactlyOnce pins panic containment plus
// exactly-once delivery to every healthy sibling around a panicking handler.
func TestPanickingHandlerDoesNotSkipSiblingsAndFanOutIsExactlyOnce(parseT *testing.T) {
	const parseTopic = "review.panic-fanout"
	var parseBefore, parseAfter atomic.Int32
	parseUnsubA := Subscribe[string](parseTopic, func(string) { parseBefore.Add(1) })
	parseUnsubB := Subscribe[string](parseTopic, func(string) { panic("boom") })
	parseUnsubC := Subscribe[string](parseTopic, func(string) { parseAfter.Add(1) })
	defer parseUnsubA()
	defer parseUnsubB()
	defer parseUnsubC()

	for range 5 {
		Publish(parseTopic, "x")
	}
	if parseBefore.Load() != 5 || parseAfter.Load() != 5 {
		parseT.Fatalf("before=%d after=%d, want 5/5", parseBefore.Load(), parseAfter.Load())
	}
}

// TestReentrantPublishAndDoubleUnsubscribe covers a handler publishing on its own topic and
// calling unsubscribe twice (the second call must be a harmless no-op).
func TestReentrantPublishAndDoubleUnsubscribe(parseT *testing.T) {
	const parseTopic = "review.reentrant"
	var parseCalls []int
	parseUnsub := Subscribe[int](parseTopic, func(parseValue int) {
		parseCalls = append(parseCalls, parseValue)
		if parseValue < 3 {
			Publish(parseTopic, parseValue+1)
		}
	})
	Publish(parseTopic, 0)
	parseUnsub()
	parseUnsub()
	if len(parseCalls) != 4 {
		parseT.Fatalf("calls = %v, want 4 nested deliveries", parseCalls)
	}
	if subscriberCount(parseTopic) != 0 {
		parseT.Fatalf("subscriber leaked after unsubscribe")
	}
}
