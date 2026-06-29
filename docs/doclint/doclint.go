// Package doclint is a documentation drift guard.
//
// It scans tracked Markdown files for repository-relative paths used as inputs
// to commands in fenced code blocks and reports any that no longer resolve on
// disk. It exists to prevent the specific class of staleness that broke the
// front-door quickstart: an example was relocated from examples/01-counter to
// examples/public/counter, but every copy-paste `gwc` command in the README,
// the getting-started chapter, and AGENTS.md kept pointing at the dead path, so
// a fresh evaluator's first command failed on a clean checkout.
//
// The guard is deliberately high precision rather than exhaustive. It only
// inspects tokens inside fenced code blocks, only treats a token as a
// repo-relative path when its first segment is an existing top-level repo
// entry, and skips command outputs and obvious user-project placeholders. The
// goal is zero false positives so the lane can fail a build without becoming
// noise that gets disabled.
package doclint

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PathRef is one repo-relative path reference found inside a doc command block.
type PathRef struct {
	// DocPath is the repo-relative path of the Markdown file that contains it.
	DocPath string
	// Line is the 1-based line number of the reference within that file.
	Line int
	// Raw is the token exactly as written in the doc (Windows or POSIX style).
	Raw string
	// Rel is the normalized, forward-slash, repo-relative path to resolve.
	Rel string
}

// dirsSkipped are subtrees the guard never descends into: VCS metadata, vendored
// or generated trees, and the historical example-site snapshot docs that
// intentionally preserve the old numbered catalog for the record.
var dirsSkipped = map[string]bool{
	".git":         true,
	"node_modules": true,
	"third_party":  true,
	"bin":          true,
	"vendor":       true,
	"site-dist":    true,
}

// snapshotDocDir is the one docs subtree that is an archival capture rather than
// living instructions, so its older paths must not fail the guard.
const snapshotDocDir = "examples/public-examples-site/assets/docs"

// outputFlags precede command arguments that name where output is written, so
// the referenced path is not expected to exist before the command runs.
var outputFlags = map[string]bool{
	"-o":                     true,
	"-out":                   true,
	"-output":                true,
	"-out-dir":               true,
	"-outdir":                true,
	"-export-static-catalog": true,
}

// fenceOpen matches a fenced code block delimiter and captures the optional
// language hint that follows the opening backticks.
var fenceOpen = regexp.MustCompile("^\\s*```+\\s*([A-Za-z0-9_+-]*)")

// shellLangs are the fenced-block languages whose contents are commands. Only
// these (plus unlabeled blocks that clearly start with a command verb) are
// scanned for path arguments, so directory-layout listings (```text) and code
// samples with URL string literals (```go, ```json) never produce false hits.
var shellLangs = map[string]bool{
	"powershell": true, "pwsh": true, "ps1": true,
	"bash": true, "sh": true, "shell": true, "zsh": true,
	"console": true, "bat": true, "cmd": true,
}

// commandVerbs lead a real command line, used to admit unlabeled fenced blocks
// without admitting prose or code.
var commandVerbs = []string{
	"go ", "gwc ", "./tools", ".\\tools", "set-location", "new-item",
	"npx ", "node ", "go.exe",
}

// pathToken captures dot-anchored relative paths (./x, .\x) and bare
// examples/... or tools/... style segmented paths used as command arguments.
var pathToken = regexp.MustCompile(`(?:\.[\\/]|\b)(?:[\w.@-]+[\\/])+[\w.@-]+`)

// placeholderHints flag tokens that are illustrative stand-ins for a user's own
// project, not real repo paths, so they are skipped rather than resolved.
var placeholderHints = []string{
	"path\\to", "path/to", "your-app", "my-app", "<", "...", "layout\\main.go",
}

// FindRepoRoot walks up from start until it finds the directory containing
// go.mod, returning the module root. It returns ok=false if none is found.
func FindRepoRoot(start string) (root string, ok bool) {
	dir := start
	for range 12 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// topLevelEntries returns the set of names directly under the repo root. A
// candidate path is only treated as repo-relative when its first segment is one
// of these, which is what distinguishes examples\public\counter\main.go (real)
// from a user-project placeholder like .\main.go.
func topLevelEntries(root string) (map[string]bool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		set[e.Name()] = true
	}
	return set, nil
}

