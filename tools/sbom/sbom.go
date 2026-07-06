// Package sbom generates a CycloneDX software bill of materials for the module's
// dependency graph, so a release can attach an SBOM whose package list matches
// go.mod. It shells out to `go list -m -json all` (the authoritative module
// list) rather than parsing go.mod directly, so it captures the full resolved
// graph including transitive dependencies.
package sbom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
)

// Component is one CycloneDX component (a Go module dependency).
type Component struct {
	Type    string `json:"type"`
	BOMRef  string `json:"bom-ref"`
	Name    string `json:"name"`
	Version string `json:"version"`
	PURL    string `json:"purl"`
}

// Document is the minimal CycloneDX 1.5 BOM shape needed to enumerate the
// dependency components.
type Document struct {
	BOMFormat   string      `json:"bomFormat"`
	SpecVersion string      `json:"specVersion"`
	Version     int         `json:"version"`
	Components  []Component `json:"components"`
}

// goModule mirrors the fields of `go list -m -json` we consume. Replace is the
// resolved replacement target (nil when the module is not replaced).
type goModule struct {
	Path    string    `json:"Path"`
	Version string    `json:"Version"`
	Main    bool      `json:"Main"`
	Replace *goModule `json:"Replace"`
}

// Generate runs the module list in parseRepoRoot and returns a CycloneDX BOM.
func Generate(parseRepoRoot string) (Document, error) {
	parseCmd := exec.Command("go", "list", "-m", "-json", "all")
	parseCmd.Dir = parseRepoRoot
	var parseStdout bytes.Buffer
	parseCmd.Stdout = &parseStdout
	parseCmd.Stderr = os.Stderr
	if parseErr := parseCmd.Run(); parseErr != nil {
		return Document{}, fmt.Errorf("go list -m -json all: %w", parseErr)
	}
	return BuildDocument(parseStdout.Bytes())
}

// BuildDocument decodes the `go list -m -json all` stream into a CycloneDX BOM.
// The main module is excluded (it is the subject, not a dependency), as are
// modules without a resolved version.
func BuildDocument(parseListJSON []byte) (Document, error) {
	parseDecoder := json.NewDecoder(bytes.NewReader(parseListJSON))
	var parseComponents []Component
	for {
		var parseModule goModule
		parseErr := parseDecoder.Decode(&parseModule)
		if parseErr == io.EOF {
			break
		}
		if parseErr != nil {
			return Document{}, fmt.Errorf("decode module list: %w", parseErr)
		}
		if parseModule.Main {
			continue
		}
		// Resolve replace directives so the SBOM reflects REAL provenance rather
		// than the original module's placeholder pseudo-version. A local-path
		// replace (Replace set, no version) is in-tree/vendored source, not a
		// fetchable upstream package — emitting a synthetic pkg:golang PURL for it
		// would actively misrepresent provenance, so it is skipped.
		parsePath := parseModule.Path
		parseVersion := parseModule.Version
		if parseModule.Replace != nil {
			if parseModule.Replace.Version == "" {
				continue
			}
			parsePath = parseModule.Replace.Path
			parseVersion = parseModule.Replace.Version
		}
		if parseVersion == "" {
			continue
		}
		parsePURL := "pkg:golang/" + parsePath + "@" + parseVersion
		parseComponents = append(parseComponents, Component{
			Type:    "library",
			BOMRef:  parsePURL,
			Name:    parsePath,
			Version: parseVersion,
			PURL:    parsePURL,
		})
	}
	sort.Slice(parseComponents, func(parseA, parseB int) bool {
		return parseComponents[parseA].Name < parseComponents[parseB].Name
	})
	return Document{
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.5",
		Version:     1,
		Components:  parseComponents,
	}, nil
}

// WriteFile generates the BOM for parseRepoRoot and writes it as indented JSON.
func WriteFile(parseRepoRoot string, parsePath string) error {
	parseDoc, parseErr := Generate(parseRepoRoot)
	if parseErr != nil {
		return parseErr
	}
	parseData, parseErr := json.MarshalIndent(parseDoc, "", "  ")
	if parseErr != nil {
		return parseErr
	}
	return os.WriteFile(parsePath, append(parseData, '\n'), 0o644)
}
