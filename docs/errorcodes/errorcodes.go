// Package errorcodes extracts the framework's diagnostic codes from the runtime
// source of truth and renders the generated error-code reference page. The page
// is never hand-maintained: TestErrorCodeReferenceIsGenerated regenerates it
// from internal/runtime/diagnostic_metadata.go and fails if the committed page
// drifts, so every code a developer (or agent) can see in a report is always
// looked-up-able with its cause anchor and remediation.
package errorcodes

import (
	"regexp"
	"sort"
	"strings"
)

// ErrorCode is one diagnostic code with its docs anchor and remediation text.
type ErrorCode struct {
	Code        string
	Docs        string
	Remediation string
}

const actionableErrorsDoc = "ACTIONABLE_ERRORS.md"

var (
	codeAssignPattern = regexp.MustCompile(`\.Code\s*=\s*"([A-Z0-9-]+)"`)
	docsAssignPattern = regexp.MustCompile(`\.Docs\s*=\s*(.+)$`)
	remedAssignPattern = regexp.MustCompile(`\.Remediation\s*=\s*"(.*)"\s*$`)
	caseCodePattern   = regexp.MustCompile(`return\s+"([A-Z0-9-]+)"`)
	caseDocsPattern   = regexp.MustCompile(`return\s+(.+)$`)
	quotedPattern     = regexp.MustCompile(`"([^"]*)"`)
)

// ExtractCodes parses the diagnostic-metadata source and returns every code,
// sorted by code, deduplicated (the descriptive switch wins over the
// phase-keyed panic helpers, which only add codes the switch omits).
func ExtractCodes(parseSource string) []ErrorCode {
	parseByCode := map[string]ErrorCode{}

	// Primary source: the diagnosticMetadata switch, which sets .Code, .Docs,
	// and .Remediation in that order per case.
	var parsePending ErrorCode
	for _, parseLine := range strings.Split(parseSource, "\n") {
		if parseMatch := codeAssignPattern.FindStringSubmatch(parseLine); parseMatch != nil {
			parsePending = ErrorCode{Code: parseMatch[1]}
			continue
		}
		if parseMatch := docsAssignPattern.FindStringSubmatch(parseLine); parseMatch != nil && parsePending.Code != "" {
			parsePending.Docs = resolveDocsExpression(parseMatch[1])
			continue
		}
		if parseMatch := remedAssignPattern.FindStringSubmatch(parseLine); parseMatch != nil && parsePending.Code != "" {
			parsePending.Remediation = parseMatch[1]
			if _, parseExists := parseByCode[parsePending.Code]; !parseExists {
				parseByCode[parsePending.Code] = parsePending
			}
			parsePending = ErrorCode{}
		}
	}

	// Secondary source: panic-phase helpers. panicDiagnosticCode maps phase ->
	// code, panicDiagnosticDocs maps phase -> docs, panicDiagnosticRemediation
	// maps phase -> remediation. They are keyed by the same `case <phase>:`
	// labels, so align them positionally per function to recover codes (notably
	// GWC-RUNTIME-PANIC-ASYNC) that have no descriptive switch entry.
	parsePhaseCodes := extractFunctionReturns(parseSource, "func panicDiagnosticCode", caseCodePattern, false)
	parsePhaseDocs := extractFunctionReturns(parseSource, "func panicDiagnosticDocs", caseDocsPattern, true)
	parsePhaseRems := extractFunctionReturns(parseSource, "func panicDiagnosticRemediation", caseDocsPattern, true)
	for parsePhase, parseCode := range parsePhaseCodes {
		if parseCode == "" || strings.EqualFold(parseCode, "GWC-RUNTIME-PANIC") {
			continue // skip the generic fallback
		}
		if _, parseExists := parseByCode[parseCode]; parseExists {
			continue
		}
		parseByCode[parseCode] = ErrorCode{
			Code:        parseCode,
			Docs:        resolveDocsExpression(parsePhaseDocs[parsePhase]),
			Remediation: cleanRemediation(parsePhaseRems[parsePhase]),
		}
	}

	parseCodes := make([]ErrorCode, 0, len(parseByCode))
	for _, parseCode := range parseByCode {
		parseCodes = append(parseCodes, parseCode)
	}
	sort.Slice(parseCodes, func(parseA, parseB int) bool {
		return parseCodes[parseA].Code < parseCodes[parseB].Code
	})
	return parseCodes
}

