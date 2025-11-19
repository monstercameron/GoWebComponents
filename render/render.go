//go:build js && wasm
// +build js,wasm

package render

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/platform/jsdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Type alias for Element
type Element = runtime.Element

var initialized = false

// ensureInitialized ensures the global runtime is initialized with WASM adapters
func ensureInitialized() {
	if !initialized {
		// Create WASM adapters
		domAdapter := jsdom.NewWASMDOMAdapter()
		eventAdapter := jsdom.NewWASMEventAdapter()
		scheduler := jsdom.NewWASMScheduler()
		browserState := jsdom.NewWASMBrowserState()

		// Initialize global runtime
		runtime.InitGlobalRuntime(runtime.Config{
			DOMAdapter:   domAdapter,
			EventAdapter: eventAdapter,
			Scheduler:    scheduler,
			BrowserState: browserState,
		})

		initialized = true
	}
}

// To renders an element to a DOM node specified by a CSS selector.
// This is the primary function for mounting your application to the page.
//
// The selector can be any valid CSS selector:
//   - ID selector: "#app", "#root"
//   - Class selector: ".container"
//   - Tag selector: "body", "main"
//   - Complex selector: "div.container", "#app > div"
//
// The function queries the DOM for the first matching element and renders
// the provided Element into it. If the selector doesn't match any element,
// the function will panic.
//
// Example:
//
//	func main() {
//	    app := dom.Div(nil,
//	        dom.H1(nil, "My App"),
//	        HomePage(nil),
//	    )
//
//	    render.To(app, "#app") // Renders into <div id="app"></div>
//	}
//
// The element is mounted and the reactive rendering system is initialized.
// Subsequent updates from hooks or state changes will automatically re-render
// only the affected parts of the DOM.
func To(element *runtime.Element, selector string) {
	ensureInitialized()
	rt := runtime.GetGlobalRuntime()
	rt.RenderTo(selector, element)
}

// ToElement renders an element to a specific DOM element (js.Value).
// This is a lower-level function for advanced use cases where you already
// have a reference to a DOM element.
//
// Example:
//
//	func main() {
//	    doc := js.Global().Get("document")
//	    container := doc.Call("getElementById", "app")
//
//	    app := dom.Div(nil,
//	        dom.H1(nil, "My App"),
//	    )
//
//	    render.ToElement(app, container)
//	}
//
// Most applications should use To() with a CSS selector instead,
// as it's more convenient and readable.
func ToElement(element *runtime.Element, domElement js.Value) {
	ensureInitialized()
	rt := runtime.GetGlobalRuntime()

	// Wrap js.Value in a DOMNode
	node := jsdom.NewWASMDOMNode(domElement)
	rt.Render(element, node)
}
