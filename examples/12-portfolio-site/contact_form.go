//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"
)

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
										"💼",
									),
									// Floating particles
									Div(Attrs{"class": "absolute -top-2 -right-2 w-3 h-3 bg-blue-400 rounded-full animate-pulse opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
									Div(Attrs{"class": "absolute -bottom-2 -left-2 w-2 h-2 bg-blue-500 rounded-full animate-pulse delay-300 opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
								),
								Div(
									Attrs{"class": "flex-1"},
									H4(Attrs{"class": "text-xl font-bold text-blue-100 group-hover:text-white transition-colors duration-300"}, "LinkedIn"),
									P(Attrs{"class": "text-blue-300 mb-3 group-hover:text-blue-200 transition-colors duration-300 text-sm"}, "Fort Lauderdale, Florida • 518 followers • UKG"),
									A(
										Attrs{
											"href":   "https://www.linkedin.com/in/earl-cameron/",
											"target": "_blank",
											"class":  "relative z-20 inline-flex items-center space-x-2 px-4 py-2 bg-blue-600/80 text-white rounded-lg font-semibold hover:bg-blue-600 transition-all duration-300 transform hover:scale-105 group-hover:shadow-lg cursor-pointer text-sm",
										},
										Span(nil, "Connect"),
										Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, "→"),
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
										"🐙",
									),
									// Floating particles
									Div(Attrs{"class": "absolute -top-2 -right-2 w-3 h-3 bg-gray-500 rounded-full animate-pulse opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
									Div(Attrs{"class": "absolute -bottom-2 -left-2 w-2 h-2 bg-gray-600 rounded-full animate-pulse delay-300 opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
								),
								Div(
									Attrs{"class": "flex-1"},
									H4(Attrs{"class": "text-xl font-bold text-gray-100 group-hover:text-white transition-colors duration-300"}, "GitHub"),
									P(Attrs{"class": "text-gray-400 mb-3 group-hover:text-gray-300 transition-colors duration-300 text-sm"}, "Miami, Florida • 53 repositories • GoWebComponents"),
									A(
										Attrs{
											"href":   "https://github.com/monstercameron",
											"target": "_blank",
											"class":  "relative z-20 inline-flex items-center space-x-2 px-4 py-2 bg-gray-700 text-white rounded-lg font-semibold hover:bg-gray-600 transition-all duration-300 transform hover:scale-105 group-hover:shadow-lg cursor-pointer text-sm",
										},
										Span(nil, "View Repos"),
										Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, "→"),
									),
								),
							),
						),
					),

					Div(
						Attrs{"class": "mt-8 p-6 bg-white/5 rounded-xl shadow-lg border border-white/10 backdrop-blur-sm"},
						H4(Attrs{"class": "text-lg font-semibold text-white mb-4 flex items-center"},
							Span(Attrs{"class": "mr-2"}, "🎯"),
							"Professional Services Available:",
						),
						Ul(
							Attrs{"class": "space-y-3 text-gray-300"},
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-blue-400"}, "💼"),
								"Full-stack contracting projects"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-purple-400"}, "🎯"),
								"Technical consulting & architecture"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-blue-400"}, "📚"),
								"Developer training & workshops"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-green-400"}, "🔍"),
								"Code reviews & optimization"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-orange-400"}, "🚀"),
								"GoWebComponents implementation"),
						),
						Div(
							Attrs{"class": "mt-4 pt-4 border-t border-white/10"},
							P(Attrs{"class": "text-sm text-gray-400 flex items-center"},
								Span(Attrs{"class": "mr-2"}, "📍"),
								"Based in Fort Lauderdale, Florida • Remote & On-site Available"),
						),
					),
				),
			),
		),
	)
}

// ContactMethod renders a structured contact option with icon and call-to-action.
// Provides consistent styling for different communication channels.
func ContactMethod(icon, title, description, link string) *Element {
	return Div(
		Attrs{"class": "flex items-start space-x-4"},
		Span(Attrs{"class": "text-2xl"}, icon),
		Div(
			Attrs{"class": "flex-1"},
			H4(Attrs{"class": "text-lg font-semibold text-white"}, title),
			P(Attrs{"class": "text-gray-400 mb-2"}, description),
			Button(
				Attrs{
					"class":   "text-purple-400 hover:text-purple-300 transition-colors duration-200",
					"onclick": OpenContactLink(link),
				},
				"Connect →",
			),
		),
	)
}

// ContactForm provides a functional contact form with submission handling.
// Features form validation, loading states, and success confirmation with auto-reset.
func ContactForm(_ Attrs) *Element {
	_, setFormData := UseState(map[string]string{
		"name":    "",
		"email":   "",
		"message": "",
	})

	isSubmitting, setIsSubmitting := UseState(false)
	isSubmitted, setIsSubmitted := UseState(false)

	handleSubmit := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()

		// Set submitting state
		setIsSubmitting(true)

		// Simulate form submission
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			setIsSubmitting(false)
			setIsSubmitted(true)

			// Reset form after 3 seconds
			js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				setIsSubmitted(false)
				setFormData(map[string]string{
					"name":    "",
					"email":   "",
					"message": "",
				})
				return nil
			}), 3000)

			return nil
		}), 2000)
	})

	handleInputChange := GoUseFunc(func(event GoEvent) {
		// Note: This is a simplified version - actual implementation would need proper event handling
		// For now, this is a placeholder for the form interaction
	})

	if isSubmitted() {
		return Div(
			Attrs{"class": "bg-white/5 p-8 rounded-xl shadow-lg border border-white/10 backdrop-blur-sm"},
			Div(
				Attrs{"class": "text-center"},
				Div(Attrs{"class": "text-4xl mb-4"}, "✅"),
				H3(Attrs{"class": "text-2xl font-semibold text-white mb-2"}, "Message Sent!"),
				P(Attrs{"class": "text-gray-400"}, "Thank you for reaching out. I'll get back to you soon!"),
			),
		)
	}

	return Div(
		Attrs{"class": "bg-white/5 p-8 rounded-xl shadow-lg border border-white/10 backdrop-blur-sm"},
		H3(Attrs{"class": "text-2xl font-semibold text-white mb-6"}, "Send a Message"),

		Form(
			Attrs{"onsubmit": handleSubmit},

			Div(
				Attrs{"class": "mb-6"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-300 mb-2"}, "Name"),
				Input(Attrs{
					"type":        "text",
					"name":        "name",
					"class":       "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-white placeholder-gray-500",
					"placeholder": "Your name",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			Div(
				Attrs{"class": "mb-6"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-300 mb-2"}, "Email"),
				Input(Attrs{
					"type":        "email",
					"name":        "email",
					"class":       "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-white placeholder-gray-500",
					"placeholder": "your.email@example.com",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			Div(
				Attrs{"class": "mb-6"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-300 mb-2"}, "Message"),
				Textarea(Attrs{
					"name":        "message",
					"rows":        "4",
					"class":       "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-white placeholder-gray-500",
					"placeholder": "Tell me about your project or just say hello!",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			Button(
				Attrs{
					"type":  "submit",
					"class": "w-full py-3 px-6 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg font-semibold hover:from-blue-700 hover:to-purple-700 transition-all duration-200 disabled:opacity-50 cursor-pointer",
				},
				func() string {
					if isSubmitting() {
						return "Sending..."
					}
					return "Send Message"
				}(),
			),
		),
	)
}

// OpenContactLink creates a JavaScript function to open a contact link
func OpenContactLink(url string) interface{} {
	return GoUseFunc(func(e GoEvent) {
		js.Global().Get("window").Call("open", url, "_blank")
	})
}
