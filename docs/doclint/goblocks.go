package doclint

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GoBlock is one fenced ```go code block in the docs that is a COMPLETE Go file (it declares a
// package). Fragments — snippets without a package clause — are intentionally not validated here,
// keeping the guard high-precision (zero false positives) like the path-reference scanner: only
// copy-paste-runnable samples are checked, which is exactly the rot that breaks a fresh reader.
type GoBlock struct {
	DocPath string // repo-relative path of the Markdown file
	Line    int    // 1-based line of the block's opening fence
	Source  string // the block's Go source
	// Compile is true when the opening fence is marked `gwc:build` — an opt-in that the sample
	// is a complete, runnable program to be type-checked (compiled), not just parsed. Native is
	// true for the `gwc:build:native` variant (a server/native sample); otherwise the sample is
	// compiled for the browser (js/wasm) target.
	Compile bool
	Native  bool
}

// GoBlockError is a complete-file ```go sample that no longer parses, so the docs would silently
// hand a reader broken Go.
type GoBlockError struct {
	DocPath string
	Line    int
	Err     string
}

// goFenceLangs are the fenced-block language hints whose contents are Go source.
var goFenceLangs = map[string]bool{"go": true, "golang": true}

// ScanGoBlocks walks every tracked Markdown file under root and returns the complete-file ```go
// blocks (those containing a top-level `package ` declaration).
func ScanGoBlocks(root string) ([]GoBlock, error) {
	var blocks []GoBlock
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if dirsSkipped[info.Name()] {
				return filepath.SkipDir
			}
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
		fileBlocks, scanErr := scanGoBlocksInFile(path, relSlash)
		if scanErr != nil {
			return scanErr
		}
		blocks = append(blocks, fileBlocks...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return blocks, nil
}

// scanGoBlocksInFile extracts the complete-file ```go blocks from one Markdown file.
func scanGoBlocksInFile(absPath, docRel string) ([]GoBlock, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var blocks []GoBlock
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	inGo := false
	fenceLine := 0
	fenceCompile := false
	fenceNative := false
	var current strings.Builder
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if m := fenceOpen.FindStringSubmatch(line); m != nil {
			if inGo {
				// Closing fence: keep only complete-file blocks (those with a package clause).
				source := current.String()
				if goBlockIsCompleteFile(source) {
					blocks = append(blocks, GoBlock{DocPath: docRel, Line: fenceLine, Source: source, Compile: fenceCompile, Native: fenceNative})
				}
				inGo = false
				current.Reset()
				continue
			}
			if goFenceLangs[strings.ToLower(m[1])] {
				inGo = true
				fenceLine = lineNo
				fenceCompile = strings.Contains(line, "gwc:build")
				fenceNative = strings.Contains(line, "gwc:build:native")
				current.Reset()
			}
			continue
		}
		if inGo {
			current.WriteString(line)
			current.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return blocks, nil
}

// goBlockIsCompleteFile reports whether a ```go block is a full Go file (has a top-level
// `package` declaration), as opposed to a fragment. Only complete files are validated.
func goBlockIsCompleteFile(source string) bool {
	for raw := range strings.SplitSeq(source, "\n") {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "package ") {
			return true
		}
	}
	return false
}

// GoBlockCompileError is a `gwc:build`-marked sample that failed to compile (type-check) against
// the real module — the type-level "docs can't silently lie" guard (a renamed/removed API used
// by a runnable sample fails here even though it still parses).
type GoBlockCompileError struct {
	DocPath string
	Line    int
	Err     string
}

// CompileMarkedGoBlocks compiles every `gwc:build`-marked complete-file sample against the real
// module: each is written to a throwaway module with a `replace` to repoRoot and built with
// `go build` (js/wasm by default, native for `gwc:build:native`). This is the opt-in type-level
// gate — only samples explicitly marked as runnable programs are compiled, so illustrative
// fragments never produce false failures. repoRoot is the module root (contains go.mod).
func CompileMarkedGoBlocks(repoRoot string) ([]GoBlockCompileError, error) {
	blocks, err := ScanGoBlocks(repoRoot)
	if err != nil {
		return nil, err
	}
	var errs []GoBlockCompileError
	for index, block := range blocks {
		if !block.Compile {
			continue
		}
		if buildErr := compileSampleInModule(repoRoot, block, index); buildErr != "" {
			errs = append(errs, GoBlockCompileError{DocPath: block.DocPath, Line: block.Line, Err: buildErr})
		}
	}
	return errs, nil
}

// compileSampleInModule writes one sample to a temp module (replace → repoRoot) and `go build`s
// it, returning "" on success or the combined build output on failure.
func compileSampleInModule(repoRoot string, block GoBlock, index int) string {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("gwc-docsample-%d-", index))
	if err != nil {
		return "mktemp: " + err.Error()
	}
	defer os.RemoveAll(tempDir)

	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(block.Source), 0o644); err != nil {
		return "write sample: " + err.Error()
	}
	goMod := "module gwcdocsample\n\ngo 1.26\n\nrequire github.com/monstercameron/GoWebComponents v0.0.0\n" +
		"replace github.com/monstercameron/GoWebComponents => " + filepath.ToSlash(repoRoot) + "\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return "write go.mod: " + err.Error()
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = tempDir
	env := append(os.Environ(), "GOFLAGS=-mod=mod")
	if !block.Native {
		env = append(env, "GOOS=js", "GOARCH=wasm")
	}
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// ValidateGoBlocks parses every complete-file ```go sample under root and returns the ones that
// no longer parse — the documentation-can't-silently-lie guard for code samples. It uses
// go/parser (offline, deterministic, no module resolution) so the lane stays fast and
// dependency-free, matching the path-reference guard's design.
func ValidateGoBlocks(root string) ([]GoBlockError, error) {
	blocks, err := ScanGoBlocks(root)
	if err != nil {
		return nil, err
	}
	var errs []GoBlockError
	for _, block := range blocks {
		fset := token.NewFileSet()
		if _, parseErr := parser.ParseFile(fset, "sample.go", block.Source, parser.AllErrors); parseErr != nil {
			errs = append(errs, GoBlockError{DocPath: block.DocPath, Line: block.Line, Err: parseErr.Error()})
		}
	}
	return errs, nil
}
