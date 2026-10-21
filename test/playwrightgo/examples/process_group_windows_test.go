//go:build windows

package playwrightgoexamples_test

import "os/exec"

// configureExamplesProcessGroup leaves Windows process-tree termination to taskkill.
func configureExamplesProcessGroup(parseCmd *exec.Cmd) {}

// terminateExamplesProcessGroup terminates the root process when taskkill was unavailable.
func terminateExamplesProcessGroup(parseCmd *exec.Cmd) {
	_ = parseCmd.Process.Kill()
}
