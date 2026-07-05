//go:build !js || !wasm

package fetch

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/interop"
)

// fetchTestNoOpScheduler keeps native fetch tests deterministic without background runtime timers.
type fetchTestNoOpScheduler struct{}

// RequestIdleCallback ignores native idle callback scheduling in fetch tests.
func (fetchTestNoOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

// SetTimeout ignores native timeout scheduling in fetch tests.
func (fetchTestNoOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

// setFetchTestStructField writes one unexported interop field so native fetch tests can build lightweight doubles.
func setFetchTestStructField(parseT *testing.T, parseTarget any, parseField string, parseValue any) {
	parseT.Helper()
	parseStructValue := reflect.ValueOf(parseTarget).Elem()
	parseFieldValue := parseStructValue.FieldByName(parseField)
	if !parseFieldValue.IsValid() {
		parseT.Fatalf("missing field %q on %T", parseField, parseTarget)
	}
	reflect.NewAt(parseFieldValue.Type(), unsafe.Pointer(parseFieldValue.UnsafeAddr())).Elem().Set(reflect.ValueOf(parseValue))
}

// resetFetchTestCacheState clears shared fetch package globals between native tests.
func resetFetchTestCacheState() {
	cachedResourceRegistry = sync.Map{}
	queryTagIndex = sync.Map{}
	ConfigurePersistentCache(PersistentCacheOptions{})
}

// installFetchTestHookContext resets the runtime and installs one active fiber for native hook-based fetch tests.
func installFetchTestHookContext(parseT *testing.T) {
	parseT.Helper()
	resetFetchTestCacheState()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: fetchTestNoOpScheduler{}, Reset: true})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
		resetFetchTestCacheState()
	})
}

// buildFetchTestPersistentStore returns one in-memory persistent-store double for native fetch tests.
func buildFetchTestPersistentStore(parseT *testing.T) (interop.PersistentStore, map[string]string) {
	parseT.Helper()
	parseStore := interop.PersistentStore{}
	parseData := map[string]string{}
	var parseMu sync.Mutex
	setFetchTestStructField(parseT, &parseStore, "backend", func() string { return "memory" })
	setFetchTestStructField(parseT, &parseStore, "getItem", func(parseCtx context.Context, parseKey string) (string, bool, error) {
		_ = parseCtx
		parseMu.Lock()
		defer parseMu.Unlock()
		parseValue, parseOk := parseData[parseKey]
		return parseValue, parseOk, nil
	})
	setFetchTestStructField(parseT, &parseStore, "setItem", func(parseCtx context.Context, parseKey string, parseValue string) error {
		_ = parseCtx
		parseMu.Lock()
		defer parseMu.Unlock()
		parseData[parseKey] = parseValue
		return nil
	})
	setFetchTestStructField(parseT, &parseStore, "removeItem", func(parseCtx context.Context, parseKey string) error {
		_ = parseCtx
		parseMu.Lock()
		defer parseMu.Unlock()
		delete(parseData, parseKey)
		return nil
	})
	setFetchTestStructField(parseT, &parseStore, "close", func() error { return nil })
	return parseStore, parseData
}

// waitFetchTestCondition waits until the provided predicate succeeds or the timeout elapses.
// Callers pass an "eventually" bound, not a precise one, so the ceiling is
// floored at 5s: 1s ceilings flaked on loaded CI runners (the v4.1.0 release
// run) while a larger ceiling only ever extends failing runs.
func waitFetchTestCondition(parseT *testing.T, parseTimeout time.Duration, parseCheck func() bool) {
	parseT.Helper()
	if parseTimeout < 5*time.Second {
		parseTimeout = 5 * time.Second
	}
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		if parseCheck() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatal("timed out waiting for fetch test condition")
}
