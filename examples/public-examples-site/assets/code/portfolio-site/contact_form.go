//go:build js && wasm

package main

// ContactSection provides multiple ways to connect with Earl Cameron.
// Features enhanced LinkedIn and GitHub cards with professional service listings
// and animated interactions for improved engagement.
func ContactSection(_ Attrs) *Element {
	return Section(
		Attrs{
			"id":    "contact",
			"class": "py-20 relative",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-white mb-4"},
					"Let's Connect",
				),
				P(
					Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"},
					"Based in Fort Lauderdale, Florida with 4+ years of full-stack development experience. Let's discuss your next project or collaboration opportunity.",
				),
			),

			Div(
				Attrs{"class": "max-w-2xl mx-auto"},

				// Contact methods
				Div(
					Attrs{"class": "space-y-8"},
					H3(Attrs{"class": "text-2xl font-semibold text-white mb-6"}, "Connect With Me"),

					// Two column layout for LinkedIn and GitHub
					Div(
						Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-6"},

						// Enhanced LinkedIn
						Div(
							Attrs{"class": "group relative overflow-hidden bg-gradient-to-br from-blue-900/20 to-blue-800/20 p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-500 transform hover:-translate-y-2 border border-blue-500/30 backdrop-blur-sm"},
							// Animated background gradient (moved to back layer)
							Div(Attrs{"class": "absolute inset-0 bg-gradient-to-r from-blue-400/5 to-blue-600/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500 rounded-2xl -z-10"}),
							Div(
								Attrs{"class": "relative z-10 flex items-center space-x-4"},
								Div(
									Attrs{"class": "relative"},
									Div(Attrs{"class": "w-16 h-16 bg-gradient-to-br from-blue-600 to-blue-700 rounded-xl flex items-center justify-center shadow-lg group-hover:shadow-blue-500/30 transition-all duration-500 group-hover:scale-110 group-hover:rotate-3"}),
									Div(
										Attrs{"class": "absolute inset-0 flex items-center justify-center text-white text-2xl font-bold transition-transform duration-500 group-hover:scale-110"},
										"ðŸ’¼",
									),
									// Floating particles
									Div(Attrs{"class": "absolute -top-2 -right-2 w-3 h-3 bg-blue-400 rounded-full animate-pulse opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
									Div(Attrs{"class": "absolute -bottom-2 -left-2 w-2 h-2 bg-blue-500 rounded-full animate-pulse delay-300 opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
								),
								Div(
									Attrs{"class": "flex-1"},
									H4(Attrs{"class": "text-xl font-bold text-blue-100 group-hover:text-white transition-colors duration-300"}, "LinkedIn"),
									P(Attrs{"class": "text-blue-300 mb-3 group-hover:text-blue-200 transition-colors duration-300 text-sm"}, "Fort Lauderdale, Florida â€¢ 518 followers â€¢ UKG"),
									A(
										Attrs{
											"href":   "https://www.linkedin.com/in/earl-cameron/",
											"target": "_blank",
											"class":  "relative z-20 inline-flex items-center space-x-2 px-4 py-2 bg-blue-600/80 text-white rounded-lg font-semibold hover:bg-blue-600 transition-all duration-300 transform hover:scale-105 group-hover:shadow-lg cursor-pointer text-sm",
										},
										Span(nil, "Connect"),
										Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, "â†’"),
									),
								),
							),
						),

						// Enhanced GitHub
						Div(
							Attrs{"class": "group relative overflow-hidden bg-gradient-to-br from-gray-800/40 to-gray-900/40 p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-500 transform hover:-translate-y-2 border border-gray-600/30 backdrop-blur-sm"},
							// Animated background gradient (moved to back layer)
							Div(Attrs{"class": "absolute inset-0 bg-gradient-to-r from-gray-400/5 to-gray-600/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500 rounded-2xl -z-10"}),
							Div(
								Attrs{"class": "relative z-10 flex items-center space-x-4"},
								Div(
									Attrs{"class": "relative"},
									Div(Attrs{"class": "w-16 h-16 bg-gradient-to-br from-gray-700 to-gray-800 rounded-xl flex items-center justify-center shadow-lg group-hover:shadow-gray-500/30 transition-all duration-500 group-hover:scale-110 group-hover:rotate-3"}),
									Div(
										Attrs{"class": "absolute inset-0 flex items-center justify-center text-white text-2xl font-bold transition-transform duration-500 group-hover:scale-110"},
										"ðŸ™",
									),
									// Floating particles
									Div(Attrs{"class": "absolute -top-2 -right-2 w-3 h-3 bg-gray-500 rounded-full animate-pulse opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
									Div(Attrs{"class": "absolute -bottom-2 -left-2 w-2 h-2 bg-gray-600 rounded-full animate-pulse delay-300 opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
								),
								Div(
									Attrs{"class": "flex-1"},
									H4(Attrs{"class": "text-xl font-bold text-gray-100 group-hover:text-white transition-colors duration-300"}, "GitHub"),
									P(Attrs{"class": "text-gray-400 mb-3 group-hover:text-gray-300 transition-colors duration-300 text-sm"}, "Miami, Florida â€¢ 53 repositories â€¢ GoWebComponents"),
									A(
										Attrs{
											"href":   "https://github.com/monstercameron",
											"target": "_blank",
											"class":  "relative z-20 inline-flex items-center space-x-2 px-4 py-2 bg-gray-700 text-white rounded-lg font-semibold hover:bg-gray-600 transition-all duration-300 transform hover:scale-105 group-hover:shadow-lg cursor-pointer text-sm",
										},
										Span(nil, "View Repos"),
										Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, "â†’"),
									),
								),
							),
						),
					),

					Div(
						Attrs{"class": "mt-8 p-6 bg-white/5 rounded-xl shadow-lg border border-white/10 backdrop-blur-sm"},
						H4(Attrs{"class": "text-lg font-semibold text-white mb-4 flex items-center"},
							Span(Attrs{"class": "mr-2"}, "ðŸŽ¯"),
							"Professional Services Available:",
						),
						Ul(
							Attrs{"class": "space-y-3 text-gray-300"},
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-blue-400"}, "ðŸ’¼"),
								"Full-stack contracting projects"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-purple-400"}, "ðŸŽ¯"),
								"Technical consulting & architecture"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-blue-400"}, "ðŸ“š"),
								"Developer training & workshops"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-green-400"}, "ðŸ”"),
								"Code reviews & optimization"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-orange-400"}, "ðŸš€"),
								"GoWebComponents implementation"),
						),
						Div(
							Attrs{"class": "mt-4 pt-4 border-t border-white/10"},
							P(Attrs{"class": "text-sm text-gray-400 flex items-center"},
								Span(Attrs{"class": "mr-2"}, "ðŸ“"),
								"Based in Fort Lauderdale, Florida â€¢ Remote & On-site Available"),
						),
					),
				),
			),
		),
	)
}
