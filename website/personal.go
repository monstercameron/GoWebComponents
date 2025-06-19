//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// PersonalHeroSection creates the main hero section for Earl Cameron
func PersonalHeroSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "home",
			"class": "relative min-h-screen flex items-center justify-center py-20 overflow-hidden",
		},
		// Animated background elements
		Div(
			Attrs{"class": "absolute inset-0 overflow-hidden"},
			Div(Attrs{"class": "absolute -top-40 -right-40 w-80 h-80 bg-purple-300 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob"}),
			Div(Attrs{"class": "absolute -bottom-40 -left-40 w-80 h-80 bg-yellow-300 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob animation-delay-2000"}),
			Div(Attrs{"class": "absolute top-40 left-40 w-80 h-80 bg-pink-300 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob animation-delay-4000"}),
		),

		// Main content
		Div(
			Attrs{"class": "relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center"},
			Div(
				Attrs{"class": "mb-8"},
				H1(
					Attrs{"class": "text-5xl md:text-7xl font-bold mb-6"},
					Span(Attrs{"class": "bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent"}, "Earl Cameron"),
				),
				H2(
					Attrs{"class": "text-2xl md:text-3xl text-gray-700 mb-4 font-light"},
					"Full-Stack Developer & Creator of GoWebComponents",
				),
				P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto mb-8"},
					"Passionate about building innovative web technologies and creating developer tools that make coding more enjoyable. Currently revolutionizing frontend development with Go and WebAssembly.",
				),
			),

			// CTA Buttons
			Div(
				Attrs{"class": "flex flex-col sm:flex-row gap-4 justify-center mb-12"},
				Button(
					Attrs{
						"class":   "px-8 py-4 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg font-semibold shadow-lg hover:shadow-xl transform hover:-translate-y-1 transition-all duration-200",
						"onclick": ScrollToSection("projects"),
					},
					"🚀 View My Projects",
				),
				Button(
					Attrs{
						"class":   "px-8 py-4 border-2 border-gray-300 text-gray-700 rounded-lg font-semibold hover:border-purple-600 hover:text-purple-600 transition-all duration-200",
						"onclick": ScrollToSection("contact"),
					},
					"📬 Get In Touch",
				),
			),

			// Quick stats
			Div(
				Attrs{"class": "grid grid-cols-2 md:grid-cols-4 gap-8 max-w-2xl mx-auto"},
				PersonalStatCard("5+", "Years Experience"),
				PersonalStatCard("50+", "Projects Built"),
				PersonalStatCard("10+", "Technologies"),
				PersonalStatCard("1", "Groundbreaking Framework"),
			),
		),
	)
}

// PersonalStatCard creates a small stat display for personal section
func PersonalStatCard(number, label string) *Element {
	return Div(
		Attrs{"class": "text-center"},
		P(Attrs{"class": "text-3xl font-bold text-purple-600"}, number),
		P(Attrs{"class": "text-sm text-gray-600"}, label),
	)
}

