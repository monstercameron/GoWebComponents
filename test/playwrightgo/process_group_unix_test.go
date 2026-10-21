//go:build !windows

package playwrightgo_test

import (
	"os/exec"
	"syscall"
)

// configureSuiteProcessGroup starts a harness command in its own process group.
func configureSuiteProcessGroup(parseCmd *exec.Cmd) {
	parseCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateSuiteProcessGroup kills the harness and all descendants in its process group.
func terminateSuiteProcessGroup(parseCmd *exec.Cmd) {
	_ = syscall.Kill(-parseCmd.Process.Pid, syscall.SIGKILL)
}
