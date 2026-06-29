//go:build playwrightgo

package playwrightgoexamples_test

import (
	"runtime"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestPortalExamplesMountIntoSelfRenderedRoots verifies the portal-target
// defect class is fixed across every affected example: each overlay or portal
// must actually mount its content into its root when triggered, under the
// generated catalog shell (which provides only #app).
func TestPortalExamplesMountIntoSelfRenderedRoots(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18106")

	parseCases := []struct {
		Name        string
		Path        string
		ReadySel    string
		TriggerSel  string
		MountedExpr string
	}{
		{
			Name:        "overlay-stack",
			Path:        "/examples/public/overlay-stack/",
			ReadySel:    "#open-overlay-stack-dialog",
			TriggerSel:  "#open-overlay-stack-dialog",
			MountedExpr: `() => { const r = document.querySelector("#overlay-stack-root"); return !!(r && r.children.length > 0); }`,
		},
		{
			Name:        "overlay-anchor",
			Path:        "/examples/public/overlay-anchor/",
			ReadySel:    "#open-overlay-anchor-menu",
			TriggerSel:  "#open-overlay-anchor-menu",
			MountedExpr: `() => { const r = document.querySelector("#overlay-anchor-root"); return !!(r && r.children.length > 0); }`,
		},
		{
			Name:        "portals",
			Path:        "/examples/public/portals/",
			ReadySel:    "text=Open Modal",
			TriggerSel:  "text=Open Modal",
			MountedExpr: `() => { const r = document.querySelector("#portal-root"); return !!(r && r.children.length > 0); }`,
		},
	}

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		for _, parseCase := range parseCases {
			parseCase2 := parseCase
			parseT.Run(parseCase2.Name, func(parseT2 *testing.T) {
				if _, parseErr := parsePage.Goto(getBaseURL+parseCase2.Path, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateLoad,
				}); parseErr != nil {
					parseT2.Fatalf("goto: %v", parseErr)
				}
				if _, parseErr := parsePage.WaitForSelector(parseCase2.ReadySel); parseErr != nil {
					parseT2.Fatalf("example never rendered: %v", parseErr)
				}
				if parseErr := parsePage.Click(parseCase2.TriggerSel); parseErr != nil {
					parseT2.Fatalf("trigger: %v", parseErr)
				}
				if _, parseErr := parsePage.WaitForFunction(parseCase2.MountedExpr, nil); parseErr != nil {
					parseT2.Fatalf("portal content never mounted into its root: %v", parseErr)
				}
			})
		}
	})
}
