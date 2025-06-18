//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// PortfolioProjectsSection showcases key projects
func PortfolioProjectsSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "projects",
			"class": "py-20 bg-white",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"Featured Projects",
				),
				P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"A selection of projects that showcase my passion for innovation",
				),
			),

			// Projects grid
			Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-8"},

				// GoWebComponents - Main project
				PortfolioProjectCard(
					"🚀 GoWebComponents",
					"Revolutionary Frontend Framework",
					"A React-like framework for building web applications entirely in Go using WebAssembly. Features component-based architecture, state management, and zero JavaScript.",
					[]string{"Go", "WebAssembly", "React-like", "State Management"},
					"https://github.com/earlcameron/gowebcomponents",
					true, // featured
				),

				PortfolioProjectCard(
					"🌐 Portfolio Website",
					"Personal Brand & Showcase",
					"This very website! Built entirely with GoWebComponents to demonstrate the framework's capabilities for creating modern, interactive web applications.",
					[]string{"GoWebComponents", "Tailwind CSS", "Responsive Design"},
					"#",
					false,
				),

				PortfolioProjectCard(
					"📊 Analytics Dashboard",
					"Real-time Data Visualization",
					"A comprehensive analytics dashboard with real-time data updates, interactive charts, and custom reporting features built for enterprise clients.",
					[]string{"Go", "PostgreSQL", "WebSocket", "Chart.js"},
					"#",
					false,
				),

				PortfolioProjectCard(
					"🤖 AI Code Assistant",
					"Developer Productivity Tool",
					"An intelligent code completion and refactoring tool that leverages machine learning to help developers write better code faster.",
					[]string{"Python", "TensorFlow", "VS Code API", "NLP"},
					"#",
					false,
				),
			),
		),
	)
}

// PortfolioProjectCard creates a project showcase card
func PortfolioProjectCard(title, subtitle, description string, technologies []string, link string, featured bool) *Element {
	techElements := make([]interface{}, len(technologies))
	for i, tech := range technologies {
		techElements[i] = Span(
			Attrs{"class": "inline-block bg-purple-100 text-purple-800 px-2 py-1 rounded text-xs font-medium"},
			tech,
		)
	}

	cardClass := "bg-white p-8 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 border border-gray-200 hover:border-purple-200"
	if featured {
		cardClass = "bg-gradient-to-br from-purple-50 to-blue-50 p-8 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 border-2 border-purple-200 hover:border-purple-300"
	}

	return Div(
		Attrs{"class": cardClass},

		// Header
		Div(
			Attrs{"class": "mb-4"},
			H3(Attrs{"class": "text-2xl font-bold text-gray-900 mb-2"}, title),
			P(Attrs{"class": "text-purple-600 font-medium"}, subtitle),
		),

		// Description
		P(Attrs{"class": "text-gray-600 mb-6 leading-relaxed"}, description),

		// Technologies
		Div(
			Attrs{"class": "flex flex-wrap gap-2 mb-6"},
			techElements...,
		),

		// CTA Button
		Button(
			Attrs{
				"class":   "w-full py-2 px-4 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors duration-200 font-medium",
				"onclick": OpenProjectLink(link),
			},
			"View Project →",
		),
	)
}

// OpenProjectLink creates a JavaScript function to open a project link
func OpenProjectLink(url string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if url != "#" {
			js.Global().Get("window").Call("open", url, "_blank")
		}
		return nil
	})
}
