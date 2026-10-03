// Package errorcodes renders the framework's diagnostic-code reference page from
// the internal/errcode registry. The page is never hand-maintained:
// TestErrorCodeReferenceIsGenerated regenerates it from errcode.All() and fails
// if the committed page drifts, so every code a developer (or agent) can see in a
// report is always looked-up-able at the anchor its Docs reference names.
package errorcodes

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/internal/errcode"
)

// RenderPage renders the committed error-code reference markdown from entries.
// Each heading is the bare code, so its lowercase form is the anchor that
// errcode.Entry.Docs points at.
func RenderPage(parseEntries []errcode.Entry) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("# Diagnostic Error Code Reference\n\n")
	parseBuilder.WriteString("> Generated from `internal/errcode` by `docs/errorcodes`.\n")
	parseBuilder.WriteString("> Do not edit by hand; run `ERRORCODES_WRITE=1 go test ./docs/errorcodes/` to regenerate.\n\n")
	parseBuilder.WriteString("Every code below is emitted in a structured runtime diagnostic. Look the code up here for its cause and the `next:` remediation.\n\n")
	for _, parseEntry := range parseEntries {
		parseBuilder.WriteString("## " + string(parseEntry.Code) + "\n\n")
		parseBuilder.WriteString(parseEntry.Headline + "\n\n")
		parseBuilder.WriteString("- **Family:** " + string(parseEntry.Family) + "\n")
		parseBuilder.WriteString("- **Severity:** " + string(parseEntry.Severity) + "\n")
		parseBuilder.WriteString("- **Docs:** `" + parseEntry.Docs + "`\n")
		if parseEntry.Remediation != "" {
			parseBuilder.WriteString("- **next:** " + parseEntry.Remediation + "\n")
		}
		parseBuilder.WriteString("\n")
	}
	return parseBuilder.String()
}
