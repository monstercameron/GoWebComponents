//go:build js && wasm

package i18n

import (
	"reflect"
	"syscall/js"
	"testing"
	"unsafe"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// i18nTestNoOpScheduler keeps wasm locale tests deterministic without background runtime scheduling.
type i18nTestNoOpScheduler struct{}

// RequestIdleCallback ignores idle callback scheduling in i18n wasm tests.
func (i18nTestNoOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

// SetTimeout ignores timeout scheduling in i18n wasm tests.
func (i18nTestNoOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

// installI18nWasmHookContext resets the runtime and installs one active fiber for hook-based locale tests.
func installI18nWasmHookContext(parseT *testing.T) *runtime.Fiber {
	parseT.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: i18nTestNoOpScheduler{}, Reset: true})
	parseFiber := &runtime.Fiber{}
	runtime.SetCurrentFiber(parseFiber)
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
	return parseFiber
}

// getI18nTestFiberEffects returns the queued effect slice from one test fiber.
func getI18nTestFiberEffects(parseT *testing.T, parseFiber *runtime.Fiber) []runtime.Effect {
	parseT.Helper()
	parseFiberValue := reflect.ValueOf(parseFiber).Elem()
	parseEffectsField := parseFiberValue.FieldByName("effects")
	return reflect.NewAt(parseEffectsField.Type(), unsafe.Pointer(parseEffectsField.UnsafeAddr())).Elem().Interface().([]runtime.Effect)
}

// setI18nTestGlobalValue swaps one browser global for the duration of a test.
func setI18nTestGlobalValue(parseName string, parseValue interface{}) func() {
	parseGlobal := js.Global()
	parsePrev := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parsePrev)
	}
}

// TestUseLocaleWasmDetectsBrowserLocaleAndPersistsSelection covers the reactive locale state path on js/wasm.
func TestUseLocaleWasmDetectsBrowserLocaleAndPersistsSelection(parseT *testing.T) {
	parseFiber := installI18nWasmHookContext(parseT)

	parseStorage := js.Global().Get("Object").New()
	parseData := map[string]string{"app-locale": "fr"}
	parseGetItem := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseData[parseArgs[0].String()]
	})
	parseSetItem := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseData[parseArgs[0].String()] = parseArgs[1].String()
		return nil
	})
	parseStorage.Set("getItem", parseGetItem)
	parseStorage.Set("setItem", parseSetItem)
	parseT.Cleanup(func() {
		parseGetItem.Release()
		parseSetItem.Release()
	})

	parseWindow := js.Global().Get("Object").New()
	parseWindow.Set("localStorage", parseStorage)
	parseRestoreWindow := setI18nTestGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	var parseChanges []string
	parseHandle := UseLocale(LocaleOptions{
		InitialLocale:    "",
		SupportedLocales: []string{"en", "fr"},
		FallbackLocale:   "en",
		DetectBrowser:    true,
		PersistenceKey:   "app-locale",
		OnChange: func(parseLocale string) {
			parseChanges = append(parseChanges, parseLocale)
		},
	})

	if parseGot := parseHandle.Get(); primaryLanguage(parseGot) != "en" {
		parseT.Fatalf("expected detected browser locale to resolve through the english branch, got %q", parseGot)
	}
	parseEffects := getI18nTestFiberEffects(parseT, parseFiber)
	if len(parseEffects) != 2 {
		parseT.Fatalf("expected two queued locale effects, got %d", len(parseEffects))
	}
	for _, parseEffect := range parseEffects {
		if parseEffect.Fn != nil {
			parseEffect.Fn()
		}
	}

	if parseGot2 := parseHandle.Get(); parseGot2 != "fr" {
		parseT.Fatalf("expected persisted locale to remain fr, got %q", parseGot2)
	}
	if parseHandle.Direction() != DirectionLTR {
		parseT.Fatalf("expected french locale direction to stay ltr, got %q", parseHandle.Direction())
	}
	if parseHandle.FallbackLocale() != "en" {
		parseT.Fatalf("expected fallback locale en, got %q", parseHandle.FallbackLocale())
	}
	parseSupported := parseHandle.SupportedLocales()
	parseSupported[0] = "mutated"
	if parseHandle.SupportedLocales()[0] != "en" {
		parseT.Fatalf("expected supported locales clone, got %v", parseHandle.SupportedLocales())
	}
	if parseData["app-locale"] != "fr" {
		parseT.Fatalf("expected persisted locale write, storage=%v", parseData)
	}
	if len(parseChanges) != 1 || parseChanges[0] != "fr" {
		parseT.Fatalf("expected one locale change callback for fr, got %v", parseChanges)
	}

	parseHandle.Set("unknown")
	if parseHandle.Get() != "en" {
		parseT.Fatalf("expected unsupported locale to fall back to en, got %q", parseHandle.Get())
	}
}

// TestUseLocaleWasmBrowserHelpersHandleMissingGlobals covers the browser helper fallbacks on js/wasm.
func TestUseLocaleWasmBrowserHelpersHandleMissingGlobals(parseT *testing.T) {
	parseRestoreWindow := setI18nTestGlobalValue("window", js.Undefined())
	defer parseRestoreWindow()

	if parseStorage := browserStorage(); parseStorage.Truthy() {
		parseT.Fatalf("expected browserStorage to be unavailable without window, got %v", parseStorage)
	}
	if parseStored := readPersistedLocale("missing"); parseStored != "" {
		parseT.Fatalf("expected empty persisted locale without storage, got %q", parseStored)
	}
	writePersistedLocale("missing", "fr")
	if parseGot := firstLocale(nil); parseGot != "" {
		parseT.Fatalf("expected firstLocale(nil) to return empty, got %q", parseGot)
	}
}
