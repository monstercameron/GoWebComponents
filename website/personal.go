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
					"Innovating at the intersection of Go and web development",
				),
			),

			Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-12 items-center"},

				// Profile content
				Div(
					Attrs{"class": "space-y-6"},
					P(
						Attrs{"class": "text-lg text-gray-700 leading-relaxed"},
						"Hi! I'm Earl Cameron, a passionate full-stack developer with a love for creating innovative solutions. My journey in tech has led me to explore the cutting edge of web development, culminating in the creation of GoWebComponents.",
					),
					P(
						Attrs{"class": "text-lg text-gray-700 leading-relaxed"},
						"When I'm not coding, you'll find me exploring new technologies, contributing to open source projects, or sharing knowledge with the developer community. I believe in the power of clean code, elegant solutions, and tools that make developers' lives easier.",
					),

					// Key highlights
					Div(
						Attrs{"class": "space-y-3"},
						PersonalHighlightItem("🎯", "Focus", "Frontend innovation with Go & WebAssembly"),
						PersonalHighlightItem("💡", "Mission", "Making web development more efficient and enjoyable"),
						PersonalHighlightItem("🌟", "Achievement", "Created GoWebComponents framework"),
					),
				),

				// Profile image placeholder / tech stack visual
				Div(
					Attrs{"class": "relative"},
					Div(
						Attrs{"class": "bg-gradient-to-br from-purple-100 to-blue-100 rounded-2xl p-8 shadow-lg"},
						Div(
							Attrs{"class": "text-center"},
							Div(Attrs{"class": "text-6xl mb-4"}, "👨‍💻"),
							P(Attrs{"class": "text-lg font-semibold text-gray-800"}, "Earl Cameron"),
							P(Attrs{"class": "text-purple-600"}, "Full-Stack Developer"),
							P(Attrs{"class": "text-sm text-gray-600 mt-2"}, "Creator of GoWebComponents"),
						),
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
					"GoWebComponents", "React", "Vue.js", "Tailwind CSS", "WebAssembly", "PWAs",
				}),
				PersonalSkillCategory("🔧 Backend", []string{
					"Node.js", "Express", "FastAPI", "PostgreSQL", "MongoDB", "Redis",
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
