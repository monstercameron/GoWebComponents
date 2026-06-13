//go:build !playwrightgo

package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdoutForStub(t *testing.T, parseRun func() error) string {
	t.Helper()
	parseOld := os.Stdout
	parseRead, parseWrite, parseErr := os.Pipe()
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	os.Stdout = parseWrite
	defer func() {
		os.Stdout = parseOld
	}()

	parseRunErr := parseRun()
	if parseRunErr != nil {
		t.Fatalf("stub returned error: %v", parseRunErr)
	}
	if parseErr := parseWrite.Close(); parseErr != nil {
		t.Fatal(parseErr)
	}
	parseData, parseErr := io.ReadAll(parseRead)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	_ = parseRead.Close()
	return string(parseData)
}

func assertStubEnvelope(t *testing.T, parseCommand string, parseOutput string) {
	t.Helper()
	var parseEnvelope agenticEnvelope
	if parseErr := json.Unmarshal([]byte(parseOutput), &parseEnvelope); parseErr != nil {
		t.Fatalf("decode envelope for %s: %v\n%s", parseCommand, parseErr, parseOutput)
	}
	if parseEnvelope.Command != parseCommand {
		t.Fatalf("command = %q, want %q", parseEnvelope.Command, parseCommand)
	}
	if parseEnvelope.OK {
		t.Fatalf("%s OK = true, want false", parseCommand)
	}
	if parseEnvelope.Error == nil || !strings.Contains(parseEnvelope.Error.Message, "-tags playwrightgo") {
		t.Fatalf("%s error = %#v, want playwrightgo guidance", parseCommand, parseEnvelope.Error)
	}
}

func TestDefaultBuildBrowserCommandsReturnPlaywrightGuidance(t *testing.T) {
	parseCommands := map[string]func() error{
		"screenshot": func() error { return runScreenshotCommand(launcher{}, nil) },
		"browser":    func() error { return runBrowserCommand(launcher{}, nil) },
		"click":      func() error { return runClickCommand(launcher{}, nil) },
		"type":       func() error { return runTypeCommand(launcher{}, nil) },
		"press":      func() error { return runPressCommand(launcher{}, nil) },
		"hover":      func() error { return runHoverCommand(launcher{}, nil) },
		"scroll":     func() error { return runScrollCommand(launcher{}, nil) },
		"console":    func() error { return runConsoleCommand(launcher{}, nil) },
		"network":    func() error { return runNetworkCommand(launcher{}, nil) },
		"dom":        func() error { return runDomCommand(launcher{}, nil) },
		"eval":       func() error { return runEvalCommand(launcher{}, nil) },
		"expect":     func() error { return runExpectCommand(launcher{}, nil) },
		"wait":       func() error { return runWaitCommand(launcher{}, nil) },
		"trace":      func() error { return runTraceCommand(launcher{}, nil) },
		"a11y":       func() error { return runA11yCommand(launcher{}, nil) },
		"select":     func() error { return runSelectCommand(launcher{}, nil) },
		"upload":     func() error { return runUploadCommand(launcher{}, nil) },
		"drag":       func() error { return runDragCommand(launcher{}, nil) },
		"mock":       func() error { return runMockCommand(launcher{}, nil) },
	}
	for parseCommand, parseRun := range parseCommands {
		t.Run(parseCommand, func(t *testing.T) {
			assertStubEnvelope(t, parseCommand, captureStdoutForStub(t, parseRun))
		})
	}
}
