//go:build !js || !wasm

package runtime

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var storeSSRHookFibers sync.Map
var storeSSRHookCount atomic.Int64

// getSSRHookOwner identifies the native call stack even in production builds,
// where the optional client hook threading diagnostic is compiled out.
func getSSRHookOwner() uint64 {
	var parseBuffer [64]byte
	parseSize := runtime.Stack(parseBuffer[:], false)
	parseFields := strings.Fields(string(parseBuffer[:parseSize]))
	if len(parseFields) < 2 {
		panic("SSR: cannot identify rendering goroutine")
	}
	parseOwner, parseErr := strconv.ParseUint(parseFields[1], 10, 64)
	if parseErr != nil || parseOwner == 0 {
		panic("SSR: invalid rendering goroutine identity")
	}
	return parseOwner
}

// getSSRHookFiber retrieves only this goroutine's transient SSR fiber.
func getSSRHookFiber() *Fiber {
	if storeSSRHookCount.Load() == 0 {
		return nil
	}
	parseValue, _ := storeSSRHookFibers.Load(getSSRHookOwner())
	parseFiber, _ := parseValue.(*Fiber)
	return parseFiber
}

// setSSRHookFiber installs a scoped native fiber without mutating the client
// render globals. Cleanup restores nested renders and removes top-level entries
// on normal return, suspension, and panic; no request survives in the map.
func setSSRHookFiber(parseFiber *Fiber, _ uint64) func() {
	parseOwner := getSSRHookOwner()
	parsePrevious, hasPrevious := storeSSRHookFibers.Load(parseOwner)
	storeSSRHookFibers.Store(parseOwner, parseFiber)
	storeSSRHookCount.Add(1)
	return func() {
		if hasPrevious {
			storeSSRHookFibers.Store(parseOwner, parsePrevious)
		} else {
			storeSSRHookFibers.Delete(parseOwner)
		}
		storeSSRHookCount.Add(-1)
	}
}
