//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

// Type aliases for convenience
type Attrs = dom.Attrs
type Element = render.Element

// Simple test component to demonstrate component references
func TestComponent(props Attrs) *Element {
	fmt.Println("🧪 TestComponent: Rendering test component")
	return dom.Div(Attrs{
		"class": "test-component bg-red-100 p-4 rounded",
	}, dom.Text("This component was passed as a reference!"))
}

// Another test component with different styling
func AnotherTestComponent(props Attrs) *Element {
	fmt.Println("🔬 AnotherTestComponent: Rendering another test component")
	return dom.Div(Attrs{
		"class": "another-test bg-green-100 p-4 rounded",
	}, dom.Text("Another component reference!"))
}

// Header component - Navigation and branding
func HeaderComponent(props Attrs) *Element {
	fmt.Println("🏠 HeaderComponent: Rendering header with navigation")
	return dom.Header(Attrs{
		"class": "bg-white shadow-md sticky top-0 z-50",
	},
		dom.Nav(Attrs{
			"class": "container mx-auto px-6 py-4",
		},
			dom.Div(Attrs{
				"class": "flex items-center justify-between",
			},
				// Logo/Brand
				dom.Div(Attrs{
					"class": "flex items-center space-x-2",
				},
					dom.H1(Attrs{
						"class": "text-2xl font-bold text-indigo-600",
					}, dom.Text("📝 TechBlog")),
					dom.Span(Attrs{
						"class": "text-sm text-gray-500",
					}, dom.Text("v1.0")),
				),
				// Navigation Menu
				dom.Ul(Attrs{
					"class": "flex space-x-6",
				},
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#home",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, dom.Text("Home")),
					),
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#about",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, dom.Text("About")),
					),
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#posts",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, dom.Text("Posts")),
					),
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#contact",
							"class": "text-gray-700 hover:text-indigo-600 font-medium transition-colors",
						}, dom.Text("Contact")),
					),
				),
			),
		),
	)
}

