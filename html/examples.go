package html

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// OptionData represents the structure of our sample data, mimicking a JSON response
type OptionData struct {
	Value string
	Label string
}

// Sample data that mimics a JSON response
var OptionsData = []OptionData{
	{Value: "1", Label: "Option A"},
	{Value: "2", Label: "Option B"},
	{Value: "3", Label: "Option C"},
}

// Helper function to create navigation links
func createNavLinks(currentPage string) Node {
	pages := []struct {
		href, label string
	}{
		{"/", "Home"},
		{"/simple", "Simple Example"},
		{"/complex", "Complex Example"},
		{"/advanced", "Advanced Example"},
	}
	var navItems []Node
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
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	content := HTML("div", nil,
		createNavLinks("/"),
		HTML("h1", nil, Text("Welcome to the HTML Rendering Demo")),
		HTML("p", nil, Text("Click on the links above to see different examples of HTML rendering.")),
	)
	renderPage(w, content)
}

// Handler for the simple example page
func SimpleHandler(w http.ResponseWriter, r *http.Request) {
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
func ComplexHandler(optionsData []OptionData) http.HandlerFunc {
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

// Handler for the advanced example page
func AdvancedHandler(w http.ResponseWriter, r *http.Request) {
	// String interpolation in a text node
	currentTime := time.Now().Format("15:04:05")
	timeNode := Text(fmt.Sprintf("The current time is %s", currentTime))

	// String interpolation in an attribute
	randomNumber := rand.Intn(100)
	randomAttr := map[string]string{"data-random": fmt.Sprintf("random-%d", randomNumber)}

	// String interpolation in a CSS class using a ternary operator
	isEven := randomNumber%2 == 0
	evenOddClass := map[string]string{"class": fmt.Sprintf("number %s", func() string {
		if isEven {
			return "even"
		}
		return "odd"
	}())}

	// Using an anonymous function
	repeatText := func(text string, times int) Node {
		return HTML("p", nil, Text(fmt.Sprintf("%s", 
			func() string {
				result := ""
				for i := 0; i < times; i++ {
					result += text
				}
				return result
			}())))
	}

	content := HTML("div", nil,
		createNavLinks("/advanced"),
		HTML("h1", nil, Text("Advanced Example")),
		HTML("p", nil, timeNode),
		HTML("div", randomAttr, Text(fmt.Sprintf("This div has a random attribute value: %d", randomNumber))),
		HTML("p", evenOddClass, Text(fmt.Sprintf("The random number %d is %s", randomNumber, func() string {
			if isEven {
				return "even"
			}
			return "odd"
		}()))),
		repeatText("Repeat this! ", 3),
	)
	renderPage(w, content)
}

// Helper function to render the full HTML page
func renderPage(w http.ResponseWriter, content Node) {
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
				.even { color: blue; }
				.odd { color: red; }
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
func mapOptionsToHTML(options []OptionData) []Node {
	var optionNodes []Node
	for _, option := range options {
		optionNode := HTML("option", map[string]string{"value": option.Value}, Text(option.Label))
		optionNodes = append(optionNodes, optionNode)
	}
	return optionNodes
}