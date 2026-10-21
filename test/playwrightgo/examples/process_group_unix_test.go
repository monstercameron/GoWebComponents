//go:build !windows

package playwrightgoexamples_test

import (
	"os/exec"
	"syscall"
)

// configureExamplesProcessGroup starts an example server in its own process group.
func configureExamplesProcessGroup(parseCmd *exec.Cmd) {
	parseCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateExamplesProcessGroup kills an example server and all descendants in its process group.
func terminateExamplesProcessGroup(parseCmd *exec.Cmd) {
	_ = syscall.Kill(-parseCmd.Process.Pid, syscall.SIGKILL)
}
