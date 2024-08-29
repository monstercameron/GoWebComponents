package main

import (
	"fmt"
	html "goHTML/html" // Import the custom HTML package with alias 'html'
	"net/http"
)

// Alias the functions from the html package for more concise usage
var (
	HTML = html.HTML
	Text = html.Text
)

// OptionData represents the structure of our sample data, mimicking a JSON response
type OptionData struct {
	Value string
	Label string
}

// Main function for the web server
func main() {
	// Sample data that mimics a JSON response
	optionsData := []OptionData{
		{Value: "1", Label: "Option A"},
		{Value: "2", Label: "Option B"},
		{Value: "3", Label: "Option C"},
	}

	// Set up the HTTP handler functions for different routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/simple", simpleHandler)
	http.HandleFunc("/complex", complexHandler(optionsData))

	// Start the HTTP server on port 8080
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Helper function to create navigation links
func createNavLinks(currentPage string) html.Node {
	pages := []struct {
		href, label string
	}{
		{"/", "Home"},
		{"/simple", "Simple Example"},
		{"/complex", "Complex Example"},
	}

	var navItems []html.Node
	for _, page := range pages {
		attrs := map[string]string{"href": page.href}
		if page.href == currentPage {
			attrs["class"] = "active"
		}
		navItems = append(navItems, HTML("li", nil, HTML("a", attrs, Text(page.label))))
	}

	return HTML("nav", nil,
		HTML("ul", nil, navItems...),
	)
}

// Handler for the home page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	content := HTML("div", nil,
		createNavLinks("/"),
		HTML("h1", nil, Text("Welcome to the HTML Rendering Demo")),
		HTML("p", nil, Text("Click on the links above to see different examples of HTML rendering.")),
	)

	renderPage(w, content)
}

// Handler for the simple example page
func simpleHandler(w http.ResponseWriter, r *http.Request) {
	content := HTML("div", nil,
		createNavLinks("/simple"),
		HTML("h1", nil, Text("Simple Example")),
		HTML("div", map[string]string{"class": "container"},
			Text("Hello, "),
			HTML("strong", nil, Text("world")),
			Text("!"),
		),
	)

	renderPage(w, content)
}

// Handler for the complex example page
func complexHandler(optionsData []OptionData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content := HTML("div", nil,
			createNavLinks("/complex"),
			HTML("h1", nil, Text("Complex Example")),
			HTML("select", map[string]string{"name": "options", "id": "selectElement"},
				HTML("option", map[string]string{"value": "1"}, Text("Option 1")),
				HTML("option", map[string]string{"value": "2"}, Text("Option 2")),
				HTML("option", map[string]string{"value": "3"}, Text("Option 3")),
			),
			HTML("h2", nil, Text("Dynamic Select")),
			HTML("select", map[string]string{"name": "dynamicOptions", "id": "dynamicSelect"},
				mapOptionsToHTML(optionsData)...,
			),
		)

		renderPage(w, content)
	}
}

// Helper function to render the full HTML page
func renderPage(w http.ResponseWriter, content html.Node) {
	page := HTML("html", nil,
		HTML("head", nil,
			HTML("title", nil, Text("HTML Rendering Demo")),
			HTML("style", nil, Text(`
				body { font-family: Arial, sans-serif; line-height: 1.6; padding: 20px; }
				nav ul { list-style-type: none; padding: 0; }
				nav ul li { display: inline; margin-right: 10px; }
				nav ul li a { text-decoration: none; color: #333; }
				nav ul li a.active { font-weight: bold; }
				select { margin: 10px 0; }
			`)),
		),
		HTML("body", nil, content),
	)

	renderedHTML, err := page.Render()
	if err != nil {
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s", renderedHTML)
}

// Function to map the OptionData slice to a slice of Node interface (HTML options)
func mapOptionsToHTML(options []OptionData) []html.Node {
	var optionNodes []html.Node
	for _, option := range options {
		optionNode := HTML("option", map[string]string{"value": option.Value}, Text(option.Label))
		optionNodes = append(optionNodes, optionNode)
	}
	return optionNodes
}