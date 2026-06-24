package ui

// PointerHandlers groups the pointer callbacks for UsePointerEvents. Any nil
// field is skipped. Each receives the raw pointer Event (call JSValue() for
// clientX/clientY/pointerId, etc.).
type PointerHandlers struct {
	OnPointerDown   func(Event)
	OnPointerMove   func(Event)
	OnPointerUp     func(Event)
	OnPointerCancel func(Event)
}
