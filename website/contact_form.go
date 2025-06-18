//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// ContactSection creates the contact section
func ContactSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "contact",
			"class": "py-20 bg-gray-50",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"Let's Work Together",
				),
				P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"Ready to build something amazing? I'm always excited to discuss new opportunities and interesting projects.",
				),
			),

			Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-12"},

				// Contact methods
				Div(
					Attrs{"class": "space-y-8"},
					H3(Attrs{"class": "text-2xl font-semibold text-gray-900 mb-6"}, "Get In Touch"),

					ContactMethod("📧", "Email", "Drop me a line anytime", "mailto:earl@earlcameron.com"),
					ContactMethod("💼", "LinkedIn", "Let's connect professionally", "https://linkedin.com/in/earlcameron"),
					ContactMethod("🐙", "GitHub", "Check out my code", "https://github.com/monstercameron"),
					ContactMethod("🐦", "Twitter", "Follow for updates", "https://twitter.com/earlcameron"),

					Div(
						Attrs{"class": "mt-8 p-6 bg-white rounded-xl shadow-lg"},
						H4(Attrs{"class": "text-lg font-semibold text-gray-900 mb-4"}, "Currently Available For:"),
						Ul(
							Attrs{"class": "space-y-2 text-gray-600"},
							Li(nil, "• Full-stack development projects"),
							Li(nil, "• GoWebComponents consulting"),
							Li(nil, "• Technical writing and documentation"),
							Li(nil, "• Open source collaboration"),
							Li(nil, "• Speaking engagements"),
						),
					),
				),

				// Contact form
				ContactForm(nil),
			),
		),
	)
}

// ContactMethod creates a contact method item
func ContactMethod(icon, title, description, link string) *Element {
	return Div(
		Attrs{"class": "flex items-start space-x-4"},
		Span(Attrs{"class": "text-2xl"}, icon),
		Div(
			Attrs{"class": "flex-1"},
			H4(Attrs{"class": "text-lg font-semibold text-gray-900"}, title),
			P(Attrs{"class": "text-gray-600 mb-2"}, description),
			Button(
				Attrs{
					"class":   "text-purple-600 hover:text-purple-800 transition-colors duration-200",
					"onclick": OpenContactLink(link),
				},
				"Connect →",
			),
		),
	)
}

// ContactForm creates an interactive contact form
func ContactForm(props Attrs) *Element {
	_, setFormData := GoUseState(map[string]string{
		"name":    "",
		"email":   "",
		"message": "",
	})

	isSubmitting, setIsSubmitting := GoUseState(false)
	isSubmitted, setIsSubmitted := GoUseState(false)

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
			Attrs{"class": "bg-white p-8 rounded-xl shadow-lg"},
			Div(
				Attrs{"class": "text-center"},
				Div(Attrs{"class": "text-4xl mb-4"}, "✅"),
				H3(Attrs{"class": "text-2xl font-semibold text-gray-900 mb-2"}, "Message Sent!"),
				P(Attrs{"class": "text-gray-600"}, "Thank you for reaching out. I'll get back to you soon!"),
			),
		)
	}

	return Div(
		Attrs{"class": "bg-white p-8 rounded-xl shadow-lg"},
		H3(Attrs{"class": "text-2xl font-semibold text-gray-900 mb-6"}, "Send a Message"),

		Form(
			Attrs{"onsubmit": handleSubmit},

			Div(
				Attrs{"class": "mb-6"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-2"}, "Name"),
				Input(Attrs{
					"type":        "text",
					"name":        "name",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent",
					"placeholder": "Your name",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			Div(
				Attrs{"class": "mb-6"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-2"}, "Email"),
				Input(Attrs{
					"type":        "email",
					"name":        "email",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent",
					"placeholder": "your.email@example.com",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			Div(
				Attrs{"class": "mb-6"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-2"}, "Message"),
				Textarea(Attrs{
					"name":        "message",
					"rows":        "4",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent",
					"placeholder": "Tell me about your project or just say hello!",
					"required":    "true",
					"oninput":     handleInputChange,
				}),
			),

			Button(
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
