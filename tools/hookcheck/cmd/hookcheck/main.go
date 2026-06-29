// Command hookcheck reports GoWebComponents "rules of hooks" violations: any
// Use* hook (including ui.UseEvent behind an On* handler) called inside a loop.
//
// Usage:
//
//	hookcheck [path ...]   # defaults to "."
//
// Exit code 0 = clean, 1 = violations found, 2 = error.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/monstercameron/GoWebComponents/v4/tools/hookcheck"
)

func main() {
	flag.Parse()
	parseRoots := flag.Args()
	if len(parseRoots) == 0 {
		parseRoots = []string{"."}
	}

	parseTotal := 0
	for _, parseRoot := range parseRoots {
		parseFindings, parseErr := hookcheck.CheckDir(parseRoot)
		if parseErr != nil {
			fmt.Fprintln(os.Stderr, "hookcheck:", parseErr)
			os.Exit(2)
		}
		for _, parseFinding := range parseFindings {
			fmt.Println(parseFinding.String())
			parseTotal++
		}
	}

	if parseTotal > 0 {
		fmt.Fprintf(os.Stderr, "hookcheck: %d hook-in-loop violation(s)\n", parseTotal)
		os.Exit(1)
	}
}
