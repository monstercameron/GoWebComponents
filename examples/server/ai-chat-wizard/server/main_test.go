package main

import "testing"

// TestMainCallsChatWizardServer verifies the server entrypoint delegates to the app runner.
func TestMainCallsChatWizardServer(parseT *testing.T) {
	parseOriginalRun := runChatWizardServer
	parseT.Cleanup(func() {
		runChatWizardServer = parseOriginalRun
	})

	isParseCalled := false
	runChatWizardServer = func() {
		isParseCalled = true
	}

	main()

	if !isParseCalled {
		parseT.Fatal("expected chat wizard server runner to be called")
	}
}
