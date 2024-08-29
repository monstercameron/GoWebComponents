// Go web server
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

	// Set up the HTTP handler function for the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Create a <div> element with mixed content: text and a <strong> element
		div := HTML("div", map[string]string{"class": "container"},
			Text("Hello, "),
			HTML("strong", nil,
				Text("world"),
			),
			Text("!"),
		)

		// More complex example: <select> with multiple <option> elements
		selectElement := HTML("select", map[string]string{"name": "options", "id": "selectElement"},
			HTML("option", map[string]string{"value": "1"}, Text("Option 1")),
			HTML("option", map[string]string{"value": "2"}, Text("Option 2")),
			HTML("option", map[string]string{"value": "3"}, Text("Option 3")),
		)

		// Create another <select> element dynamically from the optionsData
		dynamicSelect := HTML("select", map[string]string{"name": "dynamicOptions", "id": "dynamicSelect"},
			// Map over the optionsData slice to create <option> elements dynamically
			mapOptionsToHTML(optionsData)...,
		)

		// Render the HTML and write it to the HTTP response
		fmt.Fprintf(w, "%s\n%s\n%s", div.Render(), selectElement.Render(), dynamicSelect.Render())
	})

	// Start the HTTP server on port 8080
	http.ListenAndServe(":8080", nil)
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
