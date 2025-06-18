//go:build js && wasm
// +build js,wasm

package website

import . "github.com/monstercameron/GoWebComponents/fiber"

// NavBar is an atomic, composable navigation bar for the GoWebComponents docs site
func NavBar(props Attrs) *Element {
	return Nav(
		Attrs{"class": "navbar bg-white shadow-md sticky top-0 z-50"},
		Div(
			Attrs{"class": "container mx-auto flex items-center justify-between px-6 py-4"},
			// Logo/Brand
			Div(
				Attrs{"class": "flex items-center space-x-2"},
				Img(Attrs{"src": "/static/images/hero.jpg", "alt": "GoWebComponents Logo", "class": "h-8 w-8 rounded"}),
				H1(Attrs{"class": "text-2xl font-bold text-indigo-600"}, "GoWebComponents"),
			),
			// Navigation Links
			Ul(
				Attrs{"class": "flex space-x-6"},
				Li(nil, A(Attrs{"href": "#why"}, "Why GoWebComponents?")),
				Li(nil, A(Attrs{"href": "#getting-started"}, "Getting Started")),
				Li(nil, A(Attrs{"href": "#live-reload"}, "Live Reload")),
				Li(nil, A(Attrs{"href": "#api"}, "API")),
				Li(nil, A(Attrs{"href": "#examples"}, "Examples")),
				Li(nil, A(Attrs{"href": "#contributing"}, "Contributing")),
				Li(nil, A(Attrs{"href": "https://github.com/monstercameron/GoWebComponents", "target": "_blank"}, "GitHub")),
			),
		),
	)
}
