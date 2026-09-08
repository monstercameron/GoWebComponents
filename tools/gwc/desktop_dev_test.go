package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDesktopDevSnapshotExcludesGeneratedOutputs verifies the desktop supervisor contract.
func TestDesktopDevSnapshotExcludesGeneratedOutputs(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for _, parsePath := range []string{"frontend/main.go", "assets/src/app.css", "bin/app.exe", "assets/dist/app.wasm", "generated/bindings.js", "cmd/desktop/main.go"} {
		parseFull := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFull), 0o755); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseErr := os.WriteFile(parseFull, []byte(parsePath), 0o644); parseErr != nil {
			parseT.Fatal(parseErr)
		}
	}
	parseSnapshot := desktopDevSnapshot(parseRoot)
	if len(parseSnapshot) != 3 {
		parseT.Fatalf("snapshot files = %d, want 3: %#v", len(parseSnapshot), parseSnapshot)
	}
	if parseSnapshot[filepath.Join(parseRoot, "frontend", "main.go")] == (time.Time{}) {
		parseT.Fatal("frontend source omitted")
	}
}

// TestDesktopDevChangedDetectsAddModifyRemove verifies the desktop supervisor contract.
func TestDesktopDevChangedDetectsAddModifyRemove(parseT *testing.T) {
	parseNow := time.Now()
	parseLater := parseNow.Add(time.Second)
	if desktopDevChanged(map[string]time.Time{"a": parseNow}, map[string]time.Time{"a": parseNow}) {
		parseT.Fatal("identical snapshots changed")
	}
	if !desktopDevChanged(map[string]time.Time{"a": parseNow}, map[string]time.Time{"a": parseLater}) {
		parseT.Fatal("mtime change not detected")
	}
	if !desktopDevChanged(map[string]time.Time{"a": parseNow}, map[string]time.Time{"b": parseNow}) {
		parseT.Fatal("add/remove not detected")
	}
}

// TestAcquireDesktopDevLockRejectsSecondOwnerAndReleases verifies the desktop supervisor contract.
func TestAcquireDesktopDevLockRejectsSecondOwnerAndReleases(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseRelease, parseErr := acquireDesktopDevLock(parseRoot)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if _, parseSecondErr := acquireDesktopDevLock(parseRoot); parseSecondErr == nil {
		parseT.Fatal("second owner acquired lock")
	}
	parseRelease()
	parseRelease2, parseErr := acquireDesktopDevLock(parseRoot)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseRelease2()
}

