package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const desktopDevLockName = ".gwc-desktop-dev.lock"

var (
	desktopDevBuild = func(parseL launcher, parseConfig desktopConfig) (desktopSummary, error) {
		return parseL.desktopBuild(parseConfig)
	}
	desktopDevStart        = startDesktopDevHost
	desktopDevPollInterval = 250 * time.Millisecond
)

// signalContext owns process-interruption notification until its returned cancellation runs.
func signalContext() (context.Context, context.CancelFunc) {
	parseCtx, parseCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	return parseCtx, parseCancel
}

// desktopDev validates the target and gives one supervisor ownership of its native children.
func (parseL launcher) desktopDev(parseConfig desktopConfig) (desktopSummary, error) {
	parseMetadata, parseErr := desktopLoadMetadata(parseConfig.root)
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr = desktopValidateCanonicalTarget(parseMetadata.Desktop); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseRelease, parseErr := acquireDesktopDevLock(parseConfig.root)
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	defer parseRelease()
	parseContext := parseConfig.context
	if parseContext == nil {
		var parseCancel context.CancelFunc
		parseContext, parseCancel = signalContext()
		defer parseCancel()
	}
	return parseL.runDesktopDev(parseContext, parseConfig)
}

// runDesktopDev reaps each host exactly once and observes shutdown during both builds and idle time.
func (parseL launcher) runDesktopDev(parseContext context.Context, parseConfig desktopConfig) (parseSummary desktopSummary, parseErr error) {
	parseConfig.context = parseContext
	parseSnapshot, parseReadErr := readDesktopDevSnapshot(parseConfig.root)
	if parseReadErr != nil {
		return desktopSummary{}, parseReadErr
	}
	var parseHost *exec.Cmd
	var parseFinished <-chan error
	defer func() {
		if parseHost != nil {
			if parseStopErr := stopDesktopDevHost(parseHost, parseFinished); parseErr == nil {
				parseErr = parseStopErr
			}
		}
	}()
	parseBuildAndStart := func() error {
		var parseBuildErr error
		parseSummary, parseBuildErr = desktopDevBuild(parseL, parseConfig)
		parseSummary.Action = "dev"
		if parseBuildErr != nil {
			return parseBuildErr
		}
		if parseContext.Err() != nil {
			return parseContext.Err()
		}
		parseHost, parseBuildErr = desktopDevStart(parseConfig.root, filepath.Join(parseConfig.root, filepath.FromSlash(parseSummary.Artifact)))
		if parseBuildErr != nil {
			return parseBuildErr
		}
		parseDone := make(chan error, 1)
		go func(parseCommand *exec.Cmd) { parseDone <- parseCommand.Wait() }(parseHost)
		parseFinished = parseDone
		return nil
	}
	if parseErr = parseBuildAndStart(); parseErr != nil {
		if parseContext.Err() != nil {
			return parseSummary, nil
		}
		return parseSummary, parseErr
	}
	parseTicker := time.NewTicker(desktopDevPollInterval)
	defer parseTicker.Stop()
	for {
		select {
		case <-parseContext.Done():
			return parseSummary, nil
		case parseExitErr := <-parseFinished:
			parseHost = nil
			if parseExitErr != nil {
				return parseSummary, fmt.Errorf("desktop host exited: %w", parseExitErr)
			}
			return parseSummary, nil
		case <-parseTicker.C:
			parseNext, parseReadErr := readDesktopDevSnapshot(parseConfig.root)
			if parseReadErr != nil {
				return parseSummary, parseReadErr
			}
			if !desktopDevChanged(parseSnapshot, parseNext) {
				continue
			}
			if parseErr = stopDesktopDevHost(parseHost, parseFinished); parseErr != nil {
				return parseSummary, parseErr
			}
			parseHost = nil
			parseSnapshot = parseNext
			if parseErr = parseBuildAndStart(); parseErr != nil {
				if parseContext.Err() != nil {
					return parseSummary, nil
				}
				return parseSummary, parseErr
			}
		}
	}
}

// stopDesktopDevHost terminates descendants and waits for the supervisor's single Wait call.
func stopDesktopDevHost(parseHost *exec.Cmd, parseFinished <-chan error) error {
	terminateLauncherProcessTree(parseHost)
	select {
	case <-parseFinished:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("desktop host did not finish after process-tree termination")
	}
}

// desktopDevSnapshot returns source mtimes while excluding generated outputs and bin artifacts.
func desktopDevSnapshot(parseRoot string) map[string]time.Time {
	parseSnapshot, _ := readDesktopDevSnapshot(parseRoot)
	return parseSnapshot
}

