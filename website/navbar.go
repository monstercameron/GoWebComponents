//go:build js && wasm
// +build js,wasm

package website

import . "github.com/monstercameron/GoWebComponents/fiber"

// NavBar is a beautiful, professional navigation bar for the GoWebComponents docs site
func NavBar(props Attrs) *Element {
	return Nav(
		Attrs{
			"class": "navbar bg-white/95 backdrop-blur-sm shadow-lg sticky top-0 z-50 border-b border-gray-200",
		},
		Div(
			Attrs{"class": "container mx-auto flex items-center justify-between px-6 py-4"},

			// Logo/Brand Section
			Div(
				Attrs{"class": "flex items-center space-x-3"},
				Div(
					Attrs{"class": "relative"},
					Img(Attrs{
						"src":   "/static/images/hero.jpg",
						"alt":   "GoWebComponents Logo",
						"class": "h-10 w-10 rounded-full border-2 border-indigo-500 shadow-md",
					}),
					Div(Attrs{"class": "absolute -top-1 -right-1 h-4 w-4 bg-green-500 rounded-full border-2 border-white"}),
				),
				Div(
					Attrs{"class": "flex flex-col"},
					H1(Attrs{"class": "text-2xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent"}, "GoWebComponents"),
					P(Attrs{"class": "text-xs text-gray-500 -mt-1"}, "by Earl Cameron"),
				),
			),

			// Desktop Navigation Links
			Div(
				Attrs{"class": "hidden md:flex items-center space-x-8"},
				NavLink("Home", "#home", "home"),
				NavLink("About", "#about", "about"),
				NavLink("Features", "#features", "features"),
				NavLink("Getting Started", "#getting-started", "getting-started"),
				NavLink("Examples", "#examples", "examples"),
				NavLink("API Docs", "#api", "api"),
				NavLink("Live Reload", "#live-reload", "live-reload"),
			),

			// Action Buttons
			Div(
				Attrs{"class": "hidden md:flex items-center space-x-4"},
				A(
					Attrs{
						"href":   "https://github.com/monstercameron/GoWebComponents",
						"target": "_blank",
						"class":  "flex items-center space-x-2 px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors duration-200 shadow-md hover:shadow-lg",
					},
					Span(Attrs{"class": "text-lg"}, "⚡"),
					Span(nil, "GitHub"),
				),
				Button(
					Attrs{
						"class":   "px-6 py-2 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg hover:from-indigo-700 hover:to-purple-700 transition-all duration-200 shadow-md hover:shadow-lg transform hover:-translate-y-0.5",
						"onclick": "scrollToSection('getting-started')",
					},
					"Get Started",
				),
			),

			// Mobile Menu Button
			Button(
				Attrs{
					"class":   "md:hidden p-2 rounded-lg bg-gray-100 hover:bg-gray-200 transition-colors duration-200",
					"onclick": "toggleMobileMenu()",
				},
				Span(Attrs{"class": "text-xl"}, "☰"),
			),
		),

		// Mobile Menu (Hidden by default)
		MobileMenu(nil),
	)
}

// NavLink creates a styled navigation link
func NavLink(text, href, section string) *Element {
	return A(
		Attrs{
			"href":    href,
			"class":   "relative px-3 py-2 text-gray-700 hover:text-indigo-600 transition-colors duration-200 font-medium group",
			"onclick": "scrollToSection('" + section + "')",
		},
		Span(nil, text),
		Span(Attrs{"class": "absolute bottom-0 left-0 w-0 h-0.5 bg-gradient-to-r from-indigo-600 to-purple-600 group-hover:w-full transition-all duration-300"}),
	)
}

// MobileMenu creates the mobile navigation menu
func MobileMenu(props Attrs) *Element {
	return Div(
		Attrs{
			"id":    "mobile-menu",
			"class": "md:hidden bg-white border-t border-gray-200 shadow-lg absolute top-full left-0 right-0 transform -translate-y-full opacity-0 invisible transition-all duration-300",
		},
		Div(
			Attrs{"class": "px-6 py-4 space-y-4"},
			MobileNavLink("Home", "#home", "home"),
			MobileNavLink("About", "#about", "about"),
			MobileNavLink("Features", "#features", "features"),
			MobileNavLink("Getting Started", "#getting-started", "getting-started"),
			MobileNavLink("Examples", "#examples", "examples"),
			MobileNavLink("API Docs", "#api", "api"),
			MobileNavLink("Live Reload", "#live-reload", "live-reload"),
			Div(Attrs{"class": "pt-4 border-t border-gray-200 space-y-3"},
				A(
					Attrs{
						"href":   "https://github.com/monstercameron/GoWebComponents",
						"target": "_blank",
						"class":  "flex items-center space-x-2 px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors duration-200",
					},
					Span(nil, "GitHub"),
				),
				Button(
					Attrs{
						"class":   "w-full px-4 py-2 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg hover:from-indigo-700 hover:to-purple-700 transition-all duration-200",
						"onclick": "scrollToSection('getting-started')",
					},
					"Get Started",
				),
			),
		),
	)
}

// MobileNavLink creates a styled mobile navigation link
func MobileNavLink(text, href, section string) *Element {
	return A(
		Attrs{
			"href":    href,
			"class":   "block px-3 py-2 text-gray-700 hover:text-indigo-600 hover:bg-gray-50 rounded-lg transition-colors duration-200 font-medium",
			"onclick": "scrollToSection('" + section + "'); toggleMobileMenu()",
		},
		text,
	)
}
