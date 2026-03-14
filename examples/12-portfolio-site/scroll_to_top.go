//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// ScrollToTopButton creates a floating action button that appears when scrolling down.
// Features smooth animations, visibility management based on scroll position,
// and beautiful styling with hover effects. Auto-hides when near the top of the page.
func ScrollToTopButton(props Attrs) *Element {
	isVisible := ui.UseState(false)

	ui.UseEffect(func() func() {
		handleScroll := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			scrollY := js.Global().Get("window").Get("pageYOffset").Float()
			visible := scrollY > 400
			if visible != isVisible.Get() {
				isVisible.Set(visible)
			}
			return nil
		})
		js.Global().Get("window").Call("addEventListener", "scroll", handleScroll)
		return func() {
			js.Global().Get("window").Call("removeEventListener", "scroll", handleScroll)
			handleScroll.Release()
		}
	}, "scroll_listener")

	handleScrollToTop := ui.UseEvent(func() {
		js.Global().Get("window").Call("scrollTo", map[string]interface{}{
			"top":      0,
			"behavior": "smooth",
		})
	})

	buttonClasses := "fixed bottom-8 right-8 z-50 transition-all duration-300 ease-in-out transform"
	if isVisible.Get() {
		buttonClasses += " opacity-100 translate-y-0 pointer-events-auto"
	} else {
		buttonClasses += " opacity-0 translate-y-16 pointer-events-none"
	}

	return html.Button(html.Props{Class: buttonClasses, OnClick: handleScrollToTop, Title: "Scroll to top"},
		html.Div(html.Props{Class: "group relative overflow-hidden w-14 h-14 bg-gradient-to-br from-indigo-600 via-purple-600 to-pink-600 rounded-full shadow-lg hover:shadow-2xl transition-all duration-300 transform hover:scale-110 cursor-pointer"},
			html.Div(html.Props{Class: "absolute inset-0 bg-gradient-to-r from-indigo-400 to-purple-400 rounded-full opacity-0 group-hover:opacity-20 transition-opacity duration-300 animate-pulse"}),
			html.Div(html.Props{Class: "absolute inset-0 flex items-center justify-center text-white transition-transform duration-300 group-hover:-translate-y-1"},
				html.Span(html.Props{Class: "text-2xl font-bold transition-transform duration-300 group-hover:scale-110"}, html.Text("⬆️")),
			),
			html.Div(html.Props{Class: "absolute inset-0 rounded-full border-2 border-white/30 group-hover:border-white/50 transition-colors duration-300"}),
			html.Div(html.Props{Class: "absolute inset-0 rounded-full border-2 border-white/20 group-hover:animate-ping"}),
		),
	)
}
