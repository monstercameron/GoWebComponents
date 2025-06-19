//go:build js && wasm
// +build js,wasm

package website

import (
	"fmt"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// NavBar creates a modern, responsive navigation bar with glassmorphism and animations
func NavBar(props Attrs) *Element {
	// State for mobile menu and scroll behavior
	isMobileMenuOpen, setIsMobileMenuOpen := GoUseState(false)
	isScrolled, setIsScrolled := GoUseState(false)
	scrollProgress, setScrollProgress := GoUseState(0.0)

	// Handle mobile menu toggle
	handleMobileToggle := GoUseFunc(func(event GoEvent) {
		setIsMobileMenuOpen(!isMobileMenuOpen())
	})

	// Handle scroll effect and initialize smooth scrolling
	GoUseEffect(func() {
		// Initialize smooth scroll CSS
		AddSmoothScrollCSS()

		// Add scroll event listener
		scrollHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			scrollY := js.Global().Get("window").Get("scrollY").Float()
			setIsScrolled(scrollY > 20)

			// Calculate scroll progress for progress bar
			windowHeight := js.Global().Get("window").Get("innerHeight").Float()
			documentHeight := js.Global().Get("document").Get("documentElement").Get("scrollHeight").Float()
			maxScroll := documentHeight - windowHeight

			if maxScroll > 0 {
				progress := scrollY / maxScroll
				if progress > 1 {
					progress = 1
				}
				setScrollProgress(progress)
			}

			return nil
		})

		js.Global().Get("window").Call("addEventListener", "scroll", scrollHandler)

		// Add smooth scroll behavior for any missed elements
		js.Global().Get("document").Get("documentElement").Get("style").Set("scrollBehavior", "smooth")

		// Cleanup function would go here in a real useEffect
		return
	})

	// Dynamic classes based on state
	navClasses := "fixed top-0 left-0 right-0 z-50 transition-all duration-300 ease-in-out"
	if isScrolled() {
		navClasses += " bg-white/80 backdrop-blur-xl shadow-2xl border-b border-white/20"
	} else {
		navClasses += " bg-white/60 backdrop-blur-lg shadow-lg"
	}

	return Nav(
		Attrs{"class": navClasses},

		// Scroll progress bar
		Div(
			Attrs{
				"class": "absolute bottom-0 left-0 h-1 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 transition-all duration-300 ease-out",
				"style": "width: " + fmt.Sprintf("%.1f", scrollProgress()*100) + "%",
			},
		),

		// Main navbar container
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "flex items-center justify-between h-16 md:h-20 gap-4"},

				// Logo/Brand Section with enhanced animation
				Div(
					Attrs{"class": "flex items-center space-x-3 group"},

					// Animated logo container
					Div(
						Attrs{"class": "relative transform transition-transform duration-300 group-hover:scale-110"},

						// Main logo with floating animation
						Div(
							Attrs{"class": "relative h-12 w-12 md:h-14 md:w-14"},

							// Background gradient circle
							Div(Attrs{"class": "absolute inset-0 bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 rounded-full animate-pulse"}),

							// Logo image
							Img(Attrs{
								"src":   "/static/images/hero.jpg",
								"alt":   "GoWebComponents Logo",
								"class": "relative h-full w-full rounded-full border-2 border-white shadow-xl object-cover z-10",
							}),

							// Online status indicator
							Div(Attrs{"class": "absolute -top-1 -right-1 h-4 w-4 bg-emerald-500 rounded-full border-2 border-white shadow-lg animate-pulse"}),

							// Floating ring animation
							Div(Attrs{"class": "absolute inset-0 rounded-full border-2 border-indigo-400/30 animate-ping"}),
						),
					),

					// Brand text with enhanced typography
					Div(
						Attrs{"class": "hidden sm:block"},
						H1(Attrs{
							"class": "text-xl md:text-2xl font-black bg-gradient-to-r from-indigo-600 via-purple-600 to-pink-600 bg-clip-text text-transparent tracking-tight",
						}, "GoWebComponents"),
						P(Attrs{
							"class": "text-xs md:text-sm text-gray-600 font-medium -mt-1 tracking-wide",
						}, "by Earl Cameron"),
					),
				),

				// Desktop Navigation with enhanced hover effects
				Div(
					Attrs{"class": "hidden lg:flex items-center space-x-1 xl:space-x-2 flex-1 justify-center"},
					EnhancedNavLink("🏠", "Home", "#home", "home"),
					EnhancedNavLink("👨‍💻", "About", "#about", "about"),
					EnhancedNavLink("📺", "YouTube", "#youtube", "youtube"),
					EnhancedNavLink("⚡", "Features", "#features", "features"),
					EnhancedNavLink("🎨", "Examples", "#examples", "examples"),
					EnhancedNavLink("📚", "API Docs", "#api", "api"),
					EnhancedNavLink("🔥", "Contact", "#contact", "contact"),
				),

				// Action buttons with enhanced styling
				Div(
					Attrs{"class": "hidden md:flex items-center"},

					// GitHub button
					A(
						Attrs{
							"href":   "https://github.com/monstercameron/GoWebComponents",
							"target": "_blank",
							"class":  "group relative overflow-hidden px-5 py-2.5 bg-gray-900 text-white rounded-xl hover:bg-gray-800 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 cursor-pointer",
						},
						Div(
							Attrs{"class": "absolute inset-0 bg-gradient-to-r from-gray-800 to-gray-900 opacity-0 group-hover:opacity-100 transition-opacity duration-300"},
						),
						Div(
							Attrs{"class": "relative flex items-center space-x-2"},
							Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:rotate-12"}, "🐙"),
							Span(Attrs{"class": "font-semibold text-sm"}, "GitHub"),
						),
					),
				),

				// Enhanced Mobile Menu Button
				Button(
					Attrs{
						"class":   "lg:hidden relative p-3 rounded-xl bg-white/10 backdrop-blur-sm border border-white/20 hover:bg-white/20 transition-all duration-300 group",
						"onclick": handleMobileToggle,
					},
					MobileMenuIcon(isMobileMenuOpen()),
				),
			),
		),

		// Enhanced Mobile Menu
		EnhancedMobileMenu(isMobileMenuOpen(), setIsMobileMenuOpen),
	)
}

