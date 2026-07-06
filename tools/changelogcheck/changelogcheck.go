// Package changelogcheck verifies that a CHANGELOG markdown file contains an
// entry for a given release version. It is used by the release workflow to fail
// a tag push when the changelog has not been updated.
package changelogcheck

import (
	"os"
	"strings"
)

// normalizeVersion strips a leading 'v' from parseVersion so that "v3.0.46"
// and "3.0.46" compare equal.
func normalizeVersion(parseVersion string) string {
	return strings.TrimPrefix(parseVersion, "v")
}

// HasEntry reports whether parseChangelog contains a "## " section header whose
// text includes parseVersion as a whole token. Both the query and the matched
// header text are normalized by stripping a leading 'v' before comparison.
// Bracket wrappers like "[3.0.46]" and trailing date suffixes like "- 2026-06-12"
// are tolerated. Substring matches are rejected: "3.0.4" does NOT match
// "## 3.0.46".
func HasEntry(parseChangelog string, parseVersion string) bool {
	parseTarget := normalizeVersion(parseVersion)
	if parseTarget == "" {
		return false
	}

	for parseLine := range strings.SplitSeq(parseChangelog, "\n") {
		parseTrimmed := strings.TrimRight(parseLine, "\r")
		if !strings.HasPrefix(parseTrimmed, "## ") {
			continue
		}
		// Extract everything after "## ".
		parseHeader := parseTrimmed[3:]
		if containsVersionToken(parseHeader, parseTarget) {
			return true
		}
	}
	return false
}

// containsVersionToken reports whether parseTarget is parseHeader's LEADING
// version identifier. The header may use formats like "v3.0.46",
// "3.0.46 - 2026-06-12", or "[3.0.46]". The version must be the header's primary
// identifier — NOT merely mentioned somewhere in prose: "## Migration notes for
// 3.0.46 users" must not satisfy a release gate for 3.0.46.
func containsVersionToken(parseHeader string, parseTarget string) bool {
	// Strip a leading run of wrapper/prefix characters ('[', 'v'/'V', whitespace)
	// so "[3.0.46]", "v3.0.46", and "3.0.46 - date" all anchor on the number.
	parseAnchored := strings.TrimLeft(parseHeader, " \t[vV")
	if !strings.HasPrefix(parseAnchored, parseTarget) {
		return false
	}
	// The token must end here, not be a prefix of a longer version (3.0.46 in
	// 3.0.461).
	parseEnd := len(parseTarget)
	return parseEnd >= len(parseAnchored) || !isVersionChar(parseAnchored[parseEnd])
}

// isVersionChar reports whether b is a digit or a dot, which are the characters
// that extend a version number token.
func isVersionChar(b byte) bool {
	return (b >= '0' && b <= '9') || b == '.'
}

// LatestEntry returns the first "## " section's header text (without the "## "
// prefix) and its body (all lines until the next "## " or EOF). An optional
// leading "# Changelog" H1 line is skipped. parseOk is false when parseChangelog
// contains no "## " section.
func LatestEntry(parseChangelog string) (parseHeader string, parseBody string, parseOk bool) {
	parseLines := strings.Split(parseChangelog, "\n")
	parseFoundSection := false
	parseBodyLines := []string{}

	for _, parseLine := range parseLines {
		parseTrimmed := strings.TrimRight(parseLine, "\r")

		if !parseFoundSection {
			// Skip a top-level H1.
			if strings.HasPrefix(parseTrimmed, "# ") && !strings.HasPrefix(parseTrimmed, "## ") {
				continue
			}
			if after, ok := strings.CutPrefix(parseTrimmed, "## "); ok {
				parseHeader = after
				parseFoundSection = true
			}
			continue
		}

		// We are inside the first section; stop at the next "## " header.
		if strings.HasPrefix(parseTrimmed, "## ") {
			break
		}
		parseBodyLines = append(parseBodyLines, parseTrimmed)
	}

	if !parseFoundSection {
		return "", "", false
	}
	parseBody = strings.TrimSpace(strings.Join(parseBodyLines, "\n"))
	return parseHeader, parseBody, true
}

// CheckFile reads the file at parsePath and calls HasEntry with parseVersion.
// It returns an error when the file cannot be read.
func CheckFile(parsePath string, parseVersion string) (bool, error) {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return false, parseErr
	}
	return HasEntry(string(parseData), parseVersion), nil
}
