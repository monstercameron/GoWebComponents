//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type BlogPostCardProps struct {
	Title       string
	Description string
	Tag         string
	TagColor    string
	Date        string
	DateTime    string
}

func TestComponent() ui.Node {
	fmt.Println("TestComponent: rendering test component")
	return html.Div(html.Props{
		Class: "test-component bg-red-100 p-4 rounded",
	}, html.Text("This component was passed as a reference!"))
}

func AnotherTestComponent() ui.Node {
	fmt.Println("AnotherTestComponent: rendering another test component")
	return html.Div(html.Props{
		Class: "another-test bg-green-100 p-4 rounded",
	}, html.Text("Another component reference!"))
}

func HeaderComponent() ui.Node {
	fmt.Println("HeaderComponent: rendering header with navigation")
	return html.Header(html.Props{
		Class: "bg-[#0a0a0a]/80 backdrop-blur-md border-b border-white/10 sticky top-0 z-50",
	},
		html.Nav(html.Props{
			Class: "container mx-auto px-6 py-4",
		},
			html.Div(html.Props{
				Class: "flex items-center justify-between",
			},
				html.Div(html.Props{
					Class: "flex items-center space-x-2",
				},
					html.H1(html.Props{
						Class: "text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
					}, html.Text("TechBlog")),
					html.Span(html.Props{
						Class: "text-sm text-gray-500",
					}, html.Text("v1.0")),
				),
				html.Ul(html.Props{
					Class: "flex space-x-8",
				},
					html.Li(html.Props{},
						html.A(html.Props{
							Href:  "#home",
							Class: "text-gray-300 hover:text-white font-medium transition-colors",
						}, html.Text("Home")),
					),
					html.Li(html.Props{},
						html.A(html.Props{
							Href:  "#about",
							Class: "text-gray-300 hover:text-white font-medium transition-colors",
						}, html.Text("About")),
					),
					html.Li(html.Props{},
						html.A(html.Props{
							Href:  "#posts",
							Class: "text-gray-300 hover:text-white font-medium transition-colors",
						}, html.Text("Posts")),
					),
					html.Li(html.Props{},
						html.A(html.Props{
							Href:  "#contact",
							Class: "text-gray-300 hover:text-white font-medium transition-colors",
						}, html.Text("Contact")),
					),
				),
			),
		),
	)
}

func HeroSection() ui.Node {
	fmt.Println("HeroSection: rendering hero banner with CTA buttons")

	return html.Section(html.Props{
		ID:    "home",
		Class: "py-24 px-6 relative overflow-hidden",
	},
		html.Div(html.Props{
			Class: "absolute top-0 left-1/2 -translate-x-1/2 w-full h-full max-w-4xl bg-blue-500/10 blur-[100px] -z-10 rounded-full",
		}),
		html.Div(html.Props{
			Class: "container mx-auto text-center relative z-10",
		},
			html.H2(html.Props{
				Class: "text-6xl font-extrabold text-white mb-8 tracking-tight",
			}, html.Text("Welcome to TechBlog")),
			html.P(html.Props{
				Class: "text-xl text-gray-400 mb-10 max-w-2xl mx-auto leading-relaxed",
			}, html.Text("Discover the latest trends in technology, programming, and web development. Join our community of passionate developers and tech enthusiasts.")),
			html.Div(html.Props{
				Class: "flex justify-center space-x-6",
			},
				html.Button(html.Props{
					Class: "bg-gradient-to-r from-blue-500 to-purple-600 text-white px-8 py-4 rounded-lg font-bold hover:opacity-90 transition-all shadow-lg shadow-purple-500/20",
				}, html.Text("Start Reading")),
				html.Button(html.Props{
					Class: "bg-white/5 border border-white/10 text-white px-8 py-4 rounded-lg font-bold hover:bg-white/10 transition-all",
				}, html.Text("Subscribe")),
			),
			html.Div(html.Props{
				Class: "mt-16 space-y-4 opacity-50 hover:opacity-100 transition-opacity",
			},
				html.H3(html.Props{
					Class: "text-sm font-semibold text-gray-500 uppercase tracking-widest",
				}, html.Text("Component Reference Demo")),
				ui.CreateElement(TestComponent),
				ui.CreateElement(AnotherTestComponent),
			),
		),
	)
}

func BlogPostCard(props BlogPostCardProps) ui.Node {
	return html.Article(html.Props{
		Class: "bg-white/5 border border-white/10 rounded-xl p-8 hover:bg-white/10 transition-all duration-300 group",
	},
		html.Div(html.Props{
			Class: "flex items-center mb-6",
		},
			html.Span(html.Props{
				Class: props.TagColor + " px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider",
			}, html.Text(props.Tag)),
			html.Time(html.Props{
				Class: "text-gray-500 text-sm ml-auto font-mono",
				Raw:   map[string]interface{}{"datetime": props.DateTime},
			}, html.Text(props.Date)),
		),
		html.H4(html.Props{
			Class: "text-2xl font-bold mb-4 text-white group-hover:text-blue-400 transition-colors",
		}, html.Text(props.Title)),
		html.P(html.Props{
			Class: "text-gray-400 mb-6 line-clamp-3 leading-relaxed",
		}, html.Text(props.Description)),
		html.A(html.Props{
			Href:  "#",
			Class: "inline-flex items-center text-blue-400 font-semibold hover:text-blue-300 transition-colors",
		}, html.Text("Read More ->")),
	)
}

