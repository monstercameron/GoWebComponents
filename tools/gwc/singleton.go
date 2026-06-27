package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// A launcher singleton guard enforces "at most one live server per (role, host,
// port)" on a dev machine. Re-running a long-lived server command (gwc serve,
// gwc dev) terminates any predecessor that is still holding the port instead of
// spawning a second instance alongside an orphaned runaway process.
//
// The guard records the owning PID in a small JSON file under the launcher
// runtime directory. On acquire it reads any prior record, kills that process
// tree, and (as a backstop for the case where the recorder already died but
// left an orphaned child still bound to the port) frees the port directly.

type launcherSingletonRecord struct {
	Role      string `json:"role"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	PID       int    `json:"pid"`
	StartedAt string `json:"startedAt"`
}

// launcherSingletonHandle owns a held singleton slot. Call release on clean
// shutdown to remove the record; acquireLauncherSingleton also removes it
// automatically on SIGINT/SIGTERM so an interrupted server never leaves a stale
// record that a successor would treat as a live owner.
type launcherSingletonHandle struct {
	path    string
	release sync.Once
}

// Release removes the singleton record for a cleanly exiting server.
func (parseHandle *launcherSingletonHandle) Release() {
	if parseHandle == nil {
		return
	}
	parseHandle.release.Do(func() {
		_ = os.Remove(parseHandle.path)
	})
}

// Overridable for tests.
var launcherSingletonCheckPIDRunning = checkLauncherPIDRunning

var launcherSingletonTerminatePIDTree = terminateLauncherPIDTree

var launcherSingletonNowUTC = func() time.Time { return time.Now().UTC() }

// acquireLauncherSingleton ensures this process is the sole owner of the
// (role, host, port) slot. Any live predecessor is terminated (kill-old,
// take-over), the port is confirmed free, and the current PID is recorded.
func (parseL launcher) acquireLauncherSingleton(parseRole string, parseHost string, parsePort string) (*launcherSingletonHandle, error) {
	parseRecordPath, parseErr := parseL.resolveLauncherSingletonPath(parseRole, parseHost, parsePort)
	if parseErr != nil {
		return nil, parseErr
	}

	parsePrior, parseFound := readLauncherSingletonRecord(parseRecordPath)
	parseSelf := os.Getpid()
	if parseFound && parsePrior.PID != parseSelf && launcherSingletonCheckPIDRunning(parsePrior.PID) {
		fmt.Printf("GWC singleton: terminating existing %s owner (pid %d) on %s\n",
			parseRole, parsePrior.PID, joinHostPort(parseHost, parsePort))
		if parseErr2 := launcherSingletonTerminatePIDTree(parsePrior.PID); parseErr2 != nil {
			return nil, fmt.Errorf("terminate existing %s owner (pid %d): %w", parseRole, parsePrior.PID, parseErr2)
		}
	}

	// Backstop: even after killing the recorded owner, an orphaned child it
	// spawned (e.g. the compiled server behind `go run`) may still hold the
	// port. Wait for the port to drain and, failing that, free it directly.
	if parseErr2 := waitLauncherPortFree(parseHost, parsePort, 5*time.Second); parseErr2 != nil {
		if parseErr3 := freeLauncherPort(parsePort); parseErr3 != nil {
			return nil, fmt.Errorf("free %s for %s: %w", joinHostPort(parseHost, parsePort), parseRole, parseErr3)
		}
		if parseErr3 := waitLauncherPortFree(parseHost, parsePort, 5*time.Second); parseErr3 != nil {
			return nil, fmt.Errorf("port %s still in use after reclaim for %s: %w", joinHostPort(parseHost, parsePort), parseRole, parseErr3)
		}
	}

	parseRecord := launcherSingletonRecord{
		Role:      parseRole,
		Host:      parseHost,
		Port:      parsePort,
		PID:       parseSelf,
		StartedAt: launcherSingletonNowUTC().Format(time.RFC3339),
	}
	if parseErr2 := writeLauncherSingletonRecord(parseRecordPath, parseRecord); parseErr2 != nil {
		return nil, parseErr2
	}

	parseHandle := &launcherSingletonHandle{path: parseRecordPath}
	installLauncherSingletonSignalCleanup(parseHandle)
	return parseHandle, nil
}

// installLauncherSingletonSignalCleanup removes the singleton record when the
// server is interrupted, then restores default signal handling and re-raises so
// the process still exits with conventional signal semantics.
func installLauncherSingletonSignalCleanup(parseHandle *launcherSingletonHandle) {
	parseSignals := make(chan os.Signal, 1)
	signal.Notify(parseSignals, os.Interrupt, syscall.SIGTERM)
	go func() {
		parseSig := <-parseSignals
		parseHandle.Release()
		signal.Stop(parseSignals)
		if parseProcess, parseErr := os.FindProcess(os.Getpid()); parseErr == nil {
			_ = parseProcess.Signal(parseSig)
		}
	}()
}

// resolveLauncherSingletonPath resolves the per-slot record file path.
func (parseL launcher) resolveLauncherSingletonPath(parseRole string, parseHost string, parsePort string) (string, error) {
	parseRootPath := strings.TrimSpace(parseL.repoRoot)
	// Without a resolved repo root (e.g. a bare launcher in tests) fall back to a
	// machine-global temp location rather than failing — the guard is a dev-time
	// convenience and must never block the server from starting.
	if parseRootPath == "" {
		parseRuntimeDir := filepath.Join(os.TempDir(), "gwc-runtime", "singletons")
		return filepath.Join(parseRuntimeDir, launcherSingletonSlotKey(parseRole, parseHost, parsePort)+".json"), nil
	}
	parseRuntimeDir := filepath.Join(parseRootPath, "bin", "runtime", "singletons")
	if parseArtifactRoot, parseHasArtifactRoot, parseErr := resolveLauncherArtifactRoot(parseRootPath); parseErr != nil {
		return "", fmt.Errorf("resolve singleton artifact root: %w", parseErr)
	} else if parseHasArtifactRoot {
		parseRuntimeDir = filepath.Join(parseArtifactRoot, "runtime", "singletons")
	}
	return filepath.Join(parseRuntimeDir, launcherSingletonSlotKey(parseRole, parseHost, parsePort)+".json"), nil
}

var launcherSingletonUnsafeKeyChars = regexp.MustCompile(`[^a-z0-9]+`)

// launcherSingletonSlotKey builds a filesystem-safe slot key from the role and
// bind address so distinct (role, host, port) servers never share a record.
func launcherSingletonSlotKey(parseRole string, parseHost string, parsePort string) string {
	parseRaw := strings.ToLower(strings.TrimSpace(parseRole) + "-" + strings.TrimSpace(parseHost) + "-" + strings.TrimSpace(parsePort))
	parseKey := strings.Trim(launcherSingletonUnsafeKeyChars.ReplaceAllString(parseRaw, "-"), "-")
	if parseKey == "" {
		return "singleton"
	}
	return parseKey
}

// readLauncherSingletonRecord reads a slot record when present. A missing or
// corrupt record is treated as "no live owner" so a damaged file never wedges
// startup.
func readLauncherSingletonRecord(parseRecordPath string) (launcherSingletonRecord, bool) {
	parsePayload, parseErr := os.ReadFile(parseRecordPath)
	if parseErr != nil {
		return launcherSingletonRecord{}, false
	}
	var parseRecord launcherSingletonRecord
	if parseErr2 := json.Unmarshal(parsePayload, &parseRecord); parseErr2 != nil {
		return launcherSingletonRecord{}, false
	}
	if parseRecord.PID <= 0 {
		return launcherSingletonRecord{}, false
	}
	return parseRecord, true
}

// writeLauncherSingletonRecord persists a slot record, creating the runtime
// directory as needed.
func writeLauncherSingletonRecord(parseRecordPath string, parseRecord launcherSingletonRecord) error {
	if parseErr := os.MkdirAll(filepath.Dir(parseRecordPath), 0755); parseErr != nil {
		return fmt.Errorf("create singleton runtime directory: %w", parseErr)
	}
	parsePayload, parseErr := json.MarshalIndent(parseRecord, "", "  ")
	if parseErr != nil {
		return fmt.Errorf("encode singleton record: %w", parseErr)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr2 := os.WriteFile(parseRecordPath, parsePayload, 0644); parseErr2 != nil {
		return fmt.Errorf("write singleton record: %w", parseErr2)
	}
	return nil
}

// waitLauncherPortFree polls until the TCP port can be bound or the timeout
// elapses. It returns an error only when the port is still occupied at the
// deadline.
func waitLauncherPortFree(parseHost string, parsePort string, parseTimeout time.Duration) error {
	parseDeadline := launcherSingletonNowUTC().Add(parseTimeout)
	parseAddr := joinHostPort(parseHost, parsePort)
	for {
		if launcherPortIsFree(parseAddr) {
			return nil
		}
		if launcherSingletonNowUTC().After(parseDeadline) {
			return fmt.Errorf("port %s still in use", parseAddr)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

// launcherPortIsFree reports whether the bind address is currently available.
// Only a genuine address-already-in-use error counts as occupied; any other
// bind failure (e.g. a malformed address) is reported as free so the caller's
// real net.Listen surfaces the actual error instead of looping here.
func launcherPortIsFree(parseAddr string) bool {
	parseListener, parseErr := net.Listen("tcp", parseAddr)
	if parseErr != nil {
		return !errors.Is(parseErr, syscall.EADDRINUSE)
	}
	_ = parseListener.Close()
	return true
}

// freeLauncherPort terminates the process(es) currently bound to the port. It
// is the last-resort reclaim used only when a runaway orphan still holds the
// port after its recorded parent was killed.
func freeLauncherPort(parsePort string) error {
	parsePIDs, parseErr := findLauncherPortOwnerPIDs(parsePort)
	if parseErr != nil {
		return parseErr
	}
	parseSelf := os.Getpid()
	for _, parsePID := range parsePIDs {
		if parsePID <= 0 || parsePID == parseSelf {
			continue
		}
		if parseErr2 := launcherSingletonTerminatePIDTree(parsePID); parseErr2 != nil {
			return fmt.Errorf("terminate port owner pid %d: %w", parsePID, parseErr2)
		}
	}
	return nil
}

var launcherPortListeningLine = regexp.MustCompile(`LISTENING\s+(\d+)\s*$`)

// findLauncherPortOwnerPIDs returns the PIDs listening on the port. It shells
// out to platform tooling (netstat on Windows, lsof on POSIX) because Go has no
// portable API for reverse port-to-PID lookup.
func findLauncherPortOwnerPIDs(parsePort string) ([]int, error) {
	parsePort = strings.TrimSpace(parsePort)
	if parsePort == "" {
		return nil, nil
	}
	parseSeen := map[int]struct{}{}
	var parsePIDs []int
	appendPID := func(parseToken string) {
		parsePID, parseErr := strconv.Atoi(strings.TrimSpace(parseToken))
		if parseErr != nil || parsePID <= 0 {
			return
		}
		if _, parseExists := parseSeen[parsePID]; parseExists {
			return
		}
		parseSeen[parsePID] = struct{}{}
		parsePIDs = append(parsePIDs, parsePID)
	}

	if runtime.GOOS == "windows" {
		parseOutput, _ := launcherRunCommand("netstat", []string{"-ano", "-p", "tcp"}, "", nil)
		parseSuffix := ":" + parsePort
		for parseLine := range strings.SplitSeq(parseOutput, "\n") {
			parseLine = strings.TrimSpace(parseLine)
			if !strings.Contains(parseLine, "LISTENING") {
				continue
			}
			parseFields := strings.Fields(parseLine)
			if len(parseFields) < 2 || !strings.HasSuffix(parseFields[1], parseSuffix) {
				continue
			}
			if parseMatch := launcherPortListeningLine.FindStringSubmatch(parseLine); parseMatch != nil {
				appendPID(parseMatch[1])
			}
		}
		return parsePIDs, nil
	}

	parseOutput, _ := launcherRunCommand("lsof", []string{"-nP", "-iTCP:" + parsePort, "-sTCP:LISTEN", "-t"}, "", nil)
	for parseLine := range strings.SplitSeq(parseOutput, "\n") {
		appendPID(parseLine)
	}
	return parsePIDs, nil
}
