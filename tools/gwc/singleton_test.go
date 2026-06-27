package main

import (
	"os"
	"testing"
)

func TestLauncherSingletonSlotKeySanitizesAndSeparates(parseT *testing.T) {
	parseServe := launcherSingletonSlotKey("serve", "127.0.0.1", "8090")
	parseDev := launcherSingletonSlotKey("dev", "127.0.0.1", "8090")
	parseOtherPort := launcherSingletonSlotKey("serve", "127.0.0.1", "8091")
	if parseServe == parseDev || parseServe == parseOtherPort || parseDev == parseOtherPort {
		parseT.Fatalf("expected distinct slot keys per role/port, got %q %q %q", parseServe, parseDev, parseOtherPort)
	}
	if parseServe != "serve-127-0-0-1-8090" {
		parseT.Fatalf("unexpected sanitized slot key: %q", parseServe)
	}
	if parseEmpty := launcherSingletonSlotKey("", "", ""); parseEmpty != "singleton" {
		parseT.Fatalf("expected fallback key for empty inputs, got %q", parseEmpty)
	}
}

func TestAcquireLauncherSingletonTerminatesLivePredecessor(parseT *testing.T) {
	parseL := launcher{repoRoot: parseT.TempDir()}

	parsePriorPath, parseErr := parseL.resolveLauncherSingletonPath("serve", "127.0.0.1", "0")
	if parseErr != nil {
		parseT.Fatalf("resolve singleton path: %v", parseErr)
	}
	if parseErr2 := writeLauncherSingletonRecord(parsePriorPath, launcherSingletonRecord{
		Role: "serve", Host: "127.0.0.1", Port: "0", PID: 424242,
	}); parseErr2 != nil {
		parseT.Fatalf("seed prior record: %v", parseErr2)
	}

	parseTerminated := 0
	parseRestore := swapSingletonStubs(
		func(parsePID int) bool { return parsePID == 424242 },
		func(parsePID int) error { parseTerminated = parsePID; return nil },
	)
	defer parseRestore()

	parseHandle, parseErr3 := parseL.acquireLauncherSingleton("serve", "127.0.0.1", "0")
	if parseErr3 != nil {
		parseT.Fatalf("acquire singleton: %v", parseErr3)
	}
	if parseTerminated != 424242 {
		parseT.Fatalf("expected predecessor pid 424242 to be terminated, got %d", parseTerminated)
	}

	parseRecord, parseFound := readLauncherSingletonRecord(parsePriorPath)
	if !parseFound || parseRecord.PID != os.Getpid() {
		parseT.Fatalf("expected record to be owned by self (%d), got %+v found=%t", os.Getpid(), parseRecord, parseFound)
	}

	parseHandle.Release()
	if _, parseStillFound := readLauncherSingletonRecord(parsePriorPath); parseStillFound {
		parseT.Fatal("expected record removed after Release")
	}
}

func TestAcquireLauncherSingletonIgnoresDeadPredecessor(parseT *testing.T) {
	parseL := launcher{repoRoot: parseT.TempDir()}
	parsePath, _ := parseL.resolveLauncherSingletonPath("dev", "127.0.0.1", "0")
	if parseErr := writeLauncherSingletonRecord(parsePath, launcherSingletonRecord{
		Role: "dev", Host: "127.0.0.1", Port: "0", PID: 999999,
	}); parseErr != nil {
		parseT.Fatalf("seed record: %v", parseErr)
	}

	parseTerminateCalled := false
	parseRestore := swapSingletonStubs(
		func(int) bool { return false }, // predecessor is dead
		func(int) error { parseTerminateCalled = true; return nil },
	)
	defer parseRestore()

	parseHandle, parseErr := parseL.acquireLauncherSingleton("dev", "127.0.0.1", "0")
	if parseErr != nil {
		parseT.Fatalf("acquire singleton: %v", parseErr)
	}
	defer parseHandle.Release()
	if parseTerminateCalled {
		parseT.Fatal("expected no termination for a dead predecessor")
	}
}

// swapSingletonStubs overrides the PID liveness and terminate hooks and returns
// a restore func.
func swapSingletonStubs(parseCheck func(int) bool, parseTerminate func(int) error) func() {
	parsePrevCheck := launcherSingletonCheckPIDRunning
	parsePrevTerminate := launcherSingletonTerminatePIDTree
	launcherSingletonCheckPIDRunning = parseCheck
	launcherSingletonTerminatePIDTree = parseTerminate
	return func() {
		launcherSingletonCheckPIDRunning = parsePrevCheck
		launcherSingletonTerminatePIDTree = parsePrevTerminate
	}
}
