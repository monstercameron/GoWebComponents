//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"errors"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/interop"
)

// RegisterServiceWorker registers a service worker at the given URL via the browser serviceworker API.
func RegisterServiceWorker(ctx context.Context, options ServiceWorkerOptions) (ServiceWorkerRegistration, error) {
	if options.URL == "" {
		return ServiceWorkerRegistration{}, &interop.Error{Op: "RegisterServiceWorker", Code: interop.CodeInvalid, Err: errors.New("service worker URL is empty")}
	}
	container, err := serviceWorkerContainer()
	if err != nil {
		return ServiceWorkerRegistration{}, err
	}
	registerFn := container.Get("register")
	if registerFn.Type() != js.TypeFunction {
		return ServiceWorkerRegistration{}, &interop.Error{Op: "RegisterServiceWorker", Target: options.URL, Code: interop.CodeNotFunction, Err: errors.New("navigator.serviceWorker.register is not callable")}
	}
	init := js.Global().Get("Object").New()
	if options.Scope != "" {
		init.Set("scope", options.Scope)
	}
	if options.Type != "" {
		init.Set("type", options.Type)
	}
	if options.UpdateViaCache != "" {
		init.Set("updateViaCache", options.UpdateViaCache)
	}
	rawRegistration, err := awaitServiceWorkerValue(ctx, "RegisterServiceWorker", options.URL, registerFn.Invoke(options.URL, init))
	if err != nil {
		return ServiceWorkerRegistration{}, err
	}
	return newServiceWorkerRegistration(container, rawRegistration), nil
}

func newServiceWorkerRegistration(container js.Value, raw js.Value) ServiceWorkerRegistration {
	snapshot := func() ServiceWorkerSnapshot {
		return serviceWorkerSnapshot(container, raw)
	}
	return ServiceWorkerRegistration{
		snapshot: snapshot,
		backgroundSync: func() BackgroundSyncCapabilities {
			return backgroundSyncCapabilities(raw)
		},
		registerSync: func(ctx context.Context, tag string) error {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				return &interop.Error{Op: "ServiceWorkerRegistration.RegisterSync", Target: snapshot().Scope, Code: interop.CodeInvalid, Err: errors.New("background sync tag is empty")}
			}
			syncManager := raw.Get("sync")
			if syncManager.IsUndefined() || syncManager.IsNull() {
				return &interop.Error{Op: "ServiceWorkerRegistration.RegisterSync", Target: tag, Code: interop.CodeUnavailable, Err: errors.New("background sync is unavailable for this service worker registration")}
			}
			register := syncManager.Get("register")
			if register.Type() != js.TypeFunction {
				return &interop.Error{Op: "ServiceWorkerRegistration.RegisterSync", Target: tag, Code: interop.CodeNotFunction, Err: errors.New("service worker sync.register is not callable")}
			}
			_, err := awaitServiceWorkerValue(ctx, "ServiceWorkerRegistration.RegisterSync", tag, register.Invoke(tag))
			return err
		},
		update: func(ctx context.Context) error {
			_, err := awaitServiceWorkerValue(ctx, "ServiceWorkerRegistration.Update", snapshot().Scope, raw.Call("update"))
			return err
		},
		unregister: func(ctx context.Context) (bool, error) {
			value, err := awaitServiceWorkerValue(ctx, "ServiceWorkerRegistration.Unregister", snapshot().Scope, raw.Call("unregister"))
			if err != nil {
				return false, err
			}
			if value.IsUndefined() || value.IsNull() {
				return false, nil
			}
			return value.Bool(), nil
		},
		skipWaiting: func(ctx context.Context) error {
			waiting := raw.Get("waiting")
			if waiting.IsUndefined() || waiting.IsNull() {
				return &interop.Error{Op: "ServiceWorkerRegistration.SkipWaiting", Target: snapshot().Scope, Code: interop.CodeInvalid, Err: errors.New("no waiting service worker is available")}
			}
			postMessage := waiting.Get("postMessage")
			if postMessage.Type() != js.TypeFunction {
				return &interop.Error{Op: "ServiceWorkerRegistration.SkipWaiting", Target: snapshot().Scope, Code: interop.CodeNotFunction, Err: errors.New("waiting service worker postMessage is not callable")}
			}
			waiting.Call("postMessage", map[string]any{"type": "SKIP_WAITING"})
			_ = ctx
			return nil
		},
		subscribeLifecycle: func(handler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
			if handler == nil {
				return ServiceWorkerSubscription{}, &interop.Error{Op: "ServiceWorkerRegistration.SubscribeLifecycle", Target: snapshot().Scope, Code: interop.CodeInvalid, Err: errors.New("lifecycle handler is nil")}
			}
			attachments := []serviceWorkerEventAttachment{}
			attachedWorkers := map[string]bool{}
			var notify func()
			attach := func(target js.Value, eventName string, fn js.Func) {
				target.Call("addEventListener", eventName, fn)
				attachments = append(attachments, serviceWorkerEventAttachment{target: target, eventName: eventName, fn: fn})
			}
			attachWorkerStateListeners := func() {
				for _, worker := range []js.Value{raw.Get("installing"), raw.Get("waiting"), raw.Get("active")} {
					if worker.IsUndefined() || worker.IsNull() {
						continue
					}
					key := strings.TrimSpace(worker.Get("scriptURL").String()) + "|" + strings.TrimSpace(worker.Get("state").String())
					if attachedWorkers[key] {
						continue
					}
					attachedWorkers[key] = true
					listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						notify()
						return nil
					})
					attach(worker, "statechange", listener)
				}
			}
			notify = func() {
				attachWorkerStateListeners()
				handler(snapshot())
			}
			updateFound := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				notify()
				return nil
			})
			controllerChanged := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				notify()
				return nil
			})
			attach(raw, "updatefound", updateFound)
			attach(container, "controllerchange", controllerChanged)
			notify()
			return ServiceWorkerSubscription{cancel: func() {
				for _, attachment := range attachments {
					attachment.target.Call("removeEventListener", attachment.eventName, attachment.fn)
					attachment.fn.Release()
				}
			}}, nil
		},
		reloadOnControllerChange: func() (ServiceWorkerSubscription, error) {
			window := browserWindow()
			if window.IsUndefined() || window.IsNull() {
				return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.ReloadOnControllerChange", snapshot().Scope)
			}
			location := window.Get("location")
			reload := location.Get("reload")
			if reload.Type() != js.TypeFunction {
				return ServiceWorkerSubscription{}, &interop.Error{Op: "ServiceWorkerRegistration.ReloadOnControllerChange", Target: snapshot().Scope, Code: interop.CodeNotFunction, Err: errors.New("window.location.reload is not callable")}
			}
			var listener js.Func
			listener = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				location.Call("reload")
				return nil
			})
			container.Call("addEventListener", "controllerchange", listener)
			return ServiceWorkerSubscription{cancel: func() {
				container.Call("removeEventListener", "controllerchange", listener)
				listener.Release()
			}}, nil
		},
	}
}

