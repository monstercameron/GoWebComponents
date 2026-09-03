//go:build playwrightgo

package playwrightgoexamples_test

import (
	"runtime"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// TestRouterNavigationBrowserE2E verifies hash-router navigation in a real
// browser: link clicks change the rendered route, the location hash tracks,
// the catch-all handles unknown hashes, and the back button restores routes.
func TestRouterNavigationBrowserE2E(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18103")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(getBaseURL+"/examples/public/hash-router/", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
		}); parseErr != nil {
			parseT.Fatalf("goto hash-router: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Hash router basics"); parseErr != nil {
			parseT.Fatalf("home route never rendered: %v", parseErr)
		}

		if parseErr := parsePage.Click(`a[href="#/docs"]`); parseErr != nil {
			parseT.Fatalf("click docs: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Docs route"); parseErr != nil {
			parseT.Fatalf("docs route never rendered after click: %v", parseErr)
		}
		parseHash, _ := parsePage.Evaluate("() => window.location.hash")
		if parseHash != "#/docs" {
			parseT.Fatalf("hash did not track navigation: %v", parseHash)
		}

		if parseErr := parsePage.Click(`a[href="#/pricing"]`); parseErr != nil {
			parseT.Fatalf("click pricing: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Pricing route"); parseErr != nil {
			parseT.Fatalf("pricing route never rendered: %v", parseErr)
		}

		if _, parseErr := parsePage.Evaluate(`() => { window.location.hash = "#/nope" }`); parseErr != nil {
			parseT.Fatalf("set unknown hash: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Hash 404"); parseErr != nil {
			parseT.Fatalf("catch-all route never rendered: %v", parseErr)
		}

		if _, parseErr := parsePage.GoBack(); parseErr != nil {
			parseT.Fatalf("go back: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Pricing route"); parseErr != nil {
			parseT.Fatalf("back button did not restore pricing route: %v", parseErr)
		}
		if _, parseErr := parsePage.GoBack(); parseErr != nil {
			parseT.Fatalf("go back 2: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Docs route"); parseErr != nil {
			parseT.Fatalf("back button did not restore docs route: %v", parseErr)
		}
	})
}

// TestAccessibleOverlayBrowserE2E verifies the modal overlay primitive in a
// real browser: portal rendering, initial focus, Escape dismissal, focus
// restoration to the trigger, and the confirm action round-trip.
func TestAccessibleOverlayBrowserE2E(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18104")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(getBaseURL+"/examples/public/accessible-overlay/", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
		}); parseErr != nil {
			parseT.Fatalf("goto overlay example: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#open-accessible-overlay"); parseErr != nil {
			parseT.Fatalf("trigger never rendered: %v", parseErr)
		}

		if parseErr := parsePage.Click("#open-accessible-overlay"); parseErr != nil {
			parseT.Fatalf("open overlay: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#confirm-accessible-dialog"); parseErr != nil {
			parseT.Fatalf("dialog never appeared: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.activeElement && document.activeElement.id === "confirm-accessible-dialog"`, nil); parseErr != nil {
			parseT.Fatalf("initial focus did not land on confirm button: %v", parseErr)
		}

		if parseErr := parsePage.Keyboard().Press("Escape"); parseErr != nil {
			parseT.Fatalf("press escape: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => !document.querySelector("#confirm-accessible-dialog")`, nil); parseErr != nil {
			parseT.Fatalf("escape did not close dialog: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.activeElement && document.activeElement.id === "open-accessible-overlay"`, nil); parseErr != nil {
			parseT.Fatalf("focus was not restored to trigger after escape: %v", parseErr)
		}

		if parseErr := parsePage.Click("#open-accessible-overlay"); parseErr != nil {
			parseT.Fatalf("reopen overlay: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#confirm-accessible-dialog"); parseErr != nil {
			parseT.Fatalf("dialog never reappeared: %v", parseErr)
		}
		if parseErr := parsePage.Click("#confirm-accessible-dialog"); parseErr != nil {
			parseT.Fatalf("confirm: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => !document.querySelector("#confirm-accessible-dialog")`, nil); parseErr != nil {
			parseT.Fatalf("confirm did not close dialog: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.activeElement && document.activeElement.id === "open-accessible-overlay"`, nil); parseErr != nil {
			parseT.Fatalf("focus not restored after confirm: %v", parseErr)
		}
	})
}

// TestControlledFormBrowserE2E verifies controlled inputs in a real browser:
// typing updates state, state reflects back into input values, and resetting
// state to the zero value actually clears the inputs.
func TestControlledFormBrowserE2E(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18105")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(getBaseURL+"/examples/public/form/", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
		}); parseErr != nil {
			parseT.Fatalf("goto form example: %v", parseErr)
		}
		parseNameSel := `input[placeholder="Enter name"]`
		if _, parseErr := parsePage.WaitForSelector(parseNameSel); parseErr != nil {
			parseT.Fatalf("form never rendered: %v", parseErr)
		}

		if parseErr := parsePage.Fill(parseNameSel, "Cam"); parseErr != nil {
			parseT.Fatalf("fill name: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelector('input[placeholder="Enter name"]').value === "Cam"`, nil); parseErr != nil {
			parseT.Fatalf("controlled input did not hold typed value: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.body.textContent.includes("Cam")`, nil); parseErr != nil {
			parseT.Fatalf("live state snapshot never showed typed name: %v", parseErr)
		}

		if parseErr := parsePage.Click("text=Reset"); parseErr != nil {
			parseT.Fatalf("click reset: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelector('input[placeholder="Enter name"]').value === ""`, nil); parseErr != nil {
			parseT.Fatalf("reset did not clear controlled input: %v", parseErr)
		}
	})
}
