package html

import (
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Ref binds a ui.DOMRef to the element so its live DOM node is published into the
// ref on mount (and cleared on unmount). It carries the ref's sink through the
// reserved props key, which the reconciler skips as a DOM attribute. Attach at
// most one Ref per element, and attach it at mount time.
//
//	r := ui.UseDOMRef()
//	html.Input(html.Ref(r))
//
// A zero-value DOMRef (not produced by UseDOMRef) yields a no-op.
func Ref(parseRef ui.DOMRef) PropOption {
	parseSink := parseRef.Sink()
	if parseSink == nil {
		return optionFunc(func(*Props) {})
	}
	return optionFunc(func(parseProps *Props) {
		if parseProps.Raw == nil {
			parseProps.Raw = map[string]any{}
		}
		parseProps.Raw[runtime.DOMRefKey] = parseSink
	})
}
