package pwa

import (
	"context"
	"sync"
)

// UpdatePrompt is the composite "a new version is available" flow on top of the service-worker
// lifecycle primitives. An app wires it to a banner: when a new service worker is waiting,
// Available() becomes true and the OnChange callback fires; calling Apply activates the new
// worker (SkipWaiting) and reloads once it takes control (ReloadOnControllerChange). This is the
// E3 10-rung "update-prompt flow" as one object, so apps don't hand-assemble the three
// primitives (the prior gap was that the primitives existed but the composite did not).
type UpdatePrompt struct {
	registration ServiceWorkerRegistration

	mu           sync.Mutex
	available    bool
	onChange     func(bool)
	subscription ServiceWorkerSubscription
	reloadSub    ServiceWorkerSubscription
	started      bool
}

// NewUpdatePrompt builds an update-prompt controller for a registration.
func NewUpdatePrompt(parseRegistration ServiceWorkerRegistration) *UpdatePrompt {
	return &UpdatePrompt{registration: parseRegistration}
}

// OnChange registers the callback invoked whenever update-availability changes (e.g. to show or
// hide a banner). It is called with the current availability immediately if already started.
func (parsePrompt *UpdatePrompt) OnChange(parseHandler func(bool)) {
	parsePrompt.mu.Lock()
	parsePrompt.onChange = parseHandler
	parseAvailable := parsePrompt.available
	parseStarted := parsePrompt.started
	parsePrompt.mu.Unlock()
	if parseHandler != nil && parseStarted {
		parseHandler(parseAvailable)
	}
}

// Start subscribes to the service-worker lifecycle and begins tracking whether a waiting worker
// (a ready update) exists. It is idempotent.
func (parsePrompt *UpdatePrompt) Start() error {
	parsePrompt.mu.Lock()
	if parsePrompt.started {
		parsePrompt.mu.Unlock()
		return nil
	}
	parsePrompt.started = true
	parsePrompt.mu.Unlock()

	// Seed from the current snapshot so a worker that is already waiting at start is surfaced.
	parsePrompt.handleSnapshot(parsePrompt.registration.Snapshot())

	parseSubscription, parseErr := parsePrompt.registration.SubscribeLifecycle(parsePrompt.handleSnapshot)
	if parseErr != nil {
		return parseErr
	}
	parsePrompt.mu.Lock()
	parsePrompt.subscription = parseSubscription
	parsePrompt.mu.Unlock()
	return nil
}

// handleSnapshot updates availability from a lifecycle snapshot: a non-empty waiting worker means
// an update is ready to activate. Fires OnChange only on a transition.
func (parsePrompt *UpdatePrompt) handleSnapshot(parseSnapshot ServiceWorkerSnapshot) {
	parseAvailable := parseSnapshot.Waiting.ScriptURL != ""
	parsePrompt.mu.Lock()
	parseChanged := parseAvailable != parsePrompt.available
	parsePrompt.available = parseAvailable
	parseHandler := parsePrompt.onChange
	parsePrompt.mu.Unlock()
	if parseChanged && parseHandler != nil {
		parseHandler(parseAvailable)
	}
}

// Available reports whether a new service worker is waiting to take over.
func (parsePrompt *UpdatePrompt) Available() bool {
	parsePrompt.mu.Lock()
	defer parsePrompt.mu.Unlock()
	return parsePrompt.available
}

// Apply activates the waiting worker and reloads the page once it takes control. It arms the
// controller-change reload BEFORE skipping waiting so the reload fires on activation, not before.
func (parsePrompt *UpdatePrompt) Apply(parseCtx context.Context) error {
	parseReloadSub, parseErr := parsePrompt.registration.ReloadOnControllerChange()
	if parseErr != nil {
		return parseErr
	}
	// Retain the reload-on-controllerchange handle. Cancel any listener a prior
	// Apply armed (idempotence) and store the new one; on a SkipWaiting failure
	// cancel it, because no controllerchange will fire and it would otherwise
	// leak. On success the page reloads on activation, which tears everything down.
	parsePrompt.mu.Lock()
	parsePrompt.reloadSub.Cancel()
	parsePrompt.reloadSub = parseReloadSub
	parsePrompt.mu.Unlock()

	if parseErr := parsePrompt.registration.SkipWaiting(parseCtx); parseErr != nil {
		parsePrompt.mu.Lock()
		parsePrompt.reloadSub.Cancel()
		parsePrompt.reloadSub = ServiceWorkerSubscription{}
		parsePrompt.mu.Unlock()
		return parseErr
	}
	return nil
}

// Stop cancels the lifecycle subscription.
func (parsePrompt *UpdatePrompt) Stop() {
	parsePrompt.mu.Lock()
	parseSubscription := parsePrompt.subscription
	parseReloadSub := parsePrompt.reloadSub
	parsePrompt.reloadSub = ServiceWorkerSubscription{}
	parsePrompt.started = false
	parsePrompt.mu.Unlock()
	parseSubscription.Cancel()
	parseReloadSub.Cancel()
}
