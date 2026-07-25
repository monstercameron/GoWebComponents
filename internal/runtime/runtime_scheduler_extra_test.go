package runtime

import "testing"

func TestNewRuntime_AppliesConfig(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	if parseRt.domAdapter != parseAdapter {
		parseT.Fatal("expected DOM adapter to be stored on runtime")
	}
	if parseRt.scheduler != parseScheduler {
		parseT.Fatal("expected scheduler to be stored on runtime")
	}
	if parseRt.atomRegistry == nil {
		parseT.Fatal("expected atom registry to be initialized")
	}
	if parseRt.deletions == nil {
		parseT.Fatal("expected deletions slice to be initialized")
	}
}

func TestGetGlobalRuntime_ReusesExistingInstance(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseFirst := GetGlobalRuntime()
	parseSecond := GetGlobalRuntime()

	if parseFirst == nil || parseSecond == nil {
		parseT.Fatal("expected global runtime instances")
	}
	if parseFirst != parseSecond {
		parseT.Fatal("expected GetGlobalRuntime to reuse the same instance")
	}
}

func TestInfiniteDeadlineMethods(parseT *testing.T) {
	if globalInfiniteDeadline.TimeRemaining() <= 0 {
		parseT.Fatal("expected infinite deadline to report positive time remaining")
	}
	if globalInfiniteDeadline.DidTimeout() {
		parseT.Fatal("expected infinite deadline to never time out")
	}
}

