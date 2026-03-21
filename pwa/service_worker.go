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

func (c BackgroundSyncCapabilities) Available() bool {
	return c.OneShot || c.Periodic
}

type ServiceWorkerSubscription struct {
	cancel func()
}

func (s ServiceWorkerSubscription) Cancel() {
	if s.cancel != nil {
		s.cancel()
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

func (r ServiceWorkerRegistration) Snapshot() ServiceWorkerSnapshot {
	if r.snapshot == nil {
		return ServiceWorkerSnapshot{}
	}
	return r.snapshot()
}

func (r ServiceWorkerRegistration) BackgroundSyncCapabilities() BackgroundSyncCapabilities {
	if r.backgroundSync == nil {
		return BackgroundSyncCapabilities{}
	}
	return r.backgroundSync()
}

func (r ServiceWorkerRegistration) RegisterSync(ctx context.Context, tag string) error {
	if r.registerSync == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.RegisterSync", tag)
	}
	return r.registerSync(ctx, tag)
}

func (r ServiceWorkerRegistration) Update(ctx context.Context) error {
	if r.update == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.Update", "")
	}
	return r.update(ctx)
}

func (r ServiceWorkerRegistration) Unregister(ctx context.Context) (bool, error) {
	if r.unregister == nil {
		return false, serviceWorkerUnavailable("ServiceWorkerRegistration.Unregister", "")
	}
	return r.unregister(ctx)
}

func (r ServiceWorkerRegistration) SkipWaiting(ctx context.Context) error {
	if r.skipWaiting == nil {
		return serviceWorkerUnavailable("ServiceWorkerRegistration.SkipWaiting", "")
	}
	return r.skipWaiting(ctx)
}

func (r ServiceWorkerRegistration) SubscribeLifecycle(handler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
	if r.subscribeLifecycle == nil {
		return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.SubscribeLifecycle", "")
	}
	return r.subscribeLifecycle(handler)
}

func (r ServiceWorkerRegistration) ReloadOnControllerChange() (ServiceWorkerSubscription, error) {
	if r.reloadOnControllerChange == nil {
		return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.ReloadOnControllerChange", "")
	}
	return r.reloadOnControllerChange()
}
