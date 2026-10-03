// Package errcode is the single registry of framework diagnostic codes.
//
// It is tag-free and imports only the standard library so the runtime, the
// docs generator, and the gwc tool all read the same table. A code carries its
// family, headline, remediation, docs reference, and severity; nothing else in
// the repository should restate that text.
package errcode

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Code is a stable diagnostic identifier such as GWC-RUNTIME-PANIC-RENDER.
type Code string

// Family groups related codes for filtering and documentation.
type Family string

// Severity is the default severity a code is reported at.
type Severity string

// Families of registered codes.
const (
	FamilyPanic     Family = "panic"
	FamilyHook      Family = "hook"
	FamilyRuntime   Family = "runtime"
	FamilyUI        Family = "ui"
	FamilyRouter    Family = "router"
	FamilyHydration Family = "hydration"
	FamilyFramework Family = "framework"
	FamilyState     Family = "state"
	FamilyFetch     Family = "fetch"
	FamilyCSS       Family = "css"
	FamilyInterop   Family = "interop"
)

// Default severities of registered codes.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ReferencePagePath is the repository path of the generated code reference.
const ReferencePagePath = "docs/REFERENCE_MANUAL/error-codes.md"

// Entry is the full description of one registered code.
type Entry struct {
	Code     Code
	Family   Family
	Headline string
	// Remediation is the actionable next step shown in the "next:" line.
	Remediation string
	// PhaseRemediation is the phase-specific guidance used by panic reports;
	// empty for codes that are not panic phases.
	PhaseRemediation string
	// Docs is the reference-page location with the code's heading anchor.
	Docs string
	// Recoverable marks codes that are always reported as recoverable.
	Recoverable bool
	// RecoverableUnlessError marks codes recoverable whenever the reported
	// severity is below error.
	RecoverableUnlessError bool
	Severity               Severity
}

var (
	registryIndex  map[Code]Entry
	registrySorted []Entry
)

// init builds the lookup index and fills each entry's Docs reference.
func init() {
	registryIndex, registrySorted = buildRegistry(registryEntries)
}

// buildRegistry derives Docs for every entry and returns the index plus the
// code-sorted slice.
func buildRegistry(parseEntries []Entry) (map[Code]Entry, []Entry) {
	parseIndex := make(map[Code]Entry, len(parseEntries))
	parseSorted := make([]Entry, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseEntry.Docs = getDocsRef(parseEntry.Code)
		parseIndex[parseEntry.Code] = parseEntry
		parseSorted = append(parseSorted, parseEntry)
	}
	sort.Slice(parseSorted, func(parseA, parseB int) bool {
		return parseSorted[parseA].Code < parseSorted[parseB].Code
	})
	return parseIndex, parseSorted
}

// getDocsRef returns the reference-page location for a code, anchored on the
// lowercase heading the generator writes.
func getDocsRef(parseCode Code) string {
	return ReferencePagePath + "#" + strings.ToLower(string(parseCode))
}

// Lookup returns the registered entry for a code.
func Lookup(parseCode Code) (Entry, bool) {
	parseEntry, parseOk := registryIndex[parseCode]
	return parseEntry, parseOk
}

// All returns every registered entry sorted by code. The slice is a copy.
func All() []Entry {
	return append([]Entry(nil), registrySorted...)
}

// Error is an error that carries a registered code.
type Error struct {
	Code Code
	Args []any
}

// New returns an Error for a code. Args are appended to the headline and the
// first error argument, if any, is exposed through Unwrap.
func New(parseCode Code, parseArgs ...any) *Error {
	return &Error{Code: parseCode, Args: parseArgs}
}

// Error formats the code, its headline, and any arguments.
func (parseErr *Error) Error() string {
	parseText := string(parseErr.Code)
	if parseEntry, parseOk := Lookup(parseErr.Code); parseOk {
		parseText = "[" + string(parseErr.Code) + "] " + parseEntry.Headline
	}
	if len(parseErr.Args) > 0 {
		parseText += ": " + strings.TrimSuffix(fmt.Sprintln(parseErr.Args...), "\n")
	}
	return parseText
}

// Unwrap returns the first error among the arguments.
func (parseErr *Error) Unwrap() error {
	for _, parseArg := range parseErr.Args {
		if parseCause, parseOk := parseArg.(error); parseOk {
			return parseCause
		}
	}
	return nil
}

// Is reports whether the target is an *Error with the same Code.
func (parseErr *Error) Is(parseTarget error) bool {
	parseOther, parseOk := parseTarget.(*Error)
	return parseOk && parseOther != nil && parseOther.Code == parseErr.Code
}

// CodeOf returns the code of the first *Error in the chain, or "".
func CodeOf(parseErr error) Code {
	if parseFound, parseOk := errors.AsType[*Error](parseErr); parseOk {
		return parseFound.Code
	}
	return ""
}
