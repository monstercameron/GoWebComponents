package sanitize_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/sanitize"
)

// ExampleSanitize strips dangerous markup and URL schemes while keeping safe
// formatting and allowlisted links.
func ExampleSanitize() {
	parseDirty := `<p>Hello <strong>world</strong></p>` +
		`<a href="javascript:alert(1)">x</a>` +
		`<a href="https://example.test">ok</a>` +
		`<img src=x onerror=alert(1)>` +
		`<script>alert(1)</script>`
	fmt.Println(sanitize.Sanitize(parseDirty))
	// Output: <p>Hello <strong>world</strong></p><a>x</a><a href="https://example.test">ok</a><img src="x">
}
