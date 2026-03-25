//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"
)

// PortfolioProjectsSection displays Earl's featured projects with detailed information.
// Highlights GoWebComponents and gRPC Tunnel as revolutionary developments in web technology.
func PortfolioProjectsSection(_ Attrs) *Element {
	return Section(
		Attrs{
			"id":    "projects",
			"class": "py-20 relative",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-white mb-4"},
					"Recent Projects",
				),
				P(
					Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"},
					"Revolutionary projects that redefine what's possible in web development - built with cutting-edge Go and WebAssembly technology",
				),
			),

			func() *Element {
				parseProjects := portfolioProjects()
				parseProjectCards := make([]interface{}, 0, len(parseProjects))
				for _, parseProject := range parseProjects {
					parseProjectCards = append(parseProjectCards, PortfolioProjectCard(
						parseProject.Title,
						parseProject.Subtitle,
						parseProject.Description,
						parseProject.Technologies,
						parseProject.Link,
						parseProject.Featured,
					))
				}
				return Div(
					Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-8"},
					parseProjectCards...,
				)
			}(),
		),
	)
}

// PortfolioProjectCard renders a project with title, description, technologies, and CTA.
// Supports featured styling for highlighted projects and responsive layout adaptation.
func PortfolioProjectCard(parseTitle, parseSubtitle, parseDescription string, parseTechnologies []string, parseLink string, isFeatured bool) *Element {
	parseTechElements := make([]interface{}, len(parseTechnologies))
	for parseI, parseTech := range parseTechnologies {
		parseTechElements[parseI] = Span(
			Attrs{"class": "inline-block bg-purple-500/20 text-purple-300 border border-purple-500/30 px-2 py-1 rounded text-xs font-medium"},
			parseTech,
		)
	}

	parseCardClass := "bg-white/5 p-8 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 border border-white/10 hover:border-white/20 flex flex-col h-full backdrop-blur-sm"
	if isFeatured {
		parseCardClass = "bg-gradient-to-br from-purple-900/20 to-blue-900/20 p-8 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 border border-purple-500/30 hover:border-purple-500/50 flex flex-col h-full backdrop-blur-sm"
	}

	return Div(
		Attrs{"class": parseCardClass},

		// Content container (grows to fill space)
		Div(
			Attrs{"class": "flex-grow"},
			// Header
			Div(
				Attrs{"class": "mb-4"},
				H3(Attrs{"class": "text-2xl font-bold text-white mb-2"}, parseTitle),
				P(Attrs{"class": "text-purple-400 font-medium"}, parseSubtitle),
			),

			// Description
			P(Attrs{"class": "text-gray-300 mb-6 leading-relaxed"}, parseDescription),

			// Technologies
			Div(
				Attrs{"class": "flex flex-wrap gap-2"},
				parseTechElements...,
			),
		),

		// CTA Button (sticky to bottom)
		Button(
			Attrs{
				"class":   "w-full py-3 px-4 bg-white/10 text-white rounded-lg hover:bg-white/20 border border-white/10 transition-all duration-200 font-medium mt-6 cursor-pointer",
				"onclick": OpenProjectLink(parseLink),
			},
			"View Project →",
		),
	)
}

// OpenProjectLink generates a click handler that opens project URLs in new tabs.
// Safely handles invalid URLs by checking for placeholder values.
func OpenProjectLink(parseUrl string) interface{} {
	return UseEvent(func(parseE MouseEvent) {
		if parseUrl != "#" {
			js.Global().Get("window").Call("open", parseUrl, "_blank")
		}
	})
}
