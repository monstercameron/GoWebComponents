// Package dom provides HTML element constructors and DOM manipulation utilities
// for GoWebComponents.
//
// This package offers a type-safe way to create HTML elements programmatically
// in Go, with automatic conversion of children and optimized performance through
// element pooling and smart string handling.
//
// Basic usage:
//
//	import "github.com/monstercameron/GoWebComponents/dom"
//
//	// Create simple elements
//	element := dom.Div(nil,
//	    dom.H1(nil, "Hello World"),
//	    dom.P(nil, "Welcome to GoWebComponents"),
//	)
//
// The package includes constructors for all standard HTML5 elements, organized by category:
//   - Document Structure: Html, Head, Body, Title, Meta, Link, Style, Script
//   - Layout: Div, Span, P, Section, Article, Aside, Header, Footer, Main, Nav
//   - Headings: H1, H2, H3, H4, H5, H6
//   - Text: Strong, Em, Small, Mark, Code, Pre, Blockquote, Cite
//   - Lists: Ul, Ol, Li, Dl, Dt, Dd
//   - Links & Media: A, Img, Video, Audio, Source, Canvas, Svg
//   - Forms: Form, Input, Textarea, Button, Select, Option, Label, Fieldset
//   - Tables: Table, Thead, Tbody, Tfoot, Tr, Th, Td, Caption
//   - Interactive: Details, Summary, Dialog
//
// All element constructors accept attributes as a map and variadic children:
//
//	dom.Div(dom.Attrs{"class": "container", "id": "main"},
//	    dom.H1(nil, "Title"),
//	    dom.P(nil, "Content"),
//	)
//
// Helper functions are provided for common patterns:
//
//	dom.ClassProps("btn-primary")           // Creates map with class attribute
//	dom.IdProps("my-element")               // Creates map with id attribute
//	dom.HrefProps("/about")                 // Creates map with href attribute
//	dom.ClassIdProps("container", "main")   // Creates map with both class and id
package dom
