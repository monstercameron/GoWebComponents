package html_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
)

// ExampleSanitizeMarkdownHref shows the URL-scheme allowlist used by
// RenderMarkdown: safe schemes pass through, dangerous ones are dropped to "".
func ExampleSanitizeMarkdownHref() {
	fmt.Printf("%q\n", html.SanitizeMarkdownHref("https://example.test", nil))
	fmt.Printf("%q\n", html.SanitizeMarkdownHref("guide.md#section", nil))
	fmt.Printf("%q\n", html.SanitizeMarkdownHref("javascript:alert(1)", nil))
	// Output:
	// "https://example.test"
	// "guide.md#section"
	// ""
}
