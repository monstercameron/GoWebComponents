//go:build js && wasm

package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestTesterInteractiveTimeouts limits longer waits to explicitly interactive cases.
func TestTesterInteractiveTimeouts(parseTest *testing.T) {
	parseInteractive := map[string]bool{"open-file": true, "open-files": true, "open-directory": true, "save-path": true, "message-info": true, "message-question": true, "export-report": true}
	for _, parseCase := range getTesterManualCases() {
		parseExpected := 30 * time.Second
		if parseInteractive[parseCase.Action] {
			parseExpected = 5 * time.Minute
		}
		if parseActual := getTesterTimeout(parseCase.Action); parseActual != parseExpected {
			parseTest.Fatalf("%s timeout=%v want=%v", parseCase.Action, parseActual, parseExpected)
		}
	}
}

// TestTesterManualCasesCoverNativeActions verifies every button can receive a human verdict.
func TestTesterManualCasesCoverNativeActions(parseTest *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseCase := range getTesterManualCases() {
		if parseCase.Action == "" || parseCase.Label == "" || parseSeen[parseCase.Action] {
			parseTest.Fatalf("invalid/duplicate case: %+v", parseCase)
		}
		parseSeen[parseCase.Action] = true
	}
	for _, parseGroup := range testerCases {
		for _, parseCase := range parseGroup.Items {
			if !parseSeen[parseCase.Action] {
				parseTest.Fatal(parseCase.Action)
			}
		}
	}
	for _, parseAction := range []string{"context-check", "context-radio", "context-disabled", "context-dismiss", "menu-action", "shortcut", "keyboard-focus", "keyboard-ime", "edit-menu", "second-window", "export-report"} {
		if !parseSeen[parseAction] {
			parseTest.Fatalf("missing manual case %s", parseAction)
		}
	}
}

// TestTesterTaskShowsActualOutcome keeps operation results, paths and errors visible.
func TestTesterTaskShowsActualOutcome(parseTest *testing.T) {
	parseResult := contracts.APIResult{Outcome: "cancelled", Detail: "picker dismissed", WindowID: "7", Paths: []string{"report.json"}}
	parseText := formatTesterTask(ui.TaskState[contracts.APIResult]{Value: parseResult, Ready: true})
	for _, parsePart := range []string{"cancelled", "picker dismissed", "window 7", "report.json"} {
		if !strings.Contains(parseText, parsePart) {
			parseTest.Fatal(parseText)
		}
	}
	if parseText = formatTesterTask(ui.TaskState[contracts.APIResult]{Error: errors.New("export refused")}); !strings.Contains(parseText, "export refused") {
		parseTest.Fatal(parseText)
	}
	if parseText = formatTesterTask(ui.TaskState[contracts.APIResult]{Cancelled: true}); !strings.Contains(parseText, "OS dialog may still need dismissal") {
		parseTest.Fatal(parseText)
	}
}

// TestTesterPollSuppressesLateState covers both successful and failed reads after unmount.
func TestTesterPollSuppressesLateState(parseTest *testing.T) {
	for _, parseReadErr := range []error{nil, errors.New("cancelled transport")} {
		parseContext, parseCancel := context.WithCancel(context.Background())
		parseReads, parseWrites := 0, 0
		pollTesterReport(parseContext, func(context.Context) (contracts.APIReport, error) {
			parseReads++
			parseCancel()
			return contracts.APIReport{}, parseReadErr
		}, func(contracts.APIReport) { parseWrites++ }, func(error) { parseWrites++ })
		if parseReads != 1 || parseWrites != 0 {
			parseTest.Fatalf("reads=%d late writes=%d", parseReads, parseWrites)
		}
	}
}
