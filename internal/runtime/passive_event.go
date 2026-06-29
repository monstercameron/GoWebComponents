package runtime

// PassiveEventHandler marks an event handler that should be attached with
// passive listener options when the DOM adapter supports listener binding.
type PassiveEventHandler struct {
	Handler any
}
