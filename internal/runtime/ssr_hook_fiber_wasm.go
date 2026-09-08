//go:build js && wasm

package runtime

// getSSRHookFiber preserves the browser's synchronous ambient hook context.
func getSSRHookFiber() *Fiber { return nil }

// setSSRHookFiber installs the browser's transient fiber and restores its owner.
func setSSRHookFiber(parseFiber *Fiber, parseOwner uint64) func() {
	parsePrevious := GetCurrentFiber()
	parsePreviousOwner := currentFiberOwnerGoroutineID
	setCurrentFiberOwned(parseFiber, parseOwner)
	return func() { setCurrentFiberOwned(parsePrevious, parsePreviousOwner) }
}
