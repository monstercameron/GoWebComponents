//go:build js && wasm

package servercomponents

import (
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// ServerOnly emits a client placeholder on wasm targets so server-only render
// code can be excluded from the client bundle by build tooling.
func ServerOnly(parseProps Props) ui.Node {
	if parseProps.Placeholder != nil {
		return parseProps.Placeholder
	}
	return html.Tag("template", html.Props{
		Raw: map[string]any{
			"data-gwc-server-component": parseProps.ID,
			"data-gwc-component-name":   parseProps.Name,
		},
	})
}
