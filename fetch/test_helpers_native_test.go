//go:build !js || !wasm
// +build !js !wasm

package fetch

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/interop"
)

// fetchTestNoOpScheduler keeps native fetch tests deterministic without background runtime timers.
type fetchTestNoOpScheduler struct{}

// RequestIdleCallback ignores native idle callback scheduling in fetch tests.
func (fetchTestNoOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

// SetTimeout ignores native timeout scheduling in fetch tests.
func (fetchTestNoOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

// setFetchTestStructField writes one unexported interop field so native fetch tests can build lightweight doubles.
func setFetchTestStructField(parseT *testing.T, parseTarget interface{}, parseField string, parseValue interface{}) {
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
	setFetchTestStructField(parseT, &parseStore, "backend", func() string { return "memory" })
	setFetchTestStructField(parseT, &parseStore, "getItem", func(parseCtx context.Context, parseKey string) (string, bool, error) {
		_ = parseCtx
		parseValue, parseOk := parseData[parseKey]
		return parseValue, parseOk, nil
	})
	setFetchTestStructField(parseT, &parseStore, "setItem", func(parseCtx context.Context, parseKey string, parseValue string) error {
		_ = parseCtx
		parseData[parseKey] = parseValue
		return nil
	})
	setFetchTestStructField(parseT, &parseStore, "removeItem", func(parseCtx context.Context, parseKey string) error {
		_ = parseCtx
		delete(parseData, parseKey)
		return nil
	})
	setFetchTestStructField(parseT, &parseStore, "close", func() error { return nil })
	return parseStore, parseData
}

// waitFetchTestCondition waits until the provided predicate succeeds or the timeout elapses.
func waitFetchTestCondition(parseT *testing.T, parseTimeout time.Duration, parseCheck func() bool) {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		if parseCheck() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatal("timed out waiting for fetch test condition")
}