// extractFunctionReturns scans a named func body's `case <label>:` arms and maps
// each label to the captured return expression. When wantExpr is false the
// pattern's first group is taken verbatim; otherwise the whole return expression
// is captured for later resolution.
func extractFunctionReturns(parseSource string, parseFuncSignature string, parsePattern *regexp.Regexp, parseWantExpr bool) map[string]string {
	parseResult := map[string]string{}
	parseStart := strings.Index(parseSource, parseFuncSignature)
	if parseStart < 0 {
		return parseResult
	}
	parseBody := parseSource[parseStart:]
	if parseEnd := strings.Index(parseBody, "\n}\n"); parseEnd >= 0 {
		parseBody = parseBody[:parseEnd]
	}
	var parseLabel string
	for _, parseLine := range strings.Split(parseBody, "\n") {
		parseTrimmed := strings.TrimSpace(parseLine)
		if strings.HasPrefix(parseTrimmed, "case ") && strings.HasSuffix(parseTrimmed, ":") {
			parseLabel = strings.TrimSuffix(strings.TrimPrefix(parseTrimmed, "case "), ":")
			continue
		}
		if parseLabel == "" {
			continue
		}
		if parseMatch := parsePattern.FindStringSubmatch(parseLine); parseMatch != nil {
			parseResult[parseLabel] = strings.TrimSpace(parseMatch[1])
			parseLabel = ""
		}
	}
	return parseResult
}

// resolveDocsExpression turns a Go docs expression (string literal, or
// actionableErrorsDoc + "#anchor") into the concrete docs reference.
func resolveDocsExpression(parseExpr string) string {
	parseExpr = strings.TrimSpace(parseExpr)
	var parseBuilder strings.Builder
	if strings.Contains(parseExpr, "actionableErrorsDoc") {
		parseBuilder.WriteString(actionableErrorsDoc)
	}
	for _, parseMatch := range quotedPattern.FindAllStringSubmatch(parseExpr, -1) {
		parseBuilder.WriteString(parseMatch[1])
	}
	return parseBuilder.String()
}

// cleanRemediation strips the `parsePrefix + ` automation note wrapper from a
// panic-phase remediation, keeping the human-readable guidance.
func cleanRemediation(parseExpr string) string {
	if parseMatch := quotedPattern.FindStringSubmatch(parseExpr); parseMatch != nil {
		return parseMatch[1]
	}
	return strings.TrimSpace(parseExpr)
}

// RenderPage renders the committed error-code reference markdown from codes.
func RenderPage(parseCodes []ErrorCode) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("# Diagnostic Error Code Reference\n\n")
	parseBuilder.WriteString("> Generated from `internal/runtime/diagnostic_metadata.go` by `docs/errorcodes`.\n")
	parseBuilder.WriteString("> Do not edit by hand; run `ERRORCODES_WRITE=1 go test ./docs/errorcodes/` to regenerate.\n\n")
	parseBuilder.WriteString("Every code below is emitted in a structured runtime diagnostic. Look the code up here for its cause anchor and the `next:` remediation.\n\n")
	for _, parseCode := range parseCodes {
		parseBuilder.WriteString("## " + parseCode.Code + "\n\n")
		if parseCode.Docs != "" {
			// Rendered as an inline reference, not a file link: the docs anchor
			// names where deeper guidance lives, but this generated page is the
			// canonical lookup target for the code itself.
			parseBuilder.WriteString("- **Docs:** `" + parseCode.Docs + "`\n")
		}
		if parseCode.Remediation != "" {
			parseBuilder.WriteString("- **next:** " + parseCode.Remediation + "\n")
		}
		parseBuilder.WriteString("\n")
	}
	return parseBuilder.String()
}
