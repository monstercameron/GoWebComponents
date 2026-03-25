//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"flag"
	"fmt"

	"path/filepath"
)

func main() {
	outDir := flag.String("out", filepath.Join("examples", "102-static-export-site", "dist"), "output directory for prerendered files")
	flag.Parse()

	summary, err := exportExampleSite(*outDir)
	if err != nil {
		panic(err)
	}

	fmt.Printf("exported %d html files to %s\n", len(summary.HTMLFiles), *outDir)
}
