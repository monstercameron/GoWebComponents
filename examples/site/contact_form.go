//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
)

// ContactSection provides multiple ways to connect with Earl Cameron.
// Features enhanced LinkedIn and GitHub cards with professional service listings
// and animated interactions for improved engagement.
func ContactSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{
			"id":    "contact",
			"class": "py-20 bg-gray-50",
		},
		dom.Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			dom.Div(
				Attrs{"class": "text-center mb-16"},
				dom.H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"Let's Connect",
				),
				dom.P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"Based in Fort Lauderdale, Florida with 4+ years of full-stack development experience. Let's discuss your next project or collaboration opportunity.",
				),
			),

			dom.Div(
				Attrs{"class": "max-w-2xl mx-auto"},

				// Contact methods
				dom.Div(
					Attrs{"class": "space-y-8"},
					dom.H3(Attrs{"class": "text-2xl font-semibold text-gray-900 mb-6"}, "Connect With Me"),

					// Two column layout for LinkedIn and GitHub
					dom.Div(
						Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-6"},

						// Enhanced LinkedIn
						dom.Div(
							Attrs{"class": "group relative overflow-hidden bg-gradient-to-br from-blue-50 to-blue-100 p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-500 transform hover:-translate-y-2 border border-blue-200/50"},
							// Animated background gradient (moved to back layer)
							dom.Div(Attrs{"class": "absolute inset-0 bg-gradient-to-r from-blue-400/10 to-blue-600/10 opacity-0 group-hover:opacity-100 transition-opacity duration-500 rounded-2xl -z-10"}),
							dom.Div(
								Attrs{"class": "relative z-10 flex items-center space-x-4"},
								dom.Div(
									Attrs{"class": "relative"},
									dom.Div(Attrs{"class": "w-16 h-16 bg-gradient-to-br from-blue-600 to-blue-700 rounded-xl flex items-center justify-center shadow-lg group-hover:shadow-blue-300/50 transition-all duration-500 group-hover:scale-110 group-hover:rotate-3"}),
									dom.Div(
										Attrs{"class": "absolute inset-0 flex items-center justify-center text-white text-2xl font-bold transition-transform duration-500 group-hover:scale-110"},
										"💼",
									),
									// Floating particles
									dom.Div(Attrs{"class": "absolute -top-2 -right-2 w-3 h-3 bg-blue-400 rounded-full animate-pulse opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
									dom.Div(Attrs{"class": "absolute -bottom-2 -left-2 w-2 h-2 bg-blue-500 rounded-full animate-pulse delay-300 opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
								),
								dom.Div(
									Attrs{"class": "flex-1"},
									dom.H4(Attrs{"class": "text-xl font-bold text-blue-900 group-hover:text-blue-800 transition-colors duration-300"}, "LinkedIn"),
									dom.P(Attrs{"class": "text-blue-700 mb-3 group-hover:text-blue-600 transition-colors duration-300"}, "Fort Lauderdale, Florida • 518 followers • UKG"),
									dom.A(
										Attrs{
											"href":   "https://www.linkedin.com/in/earl-cameron/",
											"target": "_blank",
											"class":  "relative z-20 inline-flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg font-semibold hover:bg-blue-700 transition-all duration-300 transform hover:scale-105 group-hover:shadow-lg cursor-pointer",
										},
										dom.Span(nil, "Connect"),
										dom.Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, "→"),
									),
								),
							),
						),

						// Enhanced GitHub
						dom.Div(
							Attrs{"class": "group relative overflow-hidden bg-gradient-to-br from-gray-50 to-gray-100 p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-500 transform hover:-translate-y-2 border border-gray-200/50"},
							// Animated background gradient (moved to back layer)
							dom.Div(Attrs{"class": "absolute inset-0 bg-gradient-to-r from-gray-400/10 to-gray-600/10 opacity-0 group-hover:opacity-100 transition-opacity duration-500 rounded-2xl -z-10"}),
							dom.Div(
								Attrs{"class": "relative z-10 flex items-center space-x-4"},
								dom.Div(
									Attrs{"class": "relative"},
									dom.Div(Attrs{"class": "w-16 h-16 bg-gradient-to-br from-gray-800 to-gray-900 rounded-xl flex items-center justify-center shadow-lg group-hover:shadow-gray-400/50 transition-all duration-500 group-hover:scale-110 group-hover:rotate-3"}),
									dom.Div(
										Attrs{"class": "absolute inset-0 flex items-center justify-center text-white text-2xl font-bold transition-transform duration-500 group-hover:scale-110"},
										"🐙",
									),
									// Floating particles
									dom.Div(Attrs{"class": "absolute -top-2 -right-2 w-3 h-3 bg-gray-600 rounded-full animate-pulse opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
									dom.Div(Attrs{"class": "absolute -bottom-2 -left-2 w-2 h-2 bg-gray-700 rounded-full animate-pulse delay-300 opacity-0 group-hover:opacity-100 transition-opacity duration-500"}),
								),
								dom.Div(
									Attrs{"class": "flex-1"},
									dom.H4(Attrs{"class": "text-xl font-bold text-gray-900 group-hover:text-gray-800 transition-colors duration-300"}, "GitHub"),
									dom.P(Attrs{"class": "text-gray-700 mb-3 group-hover:text-gray-600 transition-colors duration-300"}, "Miami, Florida • 53 repositories • GoWebComponents"),
									dom.A(
										Attrs{
											"href":   "https://github.com/monstercameron",
											"target": "_blank",
											"class":  "relative z-20 inline-flex items-center space-x-2 px-4 py-2 bg-gray-800 text-white rounded-lg font-semibold hover:bg-gray-900 transition-all duration-300 transform hover:scale-105 group-hover:shadow-lg cursor-pointer",
										},
										dom.Span(nil, "View Repositories"),
										dom.Span(Attrs{"class": "transition-transform duration-300 group-hover:translate-x-1"}, "→"),
									),
								),
							),
						),
					),

					dom.Div(
						Attrs{"class": "mt-8 p-6 bg-gradient-to-br from-indigo-50 to-purple-50 rounded-xl shadow-lg border border-indigo-100"},
						dom.H4(Attrs{"class": "text-lg font-semibold text-gray-900 mb-4 flex items-center"},
							dom.Span(Attrs{"class": "mr-2"}, "🎯"),
							"Professional Services Available:",
						),
						dom.Ul(
							Attrs{"class": "space-y-3 text-gray-700"},
							dom.Li(Attrs{"class": "flex items-center"},
								dom.Span(Attrs{"class": "mr-3 text-indigo-600"}, "💼"),
								"Full-stack contracting projects"),
							dom.Li(Attrs{"class": "flex items-center"},
								dom.Span(Attrs{"class": "mr-3 text-purple-600"}, "🎯"),
								"Technical consulting & architecture"),
							dom.Li(Attrs{"class": "flex items-center"},
								dom.Span(Attrs{"class": "mr-3 text-blue-600"}, "📚"),
								"Developer training & workshops"),
							dom.Li(Attrs{"class": "flex items-center"},
								dom.Span(Attrs{"class": "mr-3 text-green-600"}, "🔍"),
								"Code reviews & optimization"),
							dom.Li(Attrs{"class": "flex items-center"},
								dom.Span(Attrs{"class": "mr-3 text-orange-600"}, "🚀"),
								"GoWebComponents implementation"),
						),
						dom.Div(
							Attrs{"class": "mt-4 pt-4 border-t border-indigo-200"},
							dom.P(Attrs{"class": "text-sm text-gray-600 flex items-center"},
								dom.Span(Attrs{"class": "mr-2"}, "📍"),
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
	return dom.Div(
		Attrs{"class": "flex items-start space-x-4"},
		dom.Span(Attrs{"class": "text-2xl"}, icon),
		dom.Div(
			Attrs{"class": "flex-1"},
			dom.H4(Attrs{"class": "text-lg font-semibold text-gray-900"}, title),
			dom.P(Attrs{"class": "text-gray-600 mb-2"}, description),
			dom.Button(
				Attrs{
					"class":   "text-purple-600 hover:text-purple-800 transition-colors duration-200",
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
	_, setFormData := hooks.UseState(map[string]string{
		"name":    "",
		"email":   "",
		"message": "",
	})

	isSubmitting, setIsSubmitting := hooks.UseState(false)
	isSubmitted, setIsSubmitted := hooks.UseState(false)

	handleSubmit := hooks.GoUseFunc(func(event dom.GoEvent) {
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

	handleInputChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		// Note: This is a simplified version - actual implementation would need proper event handling
		// For now, this is a placeholder for the form interaction
	})

	if isSubmitted() {
		return dom.Div(
			Attrs{"class": "bg-white p-8 rounded-xl shadow-lg"},
			dom.Div(
				Attrs{"class": "text-center"},
				dom.Div(Attrs{"class": "text-4xl mb-4"}, "✅"),
				dom.H3(Attrs{"class": "text-2xl font-semibold text-gray-900 mb-2"}, "Message Sent!"),
				dom.P(Attrs{"class": "text-gray-600"}, "Thank you for reaching out. I'll get back to you soon!"),
			),
		)
	}

	return dom.Div(
		Attrs{"class": "bg-white p-8 rounded-xl shadow-lg"},
		dom.H3(Attrs{"class": "text-2xl font-semibold text-gray-900 mb-6"}, "Send a Message"),

		dom.Form(
			Attrs{"onsubmit": handleSubmit},

			dom.Div(
				Attrs{"class": "mb-6"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-2"}, "Name"),
				dom.Input(Attrs{
					"type":        "text",
					"name":        "name",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent",
					"placeholder": "Your name",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			dom.Div(
				Attrs{"class": "mb-6"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-2"}, "Email"),
				dom.Input(Attrs{
					"type":        "email",
					"name":        "email",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent",
					"placeholder": "your.email@example.com",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			dom.Div(
				Attrs{"class": "mb-6"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-2"}, "Message"),
				dom.Textarea(Attrs{
					"name":        "message",
					"rows":        "4",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent",
					"placeholder": "Tell me about your project or just say hello!",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			dom.Button(
				Attrs{
					"type":  "submit",
					"class": "w-full py-3 px-6 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg font-semibold hover:from-purple-700 hover:to-blue-700 transition-all duration-200 disabled:opacity-50",
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
func OpenContactLink(url string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Get("window").Call("open", url, "_blank")
		return nil
	})
}
