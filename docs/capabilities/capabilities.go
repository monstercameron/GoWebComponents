// Package capabilities holds the curated capability table for the GoWebComponents
// framework and renders the generated capability-matrix reference page. The page
// is never hand-maintained: TestCapabilityMatrixIsGenerated regenerates it from
// the table below and fails if the committed page drifts, so the matrix always
// reflects what actually exists on disk.
package capabilities

import "strings"

// Capability is one row in the framework capability matrix, mapping a named
// capability to the packages that implement it, a representative public example
// slug, and the reference-manual chapter that covers it in depth.
//
// ExampleSlug is the subdirectory name under examples/public/; an empty string
// means no public example exists yet and the Example cell renders "—".
// Chapter is the filename (not path) of the markdown chapter under
// docs/REFERENCE_MANUAL/.
type Capability struct {
	// Name is the human-readable capability label shown in the matrix.
	Name string
	// Packages lists the Go package path suffixes (relative to the module root)
	// that implement this capability, e.g. []string{"ui"} or []string{"html",
	// "html/shorthand"}.
	Packages []string
	// ExampleSlug is the name of the directory under examples/public/ that best
	// demonstrates this capability. Empty means no public example exists yet.
	ExampleSlug string
	// Chapter is the filename of the reference-manual chapter that covers this
	// capability, e.g. "04-ui-rendering-and-hooks.md".
	Chapter string
}

// Capabilities returns the curated capability table in stable reading order.
// Do not sort the slice; callers that need a different order should copy and sort.
func Capabilities() []Capability {
	return []Capability{
		{
			Name:        "Components & hooks",
			Packages:    []string{"ui"},
			ExampleSlug: "counter",
			Chapter:     "04-ui-rendering-and-hooks.md",
		},
		{
			Name:        "HTML authoring",
			Packages:    []string{"html", "html/shorthand"},
			ExampleSlug: "semantic-html",
			Chapter:     "05-html-authoring.md",
		},
		{
			Name:        "Shared state & reactivity",
			Packages:    []string{"state"},
			ExampleSlug: "state-atoms",
			Chapter:     "06-state-and-reactivity.md",
		},
		{
			Name:        "Data loading & mutations",
			Packages:    []string{"fetch"},
			ExampleSlug: "use-fetch",
			Chapter:     "07-data-loading-and-mutations.md",
		},
		{
			Name:        "Realtime data",
			Packages:    []string{"fetch"},
			ExampleSlug: "cross-tab-sync",
			Chapter:     "07-data-loading-and-mutations.md",
		},
		{
			Name:        "Routing",
			Packages:    []string{"router"},
			ExampleSlug: "browser-router",
			Chapter:     "08-routing.md",
		},
		{
			Name:        "SSR & hydration",
			Packages:    []string{"ui", "head"},
			ExampleSlug: "hydration",
			Chapter:     "09-ssr-and-hydration.md",
		},
		{
			Name:        "Static islands",
			Packages:    []string{"ui"},
			ExampleSlug: "static-islands",
			Chapter:     "09-ssr-and-hydration.md",
		},
		{
			Name:        "Browser interop & workers",
			Packages:    []string{"interop"},
			ExampleSlug: "browser-interop",
			Chapter:     "10-browser-interop-and-workers.md",
		},
		{
			Name:        "Forms & accessibility",
			Packages:    []string{"ui"},
			ExampleSlug: "form-accessibility",
			Chapter:     "11-forms-accessibility-and-i18n.md",
		},
		{
			Name:        "Internationalization",
			Packages:    []string{"i18n"},
			ExampleSlug: "locale-switcher",
			Chapter:     "11-forms-accessibility-and-i18n.md",
		},
		{
			Name:        "Devtools & diagnostics",
			Packages:    []string{"devtools"},
			ExampleSlug: "devtools-panel",
			Chapter:     "12-devtools-testing-and-observability.md",
		},
		{
			Name:        "PWA & offline",
			Packages:    []string{"pwa"},
			ExampleSlug: "progressive-web-app-offline-cache",
			Chapter:     "13-assets-deployment-and-pwa.md",
		},
		{
			Name:        "Feature flags",
			Packages:    []string{"flags"},
			ExampleSlug: "",
			Chapter:     "06-state-and-reactivity.md",
		},
	}
}

// RenderPage renders the capability-matrix reference page as a GitHub-Flavored
// Markdown string from the provided capability rows. Row order is preserved.
// When a Capability's ExampleSlug is empty the Example cell renders "—" rather
// than a broken link.
func RenderPage(parseCaps []Capability) string {
	var parseBuilder strings.Builder

	parseBuilder.WriteString("# Capability Matrix\n\n")
	parseBuilder.WriteString("> Generated from `docs/capabilities/capabilities.go`.")
	parseBuilder.WriteString(" Do not edit by hand;")
	parseBuilder.WriteString(" run `CAPABILITIES_WRITE=1 go test ./docs/capabilities/` to regenerate.\n\n")

	parseBuilder.WriteString("| Capability | Package(s) | Example | Chapter |\n")
	parseBuilder.WriteString("| --- | --- | --- | --- |\n")

	for _, parseCap := range parseCaps {
		// Package(s) column: each package as inline code, joined by ", ".
		parsePackageParts := make([]string, len(parseCap.Packages))
		for parseIdx, parsePkg := range parseCap.Packages {
			parsePackageParts[parseIdx] = "`" + parsePkg + "`"
		}
		parsePackageCell := strings.Join(parsePackageParts, ", ")

		// Example column: link when slug is set, em-dash otherwise.
		var parseExampleCell string
		if parseCap.ExampleSlug != "" {
			parseExampleCell = "[" + parseCap.ExampleSlug + "](../../examples/public/" + parseCap.ExampleSlug + "/)"
		} else {
			parseExampleCell = "—"
		}

		// Chapter column: link to the chapter file in the same directory.
		parseChapterCell := "[" + parseCap.Chapter + "](" + parseCap.Chapter + ")"

		parseBuilder.WriteString("| " + parseCap.Name + " | " + parsePackageCell + " | " + parseExampleCell + " | " + parseChapterCell + " |\n")
	}

	return parseBuilder.String()
}
