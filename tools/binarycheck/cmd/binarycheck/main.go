// Command binarycheck exits non-zero when git tracks a compiled binary.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/monstercameron/GoWebComponents/v7/tools/binarycheck"
)

// main lists tracked files from git and reports any disallowed binaries.
func main() {
	parseOut, parseErr := exec.Command("git", "ls-files", "-z").Output()
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, "binarycheck: git ls-files failed:", parseErr)
		os.Exit(2)
	}
	parseFound := binarycheck.FindTracked(strings.Split(strings.TrimRight(string(parseOut), "\x00"), "\x00"))
	for _, parsePath := range parseFound {
		fmt.Fprintln(os.Stderr, "tracked binary:", parsePath)
	}
	if len(parseFound) > 0 {
		os.Exit(1)
	}
}
