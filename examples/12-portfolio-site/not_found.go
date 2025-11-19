//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/dom"
)

// NotFoundPage renders a user-friendly 404 error page with navigation options.
// Features a prominent 404 display, helpful navigation buttons, and quick links
// to key sections of the application for improved user experience.
func NotFoundPage(_ Attrs) *Element {
	return dom.Div(
		Attrs{"class": "min-h-screen bg-[#0a0a0a] flex items-center justify-center"},

		// Main 404 content
		dom.Div(
			Attrs{"class": "text-center max-w-2xl mx-auto px-6"},

			// 404 illustration/icon
			dom.Div(
				Attrs{"class": "mb-8"},
				dom.H1(
					Attrs{"class": "text-9xl font-bold text-white/10 mb-4"},
					"404",
				),
				dom.Div(
					Attrs{"class": "text-6xl mb-6"},
					"🔍",
				),
			),

			// Error message
			dom.H2(
				Attrs{"class": "text-3xl md:text-4xl font-bold text-white mb-4"},
				"Page Not Found",
			),
			dom.P(
				Attrs{"class": "text-xl text-gray-400 mb-8 leading-relaxed"},
				"Oops! The page you're looking for doesn't exist. It might have been moved, deleted, or you entered the wrong URL.",
			),

			// Action buttons
			dom.Div(
				Attrs{"class": "flex flex-col sm:flex-row gap-4 justify-center items-center"},
				dom.Button(
					Attrs{
						"class":   "px-8 py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 font-semibold text-lg cursor-pointer",
						"onclick": "window.location.hash = '#/'",
					},
					"🏠 Go Home",
				),
				dom.Button(
					Attrs{
						"class":   "px-8 py-4 bg-white/5 text-white rounded-xl border-2 border-white/10 hover:border-indigo-500/50 hover:bg-white/10 transition-all duration-300 transform hover:-translate-y-1 font-semibold text-lg cursor-pointer",
						"onclick": "window.location.hash = '#/docs'",
					},
					"📚 View Docs",
				),
			),

			// Helpful links
			dom.Div(
				Attrs{"class": "mt-12 pt-8 border-t border-white/10"},
				dom.P(
					Attrs{"class": "text-gray-400 mb-4"},
					"Here are some helpful links:",
				),
				dom.Div(
					Attrs{"class": "flex flex-wrap justify-center gap-4"},
					dom.A(
						Attrs{
							"href":  "#/",
							"class": "text-indigo-400 hover:text-indigo-300 font-medium transition-colors duration-200",
						},
						"🏠 Home",
					),
					dom.A(
						Attrs{
							"href":  "#/docs",
							"class": "text-indigo-400 hover:text-indigo-300 font-medium transition-colors duration-200",
						},
						"📚 Documentation",
					),
					dom.A(
						Attrs{
							"href":   "https://github.com/monstercameron/GoWebComponents",
							"target": "_blank",
							"class":  "text-indigo-400 hover:text-indigo-300 font-medium transition-colors duration-200",
						},
						"🐙 GitHub",
					),
				),
			),
		),
	)
}
