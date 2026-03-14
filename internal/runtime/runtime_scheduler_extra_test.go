package runtime

import "testing"

func TestNewRuntime_AppliesConfig(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	if rt.domAdapter != adapter {
		t.Fatal("expected DOM adapter to be stored on runtime")
	}
	if rt.scheduler != scheduler {
		t.Fatal("expected scheduler to be stored on runtime")
	}
	if rt.atomRegistry == nil {
		t.Fatal("expected atom registry to be initialized")
	}
	if rt.deletions == nil {
		t.Fatal("expected deletions slice to be initialized")
	}
}

func TestGetGlobalRuntime_ReusesExistingInstance(t *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	first := GetGlobalRuntime()
	second := GetGlobalRuntime()

	if first == nil || second == nil {
		t.Fatal("expected global runtime instances")
	}
	if first != second {
		t.Fatal("expected GetGlobalRuntime to reuse the same instance")
	}
}

func TestInfiniteDeadlineMethods(t *testing.T) {
	if globalInfiniteDeadline.TimeRemaining() <= 0 {
		t.Fatal("expected infinite deadline to report positive time remaining")
	}
	if globalInfiniteDeadline.DidTimeout() {
		t.Fatal("expected infinite deadline to never time out")
	}
}

func TestGetUIQueueSize_AfterEnqueueAndProcess(t *testing.T) {
	ProcessUIQueue()

	EnqueueUI(func() {})
	EnqueueUI(func() {})

	if got := GetUIQueueSize(); got != 2 {
		t.Fatalf("expected queue size 2 after enqueue, got %d", got)
	}

	ProcessUIQueue()

	if got := GetUIQueueSize(); got != 0 {
		t.Fatalf("expected queue size 0 after processing, got %d", got)
	}
}