func FeaturedPostsSection() ui.Node {
	fmt.Println("FeaturedPostsSection: rendering featured blog posts")
	return html.Section(html.Props{
		ID:    "posts",
		Class: "py-20 bg-black/20",
	},
		html.Div(html.Props{
			Class: "container mx-auto px-6",
		},
			html.H3(html.Props{
				Class: "text-3xl font-bold text-center mb-16 text-white",
			}, html.Text("Featured Posts")),
			html.Div(html.Props{
				Class: "grid md:grid-cols-2 lg:grid-cols-3 gap-8",
			},
				ui.CreateElement(BlogPostCard, BlogPostCardProps{
					Title:       "Modern JavaScript Frameworks in 2024",
					Description: "Explore the latest JavaScript frameworks and libraries that are shaping web development. From React to Vue, discover what is trending.",
					Tag:         "JavaScript",
					TagColor:    "bg-yellow-500/20 text-yellow-400 border border-yellow-500/30",
					Date:        "Jan 15, 2024",
					DateTime:    "2024-01-15",
				}),
				ui.CreateElement(BlogPostCard, BlogPostCardProps{
					Title:       "Building Web Components with Go and WebAssembly",
					Description: "Learn how to create reactive web components using Go compiled to WebAssembly. A new approach to frontend development.",
					Tag:         "Go",
					TagColor:    "bg-blue-500/20 text-blue-400 border border-blue-500/30",
					Date:        "Jan 10, 2024",
					DateTime:    "2024-01-10",
				}),
				ui.CreateElement(BlogPostCard, BlogPostCardProps{
					Title:       "CSS Grid vs Flexbox: When to Use What",
					Description: "Master the art of CSS layout with this practical guide comparing CSS Grid and Flexbox.",
					Tag:         "CSS",
					TagColor:    "bg-purple-500/20 text-purple-400 border border-purple-500/30",
					Date:        "Jan 5, 2024",
					DateTime:    "2024-01-05",
				}),
			),
		),
	)
}

func AboutSection() ui.Node {
	fmt.Println("AboutSection: rendering company info and statistics")
	return html.Section(html.Props{
		ID:    "about",
		Class: "py-20",
	},
		html.Div(html.Props{
			Class: "container mx-auto px-6",
		},
			html.Div(html.Props{
				Class: "max-w-4xl mx-auto text-center",
			},
				html.H3(html.Props{
					Class: "text-3xl font-bold mb-8 text-white",
				}, html.Text("About TechBlog")),
				html.Blockquote(html.Props{
					Class: "text-2xl text-gray-300 italic mb-10 border-l-4 border-blue-500 pl-8 py-2",
				}, html.Text("\"Technology is best when it brings people together.\" - Matt Mullenweg")),
				html.P(html.Props{
					Class: "text-gray-400 mb-12 leading-relaxed text-lg",
				}, html.Text("We are a community of developers, designers, and tech enthusiasts sharing knowledge and experience. Our mission is to make technology accessible and understandable for everyone.")),
				html.Div(html.Props{
					Class: "grid grid-cols-3 gap-8 text-center",
				},
					html.Div(html.Props{Class: "p-6 bg-white/5 rounded-xl border border-white/5"},
						html.Strong(html.Props{
							Class: "block text-4xl font-bold text-blue-400 mb-2",
						}, html.Text("500+")),
						html.Span(html.Props{
							Class: "text-gray-500 uppercase tracking-wider text-sm",
						}, html.Text("Articles")),
					),
					html.Div(html.Props{Class: "p-6 bg-white/5 rounded-xl border border-white/5"},
						html.Strong(html.Props{
							Class: "block text-4xl font-bold text-purple-400 mb-2",
						}, html.Text("10K+")),
						html.Span(html.Props{
							Class: "text-gray-500 uppercase tracking-wider text-sm",
						}, html.Text("Readers")),
					),
					html.Div(html.Props{Class: "p-6 bg-white/5 rounded-xl border border-white/5"},
						html.Strong(html.Props{
							Class: "block text-4xl font-bold text-pink-400 mb-2",
						}, html.Text("50+")),
						html.Span(html.Props{
							Class: "text-gray-500 uppercase tracking-wider text-sm",
						}, html.Text("Contributors")),
					),
				),
			),
		),
	)
}

