package main

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

const testEnvUnlinkat = `go: unlinkat C:\Temp\go-build1\b001\x.test.exe: Access is denied.`

// TestBenignGoCleanupPassWithUnlinkat proves a passing run plus the cleanup error is tolerated.
func TestBenignGoCleanupPassWithUnlinkat(parseT *testing.T) {
	parseOut := "go test failed: ok  \texample.com/x\t0.5s\n" + testEnvUnlinkat
	if !isBenignGoCleanupOutput(parseOut) {
		parseT.Fatal("pass plus unlinkat must be benign")
	}
	if parseErr := tolerateGoCleanupError(parseT, errors.New("exit status 1"), parseOut); parseErr != nil {
		parseT.Fatalf("expected tolerated, got %v", parseErr)
	}
}

// TestBenignGoCleanupRealFailureStaysFailed proves FAIL plus unlinkat is still a failure.
func TestBenignGoCleanupRealFailureStaysFailed(parseT *testing.T) {
	parseOut := "--- FAIL: TestA (0.00s)\n    a_test.go:1: boom\nFAIL\nFAIL\texample.com/x\t0.1s\n" + testEnvUnlinkat
	parseWant := errors.New("exit status 1")
	if isBenignGoCleanupOutput(parseOut) {
		parseT.Fatal("FAIL must not be benign")
	}
	if parseErr := tolerateGoCleanupError(parseT, parseWant, parseOut); parseErr != parseWant {
		parseT.Fatalf("real failure must be returned unchanged, got %v", parseErr)
	}
	if isBenignGoCleanupOutput("panic: boom\nok  \tx\t1s\n" + testEnvUnlinkat) {
		parseT.Fatal("panic must not be benign")
	}
	if isBenignGoCleanupOutput(testEnvUnlinkat) {
		parseT.Fatal("cleanup line without a passing run must not be benign")
	}
	if isCleanupOnlyStderr("compile error: x\n" + testEnvUnlinkat) {
		parseT.Fatal("extra stderr error must not be cleanup-only")
	}
	if !isCleanupOnlyStderr(testEnvUnlinkat) {
		parseT.Fatal("pure cleanup stderr should be cleanup-only")
	}
	if isCleanupOnlyStderr(testEnvUnlinkat + "\nexit status 1") {
		parseT.Fatal("a child exit status line means the program failed")
	}
}

// TestRemoveAllWithRetryRemovesDir proves the retry wrapper succeeds on a missing directory.
func TestRemoveAllWithRetryRemovesDir(parseT *testing.T) {
	if parseErr := removeAllWithRetry(filepath.Join(parseT.TempDir(), "missing"), time.Second); parseErr != nil {
		parseT.Fatalf("remove of missing dir: %v", parseErr)
	}
}
