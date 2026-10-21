//go:build js && wasm

package runtime

// activeSlabFiber returns the current synchronous browser render fiber.
func activeSlabFiber() *Fiber { return currentFiber }

// getSSRHookOwner is unused for browser goroutine ownership.
func getSSRHookOwner() uint64 { return 0 }

// getSSRHookFiber preserves the browser's synchronous ambient hook context.
func getSSRHookFiber() *Fiber { return nil }

// setSSRHookFiber installs the browser's transient fiber and restores its owner.
func setSSRHookFiber(parseFiber *Fiber, parseOwner uint64) func() {
	parsePrevious := GetCurrentFiber()
	parsePreviousOwner := currentFiberOwnerGoroutineID
	setCurrentFiberOwned(parseFiber, parseOwner)
	return func() { setCurrentFiberOwned(parsePrevious, parsePreviousOwner) }
}