func NewsletterSection() ui.Node {
	fmt.Println("NewsletterSection: initializing newsletter component")

	return html.Section(html.Props{
		ID:    "contact",
		Class: "py-20 bg-gradient-to-r from-blue-900/20 to-purple-900/20 border-y border-white/5",
	},
		html.Div(html.Props{
			Class: "container mx-auto px-6 text-center",
		},
			html.H3(html.Props{
				Class: "text-3xl font-bold text-white mb-4",
			}, html.Text("Stay Updated")),
			html.P(html.Props{
				Class: "text-gray-400 mb-10 max-w-2xl mx-auto",
			}, html.Text("Subscribe to our newsletter and get the latest tech articles delivered straight to your inbox every week.")),
			html.Form(html.Props{
				Class: "flex justify-center max-w-md mx-auto",
			},
				html.Input(html.Props{
					Type:        "email",
					Placeholder: "Enter your email",
					Class:       "flex-1 px-6 py-4 rounded-l-lg bg-black/40 border border-white/10 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none text-white placeholder-gray-500",
					Required:    true,
				}),
				html.Button(html.Props{
					Type:  "submit",
					Class: "bg-blue-600 text-white px-8 py-4 rounded-r-lg font-bold hover:bg-blue-700 transition-colors",
				}, html.Text("Subscribe")),
			),
		),
	)
}

func FooterComponent() ui.Node {
	fmt.Println("FooterComponent: rendering footer with links and social icons")
	return html.Footer(html.Props{
		Class: "bg-black/40 text-white py-16 border-t border-white/5",
	},
		html.Div(html.Props{
			Class: "container mx-auto px-6",
		},
			html.Div(html.Props{
				Class: "grid md:grid-cols-4 gap-12",
			},
				html.Div(html.Props{},
					html.H5(html.Props{
						Class: "font-bold text-xl mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
					}, html.Text("TechBlog")),
					html.P(html.Props{
						Class: "text-gray-400 text-sm leading-relaxed",
					}, html.Text("Sharing knowledge and building the future of technology together.")),
				),
				html.Div(html.Props{},
					html.H6(html.Props{
						Class: "font-semibold mb-6 text-gray-200",
					}, html.Text("Categories")),
					html.Ul(html.Props{
						Class: "space-y-3 text-sm",
					},
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("JavaScript"))),
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("Go"))),
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("CSS"))),
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("WebAssembly"))),
					),
				),
				html.Div(html.Props{},
					html.H6(html.Props{
						Class: "font-semibold mb-6 text-gray-200",
					}, html.Text("Resources")),
					html.Ul(html.Props{
						Class: "space-y-3 text-sm",
					},
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("Tutorials"))),
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("Documentation"))),
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("GitHub"))),
						html.Li(html.Props{}, html.A(html.Props{Href: "#", Class: "text-gray-500 hover:text-blue-400 transition-colors"}, html.Text("Community"))),
					),
				),
				html.Div(html.Props{},
					html.H6(html.Props{
						Class: "font-semibold mb-6 text-gray-200",
					}, html.Text("Connect")),
					html.Div(html.Props{
						Class: "flex space-x-4",
					},
						html.A(html.Props{Href: "#", Class: "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-blue-500 hover:text-white transition-all", Title: "Twitter"}, html.Text("TW")),
						html.A(html.Props{Href: "#", Class: "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-gray-700 hover:text-white transition-all", Title: "GitHub"}, html.Text("GH")),
						html.A(html.Props{Href: "#", Class: "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-blue-700 hover:text-white transition-all", Title: "LinkedIn"}, html.Text("IN")),
						html.A(html.Props{Href: "#", Class: "w-10 h-10 rounded-full bg-white/5 flex items-center justify-center text-gray-400 hover:bg-indigo-500 hover:text-white transition-all", Title: "Discord"}, html.Text("DS")),
					),
				),
			),
			html.Hr(html.Props{
				Class: "my-12 border-white/5",
			}),
			html.Div(html.Props{
				Class: "flex flex-col md:flex-row justify-between items-center text-sm text-gray-500",
			},
				html.P(html.Props{}, html.Text("(c) 2024 TechBlog. All rights reserved.")),
				html.P(html.Props{}, html.Text("Built with Go + WebAssembly")),
			),
		),
	)
}

func BlogLandingPage() ui.Node {
	fmt.Println("BlogLandingPage: constructing main page layout")

	return html.Div(html.Props{
		Class: "min-h-screen bg-[#0a0a0a] text-white font-sans selection:bg-blue-500/30",
	},
		ui.CreateElement(HeaderComponent),
		html.Main(html.Props{},
			ui.CreateElement(HeroSection),
			ui.CreateElement(FeaturedPostsSection),
			ui.CreateElement(AboutSection),
			ui.CreateElement(NewsletterSection),
		),
		ui.CreateElement(FooterComponent),
	)
}
