package events

import "testing"

// TestTopicsListsSubscribedTopics pins that Topics reports each live topic with
// an accurate subscriber count, sorted, and that SubscriberCount agrees.
func TestTopicsListsSubscribedTopics(t *testing.T) {
	// Use unique topic names so the shared global registry stays deterministic
	// regardless of other tests in the package.
	parseUnsubA := Subscribe[string]("introspect.alpha", func(string) {})
	defer parseUnsubA()
	parseUnsubB1 := Subscribe[int]("introspect.beta", func(int) {})
	defer parseUnsubB1()
	parseUnsubB2 := Subscribe[int]("introspect.beta", func(int) {})
	defer parseUnsubB2()

	parseByTopic := map[string]int{}
	for _, parseInfo := range Topics() {
		parseByTopic[parseInfo.Topic] = parseInfo.Subscribers
	}

	if parseByTopic["introspect.alpha"] != 1 {
		t.Fatalf("alpha: want 1 subscriber, got %d", parseByTopic["introspect.alpha"])
	}
	if parseByTopic["introspect.beta"] != 2 {
		t.Fatalf("beta: want 2 subscribers, got %d", parseByTopic["introspect.beta"])
	}
	if parseGot := SubscriberCount("introspect.beta"); parseGot != 2 {
		t.Fatalf("SubscriberCount(beta): want 2, got %d", parseGot)
	}
	if parseGot := SubscriberCount("introspect.never"); parseGot != 0 {
		t.Fatalf("SubscriberCount(never): want 0, got %d", parseGot)
	}
}

// TestTopicsSortedAndDropsAfterUnsubscribe pins ordering and that a topic's
// count returns to zero once its subscribers leave.
func TestTopicsSortedAndDropsAfterUnsubscribe(t *testing.T) {
	parseUnsub := Subscribe[string]("introspect.zzz", func(string) {})
	parseUnsubEarly := Subscribe[string]("introspect.aaa", func(string) {})

	parseTopics := Topics()
	parseLastIdx := -1
	parsePrev := ""
	for parseIdx, parseInfo := range parseTopics {
		if parsePrev != "" && parseInfo.Topic < parsePrev {
			t.Fatalf("topics not sorted at index %d: %q before %q", parseIdx, parsePrev, parseInfo.Topic)
		}
		parsePrev = parseInfo.Topic
		parseLastIdx = parseIdx
	}
	_ = parseLastIdx

	parseUnsubEarly()
	if parseGot := SubscriberCount("introspect.aaa"); parseGot != 0 {
		t.Fatalf("after unsubscribe, want 0, got %d", parseGot)
	}
	parseUnsub()
}
