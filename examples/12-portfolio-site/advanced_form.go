//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// AdvancedFormExample showcases sophisticated form handling with real-time validation.
// Demonstrates UseState, UseEffect, UseFetch, and UseEvent hooks working together
// to create a production-ready form with password strength, async submission, and error handling.
func AdvancedFormExample(_ Attrs) *Element {
	// Form field state management
	username, setUsername := UseState("")
	email, setEmail := UseState("")
	password, setPassword := UseState("")
	confirmPass, setConfirmPass := UseState("")
	bio, setBio := UseState("")
	userType, setUserType := UseState("developer")
	agreeTerms, setAgreeTerms := UseState(false)

	// Form submission and validation state
	isSubmitting, setIsSubmitting := UseState(false)
	submitStatus, setSubmitStatus := UseState("")
	passwordStrength, setPasswordStrength := UseState(0)
	fieldClass := "w-full rounded-md border border-white/10 bg-white/5 px-3 py-2 text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-400"
	fieldErrorClass := "w-full rounded-md border border-red-400/70 bg-red-500/10 px-3 py-2 text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-red-400"

	// Calculate password strength when password changes
	UseEffect(func() func() {
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
	handleUsernameChange := UseEvent(func(event InputEvent) {
		value := event.GetValue()
		setUsername(value)
	})

	handleEmailChange := UseEvent(func(event InputEvent) {
		value := event.GetValue()
		setEmail(value)
	})

	handlePasswordChange := UseEvent(func(event InputEvent) {
		value := event.GetValue()
		setPassword(value)
	})

	handleConfirmPassChange := UseEvent(func(event InputEvent) {
		value := event.GetValue()
		setConfirmPass(value)
	})

	handleBioChange := UseEvent(func(event InputEvent) {
		value := event.GetValue()
		setBio(value)
	})

	handleUserTypeChange := UseEvent(func(event ChangeEvent) {
		value := event.GetValue()
		setUserType(value)
	})

	handleTermsChange := UseEvent(func(event ChangeEvent) {
		checked := event.IsChecked()
		setAgreeTerms(checked)
	})

	// Form submission
	handleSubmit := UseEvent(func(event FormEvent) {
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
		Attrs{"class": "rounded-2xl border border-white/10 bg-slate-950/70 p-8 shadow-2xl backdrop-blur-sm"},

		// Header
		Div(
			Attrs{"class": "text-center mb-8"},
			H3(Attrs{"class": "mb-2 text-2xl font-bold text-slate-100"}, "🔧 Advanced Form Example"),
			P(Attrs{"class": "text-slate-400"}, "Comprehensive form handling with validation, state management, and async processing"),
		),

		// Form
		Form(
			Attrs{"class": "space-y-6", "onsubmit": handleSubmit},

			// Username field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Username"),
				Input(Attrs{
					"type":    "text",
					"value":   username(),
					"oninput": handleUsernameChange,
					"class": func() string {
						if usernameError() != "" {
							return fieldErrorClass
						}
						return fieldClass
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
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Email"),
				Input(Attrs{
					"type":    "email",
					"value":   email(),
					"oninput": handleEmailChange,
					"class": func() string {
						if emailError() != "" {
							return fieldErrorClass
						}
						return fieldClass
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
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Password"),
				Input(Attrs{
					"type":    "password",
					"value":   password(),
					"oninput": handlePasswordChange,
					"class": func() string {
						if passwordError() != "" {
							return fieldErrorClass
						}
						return fieldClass
					}(),
					"placeholder": "Enter password",
				}),

				// Password strength meter
				func() *Element {
					if password() != "" {
						strength := passwordStrength()
						return Div(
							Attrs{"class": "mt-2"},
							Div(Attrs{"class": "mb-1 flex justify-between text-xs text-slate-400"},
								Span(nil, "Password Strength"),
								Span(nil, Text(strconv.Itoa(strength)), "%"),
							),
							Div(Attrs{"class": "h-2 w-full rounded-full bg-white/10"},
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
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Confirm Password"),
				Input(Attrs{
					"type":    "password",
					"value":   confirmPass(),
					"oninput": handleConfirmPassChange,
					"class": func() string {
						if confirmPassError() != "" {
							return fieldErrorClass
						}
						return fieldClass
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
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Bio (optional)"),
				Textarea(Attrs{
					"value":       bio(),
					"oninput":     handleBioChange,
					"rows":        "4",
					"class":       fieldClass,
					"placeholder": "Tell us about yourself...",
				}),
				Div(
					Attrs{"class": "flex justify-between text-xs text-slate-500"},
					Span(nil, Text("Optional")),
					Span(nil, Text(fmt.Sprintf("%d/500 characters", len(bio())))),
				),
			),

			// User type selection
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "User Type"),
				Select(Attrs{
					"value":    userType(),
					"onchange": handleUserTypeChange,
					"class":    fieldClass,
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
					"class":    "h-4 w-4 rounded border-white/20 bg-slate-900 text-indigo-500 focus:ring-indigo-400 focus:ring-offset-0",
				}),
				Label(Attrs{"class": "text-sm text-slate-300"}, "I agree to the Terms and Conditions"),
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
								return "w-full cursor-not-allowed rounded-md border border-white/10 bg-slate-900 px-4 py-3 text-slate-500"
							}
							return "w-full rounded-md bg-indigo-500 px-4 py-3 text-white transition-colors hover:bg-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-400"
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
						return Div(Attrs{"class": "rounded-md border border-emerald-400/40 bg-emerald-500/10 p-4 text-emerald-200"},
							P(Attrs{"class": "font-semibold"}, "✅ Success!"),
							P(nil, "Form submitted successfully. All data has been validated and processed."),
						)
					case "error":
						return Div(Attrs{"class": "rounded-md border border-red-400/40 bg-red-500/10 p-4 text-red-200"},
							P(Attrs{"class": "font-semibold"}, "❌ Error!"),
							P(nil, "Please fix the validation errors and try again."),
						)
					default:
						return Text("")
					}
				}(),
			),
		),
	)
}

// FormField creates a labeled form field with validation
func FormField(label string, input *Element, errorMsg string, isValidating bool) *Element {
	return Div(
		Attrs{"class": "space-y-2"},
		Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, label),
		input,
		func() *Element {
			if isValidating {
				return P(Attrs{"class": "flex items-center text-sm text-cyan-300"},
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
			Span(Attrs{"class": "text-xs text-slate-400"}, "Password Strength:"),
			Span(Attrs{"class": "text-xs font-medium " + strengthColor}, strengthText),
		),
		Div(
			Attrs{"class": "h-2 w-full rounded-full bg-white/10"},
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
