//go:build js && wasm
// +build js,wasm

package website
import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
)

// PortfolioProjectsSection displays Earl's featured projects with detailed information.
// Highlights GoWebComponents and gRPC Tunnel as revolutionary developments in web technology.
func PortfolioProjectsSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{
			"id":    "projects",
			"class": "py-20 bg-white",
		},
		dom.Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			dom.Div(
				Attrs{"class": "text-center mb-16"},
				dom.H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"Recent Projects",
				),
				dom.P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"Revolutionary projects that redefine what's possible in web development - built with cutting-edge Go and WebAssembly technology",
				),
			),

			// Projects grid
			dom.Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-8"},

				// GoWebComponents - Main project
				PortfolioProjectCard(
					"🚀 GoWebComponents",
					"Revolutionary Frontend Framework",
					"A React-like framework for building web applications entirely in Go using WebAssembly. Features component-based architecture, state management, and zero JavaScript required.",
					[]string{"Go", "WebAssembly", "React-like", "State Management", "Frontend Framework"},
					"https://github.com/monstercameron/GoWebComponents",
					true, // featured
				),

				// gRPC Tunnel - Second featured project
				PortfolioProjectCard(
					"🌐 gRPC Tunnel",
					"Native gRPC-over-WebSocket Solution",
					"Innovative project that tunnels native gRPC calls over WebSocket connections, enabling full gRPC communication from browsers using WebAssembly without gRPC-Web limitations.",
					[]string{"Go", "gRPC", "WebSocket", "WebAssembly", "Protobuf"},
					"https://github.com/monstercameron/grpc-tunnel",
					true, // featured
				),
			),
		),
	)
}

// PortfolioProjectCard renders a project with title, description, technologies, and CTA.
// Supports featured styling for highlighted projects and responsive layout adaptation.
func PortfolioProjectCard(title, subtitle, description string, technologies []string, link string, featured bool) *Element {
	techElements := make([]interface{}, len(technologies))
	for i, tech := range technologies {
		techElements[i] = dom.Span(
			Attrs{"class": "inline-block bg-purple-100 text-purple-800 px-2 py-1 rounded text-xs font-medium"},
			tech,
		)
	}

	cardClass := "bg-white p-8 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 border border-gray-200 hover:border-purple-200 flex flex-col h-full"
	if featured {
		cardClass = "bg-gradient-to-br from-purple-50 to-blue-50 p-8 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 border-2 border-purple-200 hover:border-purple-300 flex flex-col h-full"
	}

	return dom.Div(
		Attrs{"class": cardClass},

		// Content container (grows to fill space)
		dom.Div(
			Attrs{"class": "flex-grow"},
			// Header
			dom.Div(
				Attrs{"class": "mb-4"},
				dom.H3(Attrs{"class": "text-2xl font-bold text-gray-900 mb-2"}, title),
				dom.P(Attrs{"class": "text-purple-600 font-medium"}, subtitle),
			),

			// Description
			dom.P(Attrs{"class": "text-gray-600 mb-6 leading-relaxed"}, description),

			// Technologies
			dom.Div(
				Attrs{"class": "flex flex-wrap gap-2"},
				techElements...,
			),
		),

		// CTA Button (sticky to bottom)
		dom.Button(
			Attrs{
				"class":   "w-full py-2 px-4 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors duration-200 font-medium mt-6",
				"onclick": OpenProjectLink(link),
			},
			"View Project →",
		),
	)
}

// OpenProjectLink generates a click handler that opens project URLs in new tabs.
// Safely handles invalid URLs by checking for placeholder values.
func OpenProjectLink(url string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if url != "#" {
			js.Global().Get("window").Call("open", url, "_blank")
		}
		return nil
	})
}




