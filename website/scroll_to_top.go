//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// ScrollToTopButton creates a floating scroll to top button
func ScrollToTopButton(props Attrs) *Element {
	// State for button visibility
	isVisible, setIsVisible := GoUseState(false)

	// Effect to handle scroll events
	GoUseEffect(func() {
		handleScroll := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			scrollY := js.Global().Get("window").Get("pageYOffset").Float()

			// Show button when scrolled down more than 400px
			if scrollY > 400 {
				setIsVisible(true)
			} else {
				setIsVisible(false)
			}

			return nil
		})

		// Add scroll event listener
		js.Global().Get("window").Call("addEventListener", "scroll", handleScroll)

		// Note: Cleanup would be handled by the framework
		return
	})

	// Click handler to scroll to top
	handleScrollToTop := GoUseFunc(func(event GoEvent) {
		// Smooth scroll to top
		js.Global().Get("window").Call("scrollTo", map[string]interface{}{
			"top":      0,
			"behavior": "smooth",
		})
	})

	// Dynamic classes based on visibility
	buttonClasses := "fixed bottom-8 right-8 z-50 transition-all duration-300 ease-in-out transform"
	if isVisible() {
		buttonClasses += " opacity-100 translate-y-0 pointer-events-auto"
	} else {
		buttonClasses += " opacity-0 translate-y-16 pointer-events-none"
	}

	return Button(
		Attrs{
			"class":   buttonClasses,
			"onclick": handleScrollToTop,
			"title":   "Scroll to top",
		},

		// Button container with beautiful styling
		Div(
			Attrs{"class": "group relative overflow-hidden w-14 h-14 bg-gradient-to-br from-indigo-600 via-purple-600 to-pink-600 rounded-full shadow-lg hover:shadow-2xl transition-all duration-300 transform hover:scale-110 cursor-pointer"},

			// Background glow effect
			Div(Attrs{"class": "absolute inset-0 bg-gradient-to-r from-indigo-400 to-purple-400 rounded-full opacity-0 group-hover:opacity-20 transition-opacity duration-300 animate-pulse"}),

			// Arrow icon - using emoji for simplicity and consistency
			Div(
				Attrs{"class": "absolute inset-0 flex items-center justify-center text-white transition-transform duration-300 group-hover:-translate-y-1"},
				Span(Attrs{
					"class": "text-2xl font-bold transition-transform duration-300 group-hover:scale-110",
				}, "⬆️"),
			),

			// Floating ring animation
			Div(Attrs{"class": "absolute inset-0 rounded-full border-2 border-white/30 group-hover:border-white/50 transition-colors duration-300"}),

			// Pulse animation on hover
			Div(Attrs{"class": "absolute inset-0 rounded-full border-2 border-white/20 group-hover:animate-ping"}),
		),
	)
}
