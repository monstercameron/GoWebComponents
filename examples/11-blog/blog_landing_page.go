//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

// Type aliases
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
		"class": "bg-[#0a0a0a]/80 backdrop-blur-md border-b border-white/10 sticky top-0 z-50",
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
						"class": "text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
					}, dom.Text("📝 TechBlog")),
					dom.Span(Attrs{
						"class": "text-sm text-gray-500",
					}, dom.Text("v1.0")),
				),
				// Navigation Menu
				dom.Ul(Attrs{
					"class": "flex space-x-8",
				},
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#home",
							"class": "text-gray-300 hover:text-white font-medium transition-colors",
						}, dom.Text("Home")),
					),
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#about",
							"class": "text-gray-300 hover:text-white font-medium transition-colors",
						}, dom.Text("About")),
					),
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#posts",
							"class": "text-gray-300 hover:text-white font-medium transition-colors",
						}, dom.Text("Posts")),
					),
					dom.Li(nil,
						dom.A(Attrs{
							"href":  "#contact",
							"class": "text-gray-300 hover:text-white font-medium transition-colors",
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
		"class": "py-24 px-6 relative overflow-hidden",
	},
		// Background glow effect
		dom.Div(Attrs{
			"class": "absolute top-0 left-1/2 -translate-x-1/2 w-full h-full max-w-4xl bg-blue-500/10 blur-[100px] -z-10 rounded-full",
		}),

		dom.Div(Attrs{
			"class": "container mx-auto text-center relative z-10",
		},
			dom.H2(Attrs{
				"class": "text-6xl font-extrabold text-white mb-8 tracking-tight",
			}, dom.Text("Welcome to TechBlog")),
			dom.P(Attrs{
				"class": "text-xl text-gray-400 mb-10 max-w-2xl mx-auto leading-relaxed",
			}, dom.Text("Discover the latest trends in technology, programming, and web development. Join our community of passionate developers and tech enthusiasts.")),
			dom.Div(Attrs{
				"class": "flex justify-center space-x-6",
			},
				dom.Button(Attrs{
					"class": "bg-gradient-to-r from-blue-500 to-purple-600 text-white px-8 py-4 rounded-lg font-bold hover:opacity-90 transition-all shadow-lg shadow-purple-500/20",
				}, dom.Text("Start Reading")),
				dom.Button(Attrs{
					"class": "bg-white/5 border border-white/10 text-white px-8 py-4 rounded-lg font-bold hover:bg-white/10 transition-all",
				}, dom.Text("Subscribe")),
			),
			// Demonstration of component references vs return values
			dom.Div(Attrs{
				"class": "mt-16 space-y-4 opacity-50 hover:opacity-100 transition-opacity",
			},
				dom.H3(Attrs{
					"class": "text-sm font-semibold text-gray-500 uppercase tracking-widest",
				}, dom.Text("Component Reference Demo")),
				// These are component references - call them with nil
				TestComponent(nil),
				AnotherTestComponent(nil),
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
		"class": "bg-white/5 border border-white/10 rounded-xl p-8 hover:bg-white/10 transition-all duration-300 group",
	},
		dom.Div(Attrs{
			"class": "flex items-center mb-6",
		},
			dom.Span(Attrs{
				"class": tagColor + " px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider",
			}, dom.Text(tag)),
			dom.Time(Attrs{
				"class":    "text-gray-500 text-sm ml-auto font-mono",
				"datetime": datetime,
			}, dom.Text(date)),
		),
		dom.H4(Attrs{
			"class": "text-2xl font-bold mb-4 text-white group-hover:text-blue-400 transition-colors",
		}, dom.Text(title)),
		dom.P(Attrs{
			"class": "text-gray-400 mb-6 line-clamp-3 leading-relaxed",
		}, dom.Text(description)),
		dom.A(Attrs{
			"href":  "#",
			"class": "inline-flex items-center text-blue-400 font-semibold hover:text-blue-300 transition-colors",
		}, dom.Text("Read More →")),
	)
}

// Featured posts section component
func FeaturedPostsSection(props Attrs) *Element {
	fmt.Println("📝 FeaturedPostsSection: Rendering 3 featured blog posts")
	return dom.Section(Attrs{
		"id":    "posts",
		"class": "py-20 bg-black/20",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6",
		},
			dom.H3(Attrs{
				"class": "text-3xl font-bold text-center mb-16 text-white",
			}, dom.Text("Featured Posts")),
			dom.Div(Attrs{
				"class": "grid md:grid-cols-2 lg:grid-cols-3 gap-8",
			},
				BlogPostCard(Attrs{
					"title":       "Modern JavaScript Frameworks in 2024",
					"description": "Explore the latest JavaScript frameworks and libraries that are shaping web development. From React to Vue, discover what's trending.",
					"tag":         "JavaScript",
					"tagColor":    "bg-yellow-500/20 text-yellow-400 border border-yellow-500/30",
					"date":        "Jan 15, 2024",
					"datetime":    "2024-01-15",
				}),
				BlogPostCard(Attrs{
					"title":       "Building Web Components with Go and WebAssembly",
					"description": "Learn how to create reactive web components using Go compiled to WebAssembly. A new approach to frontend development.",
					"tag":         "Go",
					"tagColor":    "bg-blue-500/20 text-blue-400 border border-blue-500/30",
					"date":        "Jan 10, 2024",
					"datetime":    "2024-01-10",
				}),
				BlogPostCard(Attrs{
					"title":       "CSS Grid vs Flexbox: When to Use What",
					"description": "Master the art of CSS layout with this comprehensive guide comparing CSS Grid and Flexbox. Includes practical examples.",
					"tag":         "CSS",
					"tagColor":    "bg-purple-500/20 text-purple-400 border border-purple-500/30",
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
		"class": "py-20",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6",
		},
			dom.Div(Attrs{
				"class": "max-w-4xl mx-auto text-center",
			},
				dom.H3(Attrs{
					"class": "text-3xl font-bold mb-8 text-white",
				}, dom.Text("About TechBlog")),
				dom.Blockquote(Attrs{
					"class": "text-2xl text-gray-300 italic mb-10 border-l-4 border-blue-500 pl-8 py-2",
				}, dom.Text("\"Technology is best when it brings people together.\" - Matt Mullenweg")),
				dom.P(Attrs{
					"class": "text-gray-400 mb-12 leading-relaxed text-lg",
				}, dom.Text("We're a community of developers, designers, and tech enthusiasts sharing knowledge and experiences. Our mission is to make technology accessible and understandable for everyone.")),
				dom.Div(Attrs{
					"class": "grid grid-cols-3 gap-8 text-center",
				},
					dom.Div(Attrs{"class": "p-6 bg-white/5 rounded-xl border border-white/5"},
						dom.Strong(Attrs{
							"class": "block text-4xl font-bold text-blue-400 mb-2",
						}, dom.Text("500+")),
						dom.Span(Attrs{
							"class": "text-gray-500 uppercase tracking-wider text-sm",
						}, dom.Text("Articles")),
					),
					dom.Div(Attrs{"class": "p-6 bg-white/5 rounded-xl border border-white/5"},
						dom.Strong(Attrs{
							"class": "block text-4xl font-bold text-purple-400 mb-2",
						}, dom.Text("10K+")),
						dom.Span(Attrs{
							"class": "text-gray-500 uppercase tracking-wider text-sm",
						}, dom.Text("Readers")),
					),
					dom.Div(Attrs{"class": "p-6 bg-white/5 rounded-xl border border-white/5"},
						dom.Strong(Attrs{
							"class": "block text-4xl font-bold text-pink-400 mb-2",
						}, dom.Text("50+")),
						dom.Span(Attrs{
							"class": "text-gray-500 uppercase tracking-wider text-sm",
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
		"class": "py-20 bg-gradient-to-r from-blue-900/20 to-purple-900/20 border-y border-white/5",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6 text-center",
		},
			dom.H3(Attrs{
				"class": "text-3xl font-bold text-white mb-4",
			}, dom.Text("Stay Updated")),
			dom.P(Attrs{
				"class": "text-gray-400 mb-10 max-w-2xl mx-auto",
			}, dom.Text("Subscribe to our newsletter and get the latest tech articles delivered straight to your inbox every week.")),
			dom.Form(Attrs{
				"class": "flex justify-center max-w-md mx-auto",
			},
				dom.Input(Attrs{
					"type":        "email",
					"placeholder": "Enter your email",
					"class":       "flex-1 px-6 py-4 rounded-l-lg bg-black/40 border border-white/10 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none text-white placeholder-gray-500",
					"required":    true,
				}),
				dom.Button(Attrs{
					"type":  "submit",
					"class": "bg-blue-600 text-white px-8 py-4 rounded-r-lg font-bold hover:bg-blue-700 transition-colors",
				}, dom.Text("Subscribe")),
			),
		),
	)
}

// Footer component - Site links and information
func FooterComponent(props Attrs) *Element {
	fmt.Println("🦶 FooterComponent: Rendering footer with links and social icons")
	return dom.Footer(Attrs{
		"class": "bg-black/40 text-white py-16 border-t border-white/5",
	},
		dom.Div(Attrs{
			"class": "container mx-auto px-6",
		},
			dom.Div(Attrs{
				"class": "grid md:grid-cols-4 gap-12",
			},
				// Footer Column 1
				dom.Div(nil,
					dom.H5(Attrs{
						"class": "font-bold text-xl mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
					}, dom.Text("TechBlog")),
					dom.P(Attrs{
						"class": "text-gray-400 text-sm leading-relaxed",
					}, dom.Text("Sharing knowledge and building the future of technology together.")),
				),
				// Footer Column 2
				dom.Div(nil,
					dom.H6(Attrs{
						"class": "font-semibold mb-6 text-gray-200",
					}, dom.Text("Categories")),
					dom.Ul(Attrs{
						"class": "space-y-3 text-sm",
					},
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("JavaScript")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("Go")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("CSS")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("WebAssembly")),
						),
					),
				),
				// Footer Column 3
				dom.Div(nil,
					dom.H6(Attrs{
						"class": "font-semibold mb-6 text-gray-200",
					}, dom.Text("Resources")),
					dom.Ul(Attrs{
						"class": "space-y-3 text-sm",
					},
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("Tutorials")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("Documentation")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("GitHub")),
						),
						dom.Li(nil,
							dom.A(Attrs{
								"href":  "#",
								"class": "text-gray-500 hover:text-blue-400 transition-colors",
							}, dom.Text("Community")),
						),
					),
				),
				// Footer Column 4
				dom.Div(nil,
					dom.H6(Attrs{
						"class": "font-semibold mb-6 text-gray-200",
					}, dom.Text("Connect")),
					dom.Div(Attrs{
						"class": "flex space-x-4",
					},
						dom.A(Attrs{
							"href":  "#",
							"class": "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-blue-500 hover:text-white transition-all",
							"title": "Twitter",
						}, dom.Text("🐦")),
						dom.A(Attrs{
							"href":  "#",
							"class": "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-gray-700 hover:text-white transition-all",
							"title": "GitHub",
						}, dom.Text("🐙")),
						dom.A(Attrs{
							"href":  "#",
							"class": "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-blue-700 hover:text-white transition-all",
							"title": "LinkedIn",
						}, dom.Text("💼")),
						dom.A(Attrs{
							"href":  "#",
							"class": "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-indigo-500 hover:text-white transition-all",
							"title": "Discord",
						}, dom.Text("💬")),
					),
				),
			),
			dom.Hr(Attrs{
				"class": "my-12 border-white/5",
			}),
			dom.Div(Attrs{
				"class": "flex flex-col md:flex-row justify-between items-center text-sm text-gray-500",
			},
				dom.P(nil, dom.Text("© 2024 TechBlog. All rights reserved.")),
				dom.P(nil, dom.Text("Built with ❤️ using Go + WebAssembly")),
			),
		),
	)
}

// BlogLandingPage creates a simple blog landing page composed of smaller reusable components
func BlogLandingPage(_ Attrs) *Element {
	fmt.Println("🏗️ BlogLandingPage: Constructing main page layout")

	// Main blog landing page structure using component composition
	return dom.Div(Attrs{
		"class": "min-h-screen bg-[#0a0a0a] text-white font-sans selection:bg-blue-500/30",
	},
		// Header component
		dom.CreateElement(HeaderComponent, nil),

		// Main content area
		dom.Main(nil,
			// Hero section component
			dom.CreateElement(HeroSection, nil),
			// Featured posts section component
			dom.CreateElement(FeaturedPostsSection, nil),
			// About section component
			dom.CreateElement(AboutSection, nil),
			// Newsletter section component
			dom.CreateElement(NewsletterSection, nil),
		),

		// Footer component
		dom.CreateElement(FooterComponent, nil),
	)
}
