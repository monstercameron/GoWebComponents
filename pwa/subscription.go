package pwa

// Subscription is a cancellable handle returned by the package's subscribe APIs (service-worker
// lifecycle, installability). Cancel stops delivery and is safe to call more than once. The two
// named aliases below preserve the original API surface while sharing one implementation.
type Subscription struct {
	cancel func()
}

// Cancel stops the subscription's callbacks. No-op if already cancelled or never armed.
func (parseSubscription Subscription) Cancel() {
	if parseSubscription.cancel != nil {
		parseSubscription.cancel()
	}
}

// ServiceWorkerSubscription is the handle returned by ServiceWorkerRegistration.SubscribeLifecycle
// and ReloadOnControllerChange.
type ServiceWorkerSubscription = Subscription

// InstallabilitySubscription is the handle returned by InstallabilityManager.Subscribe.
type InstallabilitySubscription = Subscription
