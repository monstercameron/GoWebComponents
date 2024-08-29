package main

import (
	"fmt"
	HTML "goHTML/html" // Import the new examples package
	"net/http"
)

// Main function for the web server
func main() {
	// Set up the HTTP handler functions using the examples package
	http.HandleFunc("/", HTML.HomeHandler)
	http.HandleFunc("/simple", HTML.SimpleHandler)
	http.HandleFunc("/complex", HTML.ComplexHandler(HTML.OptionsData))
	http.HandleFunc("/advanced", HTML.AdvancedHandler)

	// Start the HTTP server on port 8080
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