// EnhancedNavLink creates a modern navigation link with icon and fancy hover effects
func EnhancedNavLink(icon, text, href, section string) *Element {
	handleClick := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		ScrollToSectionSmooth(section)
	})

	return A(
		Attrs{
			"href":    href,
			"class":   "group relative px-3 py-2 text-gray-700 hover:text-indigo-600 transition-all duration-300 font-medium rounded-xl hover:bg-gradient-to-r hover:from-indigo-50 hover:to-purple-50 cursor-pointer",
			"onclick": handleClick,
		},

		// Content container
		Div(
			Attrs{"class": "flex items-center space-x-2"},
			Span(Attrs{"class": "text-sm transition-transform duration-300 group-hover:scale-110"}, icon),
			Span(Attrs{"class": "text-sm font-medium transition-transform duration-300 group-hover:translate-x-0.5 whitespace-nowrap"}, text),
		),

		// Animated underline
		Div(Attrs{
			"class": "absolute bottom-0 left-1/2 w-0 h-0.5 bg-gradient-to-r from-indigo-600 to-purple-600 group-hover:w-3/4 group-hover:left-5 transition-all duration-300 rounded-full",
		}),

		// Glow effect
		Div(Attrs{
			"class": "absolute inset-0 bg-gradient-to-r from-indigo-600/10 to-purple-600/10 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-300 -z-10",
		}),
	)
}

// MobileMenuIcon creates an animated hamburger menu icon
func MobileMenuIcon(isOpen bool) *Element {
	if isOpen {
		return Div(
			Attrs{"class": "w-6 h-6 flex flex-col justify-center items-center"},
			// X icon when open
			Div(Attrs{"class": "w-5 h-0.5 bg-gray-700 transform rotate-45 translate-y-0.5 transition-all duration-300"}),
			Div(Attrs{"class": "w-5 h-0.5 bg-gray-700 transform -rotate-45 -translate-y-0.5 transition-all duration-300"}),
		)
	}

	return Div(
		Attrs{"class": "w-6 h-6 flex flex-col justify-center items-center space-y-1"},
		// Hamburger icon when closed
		Div(Attrs{"class": "w-5 h-0.5 bg-gray-700 transition-all duration-300 group-hover:w-6"}),
		Div(Attrs{"class": "w-4 h-0.5 bg-gray-700 transition-all duration-300 group-hover:w-6"}),
		Div(Attrs{"class": "w-5 h-0.5 bg-gray-700 transition-all duration-300 group-hover:w-6"}),
	)
}

