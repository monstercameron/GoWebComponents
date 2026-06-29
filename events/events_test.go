// Tests for the events package. Runs natively; no build tag required because
// the core Subscribe/Publish functions do not depend on the render loop.
package events

import (
	"sync"
	"sync/atomic"
	"testing"
)

// resetTopic removes any existing registry state for parseTopic so that tests
// start from a clean slate without cross-test interference.
func resetTopic(parseTopic string) {
	globalRegistry.Delete(parseTopic)
}

// ---------------------------------------------------------------------------
// 1. Fan-out: N subscribers each receive all published values exactly once.
// ---------------------------------------------------------------------------

func TestFanOutAllSubscribersReceiveAllValues(t *testing.T) {
	parseTopic := "test.fanout"
	resetTopic(parseTopic)

	const parseN = 3
	const parseCount = 100

	parseReceived := make([][]int, parseN)
	parseUnsubs := make([]func(), parseN)

	for parseI := range parseN {
		parseIdx := parseI
		parseReceived[parseIdx] = []int{}
		parseUnsubs[parseIdx] = Subscribe(parseTopic, func(parseV int) {
			parseReceived[parseIdx] = append(parseReceived[parseIdx], parseV)
		})
	}

	for parseV := range parseCount {
		Publish(parseTopic, parseV)
	}

	for parseIdx := range parseN {
		parseUnsubs[parseIdx]()
		if len(parseReceived[parseIdx]) != parseCount {
			t.Errorf("subscriber %d: got %d values, want %d", parseIdx, len(parseReceived[parseIdx]), parseCount)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. Publish order is preserved per subscriber.
// ---------------------------------------------------------------------------

func TestPublishOrderPreserved(t *testing.T) {
	parseTopic := "test.order"
	resetTopic(parseTopic)

	parseReceived := []int{}
	parseUnsub := Subscribe(parseTopic, func(parseV int) {
		parseReceived = append(parseReceived, parseV)
	})
	defer parseUnsub()

	for parseV := range 50 {
		Publish(parseTopic, parseV)
	}

	for parseI, parseV := range parseReceived {
		if parseV != parseI {
			t.Fatalf("out of order at index %d: got %d, want %d", parseI, parseV, parseI)
		}
	}
}

// ---------------------------------------------------------------------------
// 3. Unsubscribe stops delivery and leaves no leaked entries.
// ---------------------------------------------------------------------------

func TestUnsubscribeStopsDeliveryAndLeavesNoLeak(t *testing.T) {
	parseTopic := "test.unsub"
	resetTopic(parseTopic)

	parseCount := 0
	parseUnsub := Subscribe(parseTopic, func(_ int) {
		parseCount++
	})

	Publish(parseTopic, 1)
	parseUnsub()
	Publish(parseTopic, 2)

	if parseCount != 1 {
		t.Fatalf("got %d deliveries, want 1", parseCount)
	}
	if parseN := subscriberCount(parseTopic); parseN != 0 {
		t.Fatalf("subscriber count after unsub = %d, want 0", parseN)
	}
}

func TestUnsubscribeIsIdempotent(t *testing.T) {
	parseTopic := "test.unsub.idempotent"
	resetTopic(parseTopic)

	parseUnsub := Subscribe(parseTopic, func(_ int) {})
	parseUnsub()
	parseUnsub()
	parseUnsub()

	if parseN := subscriberCount(parseTopic); parseN != 0 {
		t.Fatalf("subscriber count after repeated unsubscribe = %d, want 0", parseN)
	}
}

// ---------------------------------------------------------------------------
// 4. Late subscriber: default = no replay; WithReplayLast = one replay.
// ---------------------------------------------------------------------------

func TestLateSubscriberNoReplayByDefault(t *testing.T) {
	parseTopic := "test.noreplay"
	resetTopic(parseTopic)

	Publish(parseTopic, 42)

	parseCount := 0
	parseUnsub := Subscribe(parseTopic, func(_ int) {
		parseCount++
	})
	defer parseUnsub()

	if parseCount != 0 {
		t.Fatalf("late subscriber received %d values by default, want 0", parseCount)
	}
}

func TestLateSubscriberWithReplayLast(t *testing.T) {
	parseTopic := "test.replay"
	resetTopic(parseTopic)

	Publish(parseTopic, 99)
	Publish(parseTopic, 100)

	parseReceived := []int{}
	parseUnsub := subscribeInternal(parseTopic, func(parseV int) {
		parseReceived = append(parseReceived, parseV)
	}, true)
	defer parseUnsub()

	if len(parseReceived) != 1 {
		t.Fatalf("replay subscriber got %d values, want 1", len(parseReceived))
	}
	if parseReceived[0] != 100 {
		t.Fatalf("replay subscriber got %d, want 100 (the last published value)", parseReceived[0])
	}
}

// ---------------------------------------------------------------------------
// 5. Concurrent publishing is safe (run with -race).
// ---------------------------------------------------------------------------

func TestConcurrentPublishNoRaceAndAllDelivered(t *testing.T) {
	parseTopic := "test.concurrent"
	resetTopic(parseTopic)

	const parseGoroutines = 8
	const parsePerGoroutine = 50

	var parseTotalReceived int64
	parseUnsub := Subscribe(parseTopic, func(_ int) {
		atomic.AddInt64(&parseTotalReceived, 1)
	})
	defer parseUnsub()

	var parseWG sync.WaitGroup
	for range parseGoroutines {
		parseWG.Go(func() {
			for parseI := range parsePerGoroutine {
				Publish(parseTopic, parseI)
			}
		})
	}
	parseWG.Wait()

	parseWant := int64(parseGoroutines * parsePerGoroutine)
	parseGot := atomic.LoadInt64(&parseTotalReceived)
	if parseGot != parseWant {
		t.Fatalf("delivered %d values, want %d", parseGot, parseWant)
	}
}

func TestSubscribePublishConcurrencyWindow(t *testing.T) {
	parseTopic := "test.subscribe-publish-window"
	resetTopic(parseTopic)

	const parseSubscribers = 16
	const parsePublishes = 200

	var parseReady sync.WaitGroup
	parseReady.Add(parseSubscribers)
	var parseStart sync.WaitGroup
	parseStart.Add(1)

	var parseDelivered int64
	var parseWG sync.WaitGroup
	for range parseSubscribers {
		parseWG.Go(func() {
			parseUnsub := Subscribe(parseTopic, func(_ int) {
				atomic.AddInt64(&parseDelivered, 1)
			})
			defer parseUnsub()
			parseReady.Done()
			parseStart.Wait()
		})
	}
	parseReady.Wait()

	var parsePublishWG sync.WaitGroup
	for range 4 {
		parsePublishWG.Go(func() {
			for parseI := range parsePublishes {
				Publish(parseTopic, parseI)
			}
		})
	}
	parsePublishWG.Wait()
	parseStart.Done()
	parseWG.Wait()

	parseWant := int64(parseSubscribers * parsePublishes * 4)
	if parseGot := atomic.LoadInt64(&parseDelivered); parseGot != parseWant {
		t.Fatalf("delivered %d values, want %d", parseGot, parseWant)
	}
	if parseN := subscriberCount(parseTopic); parseN != 0 {
		t.Fatalf("subscriber count after concurrency window = %d, want 0", parseN)
	}
}

// ---------------------------------------------------------------------------
// 6. A panicking subscriber is contained; siblings still receive the value.
// ---------------------------------------------------------------------------

func TestPanickingSubscriberIsContained(t *testing.T) {
	parseTopic := "test.panic"
	resetTopic(parseTopic)

	parsePanicUnsub := Subscribe(parseTopic, func(_ string) {
		panic("intentional test panic")
	})
	defer parsePanicUnsub()

	parseSafeCount := 0
	parseSafeUnsub := Subscribe(parseTopic, func(_ string) {
		parseSafeCount++
	})
	defer parseSafeUnsub()

	// Publish must not propagate the panic and must return normally.
	func() {
		defer func() {
			if parseR := recover(); parseR != nil {
				t.Errorf("Publish propagated a panic from a subscriber: %v", parseR)
			}
		}()
		Publish(parseTopic, "hello")
	}()

	if parseSafeCount != 1 {
		t.Fatalf("safe subscriber got %d deliveries, want 1", parseSafeCount)
	}
}

// ---------------------------------------------------------------------------
// 7. Type isolation: same topic, different T, only matching type received.
// ---------------------------------------------------------------------------

func TestTypeIsolationSameTopicDifferentTypes(t *testing.T) {
	parseTopic := "test.types"
	resetTopic(parseTopic)

	parseIntCount := 0
	parseStringCount := 0

	parseIntUnsub := Subscribe(parseTopic, func(_ int) {
		parseIntCount++
	})
	defer parseIntUnsub()

	parseStringUnsub := Subscribe(parseTopic, func(_ string) {
		parseStringCount++
	})
	defer parseStringUnsub()

	Publish(parseTopic, 1)       // int publication
	Publish(parseTopic, "hello") // string publication
	Publish(parseTopic, 2)       // int publication

	if parseIntCount != 2 {
		t.Fatalf("int subscriber got %d, want 2", parseIntCount)
	}
	if parseStringCount != 1 {
		t.Fatalf("string subscriber got %d, want 1", parseStringCount)
	}
}

// ---------------------------------------------------------------------------
// 8. TopicHandle.Publish routes to the correct topic.
// ---------------------------------------------------------------------------

func TestTopicHandlePublish(t *testing.T) {
	parseTopic := "test.handle"
	resetTopic(parseTopic)

	parseReceived := []string{}
	parseUnsub := Subscribe(parseTopic, func(parseV string) {
		parseReceived = append(parseReceived, parseV)
	})
	defer parseUnsub()

	parseHandle := TopicHandle[string]{parseTopic: parseTopic}
	parseHandle.Publish("alpha")
	parseHandle.Publish("beta")

	if len(parseReceived) != 2 || parseReceived[0] != "alpha" || parseReceived[1] != "beta" {
		t.Fatalf("unexpected received values: %v", parseReceived)
	}
}
