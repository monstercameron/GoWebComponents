package shorthand

import (
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Ref binds a ui.DOMRef to an element so its live DOM node is published into the
// ref on mount. Delegates to [html.Ref].
func Ref(parseRef ui.DOMRef) PropOption { return html.Ref(parseRef) }