// readDesktopDevSnapshot preserves access failures rather than silently dropping watched source files.
func readDesktopDevSnapshot(parseRoot string) (map[string]time.Time, error) {
	parseSnapshot := map[string]time.Time{}
	parseErr := filepath.WalkDir(parseRoot, func(parsePath string, parseEntry os.DirEntry, parseErr error) error {
		if parseErr != nil {
			return parseErr
		}
		if parseEntry.IsDir() {
			if desktopDevExcludedDir(parseRoot, parsePath) {
				return filepath.SkipDir
			}
			return nil
		}
		if desktopDevExcludedFile(parseRoot, parsePath) {
			return nil
		}
		if parseInfo, parseInfoErr := parseEntry.Info(); parseInfoErr == nil {
			parseSnapshot[parsePath] = parseInfo.ModTime()
		} else {
			return parseInfoErr
		}
		return nil
	})
	return parseSnapshot, parseErr
}

// desktopDevChanged reports additions, removals, and mtime changes between snapshots.
func desktopDevChanged(parseBefore map[string]time.Time, parseAfter map[string]time.Time) bool {
	if len(parseBefore) != len(parseAfter) {
		return true
	}
	for parsePath, parseTime := range parseBefore {
		if parseAfter[parsePath] != parseTime {
			return true
		}
	}
	return false
}

// desktopDevExcludedDir omits generated outputs from rebuild ownership.
func desktopDevExcludedDir(parseRoot, parsePath string) bool {
	parseRelative, _ := filepath.Rel(parseRoot, parsePath)
	for _, parsePart := range strings.Split(filepath.ToSlash(parseRelative), "/") {
		switch parsePart {
		case ".git", "bin", "dist", "generated":
			return true
		}
	}
	return false
}

// desktopDevExcludedFile omits runtime and temporary artifacts.
func desktopDevExcludedFile(parseRoot, parsePath string) bool {
	if desktopDevExcludedDir(parseRoot, filepath.Dir(parsePath)) {
		return true
	}
	parseBase := filepath.Base(parsePath)
	return strings.HasSuffix(parseBase, ".wasm") || strings.HasSuffix(parseBase, ".exe") || strings.HasSuffix(parseBase, ".tmp") || parseBase == desktopDevLockName
}

// acquireDesktopDevLock reserves a project without taking over another supervisor.
func acquireDesktopDevLock(parseRoot string) (func(), error) {
	parsePath := filepath.Join(parseRoot, desktopDevLockName)
	if parseErr := desktopRejectSymlinkPath(parseRoot, parsePath); parseErr != nil {
		return nil, parseErr
	}
	parseFile, parseErr := os.OpenFile(parsePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if parseErr != nil {
		if errors.Is(parseErr, os.ErrExist) {
			return nil, fmt.Errorf("desktop dev already owned for %s; remove %s only after confirming no owner", parseRoot, parsePath)
		}
		return nil, fmt.Errorf("create desktop dev lock: %w", parseErr)
	}
	parsePayload, parseErr := json.Marshal(map[string]any{"pid": os.Getpid(), "root": parseRoot, "startedAt": time.Now().UTC().Format(time.RFC3339)})
	if parseErr == nil {
		_, parseErr = parseFile.Write(append(parsePayload, '\n'))
	}
	parseOwned, parseStatErr := parseFile.Stat()
	if parseErr == nil {
		parseErr = parseStatErr
	}
	if parseCloseErr := parseFile.Close(); parseErr == nil {
		parseErr = parseCloseErr
	}
	if parseErr != nil {
		_ = os.Remove(parsePath)
		return nil, fmt.Errorf("write desktop lock: %w", parseErr)
	}
	var parseOnce sync.Once
	return func() {
		parseOnce.Do(func() {
			if parseCurrent, parseErr := os.Lstat(parsePath); parseErr == nil && os.SameFile(parseOwned, parseCurrent) {
				_ = os.Remove(parsePath)
			}
		})
	}, nil
}

// startDesktopDevHost starts the visible development app and leaves Wait to its supervisor.
func startDesktopDevHost(parseRoot, parseArtifact string) (*exec.Cmd, error) {
	parseCmd := exec.Command(parseArtifact)
	parseCmd.Dir = parseRoot
	parseCmd.Stdout, parseCmd.Stderr = os.Stderr, os.Stderr
	if runtime.GOOS == "windows" {
		parseCmd.Env = desktopNativeEnv()
	}
	if parseErr := parseCmd.Start(); parseErr != nil {
		return nil, parseErr
	}
	return parseCmd, nil
}