// PersonalAboutSection creates the about section
func PersonalAboutSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "about",
			"class": "py-20 bg-white",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"About Me",
				),
				P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"Full-Stack Engineer | AI & Machine Learning Enthusiast | Innovation in Go & WebAssembly",
				),
			),

			Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-12 items-center"},

				// Profile content
				Div(
					Attrs{"class": "space-y-6"},
					P(
						Attrs{"class": "text-lg text-gray-700 leading-relaxed"},
						"Hi! I'm Earl Cameron, a passionate Full-Stack Engineer with expertise in AI & Machine Learning. Proficient in React, Node.js, Angular, and pioneering innovative solutions with Go and WebAssembly. My journey has led me to create cutting-edge frameworks like GoWebComponents and develop innovative solutions like gRPC-over-WebSocket tunneling.",
					),
					P(
						Attrs{"class": "text-lg text-gray-700 leading-relaxed"},
						"I specialize in modern web development techniques, combining server-side Go with HTMX for lightning-fast performance and JavaScript-lite interactivity. My focus is on creating scalable, efficient applications that push the boundaries of what's possible in web development, from AI-powered features to seamless real-time communication.",
					),

					// Key highlights
					Div(
						Attrs{"class": "space-y-3"},
						PersonalHighlightItem("🎯", "Expertise", "Full-Stack Engineering with AI/ML focus"),
						PersonalHighlightItem("💡", "Innovation", "Go + WebAssembly for modern web apps"),
						PersonalHighlightItem("🌟", "Achievements", "GoWebComponents & gRPC Tunnel creator"),
						PersonalHighlightItem("🚀", "Tech Leadership", "Pioneering JavaScript-lite web development"),
					),
				),

				// Profile image section
				Div(
					Attrs{"class": "relative"},
					Div(
						Attrs{"class": "relative max-w-md mx-auto"},

						// Profile image with modern styling
						Div(
							Attrs{"class": "relative"},
							Img(Attrs{
								"src":   "/static/images/profile-2025.jpg",
								"alt":   "Earl Cameron - Full-Stack Engineer",
								"class": "w-full h-auto rounded-2xl shadow-2xl object-cover border-4 border-white/50 backdrop-blur-sm",
							}),

							// Gradient overlay for better text readability
							Div(Attrs{"class": "absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-transparent rounded-2xl"}),

							// Profile info overlay
							Div(
								Attrs{"class": "absolute bottom-0 left-0 right-0 p-6 text-white"},
								H3(Attrs{"class": "text-xl font-bold mb-1 drop-shadow-lg"}, "Earl Cameron"),
								P(Attrs{"class": "text-sm opacity-90 drop-shadow-md"}, "Full-Stack Engineer"),
								P(Attrs{"class": "text-xs opacity-75 drop-shadow-md"}, "AI/ML Enthusiast & Framework Creator"),
							),
						),

						// Floating accent elements
						Div(Attrs{"class": "absolute -top-4 -right-4 w-8 h-8 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-full shadow-lg animate-pulse"}),
						Div(Attrs{"class": "absolute -bottom-4 -left-4 w-6 h-6 bg-gradient-to-br from-purple-500 to-pink-600 rounded-full shadow-lg animate-pulse delay-1000"}),
					),
				),
			),
		),
	)
}

// PersonalHighlightItem creates a highlight item with icon
func PersonalHighlightItem(icon, title, description string) *Element {
	return Div(
		Attrs{"class": "flex items-start space-x-3"},
		Span(Attrs{"class": "text-2xl"}, icon),
		Div(
			nil,
			P(Attrs{"class": "font-semibold text-gray-900"}, title),
			P(Attrs{"class": "text-gray-600"}, description),
		),
	)
}

// PersonalSkillsSection showcases technical skills
func PersonalSkillsSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "skills",
			"class": "py-20 bg-gray-50",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"Skills & Technologies",
				),
				P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"A comprehensive toolkit for modern web development",
				),
			),

			// Skills grid
			Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8"},
				PersonalSkillCategory("🚀 Languages", []string{
					"Go", "JavaScript", "TypeScript", "Python", "HTML5", "CSS3",
				}),
				PersonalSkillCategory("⚛️ Frontend", []string{
					"React", "Angular", "GoWebComponents", "Vue.js", "Tailwind CSS", "WebAssembly", "HTMX",
				}),
				PersonalSkillCategory("🔧 Backend", []string{
					"Node.js", "Go", "Express", "gRPC", "PostgreSQL", "MongoDB", "Redis",
				}),
				PersonalSkillCategory("☁️ DevOps", []string{
					"Docker", "AWS", "GitHub Actions", "Nginx", "Linux", "Git",
				}),
				PersonalSkillCategory("🎨 Design", []string{
					"Figma", "Adobe XD", "UI/UX Design", "Responsive Design", "Accessibility", "Animation",
				}),
				PersonalSkillCategory("🧠 AI & Data", []string{
					"Machine Learning", "TensorFlow", "Data Analysis", "API Design", "Microservices", "GraphQL",
				}),
			),
		),
	)
}

// PersonalSkillCategory creates a skill category card
func PersonalSkillCategory(title string, skills []string) *Element {
	skillElements := make([]interface{}, len(skills))
	for i, skill := range skills {
		skillElements[i] = Span(
			Attrs{"class": "inline-block bg-white px-3 py-1 rounded-full text-sm text-gray-700 shadow-sm"},
			skill,
		)
	}

	return Div(
		Attrs{"class": "bg-white rounded-xl p-6 shadow-lg hover:shadow-xl transition-shadow duration-300"},
		H3(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, title),
		Div(
			Attrs{"class": "flex flex-wrap gap-2"},
			skillElements...,
		),
	)
}

// ScrollToSection creates a JavaScript function to scroll to a section
func ScrollToSection(sectionId string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		element := js.Global().Get("document").Call("getElementById", sectionId)
		if !element.IsNull() {
			element.Call("scrollIntoView", map[string]interface{}{
				"behavior": "smooth",
			})
		}
		return nil
	})
}