type serviceWorkerEventAttachment struct {
	target    js.Value
	eventName string
	fn        js.Func
}

func normalizeServiceWorkerOptions(options ServiceWorkerOptions) ServiceWorkerOptions {
	options.URL = strings.TrimSpace(options.URL)
	options.Scope = strings.TrimSpace(options.Scope)
	options.Type = strings.TrimSpace(options.Type)
	options.UpdateViaCache = strings.TrimSpace(options.UpdateViaCache)
	return options
}

func serviceWorkerContainer() (js.Value, error) {
	navigator := browserNavigator()
	if navigator.IsUndefined() || navigator.IsNull() {
		return js.Undefined(), serviceWorkerUnavailable("RegisterServiceWorker", "navigator.serviceWorker")
	}
	container := navigator.Get("serviceWorker")
	if container.IsUndefined() || container.IsNull() {
		return js.Undefined(), serviceWorkerUnavailable("RegisterServiceWorker", "navigator.serviceWorker")
	}
	return container, nil
}

func serviceWorkerSnapshot(container js.Value, raw js.Value) ServiceWorkerSnapshot {
	return ServiceWorkerSnapshot{
		Scope:         strings.TrimSpace(raw.Get("scope").String()),
		HasController: !container.Get("controller").IsUndefined() && !container.Get("controller").IsNull(),
		Installing:    serviceWorkerVersion(raw.Get("installing")),
		Waiting:       serviceWorkerVersion(raw.Get("waiting")),
		Active:        serviceWorkerVersion(raw.Get("active")),
	}
}

func backgroundSyncCapabilities(raw js.Value) BackgroundSyncCapabilities {
	capabilities := BackgroundSyncCapabilities{}
	syncManager := raw.Get("sync")
	if !syncManager.IsUndefined() && !syncManager.IsNull() {
		capabilities.OneShot = syncManager.Get("register").Type() == js.TypeFunction
	}
	periodicSync := raw.Get("periodicSync")
	if !periodicSync.IsUndefined() && !periodicSync.IsNull() {
		capabilities.Periodic = periodicSync.Get("register").Type() == js.TypeFunction
	}
	return capabilities
}

func serviceWorkerVersion(worker js.Value) ServiceWorkerVersion {
	if worker.IsUndefined() || worker.IsNull() {
		return ServiceWorkerVersion{}
	}
	return ServiceWorkerVersion{
		ScriptURL: strings.TrimSpace(worker.Get("scriptURL").String()),
		State:     ServiceWorkerState(strings.TrimSpace(worker.Get("state").String())),
	}
}

func awaitServiceWorkerValue(ctx context.Context, op string, target string, value js.Value) (js.Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	then := value.Get("then")
	if then.Type() != js.TypeFunction {
		return value, nil
	}
	resolvedCh := make(chan js.Value, 1)
	rejectedCh := make(chan error, 1)
	var resolveFn js.Func
	var rejectFn js.Func
	cleanup := func() {
		resolveFn.Release()
		rejectFn.Release()
	}
	resolveFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			select {
			case resolvedCh <- js.Undefined():
			default:
			}
			return nil
		}
		select {
		case resolvedCh <- args[0]:
		default:
		}
		return nil
	})
	rejectFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message := "service worker promise rejected"
		if len(args) > 0 {
			if text := strings.TrimSpace(args[0].String()); text != "" {
				message = text
			}
		}
		select {
		case rejectedCh <- &interop.Error{Op: op, Target: target, Code: interop.CodePromiseRejected, Err: errors.New(message)}:
		default:
		}
		return nil
	})
	value.Call("then", resolveFn).Call("catch", rejectFn)
	defer cleanup()

	select {
	case resolved := <-resolvedCh:
		return resolved, nil
	case err := <-rejectedCh:
		return js.Undefined(), err
	case <-ctx.Done():
		code := interop.CodeCancelled
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			code = interop.CodeTimeout
		}
		return js.Undefined(), &interop.Error{Op: op, Target: target, Code: code, Err: ctx.Err()}
	}
}

func serviceWorkerUnavailable(op string, target string) error {
	return &interop.Error{Op: op, Target: target, Code: interop.CodeUnavailable, Err: errors.New("service workers are unavailable in this build")}
}