// Hero section component - Main banner with call-to-action
func HeroSection(props Attrs) *Element {
	fmt.Println("🎯 HeroSection: Rendering hero banner with CTA buttons")

	return dom.Section(Attrs{
		"id":    "home",
		"class": "py-20 px-6",
	},
		dom.Div(Attrs{
			"class": "container mx-auto text-center",
		},
			dom.H2(Attrs{
				"class": "text-5xl font-extrabold text-gray-900 mb-6",
			}, dom.Text("Welcome to TechBlog")),
			dom.P(Attrs{
				"class": "text-xl text-gray-600 mb-8 max-w-2xl mx-auto leading-relaxed",
			}, dom.Text("Discover the latest trends in technology, programming, and web development. Join our community of passionate developers and tech enthusiasts.")),
			dom.Div(Attrs{
				"class": "flex justify-center space-x-4",
			},
				dom.Button(Attrs{
					"class": "bg-indigo-600 text-white px-8 py-3 rounded-lg font-semibold hover:bg-indigo-700 transition-colors shadow-lg",
				}, dom.Text("Start Reading")),
				dom.Button(Attrs{
					"class": "border-2 border-indigo-600 text-indigo-600 px-8 py-3 rounded-lg font-semibold hover:bg-indigo-50 transition-colors",
				}, dom.Text("Subscribe")),
			),
			// Demonstration of component references vs return values
			dom.Div(Attrs{
				"class": "mt-8 space-y-4",
			},
				dom.H3(Attrs{
					"class": "text-lg font-semibold text-gray-800",
				}, dom.Text("Component Reference Demo:")),
				// These are component references - call them with nil
				TestComponent(nil),
				AnotherTestComponent(nil),
				// This is also a component reference
				TestComponent(nil),
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

	return dom.Article(Attrs{
		"class": "bg-gray-50 rounded-xl p-6 hover:shadow-lg transition-shadow",
	},
		dom.Div(Attrs{
			"class": "flex items-center mb-4",
		},
			dom.Span(Attrs{
				"class": tagColor,
			}, dom.Text(tag)),
			dom.Time(Attrs{
				"class":    "text-gray-500 text-sm ml-auto",
				"datetime": datetime,
			}, dom.Text(date)),
		),
		dom.H4(Attrs{
			"class": "text-xl font-semibold mb-3 text-gray-900",
		}, dom.Text(title)),
		dom.P(Attrs{
			"class": "text-gray-600 mb-4 line-clamp-3",
		}, dom.Text(description)),
		dom.A(Attrs{
			"href":  "#",
			"class": "text-indigo-600 font-medium hover:text-indigo-800 transition-colors",
		}, dom.Text("Read More →")),
	)
}

// Featured posts section component
func FeaturedPostsSection(props Attrs) *Element {
	fmt.Println("📝 FeaturedPostsSection: Rendering 3 featured blog posts")
	return dom.Section(Attrs{
		"id":    "posts",
		"class": "py-16 bg-white",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6",
		},
			dom.H3(Attrs{
				"class": "text-3xl font-bold text-center mb-12 text-gray-900",
			}, dom.Text("Featured Posts")),
			dom.Div(Attrs{
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
	fmt.Println("ℹ️ AboutSection: Rendering company info and statistics")
	return dom.Section(Attrs{
		"id":    "about",
		"class": "py-16 bg-gray-50",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6",
		},
			dom.Div(Attrs{
				"class": "max-w-3xl mx-auto text-center",
			},
				dom.H3(Attrs{
					"class": "text-3xl font-bold mb-6 text-gray-900",
				}, dom.Text("About TechBlog")),
				dom.Blockquote(Attrs{
					"class": "text-lg text-gray-600 italic mb-6 border-l-4 border-indigo-500 pl-6",
				}, dom.Text("\"Technology is best when it brings people together.\" - Matt Mullenweg")),
				dom.P(Attrs{
					"class": "text-gray-600 mb-6 leading-relaxed",
				}, dom.Text("We're a community of developers, designers, and tech enthusiasts sharing knowledge and experiences. Our mission is to make technology accessible and understandable for everyone.")),
				dom.Div(Attrs{
					"class": "flex justify-center space-x-8 text-center",
				},
					dom.Div(nil,
						dom.Strong(Attrs{
							"class": "block text-2xl font-bold text-indigo-600",
						}, dom.Text("500+")),
						dom.Span(Attrs{
							"class": "text-gray-600",
						}, dom.Text("Articles")),
					),
					dom.Div(nil,
						dom.Strong(Attrs{
							"class": "block text-2xl font-bold text-indigo-600",
						}, dom.Text("10K+")),
						dom.Span(Attrs{
							"class": "text-gray-600",
						}, dom.Text("Readers")),
					),
					dom.Div(nil,
						dom.Strong(Attrs{
							"class": "block text-2xl font-bold text-indigo-600",
						}, dom.Text("50+")),
						dom.Span(Attrs{
							"class": "text-gray-600",
						}, dom.Text("Contributors")),
					),
				),
			),
		),
	)
}

// Newsletter section component - Email subscription form
func NewsletterSection(props Attrs) *Element {
	fmt.Println("📧 NewsletterSection: Initializing newsletter component")

	return dom.Section(Attrs{
		"class": "py-16 bg-indigo-600",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6 text-center",
		},
			dom.H3(Attrs{
				"class": "text-3xl font-bold text-white mb-4",
			}, dom.Text("Stay Updated")),
			dom.P(Attrs{
				"class": "text-indigo-100 mb-8 max-w-2xl mx-auto",
			}, dom.Text("Subscribe to our newsletter and get the latest tech articles delivered straight to your inbox every week.")),
			dom.Form(Attrs{
				"class": "flex justify-center max-w-md mx-auto",
			},
				dom.Input(Attrs{
					"type":        "email",
					"placeholder": "Enter your email",
					"class":       "flex-1 px-4 py-3 rounded-l-lg border-0 focus:ring-2 focus:ring-indigo-300 outline-none",
					"required":    true,
				}),
				dom.Button(Attrs{
					"type":  "submit",
					"class": "bg-white text-indigo-600 px-6 py-3 rounded-r-lg font-semibold hover:bg-gray-100 transition-colors",
				}, dom.Text("Subscribe")),
			),
		),
	)
}

// Footer component - Site links and information
func FooterComponent(props Attrs) *Element {
	fmt.Println("🦶 FooterComponent: Rendering footer with links and social icons")
	return dom.Footer(Attrs{
		"class": "bg-gray-900 text-white py-12",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6",
		},
			dom.Div(Attrs{
				"class": "grid md:grid-cols-4 gap-8",
			},
				// Footer Column 1
				dom.Div(nil,
					dom.H5(Attrs{
						"class": "font-bold mb-4",
					}, dom.Text("TechBlog")),
					dom.P(Attrs{
						"class": "text-gray-400 text-sm",
					}, dom.Text("Sharing knowledge and building the future of technology together.")),
				),
				// Footer Column 2
				dom.Div(nil,
					dom.H6(Attrs{
						"class": "font-semibold mb-4",
					}, dom.Text("Categories")),
					dom.Ul(Attrs{
						"class": "space-y-2 text-sm",
					},
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("JavaScript")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("Go")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("CSS")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("WebAssembly")),
						),
					),
				),
				// Footer Column 3
				dom.Div(nil,
					dom.H6(Attrs{
						"class": "font-semibold mb-4",
					}, dom.Text("Resources")),
					dom.Ul(Attrs{
						"class": "space-y-2 text-sm",
					},
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("Tutorials")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("Documentation")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("GitHub")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-400 hover:text-white transition-colors",
							}, dom.Text("Community")),
						),
					),
				),
				// Footer Column 4
				dom.Div(nil,
					dom.H6(Attrs{
						"class": "font-semibold mb-4",
					}, dom.Text("Connect")),
					dom.Div(Attrs{
						"class": "flex space-x-4",
					},
						dom.A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "Twitter",
						}, dom.Text("🐦")),
						dom.A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "GitHub",
						}, dom.Text("🐙")),
						dom.A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "LinkedIn",
						}, dom.Text("💼")),
						dom.A(Attrs{
							"href":  "#",
							"class": "text-gray-400 hover:text-white transition-colors",
							"title": "Discord",
						}, dom.Text("💬")),
					),
				),
			),
			dom.Hr(Attrs{
				"class": "my-8 border-gray-700",
			}),
			dom.Div(Attrs{
				"class": "flex flex-col md:flex-row justify-between items-center text-sm text-gray-400",
			},
				dom.P(nil, dom.Text("© 2024 TechBlog. All rights reserved.")),
				dom.P(nil, dom.Text("Built with ❤️ using Go + WebAssembly")),
			),
		),
	)
}

// BlogLandingPage creates a simple blog landing page composed of smaller reusable components
func BlogLandingPage() {
	fmt.Println("🚀 BlogLandingPage: Starting to render blog landing page")

	// Main Blog Landing Page Component - composed of smaller components
	blogLandingPage := func(props Attrs) *Element {
		fmt.Println("🏗️ BlogLandingPage: Constructing main page layout")

		// Main blog landing page structure using component composition
		return dom.Div(Attrs{
			"class": "min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100",
		},
			// Header component
			HeaderComponent(nil),

			// Main content area
			dom.Main(nil,
				// Hero section component
				HeroSection(nil),
				// Featured posts section component
				FeaturedPostsSection(nil),
				// About section component
				AboutSection(nil),
				// Newsletter section component
				NewsletterSection(nil),
			),

			// Footer component
			FooterComponent(nil),
		)
	}

	fmt.Println("🔍 BlogLandingPage: Rendering blog landing page to #app selector")

	// Render the blog landing page using the new render.To API
	render.To(blogLandingPage(nil), "#app")
	fmt.Println("🎉 BlogLandingPage: Render process initiated successfully!")
}