// EnhancedMobileMenu creates a modern mobile menu with animations
func EnhancedMobileMenu(isOpen bool, setIsOpen func(bool)) *Element {
	handleClose := GoUseFunc(func(event GoEvent) {
		setIsOpen(false)
	})

	overlayClasses := "lg:hidden fixed inset-0 z-40 transition-all duration-300 ease-in-out"
	menuClasses := "lg:hidden fixed top-16 md:top-20 left-0 right-0 z-50 transition-all duration-300 ease-in-out transform"

	if isOpen {
		overlayClasses += " bg-black/50 backdrop-blur-sm opacity-100"
		menuClasses += " translate-y-0 opacity-100"
	} else {
		overlayClasses += " bg-black/0 opacity-0 pointer-events-none"
		menuClasses += " -translate-y-full opacity-0 pointer-events-none"
	}

	return Div(
		nil,

		// Overlay
		Div(
			Attrs{
				"class":   overlayClasses,
				"onclick": handleClose,
			},
		),

		// Menu content
		Div(
			Attrs{"class": menuClasses},
			Div(
				Attrs{"class": "bg-white/95 backdrop-blur-xl shadow-2xl border-b border-gray-200/50 mx-4 rounded-2xl mt-2"},
				Div(
					Attrs{"class": "px-6 py-6 space-y-4"},

					// Navigation links
					EnhancedMobileNavLink("🏠", "Home", "#home", "home", setIsOpen),
					EnhancedMobileNavLink("👨‍💻", "About", "#about", "about", setIsOpen),
					EnhancedMobileNavLink("📺", "YouTube", "#youtube", "youtube", setIsOpen),
					EnhancedMobileNavLink("⚡", "Features", "#features", "features", setIsOpen),
					EnhancedMobileNavLink("🎨", "Examples", "#examples", "examples", setIsOpen),
					EnhancedMobileNavLink("📚", "API Docs", "#api", "api", setIsOpen),
					EnhancedMobileNavLink("🔥", "Contact", "#contact", "contact", setIsOpen),

					// Divider
					Div(Attrs{"class": "border-t border-gray-200 my-6"}),

					// Action buttons
					Div(
						Attrs{"class": "flex justify-center"},
						A(
							Attrs{
								"href":   "https://github.com/monstercameron/GoWebComponents",
								"target": "_blank",
								"class":  "flex items-center justify-center space-x-3 px-6 py-3 bg-gray-900 text-white rounded-xl hover:bg-gray-800 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-0.5 cursor-pointer",
							},
							Span(Attrs{"class": "text-lg"}, "🐙"),
							Span(Attrs{"class": "font-semibold"}, "GitHub"),
						),
					),
				),
			),
		),
	)
}

// EnhancedMobileNavLink creates a styled mobile navigation link
func EnhancedMobileNavLink(icon, text, href, section string, setIsOpen func(bool)) *Element {
	handleClick := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		ScrollToSectionSmooth(section)
		setIsOpen(false)
	})

	return A(
		Attrs{
			"href":    href,
			"class":   "group flex items-center space-x-3 px-4 py-3 text-gray-700 hover:text-indigo-600 hover:bg-gradient-to-r hover:from-indigo-50 hover:to-purple-50 rounded-xl transition-all duration-300 font-medium cursor-pointer",
			"onclick": handleClick,
		},
		Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:scale-110"}, icon),
		Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, text),
		Span(Attrs{"class": "ml-auto text-gray-400 group-hover:text-indigo-600 transition-colors duration-300"}, "→"),
	)
}

// ScrollToSectionJS creates a JavaScript function string for scrolling
func ScrollToSectionJS(section string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ScrollToSectionSmoothEnhanced(section)
		return nil
	})
}

// ScrollToSectionSmooth provides basic smooth scrolling to sections
func ScrollToSectionSmooth(section string) {
	ScrollToSectionSmoothEnhanced(section)
}

// ScrollToSectionSmoothEnhanced provides advanced smooth scrolling with easing and visual feedback
func ScrollToSectionSmoothEnhanced(section string) {
	element := js.Global().Get("document").Call("getElementById", section)
	if element.IsNull() {
		return
	}

	// Get current scroll position
	currentY := js.Global().Get("window").Get("pageYOffset").Float()

	// Calculate target position with navbar offset
	rect := element.Call("getBoundingClientRect")
	targetY := rect.Get("top").Float() + currentY - 100 // Increased offset for better spacing

	// Don't scroll if we're already close to the target
	if js.Global().Get("Math").Call("abs", targetY-currentY).Float() < 10 {
		return
	}

	// Show scroll indicator (optional visual feedback)
	showScrollIndicator(section)

	// Enhanced smooth scroll with custom easing
	animateScrollTo(currentY, targetY, 800) // 800ms duration
}

