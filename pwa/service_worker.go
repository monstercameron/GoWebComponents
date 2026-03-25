package pwa

import "context"

type ServiceWorkerOptions struct {
	URL            string
	Scope          string
	Type           string
	UpdateViaCache string
}

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

type ServiceWorkerSubscription struct {
	cancel func()
}

func (parseSubscription ServiceWorkerSubscription) Cancel() {
	if parseSubscription.cancel != nil {
		parseSubscription.cancel()
	}
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
