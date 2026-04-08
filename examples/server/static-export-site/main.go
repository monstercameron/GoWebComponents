//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"flag"
	"fmt"

	"path/filepath"
)

func main() {
	parseOutDir := flag.String("out", filepath.Join("examples", "102-static-export-site", "dist"), "output directory for prerendered files")
	flag.Parse()

	parseSummary, parseErr := exportExampleSite(*parseOutDir)
	if parseErr != nil {
		panic(parseErr)
	}

	fmt.Printf("exported %d html files to %s\n", len(parseSummary.HTMLFiles), *parseOutDir)
}
