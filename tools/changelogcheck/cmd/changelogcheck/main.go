// Command changelogcheck exits non-zero when CHANGELOG.md has no section entry
// for the given version, so a release workflow can gate a tag on a changelog
// entry. Usage: changelogcheck <changelog-path> <version>.
package main

import (
	"fmt"
	"os"

	"github.com/monstercameron/GoWebComponents/tools/changelogcheck"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: changelogcheck <changelog-path> <version>")
		os.Exit(2)
	}
	parsePath := os.Args[1]
	parseVersion := os.Args[2]
	parseFound, parseErr := changelogcheck.CheckFile(parsePath, parseVersion)
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, "changelogcheck:", parseErr)
		os.Exit(2)
	}
	if !parseFound {
		fmt.Fprintf(os.Stderr, "changelogcheck: no CHANGELOG entry found for version %q in %s\n", parseVersion, parsePath)
		os.Exit(1)
	}
	fmt.Printf("changelogcheck: found CHANGELOG entry for %q\n", parseVersion)
}
