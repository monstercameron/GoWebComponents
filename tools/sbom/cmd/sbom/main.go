// Command sbom writes a CycloneDX SBOM for the module to the given path.
// Usage: sbom <output-path> [repo-root].
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/monstercameron/GoWebComponents/v6/tools/sbom"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(parseArgs []string, parseStdout io.Writer, parseStderr io.Writer) int {
	if len(parseArgs) < 2 {
		fmt.Fprintln(parseStderr, "usage: sbom <output-path> [repo-root]")
		return 2
	}
	parseOut := parseArgs[1]
	parseRoot := "."
	if len(parseArgs) >= 3 {
		parseRoot = parseArgs[2]
	}
	if parseErr := sbom.WriteFile(parseRoot, parseOut); parseErr != nil {
		fmt.Fprintln(parseStderr, "sbom:", parseErr)
		return 1
	}
	fmt.Fprintf(parseStdout, "sbom: wrote CycloneDX SBOM to %s\n", parseOut)
	return 0
}
