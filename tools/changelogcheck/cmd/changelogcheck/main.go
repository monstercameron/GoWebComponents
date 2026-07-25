// Command changelogcheck exits non-zero when CHANGELOG.md has no section entry
// for the given version, so a release workflow can gate a tag on a changelog
// entry. Usage: changelogcheck <changelog-path> <version>.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/monstercameron/GoWebComponents/v5/tools/changelogcheck"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(parseArgs []string, parseStdout io.Writer, parseStderr io.Writer) int {
	if len(parseArgs) != 3 {
		fmt.Fprintln(parseStderr, "usage: changelogcheck <changelog-path> <version>")
		return 2
	}
	parsePath := parseArgs[1]
	parseVersion := parseArgs[2]
	parseFound, parseErr := changelogcheck.CheckFile(parsePath, parseVersion)
	if parseErr != nil {
		fmt.Fprintln(parseStderr, "changelogcheck:", parseErr)
		return 2
	}
	if !parseFound {
		fmt.Fprintf(parseStderr, "changelogcheck: no CHANGELOG entry found for version %q in %s\n", parseVersion, parsePath)
		return 1
	}
	fmt.Fprintf(parseStdout, "changelogcheck: found CHANGELOG entry for %q\n", parseVersion)
	return 0
}
