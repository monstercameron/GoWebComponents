//go:build windows

package playwrightgo_test

import (
	"os/exec"
)

// configureSuiteProcessGroup leaves Windows process-tree termination to taskkill.
func configureSuiteProcessGroup(parseCmd *exec.Cmd) {}

// terminateSuiteProcessGroup terminates the root process when taskkill was unavailable.
func terminateSuiteProcessGroup(parseCmd *exec.Cmd) {
	_ = parseCmd.Process.Kill()
}
