//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
)

// ScrollToTopButton creates a floating action button that appears when scrolling down.
// Features smooth animations, visibility management based on scroll position,
// and beautiful styling with hover effects. Auto-hides when near the top of the page.
func ScrollToTopButton(props Attrs) *Element {
	// Track button visibility based on scroll position
	isVisible, setIsVisible := hooks.UseState(false)

	// Set up scroll listener to control button visibility
	hooks.UseEffect(func() func() {
		handleScroll := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			scrollY := js.Global().Get("window").Get("pageYOffset").Float()

			// Show button when scrolled down 400+ pixels
			visible := scrollY > 400

			// Update state only when visibility actually changes to prevent unnecessary re-renders
			if visible != isVisible() {
				setIsVisible(visible)
			}

			return nil
		})

		// Attach scroll listener once per component lifecycle
		js.Global().Get("window").Call("addEventListener", "scroll", handleScroll)

		// Return cleanup function
		return func() {
			handleScroll.Release()
		}
	}, "scroll_listener") // Stable dependency prevents re-running

	// Handle smooth scroll to page top
	handleScrollToTop := hooks.GoUseFunc(func(event dom.GoEvent) {
		js.Global().Get("window").Call("scrollTo", map[string]interface{}{
			"top":      0,
			"behavior": "smooth",
		})
	})

	// Dynamic styling based on visibility state
	buttonClasses := "fixed bottom-8 right-8 z-50 transition-all duration-300 ease-in-out transform"
	if isVisible() {
		buttonClasses += " opacity-100 translate-y-0 pointer-events-auto"
	} else {
		buttonClasses += " opacity-0 translate-y-16 pointer-events-none"
	}

	return dom.Button(
		Attrs{
			"class":   buttonClasses,
			"onclick": handleScrollToTop,
			"title":   "Scroll to top",
		},

		// Button container with gradient styling and animations
		dom.Div(
			Attrs{"class": "group relative overflow-hidden w-14 h-14 bg-gradient-to-br from-indigo-600 via-purple-600 to-pink-600 rounded-full shadow-lg hover:shadow-2xl transition-all duration-300 transform hover:scale-110 cursor-pointer"},

			// Background glow effect on hover
			dom.Div(Attrs{"class": "absolute inset-0 bg-gradient-to-r from-indigo-400 to-purple-400 rounded-full opacity-0 group-hover:opacity-20 transition-opacity duration-300 animate-pulse"}),

			// Arrow icon with hover animation
			dom.Div(
				Attrs{"class": "absolute inset-0 flex items-center justify-center text-white transition-transform duration-300 group-hover:-translate-y-1"},
				dom.Span(Attrs{
					"class": "text-2xl font-bold transition-transform duration-300 group-hover:scale-110",
				}, "⬆️"),
			),

			// Decorative border ring
			dom.Div(Attrs{"class": "absolute inset-0 rounded-full border-2 border-white/30 group-hover:border-white/50 transition-colors duration-300"}),

			// Pulse animation on interaction
			dom.Div(Attrs{"class": "absolute inset-0 rounded-full border-2 border-white/20 group-hover:animate-ping"}),
		),
	)
}
