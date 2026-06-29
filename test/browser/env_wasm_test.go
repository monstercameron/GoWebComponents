//go:build js && wasm

package browser

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/interop"
)

func TestInstallProvidesPathStorageAndMediaHelpers(parseT *testing.T) {
	parseEnv := Install(parseT, Options{
		Path:           "/products/42?tab=specs",
		LocalStorage:   map[string]string{"token": "abc"},
		SessionStorage: map[string]string{"draft": "1"},
		MediaMatches:   map[string]bool{"(display-mode: standalone)": true},
	})

	if parseGot := parseEnv.Window().Get("location").Get("pathname").String(); parseGot != "/products/42" {
		parseT.Fatalf("expected pathname to round-trip, got %q", parseGot)
	}
	if parseGot2 := parseEnv.Window().Get("location").Get("search").String(); parseGot2 != "?tab=specs" {
		parseT.Fatalf("expected search to round-trip, got %q", parseGot2)
	}
	if parseGot3 := parseEnv.Window().Get("localStorage").Call("getItem", "token"); parseGot3.String() != "abc" {
		parseT.Fatalf("expected localStorage to round-trip, got %q", parseGot3.String())
	}
	if parseGot4 := parseEnv.Window().Call("matchMedia", "(display-mode: standalone)").Get("matches").Bool(); !parseGot4 {
		parseT.Fatalf("expected configured media query to match")
	}
}

func TestInstallProvidesWorkerAndBroadcastChannelHelpers(parseT *testing.T) {
	parseEnv := Install(parseT)

	parseWorker, parseErr := interop.OpenWorker(parseT.Context(), interop.WorkerOptions{URL: "/workers/search.mjs"})
	if parseErr != nil {
		parseT.Fatalf("expected worker open to succeed, got %v", parseErr)
	}
	_ = parseWorker
	parseWorkers := parseEnv.Workers()
	if len(parseWorkers) != 1 || parseWorkers[0].URL() != "/workers/search.mjs" {
		parseT.Fatalf("expected one recorded worker, got %+v", parseWorkers)
	}

	parseChannel, parseErr := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: "prefs"})
	if parseErr != nil {
		parseT.Fatalf("expected cross-tab channel to open, got %v", parseErr)
	}
	defer parseChannel.Close()
	if parseErr2 := parseChannel.Publish(map[string]any{"kind": "theme"}); parseErr2 != nil {
		parseT.Fatalf("expected publish to succeed, got %v", parseErr2)
	}
	parseLog := parseEnv.BroadcastMessages()
	if len(parseLog) != 1 || parseLog[0].Channel != "prefs" {
		parseT.Fatalf("expected broadcast message log, got %+v", parseLog)
	}
}

func TestInstallProvidesWindowOpenAndOpenerHelpers(parseT *testing.T) {
	parseEnv := Install(parseT)
	parseParent := parseEnv.newMockWindow("parent")
	parseEnv.SetOpener(parseParent)

	parseChannel, parseErr := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{Name: "support", URL: "/support"})
	if parseErr != nil {
		parseT.Fatalf("expected secondary window channel to open, got %v", parseErr)
	}
	defer parseChannel.Close()

	if len(parseEnv.OpenCalls()) != 1 || parseEnv.OpenCalls()[0].URL != "/support" {
		parseT.Fatalf("expected window.open call to be recorded, got %+v", parseEnv.OpenCalls())
	}

	parseOpener, parseErr := interop.OpenWindowOpenerChannel(interop.WindowChannelOptions{Name: "opener"})
	if parseErr != nil {
		parseT.Fatalf("expected opener channel to resolve, got %v", parseErr)
	}
	defer parseOpener.Close()
}
