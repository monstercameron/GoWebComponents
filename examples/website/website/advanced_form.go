//go:build js && wasm
// +build js,wasm

package website

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// AdvancedFormExample showcases sophisticated form handling with real-time validation.
// Demonstrates GoUseState, GoUseEffect, GoUseFetch, and GoUseFunc hooks working together
// to create a production-ready form with password strength, async submission, and error handling.
func AdvancedFormExample(props Attrs) *Element {
	// Form field state management
	username, setUsername := GoUseState("")
	email, setEmail := GoUseState("")
	password, setPassword := GoUseState("")
	confirmPass, setConfirmPass := GoUseState("")
	bio, setBio := GoUseState("")
	userType, setUserType := GoUseState("developer")
	agreeTerms, setAgreeTerms := GoUseState(false)

	// Form submission and validation state
	isSubmitting, setIsSubmitting := GoUseState(false)
	submitStatus, setSubmitStatus := GoUseState("")
	passwordStrength, setPasswordStrength := GoUseState(0)

	// Dynamic source code fetching for code view functionality
	sourceUrl := "https://raw.githubusercontent.com/monstercameron/GoWebComponents/refs/heads/master/website/advanced_form.go"
	getFetchState, refetchSource := GoUseFetch(sourceUrl)
	fetchState := getFetchState()

	// Calculate password strength when password changes
	GoUseEffect(func() {
		pass := password()
		if pass == "" {
			setPasswordStrength(0)
			return
		}

		strength := 0
		if len(pass) >= 8 {
			strength += 25
		}
		if regexp.MustCompile(`[a-z]`).MatchString(pass) {
			strength += 25
		}
		if regexp.MustCompile(`[A-Z]`).MatchString(pass) {
			strength += 25
		}
		if regexp.MustCompile(`[0-9]`).MatchString(pass) {
			strength += 12
		}
		if regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(pass) {
			strength += 13
		}
		if strength > 100 {
			strength = 100
		}
		setPasswordStrength(strength)
	})

	// Input handlers
	handleUsernameChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue()
		setUsername(value)
	})

	handleEmailChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue()
		setEmail(value)
	})

	handlePasswordChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue()
		setPassword(value)
	})

	handleConfirmPassChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue()
		setConfirmPass(value)
	})

	handleBioChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue()
		setBio(value)
	})

	handleUserTypeChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue()
		setUserType(value)
	})

	handleTermsChange := GoUseFunc(func(event GoEvent) {
		checked := event.IsChecked()
		setAgreeTerms(checked)
	})

	// Form submission
	handleSubmit := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		setIsSubmitting(true)
		setSubmitStatus("")

		// Simulate async submission
		go func() {
			time.Sleep(2 * time.Second)

			// Validate form
			usernameVal := username()
			emailVal := email()
			passwordVal := password()
			confirmPassVal := confirmPass()
			agreeTermsVal := agreeTerms()

			if usernameVal == "" || emailVal == "" || passwordVal == "" {
				setSubmitStatus("error")
				setIsSubmitting(false)
				return
			}

			if passwordVal != confirmPassVal {
				setSubmitStatus("error")
				setIsSubmitting(false)
				return
			}

			if !agreeTermsVal {
				setSubmitStatus("error")
				setIsSubmitting(false)
				return
			}

			// Success
			setSubmitStatus("success")
			setIsSubmitting(false)

			// Reset form
			setUsername("")
			setEmail("")
			setPassword("")
			setConfirmPass("")
			setBio("")
			setUserType("developer")
			setAgreeTerms(false)
		}()
	})

	// Helper functions for validation display
	usernameError := func() string {
		val := username()
		if val == "" {
			return ""
		}
		if len(val) < 3 || len(val) > 20 {
			return "Username must be 3-20 characters"
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(val) {
			return "Username can only contain letters, numbers, and underscores"
		}
		return ""
	}

	emailError := func() string {
		val := email()
		if val == "" {
			return ""
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(val) {
			return "Please enter a valid email address"
		}
		return ""
	}

	passwordError := func() string {
		val := password()
		if val == "" {
			return ""
		}
		if len(val) < 8 {
			return "Password must be at least 8 characters"
		}
		return ""
	}

	confirmPassError := func() string {
		val := confirmPass()
		passVal := password()
		if val == "" {
			return ""
		}
		if val != passVal {
			return "Passwords do not match"
		}
		return ""
	}

	return Div(
		Attrs{"class": "bg-white rounded-xl border border-gray-200 p-8 shadow-lg"},

		// Header
		Div(
			Attrs{"class": "text-center mb-8"},
			H3(Attrs{"class": "text-2xl font-bold text-gray-900 mb-2"}, "🔧 Advanced Form Example"),
			P(Attrs{"class": "text-gray-600"}, "Comprehensive form handling with validation, state management, and async processing"),
		),

		// Form
		Form(
			Attrs{"class": "space-y-6", "onsubmit": handleSubmit},

			// Username field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Username"),
				Input(Attrs{
					"type":    "text",
					"value":   username(),
					"oninput": handleUsernameChange,
					"class": func() string {
						if usernameError() != "" {
							return "w-full px-3 py-2 border border-red-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 text-black"
						}
						return "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black"
					}(),
					"placeholder": "Enter username",
				}),
				func() *Element {
					if err := usernameError(); err != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return Text("")
				}(),
			),

			// Email field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Email"),
				Input(Attrs{
					"type":    "email",
					"value":   email(),
					"oninput": handleEmailChange,
					"class": func() string {
						if emailError() != "" {
							return "w-full px-3 py-2 border border-red-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 text-black"
						}
						return "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black"
					}(),
					"placeholder": "Enter email",
				}),
				func() *Element {
					if err := emailError(); err != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return Text("")
				}(),
			),

			// Password field with strength meter
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Password"),
				Input(Attrs{
					"type":    "password",
					"value":   password(),
					"oninput": handlePasswordChange,
					"class": func() string {
						if passwordError() != "" {
							return "w-full px-3 py-2 border border-red-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 text-black"
						}
						return "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black"
					}(),
					"placeholder": "Enter password",
				}),

				// Password strength meter
				func() *Element {
					if password() != "" {
						strength := passwordStrength()
						return Div(
							Attrs{"class": "mt-2"},
							Div(Attrs{"class": "flex justify-between text-xs text-gray-600 mb-1"},
								Span(nil, "Password Strength"),
								Span(nil, Text(strconv.Itoa(strength)), "%"),
							),
							Div(Attrs{"class": "w-full bg-gray-200 rounded-full h-2"},
								Div(Attrs{
									"class": func() string {
										if strength < 30 {
											return "bg-red-500 h-2 rounded-full transition-all duration-300"
										} else if strength < 70 {
											return "bg-yellow-500 h-2 rounded-full transition-all duration-300"
										}
										return "bg-green-500 h-2 rounded-full transition-all duration-300"
									}(),
									"style": "width: " + strconv.Itoa(strength) + "%;",
								}),
							),
						)
					}
					return Text("")
				}(),

				func() *Element {
					if err := passwordError(); err != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return Text("")
				}(),
			),

			// Confirm Password field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Confirm Password"),
				Input(Attrs{
					"type":    "password",
					"value":   confirmPass(),
					"oninput": handleConfirmPassChange,
					"class": func() string {
						if confirmPassError() != "" {
							return "w-full px-3 py-2 border border-red-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 text-black"
						}
						return "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black"
					}(),
					"placeholder": "Confirm password",
				}),
				func() *Element {
					if err := confirmPassError(); err != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return Text("")
				}(),
			),

			// Bio field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Bio (optional)"),
				Textarea(Attrs{
					"value":       bio(),
					"oninput":     handleBioChange,
					"rows":        "4",
					"class":       "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black",
					"placeholder": "Tell us about yourself...",
				}),
				Div(
					Attrs{"class": "flex justify-between text-xs text-gray-500"},
					Span(nil, Text("Optional")),
					Span(nil, Text(fmt.Sprintf("%d/500 characters", len(bio())))),
				),
			),

			// User type selection
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "User Type"),
				Select(Attrs{
					"value":    userType(),
					"onchange": handleUserTypeChange,
					"class":    "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black",
				},
					Option(Attrs{"value": "developer"}, "Developer"),
					Option(Attrs{"value": "designer"}, "Designer"),
					Option(Attrs{"value": "manager"}, "Project Manager"),
					Option(Attrs{"value": "other"}, "Other"),
				),
			),

			// Terms agreement
			Div(
				Attrs{"class": "flex items-center space-x-2"},
				Input(Attrs{
					"type":     "checkbox",
					"checked":  agreeTerms(),
					"onchange": handleTermsChange,
					"class":    "h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 text-black",
				}),
				Label(Attrs{"class": "text-sm text-gray-700"}, "I agree to the Terms and Conditions"),
			),

			// Submit button and status
			Div(
				Attrs{"class": "space-y-4"},
				Button(
					Attrs{
						"type":     "submit",
						"disabled": isSubmitting() || !agreeTerms(),
						"class": func() string {
							if isSubmitting() || !agreeTerms() {
								return "w-full py-3 px-4 bg-gray-400 text-white rounded-md cursor-not-allowed"
							}
							return "w-full py-3 px-4 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-colors"
						}(),
					},
					func() string {
						if isSubmitting() {
							return "⏳ Submitting..."
						}
						return "✅ Create Account"
					}(),
				),

				// Status messages
				func() *Element {
					status := submitStatus()
					if status == "success" {
						return Div(Attrs{"class": "p-4 bg-green-100 border border-green-400 text-green-700 rounded-md"},
							P(Attrs{"class": "font-semibold"}, "✅ Success!"),
							P(nil, "Form submitted successfully. All data has been validated and processed."),
						)
					} else if status == "error" {
						return Div(Attrs{"class": "p-4 bg-red-100 border border-red-400 text-red-700 rounded-md"},
							P(Attrs{"class": "font-semibold"}, "❌ Error!"),
							P(nil, "Please fix the validation errors and try again."),
						)
					}
					return Text("")
				}(),
			),
		),

		// Source Code Preview Section
		Div(
			Attrs{"class": "mt-12 border-t border-gray-200 pt-8"},
			H3(Attrs{"class": "text-xl font-semibold text-gray-900 mb-4"}, "📋 Advanced Form Source Code"),
			P(Attrs{"class": "text-gray-600 mb-4"},
				"This section fetches and displays the source code for this advanced form example using GoUseFetch."),

			// Fetch controls
			Div(
				Attrs{"class": "flex items-center gap-4 mb-4"},
				Button(
					Attrs{
						"onclick": GoUseFunc(func(event GoEvent) {
							refetchSource()
						}),
						"class": func() string {
							if fetchState.Loading {
								return "px-4 py-2 bg-gray-400 text-white rounded-md cursor-not-allowed"
							}
							return "px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
						}(),
						"disabled": fetchState.Loading,
					},
					func() string {
						if fetchState.Loading {
							return "🔄 Fetching..."
						}
						return "🔄 Refresh Source"
					}(),
				),
				Span(Attrs{"class": "text-sm text-gray-500"},
					fmt.Sprintf("Source URL: %s", sourceUrl)),
			),

			// Source code display
			func() *Element {
				if fetchState.Loading {
					return Div(
						Attrs{"class": "p-6 bg-gray-50 border border-gray-200 rounded-lg"},
						P(Attrs{"class": "text-blue-600 flex items-center gap-2"},
							Span(nil, "🔄"),
							"Loading source code...",
						),
					)
				} else if fetchState.Error != "" {
					return Div(
						Attrs{"class": "p-6 bg-red-50 border border-red-200 rounded-lg"},
						P(Attrs{"class": "text-red-600 font-semibold mb-2"}, "❌ Error fetching source code"),
						P(Attrs{"class": "text-red-600 text-sm"}, fetchState.Error),
						Button(
							Attrs{
								"onclick": GoUseFunc(func(event GoEvent) {
									refetchSource()
								}),
								"class": "mt-3 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 transition-colors",
							},
							"🔄 Retry",
						),
					)
				} else if fetchState.Data != nil {
					// Convert the fetched data to string
					sourceCode := fmt.Sprintf("%v", fetchState.Data)
					return Div(
						Attrs{"class": "bg-gray-900 rounded-lg overflow-hidden"},
						Div(
							Attrs{"class": "bg-gray-800 px-4 py-2 border-b border-gray-700"},
							P(Attrs{"class": "text-gray-300 text-sm font-mono"}, "advanced_form.go"),
						),
						Pre(
							Attrs{
								"class": "p-6 text-sm text-gray-100 font-mono overflow-x-auto",
								"style": "max-height: 500px; overflow-y: auto;",
							},
							Code(nil, sourceCode),
						),
					)
				}
				return Div(
					Attrs{"class": "p-6 bg-gray-50 border border-gray-200 rounded-lg"},
					P(Attrs{"class": "text-gray-600"}, "No source code available"),
				)
			}(),
		),
	)
}

