// Package binarycheck flags compiled binaries that have been committed to git.
// It is used by the no-tracked-binaries workflow so build outputs such as
// stray .exe files cannot slip into the repository.
package binarycheck

import (
	"path"
	"strings"
)

// binaryExtensions lists the file extensions treated as compiled binaries.
var binaryExtensions = []string{".exe", ".dll", ".so", ".dylib", ".test", ".wasm"}

// allowedBinaries lists repo-relative paths that may be tracked. Each entry
// needs a comment saying why the binary is committed. It is empty today
// because the tree tracks no binaries.
var allowedBinaries = map[string]string{}

// isBinaryPath reports whether parsePath has a binary extension.
func isBinaryPath(parsePath string) bool {
	parseExt := strings.ToLower(path.Ext(parsePath))
	for _, parseBinaryExt := range binaryExtensions {
		if parseExt == parseBinaryExt {
			return true
		}
	}
	return false
}

// FindTracked returns the entries of parseTracked that are binaries and not in
// the allowlist, preserving input order. Paths use forward slashes.
func FindTracked(parseTracked []string) []string {
	var parseFound []string
	for _, parsePath := range parseTracked {
		parseClean := strings.ReplaceAll(parsePath, "\\", "/")
		if !isBinaryPath(parseClean) {
			continue
		}
		if _, isAllowed := allowedBinaries[parseClean]; isAllowed {
			continue
		}
		parseFound = append(parseFound, parseClean)
	}
	return parseFound
}
