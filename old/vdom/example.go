package vdom

import (
	"strings"
)

func HomePage() string {
	doctype := "<!DOCTYPE html>"
	
	html := Tag("html", map[string]string{"lang": "en"},
		Tag("head", nil,
			Tag("meta", map[string]string{"charset": "UTF-8"}),
			Tag("meta", map[string]string{"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
			Tag("title", nil, Text("Google")),
			Tag("script", map[string]string{"src": "https://cdn.tailwindcss.com"}, Text("")),
		),
		Tag("body", map[string]string{"class": "min-h-screen bg-white flex flex-col"},
			// Header
			Tag("header", map[string]string{"class": "flex justify-end items-center p-4"},
				Tag("nav", map[string]string{"class": "flex space-x-4"},
					Tag("a", map[string]string{"href": "#", "class": "text-sm text-gray-700 hover:underline"}, Text("Gmail")),
					Tag("a", map[string]string{"href": "#", "class": "text-sm text-gray-700 hover:underline"}, Text("Images")),
					Tag("button", map[string]string{"class": "p-2 bg-gray-100 rounded-full hover:bg-gray-200"},
						Tag("svg", map[string]string{"class": "w-6 h-6 text-gray-600", "fill": "none", "stroke": "currentColor", "viewBox": "0 0 24 24", "xmlns": "http://www.w3.org/2000/svg"},
							Tag("path", map[string]string{"stroke-linecap": "round", "stroke-linejoin": "round", "stroke-width": "2", "d": "M4 6h16M4 12h16M4 18h16"}),
						),
					),
					Tag("button", map[string]string{"class": "ml-2 bg-blue-500 text-white px-4 py-2 rounded-md hover:bg-blue-600"}, Text("Sign in")),
				),
			),
			// Main content
			Tag("main", map[string]string{"class": "flex-grow flex flex-col items-center justify-center px-4"},
				Tag("img", map[string]string{"src": "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png", "alt": "Google", "class": "w-72 mb-8"}),
				Tag("div", map[string]string{"class": "w-full max-w-2xl"},
					Tag("div", map[string]string{"class": "flex items-center w-full border border-gray-200 rounded-full px-5 py-3 focus-within:shadow-lg"},
						Tag("svg", map[string]string{"class": "w-5 h-5 text-gray-500", "fill": "none", "stroke": "currentColor", "viewBox": "0 0 24 24", "xmlns": "http://www.w3.org/2000/svg"},
							Tag("path", map[string]string{"stroke-linecap": "round", "stroke-linejoin": "round", "stroke-width": "2", "d": "M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"}),
						),
						Tag("input", map[string]string{"type": "text", "class": "w-full outline-none ml-4", "placeholder": "Search Google or type a URL"}),
						Tag("svg", map[string]string{"class": "w-5 h-5 text-gray-500", "fill": "none", "stroke": "currentColor", "viewBox": "0 0 24 24", "xmlns": "http://www.w3.org/2000/svg"},
							Tag("path", map[string]string{"stroke-linecap": "round", "stroke-linejoin": "round", "stroke-width": "2", "d": "M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z"}),
						),
					),
					Tag("div", map[string]string{"class": "flex justify-center mt-8"},
						Tag("button", map[string]string{"class": "bg-gray-100 text-gray-800 px-4 py-2 rounded-md mr-4 hover:shadow"}, Text("Google Search")),
						Tag("button", map[string]string{"class": "bg-gray-100 text-gray-800 px-4 py-2 rounded-md hover:shadow"}, Text("I'm Feeling Lucky")),
					),
				),
			),
			// Footer
			Tag("footer", map[string]string{"class": "bg-gray-100 text-sm text-gray-600"},
				Tag("div", map[string]string{"class": "border-b border-gray-300 px-8 py-3"},
					Text("United States"),
				),
				Tag("div", map[string]string{"class": "px-8 py-3 flex flex-wrap justify-between"},
					Tag("div", map[string]string{"class": "flex space-x-6"},
						Tag("a", map[string]string{"href": "#", "class": "hover:underline"}, Text("About")),
						Tag("a", map[string]string{"href": "#", "class": "hover:underline"}, Text("Advertising")),
						Tag("a", map[string]string{"href": "#", "class": "hover:underline"}, Text("Business")),
						Tag("a", map[string]string{"href": "#", "class": "hover:underline"}, Text("How Search works")),
					),
					Tag("div", map[string]string{"class": "flex space-x-6"},
						Tag("a", map[string]string{"href": "#", "class": "hover:underline"}, Text("Privacy")),
						Tag("a", map[string]string{"href": "#", "class": "hover:underline"}, Text("Terms")),
						Tag("a", map[string]string{"href": "#", "class": "hover:underline"}, Text("Settings")),
					),
				),
			),
		),
	)

	// Combine doctype and html
	fullDocument := strings.TrimSpace(doctype + "\n" + html.Render(0))

	// fmt.Println("Complete Google Homepage Example:")
	// fmt.Println(fullDocument)
	return fullDocument
}