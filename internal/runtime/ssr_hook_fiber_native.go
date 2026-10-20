//go:build !js || !wasm

package runtime

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/v7/internal/errcode"
)

var storeSSRHookFibers sync.Map
var storeSSRHookCount atomic.Int64

// activeSlabFiber returns the client render fiber only when no native SSR
// component is active. SSR fibers never own the client render slabs.
func activeSlabFiber() *Fiber {
	if storeSSRHookCount.Load() != 0 {
		return nil
	}
	return currentFiber
}

// getSSRHookOwner identifies the native call stack even in production builds,
// where the optional client hook threading diagnostic is compiled out.
func getSSRHookOwner() uint64 {
	var parseBuffer [64]byte
	parseSize := runtime.Stack(parseBuffer[:], false)
	parseFields := strings.Fields(string(parseBuffer[:parseSize]))
	if len(parseFields) < 2 {
		panic(errcode.New(errcode.RuntimeSSRGoroutineIdentity, "SSR: cannot identify rendering goroutine (runtime.Stack returned an unparsable header; hooks must run on the goroutine that started the render)"))
	}
	parseOwner, parseErr := strconv.ParseUint(parseFields[1], 10, 64)
	if parseErr != nil || parseOwner == 0 {
		panic(errcode.New(errcode.RuntimeSSRGoroutineIdentity, "SSR: invalid rendering goroutine identity (goroutine id parsed as zero or non-numeric: "+strings.TrimSpace(parseFields[1])+")"))
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
func setSSRHookFiber(parseFiber *Fiber, parseOwner uint64) func() {
	if parseOwner == 0 {
		parseOwner = getSSRHookOwner()
	}
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
