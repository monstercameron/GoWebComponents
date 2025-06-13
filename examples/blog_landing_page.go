// ./examples/blog_landing_page.go

package examples

import (
	"fmt"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// Simple test component to demonstrate component references
func TestComponent(props Attrs) *Element {
	return Div(Attrs{
		"class": "test-component bg-red-100 p-4 rounded",
	}, Text("This component was passed as a reference!"))
}

// Another test component with different styling
func AnotherTestComponent(props Attrs) *Element {
	return Div(Attrs{
		"class": "another-test bg-green-100 p-4 rounded",
	}, Text("Another component reference!"))
}

// Header component - Navigation and branding
func HeaderComponent(props Attrs) *Element {
	return Header(Attrs{
		"class": "bg-white shadow-md sticky top-0 z-50",
	},
		Nav(Attrs{
			"class": "container mx-auto px-6 py-4",
		},
			Div(Attrs{
				"class": "flex items-center justify-between",
			},
				// Logo/Brand
				Div(Attrs{
					"class": "flex items-center space-x-2",
				},
					H1(Attrs{
						"class": "text-2xl font-bold text-indigo-600",
					}, Text("📝 TechBlog")),
					Span(Attrs{
						"class": "text-sm text-gray-500",
					}, Text("v1.0")),
				),
				// Navigation Menu
				Ul(Attrs{
					"class": "flex space-x-6",
				},
					Li(nil,
						A(Attrs{
							"href":  "#home",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, Text("Home")),
					),
					Li(nil,
						A(Attrs{
							"href":  "#about",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, Text("About")),
					),
					Li(nil,
						A(Attrs{
							"href":  "#posts",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, Text("Posts")),
					),
					Li(nil,
						A(Attrs{
							"href":  "#contact",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, Text("Contact")),
					),
				),
			),
		),
	)
}

// Hero section component - Main banner with call-to-action
func HeroSection(props Attrs) *Element {
	return Section(Attrs{
		"id":    "home",
		"class": "py-20 px-6",
	},
		Div(Attrs{
			"class": "container mx-auto text-center",
		},
			H2(Attrs{
				"class": "text-5xl font-extrabold text-gray-900 mb-6",
			}, Text("Welcome to TechBlog")),
			P(Attrs{
				"class": "text-xl text-gray-600 mb-8 max-w-2xl mx-auto leading-relaxed",
			}, Text("Discover the latest trends in technology, programming, and web development. Join our community of passionate developers and tech enthusiasts.")),
			Div(Attrs{
				"class": "flex justify-center space-x-4",
			},
				Button(Attrs{
					"class": "bg-indigo-600 text-white px-8 py-3 rounded-lg font-semibold hover:bg-indigo-700 transition-colors shadow-lg",
				}, Text("Start Reading")),
				Button(Attrs{
					"class": "border-2 border-indigo-600 text-indigo-600 px-8 py-3 rounded-lg font-semibold hover:bg-indigo-50 transition-colors",
				}, Text("Subscribe")),
			),
			// Demonstration of component references vs return values
			Div(Attrs{
				"class": "mt-8 space-y-4",
			},
				H3(Attrs{
					"class": "text-lg font-semibold text-gray-800",
				}, Text("Component Reference Demo:")),
				// These are component references (functions) - will be called automatically
				TestComponent,
				AnotherTestComponent,
				// This is also a component reference
				TestComponent,
			),
		),
	)
}

// Blog post card component - Individual blog post preview
func BlogPostCard(props Attrs) *Element {
	title := props["title"].(string)
	description := props["description"].(string)
	tag := props["tag"].(string)
	tagColor := props["tagColor"].(string)
	date := props["date"].(string)
	datetime := props["datetime"].(string)

	return Article(Attrs{
		"class": "bg-gray-50 rounded-xl p-6 hover:shadow-lg transition-shadow",
	},
		Div(Attrs{
			"class": "flex items-center mb-4",
		},
			Span(Attrs{
				"class": tagColor,
			}, Text(tag)),
			Time(Attrs{
				"class":    "text-gray-500 text-sm ml-auto",
				"datetime": datetime,
			}, Text(date)),
		),
		H4(Attrs{
			"class": "text-xl font-semibold mb-3 text-gray-900",
		}, Text(title)),
		P(Attrs{
			"class": "text-gray-600 mb-4 line-clamp-3",
		}, Text(description)),
		A(Attrs{
			"href":  "#",
			"class": "text-indigo-600 font-medium hover:text-indigo-800 transition-colors",
		}, Text("Read More →")),
	)
}

// Featured posts section component
func FeaturedPostsSection(props Attrs) *Element {
	return Section(Attrs{
		"id":    "posts",
		"class": "py-16 bg-white",
	},
		Div(Attrs{
			"class": "container mx-auto px-6",
		},
			H3(Attrs{
				"class": "text-3xl font-bold text-center mb-12 text-gray-900",
			}, Text("Featured Posts")),
			Div(Attrs{
				"class": "grid md:grid-cols-2 lg:grid-cols-3 gap-8",
			},
				BlogPostCard(Attrs{
					"title":       "Modern JavaScript Frameworks in 2024",
					"description": "Explore the latest JavaScript frameworks and libraries that are shaping web development. From React to Vue, discover what's trending.",
					"tag":         "JavaScript",
					"tagColor":    "bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded-full",
					"date":        "Jan 15, 2024",
					"datetime":    "2024-01-15",
				}),
				BlogPostCard(Attrs{
					"title":       "Building Web Components with Go and WebAssembly",
					"description": "Learn how to create reactive web components using Go compiled to WebAssembly. A new approach to frontend development.",
					"tag":         "Go",
					"tagColor":    "bg-green-100 text-green-800 text-xs px-2 py-1 rounded-full",
					"date":        "Jan 10, 2024",
					"datetime":    "2024-01-10",
				}),
				BlogPostCard(Attrs{
					"title":       "CSS Grid vs Flexbox: When to Use What",
					"description": "Master the art of CSS layout with this comprehensive guide comparing CSS Grid and Flexbox. Includes practical examples.",
					"tag":         "CSS",
					"tagColor":    "bg-purple-100 text-purple-800 text-xs px-2 py-1 rounded-full",
					"date":        "Jan 5, 2024",
					"datetime":    "2024-01-05",
				}),
			),
		),
	)
}

// About section component - Company information and statistics
func AboutSection(props Attrs) *Element {
	return Section(Attrs{
		"id":    "about",
		"class": "py-16 bg-gray-50",
	},
		Div(Attrs{
			"class": "container mx-auto px-6",
		},
			Div(Attrs{
				"class": "max-w-3xl mx-auto text-center",
			},
				H3(Attrs{
					"class": "text-3xl font-bold mb-6 text-gray-900",
				}, Text("About TechBlog")),
				Blockquote(Attrs{
					"class": "text-lg text-gray-600 italic mb-6 border-l-4 border-indigo-500 pl-6",
				}, Text("\"Technology is best when it brings people together.\" - Matt Mullenweg")),
				P(Attrs{
					"class": "text-gray-600 mb-6 leading-relaxed",
				}, Text("We're a community of developers, designers, and tech enthusiasts sharing knowledge and experiences. Our mission is to make technology accessible and understandable for everyone.")),
				Div(Attrs{
					"class": "flex justify-center space-x-8 text-center",
				},
					Div(nil,
						Strong(Attrs{
							"class": "block text-2xl font-bold text-indigo-600",
						}, Text("500+")),
						Span(Attrs{
							"class": "text-gray-600",
						}, Text("Articles")),
					),
					Div(nil,
						Strong(Attrs{
							"class": "block text-2xl font-bold text-indigo-600",
						}, Text("10K+")),
						Span(Attrs{
							"class": "text-gray-600",
						}, Text("Readers")),
					),
					Div(nil,
						Strong(Attrs{
							"class": "block text-2xl font-bold text-indigo-600",
						}, Text("50+")),
						Span(Attrs{
							"class": "text-gray-600",
						}, Text("Contributors")),
					),
				),
			),
		),
	)
}

// Newsletter section component - Email subscription form
func NewsletterSection(props Attrs) *Element {
	// State for newsletter subscription using Go-branded hooks
	email, setEmail := GoUseState("")
	subscribed, setSubscribed := GoUseState(false)

	// Handle newsletter subscription using GoUseFunc with GoEvent
	handleSubscribe := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if email() != "" {
			setSubscribed(true)
			fmt.Println("Newsletter subscription for:", email())
		}
	})

	// Handle email input change using GoUseFunc with GoEvent
	handleEmailChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue()
		setEmail(value)
	})

	return Section(Attrs{
		"class": "py-16 bg-indigo-600",
	},
		Div(Attrs{
			"class": "container mx-auto px-6 text-center",
		},
			H3(Attrs{
				"class": "text-3xl font-bold text-white mb-4",
			}, Text("Stay Updated")),
			P(Attrs{
				"class": "text-indigo-100 mb-8 max-w-2xl mx-auto",
			}, Text("Subscribe to our newsletter and get the latest tech articles delivered straight to your inbox every week.")),
			func() *Element {
				if subscribed() {
					return Div(Attrs{
						"class": "bg-green-500 text-white px-6 py-3 rounded-lg inline-block",
					}, Text("✅ Thank you for subscribing!"))
				}
				return Form(Attrs{
					"class":    "flex justify-center max-w-md mx-auto",
					"onsubmit": handleSubscribe,
				},
					Input(Attrs{
						"type":        "email",
						"placeholder": "Enter your email",
						"class":       "flex-1 px-4 py-3 rounded-l-lg border-0 focus:ring-2 focus:ring-indigo-300 outline-none",
						"value":       email(),
						"oninput":     handleEmailChange,
						"required":    true,
					}),
					Button(Attrs{
						"type":  "submit",
						"class": "bg-white text-indigo-600 px-6 py-3 rounded-r-lg font-semibold hover:bg-gray-100 transition-colors",
					}, Text("Subscribe")),
				)
			}(),
		),
	)
}

// Footer component - Site links and information
func FooterComponent(props Attrs) *Element {
	return Footer(Attrs{
		"class": "bg-gray-900 text-white py-12",
	},
		Div(Attrs{
			"class": "container mx-auto px-6",
		},
			Div(Attrs{
				"class": "grid md:grid-cols-4 gap-8",
			},
				// Footer Column 1
				Div(nil,
					H5(Attrs{
						"class": "font-bold mb-4",
					}, Text("TechBlog")),
					P(Attrs{
						"class": "text-gray-400 text-sm",
					}, Text("Sharing knowledge and building the future of technology together.")),
				),
				// Footer Column 2
				Div(nil,
					H6(Attrs{
						"class": "font-semibold mb-4",
					}, Text("Categories")),
					Ul(Attrs{
						"class": "space-y-2 text-sm",
					},
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("JavaScript")),
						),
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("Go")),
						),
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("CSS")),
						),
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("WebAssembly")),
						),
					),
				),
				// Footer Column 3
				Div(nil,
					H6(Attrs{
						"class": "font-semibold mb-4",
					}, Text("Resources")),
					Ul(Attrs{
						"class": "space-y-2 text-sm",
					},
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("Tutorials")),
						),
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("Documentation")),
						),
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("GitHub")),
						),
						Li(nil,
							A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, Text("Community")),
						),
					),
				),
				// Footer Column 4
				Div(nil,
					H6(Attrs{
						"class": "font-semibold mb-4",
					}, Text("Connect")),
					Div(Attrs{
						"class": "flex space-x-4",
					},
						A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "Twitter",
						}, Text("🐦")),
						A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "GitHub",
						}, Text("🐙")),
						A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "LinkedIn",
						}, Text("💼")),
						A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "Discord",
						}, Text("💬")),
					),
				),
			),
			Hr(Attrs{
				"class": "my-8 border-gray-700",
			}),
			Div(Attrs{
				"class": "flex flex-col md:flex-row justify-between items-center text-sm text-gray-400",
			},
				P(nil, Text("© 2024 TechBlog. All rights reserved.")),
				P(nil, Text("Built with ❤️ using Go + WebAssembly")),
			),
		),
	)
}

// BlogLandingPage creates a simple blog landing page composed of smaller reusable components
func BlogLandingPage() {
	fmt.Println("BlogLandingPage: Starting to render blog landing page")

	// Main Blog Landing Page Component - composed of smaller components
	blogLandingPage := func(props Attrs) *Element {
		// Main blog landing page structure using component composition
		return Div(Attrs{
			"class": "min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100",
		},
			// Header component
			HeaderComponent,

			// Main content area
			Main(nil,
				// Hero section component
				HeroSection,
				// Featured posts section component
				FeaturedPostsSection,
				// About section component
				AboutSection,
				// Newsletter section component
				NewsletterSection,
			),

			// Footer component
			FooterComponent,
		)
	}

	// Render the blog landing page
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("BlogLandingPage: Error - No element with id 'root' found in the DOM")
		return
	}

	fmt.Println("BlogLandingPage: Rendering blog landing page into the container")
	Render(CreateElement(blogLandingPage, nil), container)
}
