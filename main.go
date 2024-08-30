package main

import (
	"fmt"
	"goHTML/vdom"
)

var (
	Tag        = vdom.Tag
	GenerateID = vdom.GenerateID
	Text       = vdom.Text
)

func main() {
	// Create a root element
	root := Tag("html", nil,
		Tag("head", nil,
			Tag("title", nil, Text("My VDOM Example")),
		),
		Tag("body", map[string]string{"class": "main-content"},
			Tag("header", nil,
				Tag("h1", nil, Text("Welcome to VDOM")),
				Text("This is a header text."),
			),
			Tag("nav", nil,
				Text("Navigation: "),
				Tag("ul", nil,
					Tag("li", nil, Tag("a", map[string]string{"href": "#"}, Text("Home"))),
					Tag("li", nil, Tag("a", map[string]string{"href": "#"}, Text("About"))),
					Tag("li", nil, Tag("a", map[string]string{"href": "#"}, Text("Contact"))),
				),
			),
			Tag("main", nil,
				Tag("p", nil,
					Text("This is a paragraph with "),
					Tag("strong", nil, Text("bold")),
					Text(" and "),
					Tag("em", nil, Text("italic")),
					Text(" text."),
				),
				Tag("div", map[string]string{"class": "info-box"},
					Tag("h2", nil, Text("Information")),
					Tag("p", nil, 
						Text("This is some information in a box. "),
						Tag("br", nil),
						Text("It spans multiple lines."),
					),
				),
				Text("This is some text directly inside the main element."),
			),
			Tag("footer", nil,
				Text("Copyright © 2023"),
				Tag("br", nil),
				Text("All rights reserved."),
			),
		),
	)

	// Render the HTML
	fmt.Println("Rendered HTML:")
	fmt.Print(root.Render(0))

	fmt.Println("\n\n\nState Example:")
	vdom.ExampleUsage4()
}