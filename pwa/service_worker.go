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

func (parseC BackgroundSyncCapabilities) Available() bool {
	return parseC.OneShot || parseC.Periodic
}

type ServiceWorkerSubscription struct {
	cancel func()
}

func (parseS ServiceWorkerSubscription) Cancel() {
	if parseS.cancel != nil {
		parseS.cancel()
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

func (parseR ServiceWorkerRegistration) Snapshot() ServiceWorkerSnapshot {
	if parseR.snapshot == nil {
		return ServiceWorkerSnapshot{}
	}
	return parseR.snapshot()
}

func (parseR ServiceWorkerRegistration) BackgroundSyncCapabilities() BackgroundSyncCapabilities {
	if parseR.backgroundSync == nil {
		return BackgroundSyncCapabilities{}
	}
	return parseR.backgroundSync()
}

func (parseR ServiceWorkerRegistration) RegisterSync(parseCtx context.Context, parseTag string) error {
	if parseR.registerSync == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.RegisterSync", parseTag)
	}
	return parseR.registerSync(parseCtx, parseTag)
}

func (parseR ServiceWorkerRegistration) Update(parseCtx context.Context) error {
	if parseR.update == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.Update", "")
	}
	return parseR.update(parseCtx)
}

func (parseR ServiceWorkerRegistration) Unregister(parseCtx context.Context) (bool, error) {
	if parseR.unregister == nil {
		return false, serviceWorkerUnavailable("ServiceWorkerRegistration.Unregister", "")
	}
	return parseR.unregister(parseCtx)
}

func (parseR ServiceWorkerRegistration) SkipWaiting(parseCtx context.Context) error {
	if parseR.skipWaiting == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.SkipWaiting", "")
	}
	return parseR.skipWaiting(parseCtx)
}

func (parseR ServiceWorkerRegistration) SubscribeLifecycle(parseHandler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
	if parseR.subscribeLifecycle == nil {
		return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.SubscribeLifecycle", "")
	}
	return parseR.subscribeLifecycle(parseHandler)
}

func (parseR ServiceWorkerRegistration) ReloadOnControllerChange() (ServiceWorkerSubscription, error) {
	if parseR.reloadOnControllerChange == nil {
		return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.ReloadOnControllerChange", "")
	}
	return parseR.reloadOnControllerChange()
}
