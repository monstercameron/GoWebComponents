//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/router"
)

// NavBar renders a sophisticated navigation bar with glassmorphism effects and animations.
// Features responsive design, dark mode toggle, scroll progress indicator, and smooth
// section navigation with enhanced mobile menu support.
func NavBar(_ Attrs) *Element {
	// Component state management
	isMobileMenuOpen, setIsMobileMenuOpen := UseState(false)
	isScrolled, setIsScrolled := UseState(false)
	parseScrollProgress, setScrollProgress := UseState(0.0)
	isDark, setIsDark := UseState(getInitialDarkPref()) // Persisted dark mode preference

	// Initialize dark mode CSS on component mount
	UseEffect(func() func() {
		initDarkModeCSS()
		return nil
	}, true)

	// Sync dark mode state with DOM and localStorage
	UseEffect(func() func() {
		applyDarkClass(isDark())
		saveDarkPref(isDark())
		return nil
	}, isDark())

	// Event handlers
	handleMobileToggle := UseEvent(UseCallback(func(parseEvent MouseEvent) {
		setIsMobileMenuOpen(!isMobileMenuOpen())
	}, isMobileMenuOpen()))

	handleDarkToggle := UseEvent(UseCallback(func(parseEvent2 MouseEvent) {
		isParseNewVal := !isDark()
		setIsDark(isParseNewVal)
		applyDarkClass(isParseNewVal)
		saveDarkPref(isParseNewVal)
	}, isDark()))

	// Initialize scroll tracking and smooth scrolling behavior
	UseEffect(func() func() {
		AddSmoothScrollCSS() // Inject smooth scroll CSS once

		// Setup scroll listener for navbar effects and progress tracking
		parseScrollHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseScrollY := js.Global().Get("window").Get("scrollY").Float()
			setIsScrolled(parseScrollY > 20) // Trigger glassmorphism effect

			// Calculate scroll progress for progress bar
			parseWindowHeight := js.Global().Get("window").Get("innerHeight").Float()
			parseDocumentHeight := js.Global().Get("document").Get("documentElement").Get("scrollHeight").Float()
			parseMaxScroll := parseDocumentHeight - parseWindowHeight

			if parseMaxScroll > 0 {
				parseProgress := parseScrollY / parseMaxScroll
				if parseProgress > 1 {
					parseProgress = 1
				}
				setScrollProgress(parseProgress)
			}

			return nil
		})

		js.Global().Get("window").Call("addEventListener", "scroll", parseScrollHandler)
		return func() {
			js.Global().Get("window").Call("removeEventListener", "scroll", parseScrollHandler)
			parseScrollHandler.Release()
		}
	}, true)

	// Apply glassmorphism effect based on scroll state
	parseNavClasses := "fixed top-0 left-0 right-0 z-50 transition-all duration-300 ease-in-out"
	if isScrolled() {
		parseNavClasses += " bg-[#0a0a0a]/80 backdrop-blur-xl shadow-2xl border-b border-white/10"
	} else {
		parseNavClasses += " bg-transparent backdrop-blur-lg"
	}

	return Nav(
		Attrs{"class": parseNavClasses},

		// Scroll progress bar
		Div(
			Attrs{
				"class": "absolute bottom-0 left-0 h-1 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 transition-all duration-300 ease-out",
				"style": "width: " + fmt.Sprintf("%.1f", parseScrollProgress()*100) + "%",
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
					EnhancedNavLink("🚀", "Projects", "#projects", "projects"),
					EnhancedNavLink("🎨", "Examples", "#examples", "examples"),
					EnhancedNavLink("📚", "API Docs", "#api", "api"),
					EnhancedNavLink("🔥", "Contact", "#contact", "contact"),
				),

				// Dark-mode toggle and GitHub button container
				Div(
					Attrs{"class": "hidden md:flex items-center space-x-2"},

					// Dark-mode toggle button
					Button(
						Attrs{
							"class":   "group relative overflow-hidden p-2.5 bg-gray-900 text-white rounded-xl hover:bg-gray-800 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 cursor-pointer",
							"onclick": handleDarkToggle,
							"title":   "Toggle dark mode",
						},
						Span(Attrs{"class": "text-lg"}, func() string {
							if isDark() {
								return "☀️" // sun icon when in dark mode
							}
							return "🌙" // moon icon when in light mode
						}()),
					),

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
		EnhancedMobileMenu(isMobileMenuOpen(), func(isB bool) { setIsMobileMenuOpen(isB) }),
	)
}

// EnhancedNavLink renders a navigation item with icon, smooth animations and section routing.
// Handles both hash navigation for sections and route navigation for pages like documentation.
func EnhancedNavLink(parseIcon, parseText, parseHref, parseSection string) *Element {
	handleClick := UseEvent(UseCallback(func(parseEvent MouseEvent) {
		parseEvent.PreventDefault()
		// Handle docs route vs section scrolling
		if parseSection == "docs" {
			router.Navigate(parseSection)
		} else {
			// For sections, use smooth scrolling
			ScrollToSectionSmoothEnhanced(parseSection)
		}
	}, parseSection))

	return A(
		Attrs{
			"href":    parseHref,
			"class":   "group relative px-3 py-2 text-gray-300 hover:text-white transition-all duration-300 font-medium rounded-xl hover:bg-white/5 cursor-pointer",
			"onclick": handleClick,
		},

		// Content container
		Div(
			Attrs{"class": "flex items-center space-x-2"},
			Span(Attrs{"class": "text-sm transition-transform duration-300 group-hover:scale-110"}, parseIcon),
			Span(Attrs{"class": "text-sm font-medium transition-transform duration-300 group-hover:translate-x-0.5 whitespace-nowrap"}, parseText),
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
	handleClose := UseEvent(func(parseEvent MouseEvent) {
		setIsOpen(false)
	})

	parseOverlayClasses := "lg:hidden fixed inset-0 z-40 transition-all duration-300 ease-in-out"
	parseMenuClasses := "lg:hidden fixed top-16 md:top-20 left-0 right-0 z-50 transition-all duration-300 ease-in-out transform"

	if isOpen {
		parseOverlayClasses += " bg-black/50 backdrop-blur-sm opacity-100"
		parseMenuClasses += " translate-y-0 opacity-100"
	} else {
		parseOverlayClasses += " bg-black/0 opacity-0 pointer-events-none"
		parseMenuClasses += " -translate-y-full opacity-0 pointer-events-none"
	}

	return Div(
		nil,

		// Overlay
		Div(
			Attrs{
				"class":   parseOverlayClasses,
				"onclick": handleClose,
			},
		),

		// Menu content
		Div(
			Attrs{"class": parseMenuClasses},
			Div(
				Attrs{"class": "bg-[#0a0a0a]/95 backdrop-blur-xl shadow-2xl border-b border-white/10 mx-4 rounded-2xl mt-2"},
				Div(
					Attrs{"class": "px-6 py-6 space-y-4"},

					// Navigation links
					EnhancedMobileNavLink("🏠", "Home", "#home", "home", setIsOpen),
					EnhancedMobileNavLink("👨‍💻", "About", "#about", "about", setIsOpen),
					EnhancedMobileNavLink("📺", "YouTube", "#youtube", "youtube", setIsOpen),
					EnhancedMobileNavLink("🚀", "Projects", "#projects", "projects", setIsOpen),
					EnhancedMobileNavLink("🎨", "Examples", "#examples", "examples", setIsOpen),
					EnhancedMobileNavLink("📚", "API Docs", "#api", "api", setIsOpen),
					EnhancedMobileNavLink("🔥", "Contact", "#contact", "contact", setIsOpen),

					// Divider
					Div(Attrs{"class": "my-6 border-t border-white/10"}),

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
func EnhancedMobileNavLink(parseIcon, parseText, parseHref, parseSection string, setIsOpen func(bool)) *Element {
	handleClick := UseEvent(func(parseEvent MouseEvent) {
		parseEvent.PreventDefault()
		// Handle docs route vs section scrolling
		if parseSection == "docs" {
			router.Navigate(parseSection)
		} else {
			// For sections, use smooth scrolling
			ScrollToSectionSmoothEnhanced(parseSection)
		}
		setIsOpen(false)
	})

	return A(
		Attrs{
			"href":    parseHref,
			"class":   "group flex items-center space-x-3 px-4 py-3 text-gray-300 hover:text-white hover:bg-white/5 rounded-xl transition-all duration-300 font-medium cursor-pointer",
			"onclick": handleClick,
		},
		Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:scale-110"}, parseIcon),
		Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, parseText),
		Span(Attrs{"class": "ml-auto text-gray-400 group-hover:text-indigo-600 transition-colors duration-300"}, "→"),
	)
}

// ScrollToSectionJS creates a JavaScript function string for scrolling
func ScrollToSectionJS(parseSection string) interface{} {
	return UseEvent(func(parseE MouseEvent) {
		ScrollToSectionSmoothEnhanced(parseSection)
	})
}

// ScrollToSectionSmooth provides basic smooth scrolling to sections
func ScrollToSectionSmooth(parseSection string) {
	ScrollToSectionSmoothEnhanced(parseSection)
}

// ScrollToSectionSmoothEnhanced provides advanced smooth scrolling with easing and visual feedback
func ScrollToSectionSmoothEnhanced(parseSection string) {
	parseElement := js.Global().Get("document").Call("getElementById", parseSection)
	if parseElement.IsNull() {
		return
	}

	// Get current scroll position
	parseCurrentY := js.Global().Get("window").Get("pageYOffset").Float()

	// Calculate target position with navbar offset
	parseRect := parseElement.Call("getBoundingClientRect")
	parseTargetY := parseRect.Get("top").Float() + parseCurrentY - 100 // Increased offset for better spacing

	// Don't scroll if we're already close to the target
	if js.Global().Get("Math").Call("abs", parseTargetY-parseCurrentY).Float() < 10 {
		return
	}

	// Show scroll indicator (optional visual feedback)
	showScrollIndicator(parseSection)

	// Enhanced smooth scroll with custom easing
	animateScrollTo(parseCurrentY, parseTargetY, 800) // 800ms duration
}

// animateScrollTo provides custom smooth scrolling with easing
func animateScrollTo(parseStartY, parseTargetY, parseDuration float64) {
	parseStartTime := js.Global().Get("performance").Call("now").Float()
	parseDistance := parseTargetY - parseStartY

	// Easing function (ease-in-out-cubic)
	parseEaseInOutCubic := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseT := parseArgs[0].Float()
		if parseT < 0.5 {
			return 4 * parseT * parseT * parseT
		}
		return 1 - js.Global().Get("Math").Call("pow", -2*parseT+2, 3).Float()/2
	})

	// Animation function
	var parseAnimate js.Func
	parseAnimate = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseCurrentTime := js.Global().Get("performance").Call("now").Float()
		parseElapsed := parseCurrentTime - parseStartTime

		if parseElapsed >= parseDuration {
			// Animation complete - ensure exact final position
			js.Global().Get("window").Call("scrollTo", 0, parseTargetY)
			hideScrollIndicator()
			return nil
		}

		// Calculate progress (0 to 1)
		parseProgress := parseElapsed / parseDuration

		// Apply easing
		parseEasedProgress := parseEaseInOutCubic.Invoke(parseProgress).Float()

		// Calculate current position
		parseCurrentY := parseStartY + (parseDistance * parseEasedProgress)

		// Apply scroll
		js.Global().Get("window").Call("scrollTo", 0, parseCurrentY)

		// Continue animation
		js.Global().Call("requestAnimationFrame", parseAnimate)

		return nil
	})

	// Start animation
	js.Global().Call("requestAnimationFrame", parseAnimate)
}

// showScrollIndicator shows a visual indicator during scrolling
func showScrollIndicator(parseSection string) {
	// Create or update scroll indicator
	parseIndicator := js.Global().Get("document").Call("getElementById", "scroll-indicator")

	if parseIndicator.IsNull() {
		// Create indicator element
		parseIndicator = js.Global().Get("document").Call("createElement", "div")
		parseIndicator.Set("id", "scroll-indicator")
		parseIndicator.Get("style").Set("cssText", `
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
		js.Global().Get("document").Get("body").Call("appendChild", parseIndicator)
	}

	// Update indicator text and show
	parseSectionName := formatSectionName(parseSection)
	parseIndicator.Set("textContent", "📍 Scrolling to "+parseSectionName)
	parseIndicator.Get("style").Set("opacity", "1")
	parseIndicator.Get("style").Set("transform", "translateY(-50%) translateX(0)")
}

// hideScrollIndicator hides the scroll indicator
func hideScrollIndicator() {
	parseIndicator := js.Global().Get("document").Call("getElementById", "scroll-indicator")
	if !parseIndicator.IsNull() {
		parseIndicator.Get("style").Set("opacity", "0")
		parseIndicator.Get("style").Set("transform", "translateY(-50%) translateX(20px)")

		// Remove after transition
		js.Global().Call("setTimeout", js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			if !parseIndicator.IsNull() {
				parseParent := parseIndicator.Get("parentNode")
				if !parseParent.IsNull() {
					parseParent.Call("removeChild", parseIndicator)
				}
			}
			return nil
		}), 300)
	}
}

// formatSectionName formats section ID to display name
func formatSectionName(parseSection string) string {
	switch parseSection {
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
		return parseSection
	}
}

// AddSmoothScrollCSS adds CSS for enhanced smooth scrolling
func AddSmoothScrollCSS() {
	// Add CSS for smooth scroll behavior to the document
	parseStyle := js.Global().Get("document").Call("createElement", "style")
	parseStyle.Set("textContent", `
		html {
			scroll-behavior: smooth;
		}
		
		/* Custom scrollbar for webkit browsers */
		::-webkit-scrollbar {
			width: 8px;
		}
		
		::-webkit-scrollbar-track {
			background: #1a1a1a;
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
	js.Global().Get("document").Get("head").Call("appendChild", parseStyle)
}
