//go:build production

package runtime

const hookThreadingGuardEnabled = false

func currentHookGoroutineID() uint64 {
	return 0
}
