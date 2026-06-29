package interop

import "testing"

func TestRecoverContainedPanicRoutesRecoveredValueToHandler(parseT *testing.T) {
	parseCalled := false
	SetContainedPanicHandler(func(parseSubject string, parseRecovered any) {
		parseCalled = true
		if parseSubject != "callback" || parseRecovered != "boom" {
			parseT.Fatalf("handler received %q/%#v", parseSubject, parseRecovered)
		}
	})
	parseT.Cleanup(func() { SetContainedPanicHandler(nil) })

	func() {
		defer RecoverContainedPanic("callback")
		panic("boom")
	}()
	if !parseCalled {
		parseT.Fatal("contained panic handler was not called")
	}
}

func TestRecoverContainedPanicNoopsWithoutPanic(parseT *testing.T) {
	SetContainedPanicHandler(func(parseSubject string, parseRecovered any) {
		parseT.Fatalf("handler should not be called without panic: %q/%#v", parseSubject, parseRecovered)
	})
	parseT.Cleanup(func() { SetContainedPanicHandler(nil) })

	func() {
		defer RecoverContainedPanic("quiet")
	}()
}
