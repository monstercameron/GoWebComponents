package interop

// KeepAlive blocks the calling goroutine forever.
//
// In js/wasm programs it keeps main alive after the UI mounts: if main returned,
// the Go runtime would exit and invalidate every registered event callback, so
// the app would die on the first interaction. It is the single keep-alive
// primitive shared by utils.WaitForever and ui.Run, kept in this leaf package so
// both can reuse it without an import cycle.
func KeepAlive() {
	select {}
}
