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

func ObserveInstallability(options InstallabilityOptions) (InstallabilityManager, error) {
	window := browserWindow()
	if window.IsUndefined() || window.IsNull() {
		return InstallabilityManager{}, installabilityUnavailable("ObserveInstallability", "window")
	}
	manifestValid := true
	manifestError := ""
	if options.Manifest == nil {
		manifestValid = false
		manifestError = "web app manifest was not supplied for validation"
	} else if err := options.Manifest.Validate(); err != nil {
		manifestValid = false
		manifestError = err.Error()
	}
	var promptEvent js.Value
	installed := detectInstalledDisplayMode(window)
	computeState := func() InstallabilityState {
		state := InstallabilityState{
			ManifestValid:   manifestValid,
			ManifestError:   manifestError,
			PromptAvailable: promptEvent.Truthy(),
			Installed:       installed,
		}
		state.Reasons = installabilityReasons(window, state)
		return state
	}
	return InstallabilityManager{
		state: computeState,
		prompt: func(ctx context.Context) (InstallPromptResult, error) {
			if ctx == nil {
				ctx = context.Background()
			}
			if !promptEvent.Truthy() {
				return InstallPromptResult{}, &interop.Error{Op: "InstallabilityManager.Prompt", Code: interop.CodeInvalid, Err: errors.New("install prompt is not currently available")}
			}
			promptFn := promptEvent.Get("prompt")
			if promptFn.Type() != js.TypeFunction {
				return InstallPromptResult{}, &interop.Error{Op: "InstallabilityManager.Prompt", Code: interop.CodeNotFunction, Err: errors.New("beforeinstallprompt.prompt is not callable")}
			}
			if _, err := awaitInstallabilityValue(ctx, "InstallabilityManager.Prompt", "beforeinstallprompt.prompt", promptFn.Invoke()); err != nil {
				return InstallPromptResult{}, err
			}
			choice := promptEvent.Get("userChoice")
			resolved, err := awaitInstallabilityValue(ctx, "InstallabilityManager.Prompt", "beforeinstallprompt.userChoice", choice)
			if err != nil {
				return InstallPromptResult{}, err
			}
			result := InstallPromptResult{}
			if !resolved.IsUndefined() && !resolved.IsNull() {
				result.Outcome = strings.TrimSpace(resolved.Get("outcome").String())
				result.Platform = strings.TrimSpace(resolved.Get("platform").String())
			}
			promptEvent = js.Undefined()
			return result, nil
		},
		subscribe: func(handler func(InstallabilityState)) (InstallabilitySubscription, error) {
			if handler == nil {
				return InstallabilitySubscription{}, &interop.Error{Op: "InstallabilityManager.Subscribe", Code: interop.CodeInvalid, Err: errors.New("installability handler is nil")}
			}
			beforeInstall := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				if len(args) > 0 {
					promptEvent = args[0]
					preventDefault := promptEvent.Get("preventDefault")
					if preventDefault.Type() == js.TypeFunction {
						preventDefault.Invoke()
					}
				}
				handler(computeState())
				return nil
			})
			appInstalled := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				installed = true
				promptEvent = js.Undefined()
				handler(computeState())
				return nil
			})
			window.Call("addEventListener", "beforeinstallprompt", beforeInstall)
			window.Call("addEventListener", "appinstalled", appInstalled)
			handler(computeState())
			return InstallabilitySubscription{cancel: func() {
				window.Call("removeEventListener", "beforeinstallprompt", beforeInstall)
				window.Call("removeEventListener", "appinstalled", appInstalled)
				beforeInstall.Release()
				appInstalled.Release()
			}}, nil
		},
	}, nil
}

func detectInstalledDisplayMode(window js.Value) bool {
	matchMedia := window.Get("matchMedia")
	if matchMedia.Type() == js.TypeFunction {
		result := matchMedia.Invoke("(display-mode: standalone)")
		if result.Truthy() && result.Get("matches").Bool() {
			return true
		}
	}
	navigator := browserNavigator()
	if !navigator.IsUndefined() && !navigator.IsNull() {
		standalone := navigator.Get("standalone")
		if standalone.Type() == js.TypeBoolean && standalone.Bool() {
			return true
		}
	}
	return false
}

func installabilityReasons(window js.Value, state InstallabilityState) []string {
	reasons := make([]string, 0, 4)
	if !state.ManifestValid {
		reasons = append(reasons, state.ManifestError)
	}
	isSecure := window.Get("isSecureContext")
	if isSecure.Type() == js.TypeBoolean && !isSecure.Bool() {
		reasons = append(reasons, "app is not running in a secure context")
	}
	if state.Installed {
		reasons = append(reasons, "app is already running in an installed display mode")
		return reasons
	}
	if !state.PromptAvailable {
		reasons = append(reasons, "browser has not exposed an install prompt for this app yet")
	}
	return reasons
}

func awaitInstallabilityValue(ctx context.Context, op string, target string, value js.Value) (js.Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if value.IsUndefined() || value.IsNull() {
		return value, nil
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
		resolved := js.Undefined()
		if len(args) > 0 {
			resolved = args[0]
		}
		select {
		case resolvedCh <- resolved:
		default:
		}
		return nil
	})
	rejectFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message := "installability promise rejected"
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

func installabilityUnavailable(op string, target string) error {
	return &interop.Error{Op: op, Target: target, Code: interop.CodeUnavailable, Err: errors.New("installability helpers are unavailable in this build")}
}
