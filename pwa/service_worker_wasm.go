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
func RegisterServiceWorker(parseCtx context.Context, parseOptions ServiceWorkerOptions) (ServiceWorkerRegistration, error) {
	if parseOptions.URL == "" {
		return ServiceWorkerRegistration{}, &interop.Error{Op: "RegisterServiceWorker", Code: interop.CodeInvalid, Err: errors.New("service worker URL is empty")}
	}
	parseContainer, parseErr := serviceWorkerContainer()
	if parseErr != nil {
		return ServiceWorkerRegistration{}, parseErr
	}
	parseRegisterFn := parseContainer.Get("register")
	if parseRegisterFn.Type() != js.TypeFunction {
		return ServiceWorkerRegistration{}, &interop.Error{Op: "RegisterServiceWorker", Target: parseOptions.URL, Code: interop.CodeNotFunction, Err: errors.New("navigator.serviceWorker.register is not callable")}
	}
	parseInit := js.Global().Get("Object").New()
	if parseOptions.Scope != "" {
		parseInit.Set("scope", parseOptions.Scope)
	}
	if parseOptions.Type != "" {
		parseInit.Set("type", parseOptions.Type)
	}
	if parseOptions.UpdateViaCache != "" {
		parseInit.Set("updateViaCache", parseOptions.UpdateViaCache)
	}
	parseRawRegistration, parseErr := awaitServiceWorkerValue(parseCtx, "RegisterServiceWorker", parseOptions.URL, parseRegisterFn.Invoke(parseOptions.URL, parseInit))
	if parseErr != nil {
		return ServiceWorkerRegistration{}, parseErr
	}
	return newServiceWorkerRegistration(parseContainer, parseRawRegistration), nil
}

