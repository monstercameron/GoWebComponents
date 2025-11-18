//go:build js && wasm
// +build js,wasm

package website

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
)
// AdvancedFormExample showcases sophisticated form handling with real-time validation.
// Demonstrates hooks.UseState, hooks.UseEffect, hooks.UseFetch, and hooks.GoUseFunc hooks working together
// to create a production-ready form with password strength, async submission, and error handling.
func AdvancedFormExample(_ Attrs) *Element {
	// Form field state management
	username, setUsername := hooks.UseState("")
	email, setEmail := hooks.UseState("")
	password, setPassword := hooks.UseState("")
	confirmPass, setConfirmPass := hooks.UseState("")
	bio, setBio := hooks.UseState("")
	userType, setUserType := hooks.UseState("developer")
	agreeTerms, setAgreeTerms := hooks.UseState(false)

	// Form submission and validation state
	isSubmitting, setIsSubmitting := hooks.UseState(false)
	submitStatus, setSubmitStatus := hooks.UseState("")
	passwordStrength, setPasswordStrength := hooks.UseState(0)

	// Dynamic source code fetching for code view functionality
	sourceUrl := "https://raw.githubusercontent.com/monstercameron/GoWebComponents/refs/heads/master/website/advanced_form.go"
	getFetchState, refetchSource := hooks.UseFetch(sourceUrl)
	fetchState := getFetchState()

	// Calculate password strength when password changes
	hooks.UseEffect(func() func() {
		pass := password()
		if pass == "" {
			setPasswordStrength(0)
			return nil
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
		return nil
	}, password())

	// Input handlers
	handleUsernameChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		value := event.GetValue()
		setUsername(value)
	})

	handleEmailChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		value := event.GetValue()
		setEmail(value)
	})

	handlePasswordChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		value := event.GetValue()
		setPassword(value)
	})

	handleConfirmPassChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		value := event.GetValue()
		setConfirmPass(value)
	})

	handleBioChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		value := event.GetValue()
		setBio(value)
	})

	handleUserTypeChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		value := event.GetValue()
		setUserType(value)
	})

	handleTermsChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		checked := event.IsChecked()
		setAgreeTerms(checked)
	})

	// Form submission
	handleSubmit := hooks.GoUseFunc(func(event dom.GoEvent) {
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

	return dom.Div(
		Attrs{"class": "bg-white rounded-xl border border-gray-200 p-8 shadow-lg"},

		// Header
		dom.Div(
			Attrs{"class": "text-center mb-8"},
			dom.H3(Attrs{"class": "text-2xl font-bold text-gray-900 mb-2"}, "🔧 Advanced Form Example"),
			dom.P(Attrs{"class": "text-gray-600"}, "Comprehensive form handling with validation, state management, and async processing"),
		),

		// Form
		dom.Form(
			Attrs{"class": "space-y-6", "onsubmit": handleSubmit},

			// Username field
			dom.Div(
				Attrs{"class": "space-y-2"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Username"),
				dom.Input(Attrs{
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
						return dom.P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return dom.Text("")
				}(),
			),

			// Email field
			dom.Div(
				Attrs{"class": "space-y-2"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Email"),
				dom.Input(Attrs{
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
						return dom.P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return dom.Text("")
				}(),
			),

			// Password field with strength meter
			dom.Div(
				Attrs{"class": "space-y-2"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Password"),
				dom.Input(Attrs{
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
						return dom.Div(
							Attrs{"class": "mt-2"},
							dom.Div(Attrs{"class": "flex justify-between text-xs text-gray-600 mb-1"},
								dom.Span(nil, "Password Strength"),
								dom.Span(nil, dom.Text(strconv.Itoa(strength)), "%"),
							),
							dom.Div(Attrs{"class": "w-full bg-gray-200 rounded-full h-2"},
								dom.Div(Attrs{
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
					return dom.Text("")
				}(),

				func() *Element {
					if err := passwordError(); err != "" {
						return dom.P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return dom.Text("")
				}(),
			),

			// Confirm Password field
			dom.Div(
				Attrs{"class": "space-y-2"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Confirm Password"),
				dom.Input(Attrs{
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
						return dom.P(Attrs{"class": "text-sm text-red-600"}, err)
					}
					return dom.Text("")
				}(),
			),

			// Bio field
			dom.Div(
				Attrs{"class": "space-y-2"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "Bio (optional)"),
				dom.Textarea(Attrs{
					"value":       bio(),
					"oninput":     handleBioChange,
					"rows":        "4",
					"class":       "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black",
					"placeholder": "Tell us about yourself...",
				}),
				dom.Div(
					Attrs{"class": "flex justify-between text-xs text-gray-500"},
					dom.Span(nil, dom.Text("Optional")),
					dom.Span(nil, dom.Text(fmt.Sprintf("%d/500 characters", len(bio())))),
				),
			),

			// User type selection
			dom.Div(
				Attrs{"class": "space-y-2"},
				dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, "User Type"),
				dom.Select(Attrs{
					"value":    userType(),
					"onchange": handleUserTypeChange,
					"class":    "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 text-black",
				},
					dom.Option(Attrs{"value": "developer"}, "Developer"),
					dom.Option(Attrs{"value": "designer"}, "Designer"),
					dom.Option(Attrs{"value": "manager"}, "Project Manager"),
					dom.Option(Attrs{"value": "other"}, "Other"),
				),
			),

			// Terms agreement
			dom.Div(
				Attrs{"class": "flex items-center space-x-2"},
				dom.Input(Attrs{
					"type":     "checkbox",
					"checked":  agreeTerms(),
					"onchange": handleTermsChange,
					"class":    "h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 text-black",
				}),
				dom.Label(Attrs{"class": "text-sm text-gray-700"}, "I agree to the Terms and Conditions"),
			),

			// Submit button and status
			dom.Div(
				Attrs{"class": "space-y-4"},
				dom.Button(
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
					switch status {
					case "success":
						return dom.Div(Attrs{"class": "p-4 bg-green-100 border border-green-400 text-green-700 rounded-md"},
							dom.P(Attrs{"class": "font-semibold"}, "✅ Success!"),
							dom.P(nil, "Form submitted successfully. All data has been validated and processed."),
						)
					case "error":
						return dom.Div(Attrs{"class": "p-4 bg-red-100 border border-red-400 text-red-700 rounded-md"},
							dom.P(Attrs{"class": "font-semibold"}, "❌ Error!"),
							dom.P(nil, "Please fix the validation errors and try again."),
						)
					default:
						return dom.Text("")
					}
				}(),
			),
		),

		// Source Code Preview Section
		dom.Div(
			Attrs{"class": "mt-12 border-t border-gray-200 pt-8"},
			dom.H3(Attrs{"class": "text-xl font-semibold text-gray-900 mb-4"}, "📋 Advanced Form Source Code"),
			dom.P(Attrs{"class": "text-gray-600 mb-4"},
				"This section fetches and displays the source code for this advanced form example using hooks.UseFetch."),

			// Fetch controls
			dom.Div(
				Attrs{"class": "flex items-center gap-4 mb-4"},
				dom.Button(
					Attrs{
						"onclick": hooks.GoUseFunc(func(event dom.GoEvent) {
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
				dom.Span(Attrs{"class": "text-sm text-gray-500"},
					fmt.Sprintf("Source URL: %s", sourceUrl)),
			),

			// Source code display
			func() *Element {
				if fetchState.Loading {
					return dom.Div(
						Attrs{"class": "p-6 bg-gray-50 border border-gray-200 rounded-lg"},
						dom.P(Attrs{"class": "text-blue-600 flex items-center gap-2"},
							dom.Span(nil, "🔄"),
							"Loading source code...",
						),
					)
				} else if fetchState.Error != "" {
					return dom.Div(
						Attrs{"class": "p-6 bg-red-50 border border-red-200 rounded-lg"},
						dom.P(Attrs{"class": "text-red-600 font-semibold mb-2"}, "❌ Error fetching source code"),
						dom.P(Attrs{"class": "text-red-600 text-sm"}, fetchState.Error),
						dom.Button(
							Attrs{
								"onclick": hooks.GoUseFunc(func(event dom.GoEvent) {
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
					return dom.Div(
						Attrs{"class": "bg-gray-900 rounded-lg overflow-hidden"},
						dom.Div(
							Attrs{"class": "bg-gray-800 px-4 py-2 border-b border-gray-700"},
							dom.P(Attrs{"class": "text-gray-300 text-sm font-mono"}, "advanced_form.go"),
						),
						dom.Pre(
							Attrs{
								"class": "p-6 text-sm text-gray-100 font-mono overflow-x-auto",
								"style": "max-height: 500px; overflow-y: auto;",
							},
							dom.Code(nil, sourceCode),
						),
					)
				}
				return dom.Div(
					Attrs{"class": "p-6 bg-gray-50 border border-gray-200 rounded-lg"},
					dom.P(Attrs{"class": "text-gray-600"}, "No source code available"),
				)
			}(),
		),
	)
}

// FormField creates a labeled form field with validation
func FormField(label string, input *Element, errorMsg string, isValidating bool) *Element {
	return dom.Div(
		Attrs{"class": "space-y-2"},
		dom.Label(Attrs{"class": "block text-sm font-medium text-gray-700"}, label),
		input,
		func() *Element {
			if isValidating {
				return dom.P(Attrs{"class": "text-sm text-blue-600 flex items-center"},
					dom.Span(Attrs{"class": "mr-2"}, "⏳"),
					"Validating...",
				)
			}
			if errorMsg != "" {
				return dom.P(Attrs{"class": "text-sm text-red-600 flex items-center"},
					dom.Span(Attrs{"class": "mr-2"}, "❌"),
					errorMsg,
				)
			}
			return dom.Div(nil)
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

	return dom.Div(
		Attrs{"class": "space-y-1"},
		dom.Div(
			Attrs{"class": "flex justify-between items-center"},
			dom.Span(Attrs{"class": "text-xs text-gray-600"}, "Password Strength:"),
			dom.Span(Attrs{"class": "text-xs font-medium " + strengthColor}, strengthText),
		),
		dom.Div(
			Attrs{"class": "w-full bg-gray-200 rounded-full h-2"},
			dom.Div(Attrs{
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