// animateScrollTo provides custom smooth scrolling with easing
func animateScrollTo(startY, targetY, duration float64) {
	startTime := js.Global().Get("performance").Call("now").Float()
	distance := targetY - startY

	// Easing function (ease-in-out-cubic)
	easeInOutCubic := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		t := args[0].Float()
		if t < 0.5 {
			return 4 * t * t * t
		}
		return 1 - js.Global().Get("Math").Call("pow", -2*t+2, 3).Float()/2
	})

	// Animation function
	var animate js.Func
	animate = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		currentTime := js.Global().Get("performance").Call("now").Float()
		elapsed := currentTime - startTime

		if elapsed >= duration {
			// Animation complete - ensure exact final position
			js.Global().Get("window").Call("scrollTo", 0, targetY)
			hideScrollIndicator()
			return nil
		}

		// Calculate progress (0 to 1)
		progress := elapsed / duration

		// Apply easing
		easedProgress := easeInOutCubic.Invoke(progress).Float()

		// Calculate current position
		currentY := startY + (distance * easedProgress)

		// Apply scroll
		js.Global().Get("window").Call("scrollTo", 0, currentY)

		// Continue animation
		js.Global().Call("requestAnimationFrame", animate)

		return nil
	})

	// Start animation
	js.Global().Call("requestAnimationFrame", animate)
}

// showScrollIndicator shows a visual indicator during scrolling
func showScrollIndicator(section string) {
	// Create or update scroll indicator
	indicator := js.Global().Get("document").Call("getElementById", "scroll-indicator")

	if indicator.IsNull() {
		// Create indicator element
		indicator = js.Global().Get("document").Call("createElement", "div")
		indicator.Set("id", "scroll-indicator")
		indicator.Get("style").Set("cssText", `
			position: fixed;
			top: 50%;
			right: 20px;
			transform: translateY(-50%);
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			color: white;
			padding: 12px 20px;
			border-radius: 25px;
			font-size: 14px;
			font-weight: 600;
			box-shadow: 0 10px 25px rgba(0,0,0,0.2);
			backdrop-filter: blur(10px);
			z-index: 9999;
			opacity: 0;
			transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
			pointer-events: none;
		`)
		js.Global().Get("document").Get("body").Call("appendChild", indicator)
	}

	// Update indicator text and show
	sectionName := formatSectionName(section)
	indicator.Set("textContent", "📍 Scrolling to "+sectionName)
	indicator.Get("style").Set("opacity", "1")
	indicator.Get("style").Set("transform", "translateY(-50%) translateX(0)")
}

// hideScrollIndicator hides the scroll indicator
func hideScrollIndicator() {
	indicator := js.Global().Get("document").Call("getElementById", "scroll-indicator")
	if !indicator.IsNull() {
		indicator.Get("style").Set("opacity", "0")
		indicator.Get("style").Set("transform", "translateY(-50%) translateX(20px)")

		// Remove after transition
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if !indicator.IsNull() {
				parent := indicator.Get("parentNode")
				if !parent.IsNull() {
					parent.Call("removeChild", indicator)
				}
			}
			return nil
		}), 300)
	}
}

// formatSectionName formats section ID to display name
func formatSectionName(section string) string {
	switch section {
	case "home":
		return "Home"
	case "about":
		return "About"
	case "features":
		return "Features"
	case "getting-started":
		return "Getting Started"
	case "examples":
		return "Examples"
	case "api":
		return "API Documentation"
	case "contact":
		return "Contact"
	default:
		return section
	}
}

// AddSmoothScrollCSS adds CSS for enhanced smooth scrolling
func AddSmoothScrollCSS() {
	// Add CSS for smooth scroll behavior to the document
	style := js.Global().Get("document").Call("createElement", "style")
	style.Set("textContent", `
		html {
			scroll-behavior: smooth;
		}
		
		/* Custom scrollbar for webkit browsers */
		::-webkit-scrollbar {
			width: 8px;
		}
		
		::-webkit-scrollbar-track {
			background: #f1f1f1;
			border-radius: 4px;
		}
		
		::-webkit-scrollbar-thumb {
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			border-radius: 4px;
		}
		
		::-webkit-scrollbar-thumb:hover {
			background: linear-gradient(135deg, #5a6fd8 0%, #6a4190 100%);
		}
		
		/* Scroll indicator animation */
		@keyframes scrollPulse {
			0%, 100% { transform: translateY(-50%) scale(1); }
			50% { transform: translateY(-50%) scale(1.05); }
		}
		
		#scroll-indicator {
			animation: scrollPulse 2s ease-in-out infinite;
		}
	`)
	js.Global().Get("document").Get("head").Call("appendChild", style)
}
