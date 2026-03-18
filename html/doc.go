// Package html provides typed HTML element builders for GoWebComponents.
//
// The package is designed to pair with the ui package:
//
//	func Counter() ui.Node {
//	    count := ui.UseState(0)
//	    increment := ui.UseEvent(func() {
//	        count.Update(func(v int) int { return v + 1 })
//	    })
//
//	    return html.Div(html.Props{},
//	        html.H1(html.Props{}, html.Text("Counter")),
//	        html.Button(html.Props{OnClick: increment}, html.Text("Increment")),
//	    )
//	}
//
// html.Props keeps common DOM metadata explicit while still exposing Raw for
// escape-hatch attributes that do not need first-class fields yet. Small
// convenience builders such as HiddenInput help with repetitive form markup
// without changing the underlying explicit props model.
package html
