package pwa

import (
	"context"
	"testing"
)

// TestUpdatePromptFlow proves the E3 composite update-prompt flow end to end against a stubbed
// registration: no update at start, a waiting worker flips Available + fires OnChange, and Apply
// arms the controller-change reload before skipping waiting.
func TestUpdatePromptFlow(parseT *testing.T) {
	var parseLifecycleHandler func(ServiceWorkerSnapshot)
	var parseSkipCalled, parseReloadArmed bool
	var parseReloadArmedBeforeSkip bool

	parseRegistration := ServiceWorkerRegistration{
		snapshot: func() ServiceWorkerSnapshot { return ServiceWorkerSnapshot{Scope: "/"} },
		subscribeLifecycle: func(parseHandler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
			parseLifecycleHandler = parseHandler
			return ServiceWorkerSubscription{cancel: func() {}}, nil
		},
		skipWaiting: func(context.Context) error {
			parseSkipCalled = true
			parseReloadArmedBeforeSkip = parseReloadArmed // reload must be armed first
			return nil
		},
		reloadOnControllerChange: func() (ServiceWorkerSubscription, error) {
			parseReloadArmed = true
			return ServiceWorkerSubscription{cancel: func() {}}, nil
		},
	}

	parsePrompt := NewUpdatePrompt(parseRegistration)
	var parseChanges []bool
	parsePrompt.OnChange(func(parseAvailable bool) { parseChanges = append(parseChanges, parseAvailable) })

	if parseErr := parsePrompt.Start(); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if parsePrompt.Available() {
		parseT.Fatal("no waiting worker at start — update must not be available")
	}

	// A new worker becomes waiting → update available, OnChange fires true.
	parseLifecycleHandler(ServiceWorkerSnapshot{Waiting: ServiceWorkerVersion{ScriptURL: "/sw.js", State: ServiceWorkerStateInstalled}})
	if !parsePrompt.Available() {
		parseT.Fatal("a waiting worker must mark the update available")
	}
	if len(parseChanges) != 1 || parseChanges[0] != true {
		parseT.Fatalf("expected one OnChange(true), got %v", parseChanges)
	}

	// Idempotent: same waiting snapshot does not refire.
	parseLifecycleHandler(ServiceWorkerSnapshot{Waiting: ServiceWorkerVersion{ScriptURL: "/sw.js"}})
	if len(parseChanges) != 1 {
		parseT.Fatalf("OnChange must only fire on transitions, got %v", parseChanges)
	}

	// Apply activates the waiting worker and reloads on controller change.
	if parseErr := parsePrompt.Apply(context.Background()); parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if !parseSkipCalled || !parseReloadArmed {
		parseT.Fatalf("Apply must skip waiting (%v) and arm reload (%v)", parseSkipCalled, parseReloadArmed)
	}
	if !parseReloadArmedBeforeSkip {
		parseT.Fatal("Apply must arm reload-on-controller-change BEFORE skipping waiting")
	}
}