// FormField creates a labeled form field with validation
func FormField(label string, input *Element, errorMsg string, isValidating bool) *Element {
	return Div(
		Attrs{"class": "space-y-2"},
		Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, label),
		input,
		func() *Element {
			if isValidating {
				return P(Attrs{"class": "text-sm text-blue-600 flex items-center"},
					Span(Attrs{"class": "mr-2"}, "⏳"),
					"Validating...",
				)
			}
			if errorMsg != "" {
				return P(Attrs{"class": "text-sm text-red-600 flex items-center"},
					Span(Attrs{"class": "mr-2"}, "❌"),
					errorMsg,
				)
			}
			return Div(nil)
		}(),
	)
}

// PasswordStrengthMeter shows password strength visualization
func PasswordStrengthMeter(strength int) *Element {
	var strengthText string
	var strengthColor string
	var barWidth string

	switch {
	case strength < 30:
		strengthText = "Weak"
		strengthColor = "text-red-600"
		barWidth = "25%"
	case strength < 60:
		strengthText = "Fair"
		strengthColor = "text-yellow-600"
		barWidth = "50%"
	case strength < 80:
		strengthText = "Good"
		strengthColor = "text-blue-600"
		barWidth = "75%"
	default:
		strengthText = "Strong"
		strengthColor = "text-green-600"
		barWidth = "100%"
	}

	return Div(
		Attrs{"class": "space-y-1"},
		Div(
			Attrs{"class": "flex justify-between items-center"},
			Span(Attrs{"class": "text-xs text-gray-600"}, "Password Strength:"),
			Span(Attrs{"class": "text-xs font-medium " + strengthColor}, strengthText),
		),
		Div(
			Attrs{"class": "w-full bg-gray-200 rounded-full h-2"},
			Div(Attrs{
				"class": "h-2 rounded-full transition-all duration-300 " + getStrengthBarColor(strength),
				"style": "width: " + barWidth,
			}),
		),
	)
}

// Helper functions
func getStrengthBarColor(strength int) string {
	switch {
	case strength < 30:
		return "bg-red-500"
	case strength < 60:
		return "bg-yellow-500"
	case strength < 80:
		return "bg-blue-500"
	default:
		return "bg-green-500"
	}
}
