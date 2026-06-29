package pwa

import "context"

type ServiceWorkerOptions struct {
	URL   string
	Scope string
	// Type is the worker script type: use ServiceWorkerTypeClassic / ServiceWorkerTypeModule
	// (a typo'd string silently registers nothing useful).
	Type string
	// UpdateViaCache is the HTTP-cache policy for the worker script + its imports: use
	// UpdateViaCacheImports / UpdateViaCacheAll / UpdateViaCacheNone.
	UpdateViaCache string
}

// Service-worker script type values for ServiceWorkerOptions.Type (the standard
// ServiceWorkerContainer.register `type` option). String constants — assignable to the field —
// so a typo is a compile error at the use site instead of a silent registration quirk.
const (
	ServiceWorkerTypeClassic string = "classic"
	ServiceWorkerTypeModule  string = "module"
)

// HTTP-cache policy values for ServiceWorkerOptions.UpdateViaCache (the standard `updateViaCache`
// option): "imports" (default — bypass cache for the top script, use it for imports), "all" (use
// the cache for both), "none" (bypass for both).
const (
	UpdateViaCacheImports string = "imports"
	UpdateViaCacheAll     string = "all"
	UpdateViaCacheNone    string = "none"
)

type ServiceWorkerState string

const (
	ServiceWorkerStateInstalling ServiceWorkerState = "installing"
	ServiceWorkerStateInstalled  ServiceWorkerState = "installed"
	ServiceWorkerStateActivating ServiceWorkerState = "activating"
	ServiceWorkerStateActivated  ServiceWorkerState = "activated"
	ServiceWorkerStateRedundant  ServiceWorkerState = "redundant"
)

type ServiceWorkerVersion struct {
	ScriptURL string
	State     ServiceWorkerState
}

type ServiceWorkerSnapshot struct {
	Scope         string
	HasController bool
	Installing    ServiceWorkerVersion
	Waiting       ServiceWorkerVersion
	Active        ServiceWorkerVersion
}

type BackgroundSyncCapabilities struct {
	OneShot  bool
	Periodic bool
}

func (parseCapabilities BackgroundSyncCapabilities) Available() bool {
	return parseCapabilities.OneShot || parseCapabilities.Periodic
}

type ServiceWorkerRegistration struct {
	snapshot                 func() ServiceWorkerSnapshot
	backgroundSync           func() BackgroundSyncCapabilities
	registerSync             func(context.Context, string) error
	update                   func(context.Context) error
	unregister               func(context.Context) (bool, error)
	skipWaiting              func(context.Context) error
	subscribeLifecycle       func(func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error)
	reloadOnControllerChange func() (ServiceWorkerSubscription, error)
}

func (parseRegistration ServiceWorkerRegistration) Snapshot() ServiceWorkerSnapshot {
	if parseRegistration.snapshot == nil {
		return ServiceWorkerSnapshot{}
	}
	return parseRegistration.snapshot()
}

func (parseRegistration ServiceWorkerRegistration) BackgroundSyncCapabilities() BackgroundSyncCapabilities {
	if parseRegistration.backgroundSync == nil {
		return BackgroundSyncCapabilities{}
	}
	return parseRegistration.backgroundSync()
}

func (parseRegistration ServiceWorkerRegistration) RegisterSync(parseSyncCtx context.Context, parseSyncTag string) error {
	if parseRegistration.registerSync == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.RegisterSync", parseSyncTag)
	}
	return parseRegistration.registerSync(parseSyncCtx, parseSyncTag)
}

func (parseRegistration ServiceWorkerRegistration) Update(parseUpdateCtx context.Context) error {
	if parseRegistration.update == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.Update", "")
	}
	return parseRegistration.update(parseUpdateCtx)
}

func (parseRegistration ServiceWorkerRegistration) Unregister(parseUnregisterCtx context.Context) (bool, error) {
	if parseRegistration.unregister == nil {
		return false, serviceWorkerUnavailable("ServiceWorkerRegistration.Unregister", "")
	}
	return parseRegistration.unregister(parseUnregisterCtx)
}

func (parseRegistration ServiceWorkerRegistration) SkipWaiting(parseSkipCtx context.Context) error {
	if parseRegistration.skipWaiting == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.SkipWaiting", "")
	}
	return parseRegistration.skipWaiting(parseSkipCtx)
}

func (parseRegistration ServiceWorkerRegistration) SubscribeLifecycle(parseLifecycleHandler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
	if parseRegistration.subscribeLifecycle == nil {
		return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.SubscribeLifecycle", "")
	}
	return parseRegistration.subscribeLifecycle(parseLifecycleHandler)
}

func (parseRegistration ServiceWorkerRegistration) ReloadOnControllerChange() (ServiceWorkerSubscription, error) {
	if parseRegistration.reloadOnControllerChange == nil {
		return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.ReloadOnControllerChange", "")
	}
	return parseRegistration.reloadOnControllerChange()
}
