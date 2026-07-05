//go:build production

package runtime

const hookThreadingGuardEnabled = false

func computeHookGoroutineID() uint64 {
	return 0
}
