// package html

// import (
// 	"fmt"
// 	"math/rand"
// 	"time"
// )

// func CombinedExampleMarkup() string {
// 	// Dynamic content generation
// 	currentTime := time.Now().Format("15:04:05")
// 	randomNumber := rand.Intn(100)
// 	isEven := randomNumber%2 == 0

// 	// Create the content
// 	content := HTML("div", nil,
// 		HTML("h1", nil, Text("HTML Rendering Examples")),

// 		// Simple example
// 		HTML("h2", nil, Text("Simple Example")),
// 		HTML("div", map[string]string{"class": "container"},
// 			Text("Hello, "),
// 			HTML("strong", nil, Text("world")),
// 			Text("!"),
// 		),

// 		// Complex example
// 		HTML("h2", nil, Text("Complex Example")),
// 		HTML("select", map[string]string{"name": "options", "id": "selectElement"},
// 			HTML("option", map[string]string{"value": "1"}, Text("Option 1")),
// 			HTML("option", map[string]string{"value": "2"}, Text("Option 2")),
// 			HTML("option", map[string]string{"value": "3"}, Text("Option 3")),
// 		),
// 		HTML("h3", nil, Text("Dynamic Select")),
// 		HTML("select", map[string]string{"name": "dynamicOptions", "id": "dynamicSelect"},
// 			mapOptionsToHTML(OptionsData)...,
// 		),

// 		// Advanced examples
// 		HTML("h2", nil, Text("Advanced Examples")),
// 		HTML("p", nil, Text(fmt.Sprintf("The current time is %s", currentTime))),
// 		HTML("div", map[string]string{"data-random": fmt.Sprintf("random-%d", randomNumber)},
// 			Text(fmt.Sprintf("This div has a random attribute value: %d", randomNumber)),
// 		),
// 		HTML("p", map[string]string{"class": fmt.Sprintf("number %s", func() string {
// 			if isEven {
// 				return "even"
// 			}
// 			return "odd"
// 		}())},
// 			Text(fmt.Sprintf("The random number %d is %s", randomNumber, func() string {
// 				if isEven {
// 					return "even"
// 				}
// 				return "odd"
// 			}())),
// 		),
// 		HTML("p", nil, Text(func() string {
// 			result := ""
// 			for i := 0; i < 3; i++ {
// 				result += "Repeat this! "
// 			}
// 			return result
// 		}())),
// 	)

// 	// Create the full page
// 	page := HTML("html", nil,
// 		HTML("head", nil,
// 			HTML("title", nil, Text("HTML Rendering Demo")),
// 			HTML("style", nil, Text(`
// 				body { font-family: Arial, sans-serif; line-height: 1.6; padding: 20px; }
// 				.container { border: 1px solid #ddd; padding: 10px; margin: 10px 0; }
// 				select { margin: 10px 0; }
// 				.even { color: blue; }
// 				.odd { color: red; }
// 			`)),
// 		),
// 		HTML("body", nil, content),
// 	)

// 	// Render the page
// 	renderedHTML, err := page.Render()
// 	if err != nil {
// 		return fmt.Sprintf("Error rendering page: %v", err)
// 	}
// 	return renderedHTML
// }

package html

