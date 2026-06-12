// Command sbom writes a CycloneDX SBOM for the module to the given path.
// Usage: sbom <output-path> [repo-root].
package main

import (
	"fmt"
	"os"

	"github.com/monstercameron/GoWebComponents/tools/sbom"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sbom <output-path> [repo-root]")
		os.Exit(2)
	}
	parseOut := os.Args[1]
	parseRoot := "."
	if len(os.Args) >= 3 {
		parseRoot = os.Args[2]
	}
	if parseErr := sbom.WriteFile(parseRoot, parseOut); parseErr != nil {
		fmt.Fprintln(os.Stderr, "sbom:", parseErr)
		os.Exit(1)
	}
	fmt.Printf("sbom: wrote CycloneDX SBOM to %s\n", parseOut)
}
