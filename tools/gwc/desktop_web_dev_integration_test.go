//go:build desktopintegration

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDesktopWebDevIntegration proves real serving and rebuilds outside the served asset directory.
func TestDesktopWebDevIntegration(parseTest *testing.T) {
	parseTest.Run("ordinary", func(parseTest *testing.T) { runDesktopWebDevIntegration(parseTest, false) })
	parseTest.Run("agent", func(parseTest *testing.T) { runDesktopWebDevIntegration(parseTest, true) })
}

// runDesktopWebDevIntegration executes the same live build and watcher checks for each launcher mode.
func runDesktopWebDevIntegration(parseTest *testing.T, isAgent bool) {
	parseRepo := mustDesktopRepoRoot(parseTest)
	parseRoot := filepath.Join(parseTest.TempDir(), "desktop-web-dev")
	if _, parseErr := (launcher{repoRoot: parseRepo}).desktopInit(desktopConfig{root: parseRoot}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	// An inherited desktop build tag must not leak into either the initial build or watcher rebuild.
	parseGuard := filepath.Join(parseRoot, "frontend", "desktop_leak.go")
	if parseErr := os.WriteFile(parseGuard, []byte("//go:build gwc_desktop\n\npackage main\nvar _ = desktopTagMustNotLeakIntoWeb\n"), 0o644); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseAddress := parseListener.Addr().String()
	_, parsePort, _ := net.SplitHostPort(parseAddress)
	_ = parseListener.Close()
	parseCommand := exec.Command("go", "run", "./tools/gwc", "dev", "-target", "web", "-root", parseRoot, "-host", "127.0.0.1", "-port", parsePort, "-no-doctor")
	if isAgent {
		parseCommand.Args = append(parseCommand.Args, "-agent")
	}
	parseCommand.Dir = parseRepo
	parseCommand.Env = append(desktopNativeEnv(), "GOFLAGS=-tags=gwc_desktop")
	parseOutput := &desktopProcessOutput{}
	parseCommand.Stdout, parseCommand.Stderr = parseOutput, parseOutput
	if parseErr = parseCommand.Start(); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseDone := make(chan error, 1)
	go func() { parseDone <- parseCommand.Wait() }()
	isReaped := false
	parseTest.Cleanup(func() {
		if !isReaped {
			terminateLauncherProcessTree(parseCommand)
			select {
			case <-parseDone:
			case <-time.After(10 * time.Second):
				parseTest.Error("owned web dev process failed to reap")
			}
		}
	})
	parseClient := &http.Client{Timeout: 2 * time.Second}
	parseRead := func(parsePath string) ([]byte, bool) {
		parseResponse, parseReadErr := parseClient.Get("http://" + parseAddress + parsePath)
		if parseReadErr != nil {
			return nil, false
		}
		defer parseResponse.Body.Close()
		parseData, parseReadErr := io.ReadAll(io.LimitReader(parseResponse.Body, 32<<20))
		return parseData, parseReadErr == nil && parseResponse.StatusCode == http.StatusOK
	}
	parseWait := func(parseDescription string, parseCondition func() bool) {
		parseTest.Helper()
		parseDeadline := time.Now().Add(100 * time.Second)
		for time.Now().Before(parseDeadline) {
			select {
			case parseExit := <-parseDone:
				isReaped = true
				parseTest.Fatalf("dev exited while waiting for %s: %v\n%s", parseDescription, parseExit, parseOutput.parseBytes)
			default:
			}
			if parseCondition() {
				return
			}
			time.Sleep(300 * time.Millisecond)
		}
		parseOutput.parseMutex.Lock()
		parseLog := string(parseOutput.parseBytes)
		parseOutput.parseMutex.Unlock()
		parseTest.Fatalf("timed out waiting for %s\n%s", parseDescription, parseLog)
	}
	parseWait("complete browser asset bundle", func() bool {
		parseData, isOK := parseRead("/app.wasm")
		return isOK && bytes.HasPrefix(parseData, []byte{0, 'a', 's', 'm'})
	})
	for _, parsePath := range []string{"/", "/bootstrap.js", "/app.css", "/tester.css", "/wasm_exec.js"} {
		parseData, isOK := parseRead(parsePath)
		if !isOK || len(parseData) == 0 {
			parseTest.Fatalf("missing served browser asset %s", parsePath)
		}
		if parsePath == "/bootstrap.js" && (bytes.Contains(parseData, []byte("/wails/")) || bytes.Contains(parseData, []byte("createDesktopTransport"))) {
			parseTest.Fatal("web dev served native bootstrap")
		}
	}
	if _, isOK := parseRead("/go.mod"); isOK {
		parseTest.Fatal("web dev exposed module files outside assets/web")
	}
	parseWait("initial watcher build to finish before source edit", func() bool {
		parseData, isOK := parseRead("/__gwc/status")
		var parseStatus struct {
			LastBuild *struct {
				Success bool `json:"success"`
			} `json:"lastBuild"`
		}
		return isOK && json.Unmarshal(parseData, &parseStatus) == nil && parseStatus.LastBuild != nil && parseStatus.LastBuild.Success
	})
	parseBefore, _ := parseRead("/app.wasm")
	parseHash := sha256.Sum256(parseBefore)
	parseSourcePath := filepath.Join(parseRoot, "frontend", "main.go")
	parseSource, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseChanged := strings.Replace(string(parseSource), `Text("Increment")`, `Text("Increment verified web rebuild")`, 1)
	if parseChanged == string(parseSource) {
		parseTest.Fatal("rebuild fixture did not change frontend source")
	}
	if parseErr = os.WriteFile(parseSourcePath, []byte(parseChanged), 0o644); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseWait("frontend Go edit to rebuild served Wasm", func() bool {
		parseData, isOK := parseRead("/app.wasm")
		return isOK && bytes.HasPrefix(parseData, []byte{0, 'a', 's', 'm'}) && sha256.Sum256(parseData) != parseHash && bytes.Contains(parseData, []byte("Increment verified web rebuild"))
	})
	if _, parseErr = os.Stat(filepath.Join(parseRoot, "assets", "dist")); !os.IsNotExist(parseErr) {
		parseTest.Fatalf("web dev mutated native dist: %v", parseErr)
	}
	parseTest.Log("real web dev served root/bootstrap/CSS/runtime/Wasm; source outside assets/web rebuilt with inherited desktop tags neutralized")
}
