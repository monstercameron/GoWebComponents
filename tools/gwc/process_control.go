package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// terminateLauncherProcessTree terminates a command process and, on Windows, its descendant processes.
func terminateLauncherProcessTree(parseCmd *exec.Cmd) {
	if parseCmd == nil || parseCmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(parseCmd.Process.Pid)).Run()
		return
	}
	_ = parseCmd.Process.Kill()
}

// terminateLauncherPIDTree terminates a process by PID and, on Windows, its descendant processes.
func terminateLauncherPIDTree(parsePID int) error {
	if parsePID <= 0 {
		return nil
	}
	if runtime.GOOS == "windows" {
		parseOutput, parseErr := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(parsePID)).CombinedOutput()
		if parseErr == nil {
			return nil
		}
		parseMessage := strings.ToLower(strings.TrimSpace(string(parseOutput)))
		if strings.Contains(parseMessage, "not found") || strings.Contains(parseMessage, "no running instance") {
			return nil
		}
		return fmt.Errorf("taskkill /pid %d: %s", parsePID, strings.TrimSpace(string(parseOutput)))
	}
	parseProcess, parseErr := os.FindProcess(parsePID)
	if parseErr != nil {
		return nil
	}
	if parseErr2 := parseProcess.Kill(); parseErr2 != nil && !errorsIsProcessDone(parseErr2) {
		return fmt.Errorf("kill pid %d: %w", parsePID, parseErr2)
	}
	return nil
}

// checkLauncherPIDRunning reports whether a process ID currently resolves to a live process.
func checkLauncherPIDRunning(parsePID int) bool {
	if parsePID <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		parseOutput, parseErr := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", parsePID), "/FO", "CSV", "/NH").CombinedOutput()
		if parseErr != nil && len(parseOutput) == 0 {
			return false
		}
		parseMessage := strings.ToLower(strings.TrimSpace(string(parseOutput)))
		if parseMessage == "" || strings.Contains(parseMessage, "no tasks are running") {
			return false
		}
		parsePIDToken := fmt.Sprintf("\"%d\"", parsePID)
		return strings.Contains(parseMessage, parsePIDToken)
	}
	parseProcess, parseErr := os.FindProcess(parsePID)
	if parseErr != nil {
		return false
	}
	parseSignalErr := parseProcess.Signal(syscall.Signal(0))
	if parseSignalErr == nil {
		return true
	}
	return !errorsIsProcessDone(parseSignalErr)
}

// errorsIsProcessDone reports whether an error means the target process is no longer running.
func errorsIsProcessDone(parseErr error) bool {
	if parseErr == nil {
		return false
	}
	if strings.Contains(strings.ToLower(parseErr.Error()), "process already finished") {
		return true
	}
	return false
}
