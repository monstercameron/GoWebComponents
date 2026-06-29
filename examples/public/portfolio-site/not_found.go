//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// NotFoundPage renders a user-friendly 404 error page with navigation options.
// Features a prominent 404 display, helpful navigation buttons, and quick links
// to key sections of the application for improved user experience.
func NotFoundPage(_ Attrs) *Element {
	parseHome := ui.UseEvent(func() { router.Navigate("/") })
	parseDocs := ui.UseEvent(func() { router.Navigate("/docs") })

	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] flex items-center justify-center"},
		html.Div(html.Props{Class: "text-center max-w-2xl mx-auto px-6"},
			html.Div(html.Props{Class: "mb-8"},
				html.H1(html.Props{Class: "text-9xl font-bold text-white/10 mb-4"}, html.Text("404")),
				html.Div(html.Props{Class: "text-6xl mb-6"}, html.Text("🔍")),
			),
			html.H2(html.Props{Class: "text-3xl md:text-4xl font-bold text-white mb-4"}, html.Text("Page Not Found")),
			html.P(html.Props{Class: "text-xl text-gray-400 mb-8 leading-relaxed"}, html.Text("Oops! The page you're looking for doesn't exist. It might have been moved, deleted, or you entered the wrong URL.")),
			html.Div(html.Props{Class: "flex flex-col sm:flex-row gap-4 justify-center items-center"},
				html.Button(html.Props{OnClick: parseHome, Class: "px-8 py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 font-semibold text-lg cursor-pointer"}, html.Text("🏠 Go Home")),
				html.Button(html.Props{OnClick: parseDocs, Class: "px-8 py-4 bg-white/5 text-white rounded-xl border-2 border-white/10 hover:border-indigo-500/50 hover:bg-white/10 transition-all duration-300 transform hover:-translate-y-1 font-semibold text-lg cursor-pointer"}, html.Text("📚 View Docs")),
			),
			html.Div(html.Props{Class: "mt-12 pt-8 border-t border-white/10"},
				html.P(html.Props{Class: "text-gray-400 mb-4"}, html.Text("Here are some helpful links:")),
				html.Div(html.Props{Class: "flex flex-wrap justify-center gap-4"},
					html.A(html.Props{Href: "#/", Class: "text-indigo-400 hover:text-indigo-300 font-medium transition-colors duration-200"}, html.Text("🏠 Home")),
					html.A(html.Props{Href: "#/docs", Class: "text-indigo-400 hover:text-indigo-300 font-medium transition-colors duration-200"}, html.Text("📚 Documentation")),
					html.A(html.Props{Href: "https://github.com/monstercameron/GoWebComponents", Target: "_blank", Class: "text-indigo-400 hover:text-indigo-300 font-medium transition-colors duration-200"}, html.Text("🐙 GitHub")),
				),
			),
		),
	)
}
