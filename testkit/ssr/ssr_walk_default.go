//go:build !js || !wasm

package ssr

import (
	"os"
	"path/filepath"
	"strings"
)

func collectStaticExportFiles(parseRoot string, parseRel string, parseExport *StaticExport) error {
	parseDirPath := parseRoot
	if parseRel != "" {
		parseDirPath = filepath.Join(parseRoot, parseRel)
	}
	parseEntries, parseErr := os.ReadDir(parseDirPath)
	if parseErr != nil {
		return parseErr
	}
	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		parseChildRel := parseName
		if parseRel != "" {
			parseChildRel = filepath.Join(parseRel, parseName)
		}
		if parseEntry.IsDir() {
			if parseErr2 := collectStaticExportFiles(parseRoot, parseChildRel, parseExport); parseErr2 != nil {
				return parseErr2
			}
			continue
		}
		parsePath := filepath.Join(parseRoot, parseChildRel)
		parseData, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		parseRelative := filepath.ToSlash(parseChildRel)
		if strings.HasSuffix(parseRelative, ".html") {
			parseExport.HTMLFiles[parseRelative] = Snapshot{HTML: string(parseData)}
			continue
		}
		if strings.HasPrefix(parseRelative, "bootstrap/") {
			parseExport.Bootstrap[parseRelative] = append([]byte(nil), parseData...)
		}
	}
	return nil
}
