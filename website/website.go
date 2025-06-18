//go:build js && wasm
// +build js,wasm

package website

import . "github.com/monstercameron/GoWebComponents/fiber"

// DocsWebsite is the main entrypoint for the documentation site
func DocsWebsite(props Attrs) *Element {
	return Div(
		Attrs{"class": "min-h-screen bg-gray-50"},
		NavBar(nil),
		FeaturesSection(nil),
		Main(
			Attrs{"class": "container mx-auto px-4 py-8"},
			H1(nil, "Welcome to GoWebComponents Documentation"),
			P(nil, "This is a placeholder for the main content."),
		),
	)
}
