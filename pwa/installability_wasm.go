//go:build js && wasm

package pwa

import (
	"context"
	"errors"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// ObserveInstallability returns an InstallabilityManager that tracks browser install prompt events.
func ObserveInstallability(parseOptions InstallabilityOptions) (InstallabilityManager, error) {
	parseWindow := browserWindow()
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return InstallabilityManager{}, installabilityUnavailable("ObserveInstallability", "window")
	}
	isParseManifestValid := true
	parseManifestError := ""
	if parseOptions.Manifest == nil {
		isParseManifestValid = false
		parseManifestError = "web app manifest was not supplied for validation"
	} else if parseErr := parseOptions.Manifest.Validate(); parseErr != nil {
		isParseManifestValid = false
		parseManifestError = parseErr.Error()
	}
	var parsePromptEvent js.Value
	parseInstalled := detectInstalledDisplayMode(parseWindow)
	parseComputeState := func() InstallabilityState {
		parseState := InstallabilityState{
			ManifestValid:   isParseManifestValid,
			ManifestError:   parseManifestError,
			PromptAvailable: parsePromptEvent.Truthy(),
			Installed:       parseInstalled,
		}
		parseState.Reasons = installabilityReasons(parseWindow, parseState)
		return parseState
	}
	return InstallabilityManager{
		state: parseComputeState,
		prompt: func(parseCtx context.Context) (InstallPromptResult, error) {
			if parseCtx == nil {
				parseCtx = context.Background()
			}
			if !parsePromptEvent.Truthy() {
				return InstallPromptResult{}, &interop.Error{Op: "InstallabilityManager.Prompt", Code: interop.CodeInvalid, Err: errors.New("install prompt is not currently available")}
			}
			parsePromptFn := parsePromptEvent.Get("prompt")
			if parsePromptFn.Type() != js.TypeFunction {
				return InstallPromptResult{}, &interop.Error{Op: "InstallabilityManager.Prompt", Code: interop.CodeNotFunction, Err: errors.New("beforeinstallprompt.prompt is not callable")}
			}
			if _, parseErr2 := awaitInstallabilityValue(parseCtx, "InstallabilityManager.Prompt", "beforeinstallprompt.prompt", parsePromptFn.Invoke()); parseErr2 != nil {
				return InstallPromptResult{}, parseErr2
			}
			parseChoice := parsePromptEvent.Get("userChoice")
			parseResolved, parseErr3 := awaitInstallabilityValue(parseCtx, "InstallabilityManager.Prompt", "beforeinstallprompt.userChoice", parseChoice)
			if parseErr3 != nil {
				return InstallPromptResult{}, parseErr3
			}
			parseResult := InstallPromptResult{}
			if !parseResolved.IsUndefined() && !parseResolved.IsNull() {
				parseResult.Outcome = strings.TrimSpace(parseResolved.Get("outcome").String())
				parseResult.Platform = strings.TrimSpace(parseResolved.Get("platform").String())
			}
			parsePromptEvent = js.Undefined()
			return parseResult, nil
		},
		subscribe: func(handler func(InstallabilityState)) (InstallabilitySubscription, error) {
			if handler == nil {
				return InstallabilitySubscription{}, &interop.Error{Op: "InstallabilityManager.Subscribe", Code: interop.CodeInvalid, Err: errors.New("installability handler is nil")}
			}
			parseBeforeInstall := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				if len(parseArgs) > 0 {
					parsePromptEvent = parseArgs[0]
					parsePreventDefault := parsePromptEvent.Get("preventDefault")
					if parsePreventDefault.Type() == js.TypeFunction {
						parsePreventDefault.Invoke()
					}
				}
				handler(parseComputeState())
				return nil
			})
			parseAppInstalled := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
				parseInstalled = true
				parsePromptEvent = js.Undefined()
				handler(parseComputeState())
				return nil
			})
			parseWindow.Call("addEventListener", "beforeinstallprompt", parseBeforeInstall)
			parseWindow.Call("addEventListener", "appinstalled", parseAppInstalled)
			handler(parseComputeState())
			return InstallabilitySubscription{cancel: func() {
				parseWindow.Call("removeEventListener", "beforeinstallprompt", parseBeforeInstall)
				parseWindow.Call("removeEventListener", "appinstalled", parseAppInstalled)
				parseBeforeInstall.Release()
				parseAppInstalled.Release()
			}}, nil
		},
	}, nil
}

// detectInstalledDisplayMode reports whether the app is already running in an installed display mode.
func detectInstalledDisplayMode(parseWindow js.Value) bool {
	parseMatchMedia := parseWindow.Get("matchMedia")
	if parseMatchMedia.Type() == js.TypeFunction {
		parseResult := parseMatchMedia.Invoke("(display-mode: standalone)")
		if parseResult.Truthy() && parseResult.Get("matches").Bool() {
			return true
		}
	}
	parseNavigator := browserNavigator()
	if !parseNavigator.IsUndefined() && !parseNavigator.IsNull() {
		parseStandalone := parseNavigator.Get("standalone")
		if parseStandalone.Type() == js.TypeBoolean && parseStandalone.Bool() {
			return true
		}
	}
	return false
}

// installabilityReasons explains why the app is or is not currently installable.
func installabilityReasons(parseWindow js.Value, parseState InstallabilityState) []string {
	parseReasons := make([]string, 0, 4)
	if !parseState.ManifestValid {
		parseReasons = append(parseReasons, parseState.ManifestError)
	}
	isSecure := parseWindow.Get("isSecureContext")
	if isSecure.Type() == js.TypeBoolean && !isSecure.Bool() {
		parseReasons = append(parseReasons, "app is not running in a secure context")
	}
	if parseState.Installed {
		parseReasons = append(parseReasons, "app is already running in an installed display mode")
		return parseReasons
	}
	if !parseState.PromptAvailable {
		parseReasons = append(parseReasons, "browser has not exposed an install prompt for this app yet")
	}
	return parseReasons
}

// awaitInstallabilityValue resolves one installability return value that may already be settled or may be promise-like.
func awaitInstallabilityValue(parseCtx context.Context, parseOp string, parseTarget string, parseValue js.Value) (js.Value, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return parseValue, nil
	}
	parseValueType := parseValue.Type()
	// Primitive results are already settled values under syscall/js and do not safely expose promise methods.
	if parseValueType != js.TypeObject && parseValueType != js.TypeFunction {
		return parseValue, nil
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
		parseResolved := js.Undefined()
		if len(parseArgs) > 0 {
			parseResolved = parseArgs[0]
		}
		select {
		case parseResolvedCh <- parseResolved:
		default:
		}
		return nil
	})
	parseRejectFn = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseMessage := "installability promise rejected"
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
	case parseResolved2 := <-parseResolvedCh:
		return parseResolved2, nil
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

// installabilityUnavailable builds one consistent installability unavailable error.
func installabilityUnavailable(parseOp string, parseTarget string) error {
	return &interop.Error{Op: parseOp, Target: parseTarget, Code: interop.CodeUnavailable, Err: errors.New("installability helpers are unavailable in this build")}
}