func newServiceWorkerRegistration(parseContainer js.Value, parseRaw js.Value) ServiceWorkerRegistration {
	parseSnapshot := func() ServiceWorkerSnapshot {
		return serviceWorkerSnapshot(parseContainer, parseRaw)
	}
	return ServiceWorkerRegistration{
		snapshot: parseSnapshot,
		backgroundSync: func() BackgroundSyncCapabilities {
			return backgroundSyncCapabilities(parseRaw)
		},
		registerSync: func(parseCtx context.Context, parseTag string) error {
			parseTag = strings.TrimSpace(parseTag)
			if parseTag == "" {
				return &interop.Error{Op: "ServiceWorkerRegistration.RegisterSync", Target: parseSnapshot().Scope, Code: interop.CodeInvalid, Err: errors.New("background sync tag is empty")}
			}
			parseSyncManager := parseRaw.Get("sync")
			if parseSyncManager.IsUndefined() || parseSyncManager.IsNull() {
				return &interop.Error{Op: "ServiceWorkerRegistration.RegisterSync", Target: parseTag, Code: interop.CodeUnavailable, Err: errors.New("background sync is unavailable for this service worker registration")}
			}
			parseRegister := parseSyncManager.Get("register")
			if parseRegister.Type() != js.TypeFunction {
				return &interop.Error{Op: "ServiceWorkerRegistration.RegisterSync", Target: parseTag, Code: interop.CodeNotFunction, Err: errors.New("service worker sync.register is not callable")}
			}
			_, parseErr := awaitServiceWorkerValue(parseCtx, "ServiceWorkerRegistration.RegisterSync", parseTag, parseRegister.Invoke(parseTag))
			return parseErr
		},
		update: func(parseCtx2 context.Context) error {
			_, parseErr2 := awaitServiceWorkerValue(parseCtx2, "ServiceWorkerRegistration.Update", parseSnapshot().Scope, parseRaw.Call("update"))
			return parseErr2
		},
		unregister: func(parseCtx3 context.Context) (bool, error) {
			parseValue, parseErr3 := awaitServiceWorkerValue(parseCtx3, "ServiceWorkerRegistration.Unregister", parseSnapshot().Scope, parseRaw.Call("unregister"))
			if parseErr3 != nil {
				return false, parseErr3
			}
			if parseValue.IsUndefined() || parseValue.IsNull() {
				return false, nil
			}
			return parseValue.Bool(), nil
		},
		skipWaiting: func(parseCtx4 context.Context) error {
			parseWaiting := parseRaw.Get("waiting")
			if parseWaiting.IsUndefined() || parseWaiting.IsNull() {
				return &interop.Error{Op: "ServiceWorkerRegistration.SkipWaiting", Target: parseSnapshot().Scope, Code: interop.CodeInvalid, Err: errors.New("no waiting service worker is available")}
			}
			parsePostMessage := parseWaiting.Get("postMessage")
			if parsePostMessage.Type() != js.TypeFunction {
				return &interop.Error{Op: "ServiceWorkerRegistration.SkipWaiting", Target: parseSnapshot().Scope, Code: interop.CodeNotFunction, Err: errors.New("waiting service worker postMessage is not callable")}
			}
			parseWaiting.Call("postMessage", map[string]any{"type": "SKIP_WAITING"})
			_ = parseCtx4
			return nil
		},
		subscribeLifecycle: func(handler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
			if handler == nil {
				return ServiceWorkerSubscription{}, &interop.Error{Op: "ServiceWorkerRegistration.SubscribeLifecycle", Target: parseSnapshot().Scope, Code: interop.CodeInvalid, Err: errors.New("lifecycle handler is nil")}
			}
			parseAttachments := []serviceWorkerEventAttachment{}
			parseAttachedWorkers := map[string]bool{}
			var parseNotify func()
			parseAttach := func(parseTarget js.Value, parseEventName string, parseFn js.Func) {
				parseTarget.Call("addEventListener", parseEventName, parseFn)
				parseAttachments = append(parseAttachments, serviceWorkerEventAttachment{target: parseTarget, eventName: parseEventName, fn: parseFn})
			}
			parseAttachWorkerStateListeners := func() {
				for _, parseWorker := range []js.Value{parseRaw.Get("installing"), parseRaw.Get("waiting"), parseRaw.Get("active")} {
					if parseWorker.IsUndefined() || parseWorker.IsNull() {
						continue
					}
					parseKey := strings.TrimSpace(parseWorker.Get("scriptURL").String()) + "|" + strings.TrimSpace(parseWorker.Get("state").String())
					if parseAttachedWorkers[parseKey] {
						continue
					}
					parseAttachedWorkers[parseKey] = true
					parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
						parseNotify()
						return nil
					})
					parseAttach(parseWorker, "statechange", parseListener)
				}
			}
			parseNotify = func() {
				parseAttachWorkerStateListeners()
				handler(parseSnapshot())
			}
			parseUpdateFound := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
				parseNotify()
				return nil
			})
			parseControllerChanged := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
				parseNotify()
				return nil
			})
			parseAttach(parseRaw, "updatefound", parseUpdateFound)
			parseAttach(parseContainer, "controllerchange", parseControllerChanged)
			parseNotify()
			return ServiceWorkerSubscription{cancel: func() {
				for _, parseAttachment := range parseAttachments {
					parseAttachment.target.Call("removeEventListener", parseAttachment.eventName, parseAttachment.fn)
					parseAttachment.fn.Release()
				}
			}}, nil
		},
		reloadOnControllerChange: func() (ServiceWorkerSubscription, error) {
			parseWindow := browserWindow()
			if parseWindow.IsUndefined() || parseWindow.IsNull() {
				return ServiceWorkerSubscription{}, serviceWorkerUnavailable("ServiceWorkerRegistration.ReloadOnControllerChange", parseSnapshot().Scope)
			}
			parseLocation := parseWindow.Get("location")
			parseReload := parseLocation.Get("reload")
			if parseReload.Type() != js.TypeFunction {
				return ServiceWorkerSubscription{}, &interop.Error{Op: "ServiceWorkerRegistration.ReloadOnControllerChange", Target: parseSnapshot().Scope, Code: interop.CodeNotFunction, Err: errors.New("window.location.reload is not callable")}
			}
			var parseListener2 js.Func
			parseListener2 = js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
				parseLocation.Call("reload")
				return nil
			})
			parseContainer.Call("addEventListener", "controllerchange", parseListener2)
			return ServiceWorkerSubscription{cancel: func() {
				parseContainer.Call("removeEventListener", "controllerchange", parseListener2)
				parseListener2.Release()
			}}, nil
		},
	}
}

type serviceWorkerEventAttachment struct {
	target    js.Value
	eventName string
	fn        js.Func
}

func normalizeServiceWorkerOptions(parseOptions ServiceWorkerOptions) ServiceWorkerOptions {
	parseOptions.URL = strings.TrimSpace(parseOptions.URL)
	parseOptions.Scope = strings.TrimSpace(parseOptions.Scope)
	parseOptions.Type = strings.TrimSpace(parseOptions.Type)
	parseOptions.UpdateViaCache = strings.TrimSpace(parseOptions.UpdateViaCache)
	return parseOptions
}

