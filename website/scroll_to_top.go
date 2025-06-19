//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"
	"time"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// ScrollToTopButton creates a floating scroll to top button and handles dynamic page titles
func ScrollToTopButton(props Attrs) *Element {
	// State for button visibility
	isVisible, setIsVisible := GoUseState(false)
	// State for current section
	_, setCurrentSection := GoUseState("Personal Website 2025")

	// Effect to handle scroll events with Go-based throttling
	GoUseEffect(func() {
		// Create a channel for scroll events
		scrollChan := make(chan float64, 10)

		// Start a Go routine to process scroll events with throttling
		go func() {
			var lastProcessTime time.Time
			throttleDelay := 100 * time.Millisecond

			for scrollY := range scrollChan {
				// Throttle processing
				if time.Since(lastProcessTime) < throttleDelay {
					continue
				}
				lastProcessTime = time.Now()

				// Process scroll position
				processScrollPosition(scrollY, setIsVisible, setCurrentSection)
			}
		}()

		// JavaScript scroll event listener (minimal JS usage)
		handleScroll := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			scrollY := js.Global().Get("window").Get("pageYOffset").Float()

			// Send to Go channel for processing
			select {
			case scrollChan <- scrollY:
			default:
				// Channel full, skip this event
			}

			return nil
		})

		// Add scroll event listener with passive option
		js.Global().Get("window").Call("addEventListener", "scroll", handleScroll, map[string]interface{}{
			"passive": true,
		})

		// Set initial state
		processScrollPosition(0, setIsVisible, setCurrentSection)

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

// processScrollPosition handles scroll position processing in Go
func processScrollPosition(scrollY float64, setIsVisible func(bool), setCurrentSection func(string)) {
	// Show button when scrolled down more than 400px
	if scrollY > 400 {
		setIsVisible(true)
	} else {
		setIsVisible(false)
	}

	// Update page title based on current section in view
	updatePageTitle(scrollY, setCurrentSection)
}

// updatePageTitle updates the page title based on the current scroll position
func updatePageTitle(scrollY float64, setCurrentSection func(string)) {
	var title string
	var section string

	// Get viewport height for more accurate calculations
	viewportHeight := js.Global().Get("window").Get("innerHeight").Float()

	// Use element-based detection for more accurate section tracking
	doc := js.Global().Get("document")

	// Check which section is most visible in the viewport
	sections := []struct {
		id      string
		title   string
		section string
	}{
		{"", "Personal Website 2025 - Earl Cameron", "Home"}, // Top of page
		{"about", "About Me - Earl Cameron", "About"},
		{"skills", "Skills & Technologies - Earl Cameron", "Skills"},
		{"youtube", "YouTube Channel - Earl Cameron", "YouTube"},
		{"projects", "Recent Projects - Earl Cameron", "Projects"},
		{"gwc-showcase", "GoWebComponents Showcase - Earl Cameron", "Showcase"},
		{"examples", "Mini Apps Gallery - Earl Cameron", "Examples"},
		{"api", "Why GoWebComponents? - Earl Cameron", "API Documentation"},
		{"contact", "Contact - Earl Cameron", "Contact"},
	}

	// Default to first section
	title = sections[0].title
	section = sections[0].section

	// Find the most visible section
	for i := len(sections) - 1; i >= 0; i-- {
		s := sections[i]
		if s.id == "" {
			// Handle top of page
			if scrollY < viewportHeight/2 {
				title = s.title
				section = s.section
			}
			continue
		}

		element := doc.Call("getElementById", s.id)
		if !element.IsNull() {
			rect := element.Call("getBoundingClientRect")
			top := rect.Get("top").Float()
			bottom := rect.Get("bottom").Float()

			// Check if section is visible in viewport (top half determines active section)
			if top <= viewportHeight/2 && bottom > 0 {
				title = s.title
				section = s.section
				break
			}
		}
	}

	// Only update document title if it's different from current title
	currentTitle := js.Global().Get("document").Get("title").String()
	if currentTitle != title {
		// Use Go timer to delay title update (replaces JavaScript setTimeout)
		go func() {
			timer := time.NewTimer(50 * time.Millisecond)
			<-timer.C

			// Update title after delay to ensure it runs after conflicting JS
			js.Global().Get("document").Set("title", title)
		}()
	}

	// Update current section state
	setCurrentSection(section)
}