// TestDesktopDevReapsRestartedHost verifies the deferred cleanup owns the current child, not the initial child.
func TestDesktopDevReapsRestartedHost(parseT *testing.T) {
	for _, parseFailRebuild := range []bool{false, true} {
		parseT.Run(fmt.Sprint("failure=", parseFailRebuild), func(parseT *testing.T) {
			parseRoot := parseT.TempDir()
			parseSource := filepath.Join(parseRoot, "source.go")
			if parseErr := os.WriteFile(parseSource, []byte("initial"), 0o644); parseErr != nil {
				parseT.Fatal(parseErr)
			}
			parseBuildBefore, parseStartBefore, parsePollBefore := desktopDevBuild, desktopDevStart, desktopDevPollInterval
			parseT.Cleanup(func() {
				desktopDevBuild, desktopDevStart, desktopDevPollInterval = parseBuildBefore, parseStartBefore, parsePollBefore
			})
			desktopDevPollInterval = 5 * time.Millisecond
			parseBuilds := 0
			desktopDevBuild = func(launcher, desktopConfig) (desktopSummary, error) {
				parseBuilds++
				if parseFailRebuild && parseBuilds == 2 {
					return desktopSummary{}, errors.New("rebuild failed")
				}
				return desktopSummary{Artifact: "fake.exe"}, nil
			}
			parseStarted := make(chan *exec.Cmd, 3)
			desktopDevStart = func(string, string) (*exec.Cmd, error) {
				parseExe, parseErr := os.Executable()
				if parseErr != nil {
					return nil, parseErr
				}
				parseCommand := exec.Command(parseExe, "-test.run=^TestDesktopProcessHelper$")
				parseCommand.Env = append(os.Environ(), "GWC_DESKTOP_PROCESS_HELPER=1")
				if parseErr := parseCommand.Start(); parseErr != nil {
					return nil, parseErr
				}
				parseStarted <- parseCommand
				return parseCommand, nil
			}
			parseContext, parseCancel := context.WithCancel(context.Background())
			defer parseCancel()
			parseResult := make(chan error, 1)
			go func() {
				_, parseErr := (launcher{}).runDesktopDev(parseContext, desktopConfig{root: parseRoot})
				parseResult <- parseErr
			}()
			var parseFirst *exec.Cmd
			select {
			case parseFirst = <-parseStarted:
			case <-time.After(3 * time.Second):
				parseT.Fatal("initial host missing")
			}
			parseNextTime := time.Now().Add(time.Second)
			if parseErr := os.Chtimes(parseSource, parseNextTime, parseNextTime); parseErr != nil {
				parseT.Fatal(parseErr)
			}
			var parseSecond *exec.Cmd
			if !parseFailRebuild {
				select {
				case parseSecond = <-parseStarted:
				case <-time.After(3 * time.Second):
					parseT.Fatal("restarted host missing")
				}
				parseCancel()
			}
			select {
			case parseErr := <-parseResult:
				if parseFailRebuild && (parseErr == nil || !strings.Contains(parseErr.Error(), "rebuild failed")) {
					parseT.Fatalf("rebuild result=%v", parseErr)
				}
				if !parseFailRebuild && parseErr != nil {
					parseT.Fatal(parseErr)
				}
			case <-time.After(3 * time.Second):
				parseT.Fatal("supervisor failed to terminate")
			}
			if parseFirst.ProcessState == nil || (parseSecond != nil && parseSecond.ProcessState == nil) {
				parseT.Fatal("owned child not reaped")
			}
		})
	}
}

// TestDesktopDevObservesHostExitAndBuildCancellation covers shutdown without another file change.
func TestDesktopDevObservesHostExitAndBuildCancellation(parseT *testing.T) {
	parseBuildBefore, parseStartBefore := desktopDevBuild, desktopDevStart
	parseT.Cleanup(func() { desktopDevBuild, desktopDevStart = parseBuildBefore, parseStartBefore })
	desktopDevBuild = func(launcher, desktopConfig) (desktopSummary, error) {
		return desktopSummary{Artifact: "fake.exe"}, nil
	}
	desktopDevStart = func(string, string) (*exec.Cmd, error) {
		parseExe, parseErr := os.Executable()
		if parseErr != nil {
			return nil, parseErr
		}
		parseCommand := exec.Command(parseExe, "-test.run=^TestDesktopProcessHelper$")
		if parseErr := parseCommand.Start(); parseErr != nil {
			return nil, parseErr
		}
		return parseCommand, nil
	}
	parseContext, parseCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer parseCancel()
	if _, parseErr := (launcher{}).runDesktopDev(parseContext, desktopConfig{root: parseT.TempDir()}); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseContext.Err() != nil {
		parseT.Fatal("host exit was not observed before context expired")
	}
	parseContext, parseCancel2 := context.WithCancel(context.Background())
	desktopDevBuild = func(_ launcher, parseConfig desktopConfig) (desktopSummary, error) {
		parseCancel2()
		<-parseConfig.context.Done()
		return desktopSummary{}, parseConfig.context.Err()
	}
	desktopDevStart = func(string, string) (*exec.Cmd, error) {
		parseT.Error("started a host after build cancellation")
		return nil, errors.New("unexpected start")
	}
	if _, parseErr := (launcher{}).runDesktopDev(parseContext, desktopConfig{root: parseT.TempDir()}); parseErr != nil {
		parseT.Fatal(parseErr)
	}
}