func serviceWorkerContainer() (js.Value, error) {
	parseNavigator := browserNavigator()
	if parseNavigator.IsUndefined() || parseNavigator.IsNull() {
		return js.Undefined(), serviceWorkerUnavailable("RegisterServiceWorker", "navigator.serviceWorker")
	}
	parseContainer := parseNavigator.Get("serviceWorker")
	if parseContainer.IsUndefined() || parseContainer.IsNull() {
		return js.Undefined(), serviceWorkerUnavailable("RegisterServiceWorker", "navigator.serviceWorker")
	}
	return parseContainer, nil
}

func serviceWorkerSnapshot(parseContainer js.Value, parseRaw js.Value) ServiceWorkerSnapshot {
	return ServiceWorkerSnapshot{
		Scope:         strings.TrimSpace(parseRaw.Get("scope").String()),
		HasController: !parseContainer.Get("controller").IsUndefined() && !parseContainer.Get("controller").IsNull(),
		Installing:    serviceWorkerVersion(parseRaw.Get("installing")),
		Waiting:       serviceWorkerVersion(parseRaw.Get("waiting")),
		Active:        serviceWorkerVersion(parseRaw.Get("active")),
	}
}

func backgroundSyncCapabilities(parseRaw js.Value) BackgroundSyncCapabilities {
	parseCapabilities := BackgroundSyncCapabilities{}
	parseSyncManager := parseRaw.Get("sync")
	if !parseSyncManager.IsUndefined() && !parseSyncManager.IsNull() {
		parseCapabilities.OneShot = parseSyncManager.Get("register").Type() == js.TypeFunction
	}
	parsePeriodicSync := parseRaw.Get("periodicSync")
	if !parsePeriodicSync.IsUndefined() && !parsePeriodicSync.IsNull() {
		parseCapabilities.Periodic = parsePeriodicSync.Get("register").Type() == js.TypeFunction
	}
	return parseCapabilities
}

func serviceWorkerVersion(parseWorker js.Value) ServiceWorkerVersion {
	if parseWorker.IsUndefined() || parseWorker.IsNull() {
		return ServiceWorkerVersion{}
	}
	return ServiceWorkerVersion{
		ScriptURL: strings.TrimSpace(parseWorker.Get("scriptURL").String()),
		State:     ServiceWorkerState(strings.TrimSpace(parseWorker.Get("state").String())),
	}
}

func awaitServiceWorkerValue(parseCtx context.Context, parseOp string, parseTarget string, parseValue js.Value) (js.Value, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseThen := parseValue.Get("then")
	if parseThen.Type() != js.TypeFunction {
		return parseValue, nil
	}
	parseResolvedCh := make(chan js.Value, 1)
	parseRejectedCh := make(chan error, 1)
	var parseResolveFn js.Func
	var parseRejectFn js.Func
	parseCleanup := func() {
		parseResolveFn.Release()
		parseRejectFn.Release()
	}
	parseResolveFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			select {
			case parseResolvedCh <- js.Undefined():
			default:
			}
			return nil
		}
		select {
		case parseResolvedCh <- parseArgs[0]:
		default:
		}
		return nil
	})
	parseRejectFn = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseMessage := "service worker promise rejected"
		if len(parseArgs2) > 0 {
			if parseText := strings.TrimSpace(parseArgs2[0].String()); parseText != "" {
				parseMessage = parseText
			}
		}
		select {
		case parseRejectedCh <- &interop.Error{Op: parseOp, Target: parseTarget, Code: interop.CodePromiseRejected, Err: errors.New(parseMessage)}:
		default:
		}
		return nil
	})
	parseValue.Call("then", parseResolveFn).Call("catch", parseRejectFn)
	defer parseCleanup()

	select {
	case parseResolved := <-parseResolvedCh:
		return parseResolved, nil
	case parseErr := <-parseRejectedCh:
		return js.Undefined(), parseErr
	case <-parseCtx.Done():
		parseCode := interop.CodeCancelled
		if errors.Is(parseCtx.Err(), context.DeadlineExceeded) {
			parseCode = interop.CodeTimeout
		}
		return js.Undefined(), &interop.Error{Op: parseOp, Target: parseTarget, Code: parseCode, Err: parseCtx.Err()}
	}
}

func serviceWorkerUnavailable(parseOp string, parseTarget string) error {
	return &interop.Error{Op: parseOp, Target: parseTarget, Code: interop.CodeUnavailable, Err: errors.New("service workers are unavailable in this build")}
}