// ScanRepo walks every tracked Markdown file under root and returns the
// repo-relative input path references found inside fenced command blocks whose
// first segment is a real top-level repo entry.
func ScanRepo(root string) ([]PathRef, error) {
	topLevel, err := topLevelEntries(root)
	if err != nil {
		return nil, err
	}
	var refs []PathRef
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if dirsSkipped[info.Name()] {
				return filepath.SkipDir
			}
			// Skip generated browser-compiler package archives anywhere.
			if info.Name() == "pkg" && strings.Contains(filepath.ToSlash(path), "browser-compiler") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		relSlash := filepath.ToSlash(rel)
		if strings.HasPrefix(relSlash, snapshotDocDir) {
			return nil
		}
		fileRefs, scanErr := scanFile(path, relSlash, topLevel)
		if scanErr != nil {
			return scanErr
		}
		refs = append(refs, fileRefs...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return refs, nil
}

// scanFile extracts candidate path references from one Markdown file.
func scanFile(absPath, docRel string, topLevel map[string]bool) ([]PathRef, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var refs []PathRef
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	inFence := false
	blockIsShell := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if m := fenceOpen.FindStringSubmatch(line); m != nil {
			if inFence {
				inFence = false
				blockIsShell = false
			} else {
				inFence = true
				blockIsShell = shellLangs[strings.ToLower(m[1])]
			}
			continue
		}
		if !inFence {
			continue
		}
		// Scan labeled shell blocks, and unlabeled blocks only on lines that
		// clearly begin a command. This keeps prose and code samples out.
		if !blockIsShell && !looksLikeCommand(line) {
			continue
		}
		for _, ref := range candidatesInLine(line, topLevel) {
			ref.DocPath = docRel
			ref.Line = lineNo
			refs = append(refs, ref)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return refs, nil
}

// candidatesInLine returns the repo-relative path references in a single command
// line, skipping output-flag arguments and placeholder stand-ins.
func candidatesInLine(line string, topLevel map[string]bool) []PathRef {
	fields := strings.Fields(line)
	var refs []PathRef
	for i, field := range fields {
		if i > 0 && outputFlags[strings.ToLower(fields[i-1])] {
			continue
		}
		token := strings.Trim(field, "\"'`,;:()")
		if !pathToken.MatchString(token) {
			continue
		}
		if isPlaceholder(token) {
			continue
		}
		rel := normalizeToken(token)
		if rel == "" {
			continue
		}
		first := rel
		if before, _, ok := strings.Cut(rel, "/"); ok {
			first = before
		}
		if !topLevel[first] {
			continue
		}
		// Generated output trees are never expected to exist on a clean tree,
		// at any depth (a project's bin/ or dist/ holds build and runtime
		// artifacts: wasm, logs, sqlite files).
		if strings.HasPrefix(rel, "bin/") || strings.Contains(rel, "/bin/") ||
			strings.Contains(rel, "/dist/") || strings.HasSuffix(rel, "/dist") {
			continue
		}
		refs = append(refs, PathRef{Raw: token, Rel: rel})
	}
	return refs
}

// looksLikeCommand reports whether a line (inside an unlabeled fenced block)
// begins with a recognized command verb.
func looksLikeCommand(line string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(line))
	for _, verb := range commandVerbs {
		if strings.HasPrefix(trimmed, verb) {
			return true
		}
	}
	return false
}

// isPlaceholder reports whether a token is an illustrative stand-in path.
func isPlaceholder(token string) bool {
	lower := strings.ToLower(filepath.ToSlash(token))
	for _, hint := range placeholderHints {
		if strings.Contains(lower, strings.ToLower(filepath.ToSlash(hint))) {
			return true
		}
	}
	return false
}

// normalizeToken converts a Windows or POSIX command token into a clean
// forward-slash repo-relative path, or "" if it is not a real path candidate.
func normalizeToken(token string) string {
	clean := filepath.ToSlash(token)
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.TrimPrefix(clean, ".\\")
	clean = strings.TrimPrefix(clean, "/")
	clean = strings.TrimSpace(clean)
	// A bare token with no separators and no extension is not a path argument.
	if !strings.Contains(clean, "/") {
		return ""
	}
	return clean
}

// Resolve reports the references whose target does not exist under root. A reference is
// considered live if it resolves EITHER repo-root-relative (the front-door case: README/getting-
// started commands run from the repo root) OR relative to its own doc file's directory (a subdir
// README whose command runs from that subdir, e.g. tools/devtools-extension/README.md running
// `node --test test/bridge.test.mjs` from the extension dir). Accepting the doc-dir-relative
// resolution removes a false-positive class without weakening the front-door drift guard.
func Resolve(root string, refs []PathRef) []PathRef {
	var broken []PathRef
	for _, ref := range refs {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(ref.Rel))); err == nil {
			continue
		}
		if ref.DocPath != "" {
			docDir := filepath.Dir(filepath.FromSlash(ref.DocPath))
			if _, err := os.Stat(filepath.Join(root, docDir, filepath.FromSlash(ref.Rel))); err == nil {
				continue
			}
		}
		broken = append(broken, ref)
	}
	return broken
}
