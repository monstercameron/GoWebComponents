//go:build js && wasm
// +build js,wasm

package browser

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestInstallProvidesPathStorageAndMediaHelpers(t *testing.T) {
	env := Install(t, Options{
		Path:           "/products/42?tab=specs",
		LocalStorage:   map[string]string{"token": "abc"},
		SessionStorage: map[string]string{"draft": "1"},
		MediaMatches:   map[string]bool{"(display-mode: standalone)": true},
	})

	if got := env.Window().Get("location").Get("pathname").String(); got != "/products/42" {
		t.Fatalf("expected pathname to round-trip, got %q", got)
	}
	if got := env.Window().Get("location").Get("search").String(); got != "?tab=specs" {
		t.Fatalf("expected search to round-trip, got %q", got)
	}
	if got := env.Window().Get("localStorage").Call("getItem", "token"); got.String() != "abc" {
		t.Fatalf("expected localStorage to round-trip, got %q", got.String())
	}
	if got := env.Window().Call("matchMedia", "(display-mode: standalone)").Get("matches").Bool(); !got {
		t.Fatalf("expected configured media query to match")
	}
}

func TestInstallProvidesWorkerAndBroadcastChannelHelpers(t *testing.T) {
	env := Install(t)

	worker, err := interop.OpenWorker(t.Context(), interop.WorkerOptions{URL: "/workers/search.mjs"})
	if err != nil {
		t.Fatalf("expected worker open to succeed, got %v", err)
	}
	_ = worker
	workers := env.Workers()
	if len(workers) != 1 || workers[0].URL() != "/workers/search.mjs" {
		t.Fatalf("expected one recorded worker, got %+v", workers)
	}

	channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: "prefs"})
	if err != nil {
		t.Fatalf("expected cross-tab channel to open, got %v", err)
	}
	defer channel.Close()
	if err := channel.Publish(map[string]any{"kind": "theme"}); err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}
	log := env.BroadcastMessages()
	if len(log) != 1 || log[0].Channel != "prefs" {
		t.Fatalf("expected broadcast message log, got %+v", log)
	}
}

func TestInstallProvidesWindowOpenAndOpenerHelpers(t *testing.T) {
	env := Install(t)
	parent := env.newMockWindow("parent")
	env.SetOpener(parent)

	channel, err := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{Name: "support", URL: "/support"})
	if err != nil {
		t.Fatalf("expected secondary window channel to open, got %v", err)
	}
	defer channel.Close()

	if len(env.OpenCalls()) != 1 || env.OpenCalls()[0].URL != "/support" {
		t.Fatalf("expected window.open call to be recorded, got %+v", env.OpenCalls())
	}

	opener, err := interop.OpenWindowOpenerChannel(interop.WindowChannelOptions{Name: "opener"})
	if err != nil {
		t.Fatalf("expected opener channel to resolve, got %v", err)
	}
	defer opener.Close()
}
