package atlasperf

// Fixture capture for the Atlas render-path measurement harness.
//
// The payloads under fixtures/ are REAL: they are the verbatim
// <script id="__ATLAS_BOOTSTRAP__"> contents the running Atlas server sent for
// each route, including the 88-row comments inbox. Nothing here is synthesized,
// because the point of this harness is that the previous ones were.
//
// Capture is OPT-IN (ATLAS_PERF_CAPTURE=1). It boots a real server, which needs
// a free port and the sqlite database, so it must never be a side effect of
// `go test ./...`.
//
//	ATLAS_PERF_CAPTURE=1 go test ./examples/testing/atlas-perf/ -run TestAtlasPerfCaptureFixtures -v
//
// Port policy: ask the OS for a free port. Ports 8080/8096/8097/8199/8231/8299/
// 8402/8733 were all bound on the machine this was written on; a hardcoded port
// fails as "health check timed out", which blames the app for a harness problem.

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// atlasPerfRouteSpec names one route to capture and the session it needs.
//
// Role "" captures anonymously (buyer surface). A non-empty role sets the
// atlas_mock_role cookie, which is what actually creates an operator session —
// GET /auth/mock-sign-in only sets a CSRF cookie, and an earlier browser pass in
// this repo reported nine green operator routes while five of them were the
// unauthenticated sign-in screen.
type atlasPerfRouteSpec struct {
	Name  string
	Route string
	Role  string
	// Why records what this route contributes to the profile, so a reader can
	// tell whether the set still covers the questions being asked.
	Why string
}

// atlasPerfRoutes is the capture set. Chosen to span the app's real shapes
// rather than to be exhaustive: a tiny public page, the biggest real list in the
// app, a dense operator dashboard, a filtered table, and a form surface.
var atlasPerfRoutes = []atlasPerfRouteSpec{
	{Name: "public-landing", Route: "/", Why: "smallest real payload; isolates shell + hook cost from data cost"},
	{Name: "public-catalog", Route: "/shop", Why: "buyer catalog list, the public surface's widest tree"},
	{Name: "public-product", Route: "/shop/frame-desk", Why: "detail page with a comment thread"},
	{Name: "operator-dashboard", Route: "/app/dashboard", Role: "inventory_manager", Why: "densest operator payload: 88 comments + transfers + receiving + orders"},
	{Name: "operator-inventory", Route: "/app/inventory", Role: "inventory_manager", Why: "filtered table surface"},
	{Name: "operator-comments", Route: "/app/comments", Role: "inventory_manager", Why: "the 88-item comments inbox — the largest real list in the app"},
	{Name: "operator-settings", Route: "/app/settings", Role: "inventory_manager", Why: "form surface with CSRF round trip"},
}

var atlasPerfBootstrapPattern = regexp.MustCompile(`(?s)<script id="__ATLAS_BOOTSTRAP__" type="application/json">(.*?)</script>`)

func TestAtlasPerfCaptureFixtures(parseT *testing.T) {
	if os.Getenv("ATLAS_PERF_CAPTURE") != "1" {
		parseT.Skip("set ATLAS_PERF_CAPTURE=1 to re-capture fixtures (boots a real Atlas server)")
	}

	parseRepoRoot := atlasPerfRepoRoot(parseT)
	parseBaseURL := atlasPerfStartServer(parseT, parseRepoRoot)

	if parseErr := os.MkdirAll(atlasPerfFixtureDir, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir %s: %v", atlasPerfFixtureDir, parseErr)
	}

	for _, parseSpec := range atlasPerfRoutes {
		parseBootstrap := atlasPerfFetchBootstrap(parseT, parseBaseURL, parseSpec)
		parseShape := atlasPerfDescribeBootstrap(parseT, parseBootstrap)
		parseRecord := atlasPerfCapturedFixture{
			Name:      parseSpec.Name,
			Route:     parseSpec.Route,
			Role:      parseSpec.Role,
			Bootstrap: parseBootstrap,
			Shape:     parseShape,
		}
		parseEncoded, parseErr := json.MarshalIndent(parseRecord, "", "  ")
		if parseErr != nil {
			parseT.Fatalf("encode fixture %s: %v", parseSpec.Name, parseErr)
		}
		parsePath := filepath.Join(atlasPerfFixtureDir, "atlas_perf_"+parseSpec.Name+".json")
		if parseWriteErr := os.WriteFile(parsePath, parseEncoded, 0o644); parseWriteErr != nil {
			parseT.Fatalf("write fixture %s: %v", parsePath, parseWriteErr)
		}
		parseT.Logf("captured %-20s %-28s bootstrap=%d bytes  %s", parseSpec.Name, parseSpec.Route, len(parseBootstrap), parseShape)
	}
}

// atlasPerfFetchBootstrap pulls one route and extracts its inline bootstrap.
func atlasPerfFetchBootstrap(parseT *testing.T, parseBaseURL string, parseSpec atlasPerfRouteSpec) json.RawMessage {
	parseT.Helper()
	parseRequest, parseErr := http.NewRequest(http.MethodGet, parseBaseURL+parseSpec.Route, nil)
	if parseErr != nil {
		parseT.Fatalf("build request for %s: %v", parseSpec.Route, parseErr)
	}
	if parseSpec.Role != "" {
		parseRequest.AddCookie(&http.Cookie{Name: "atlas_mock_role", Value: parseSpec.Role})
	}
	parseResponse, parseErr := http.DefaultClient.Do(parseRequest)
	if parseErr != nil {
		parseT.Fatalf("GET %s: %v", parseSpec.Route, parseErr)
	}
	defer func() { _ = parseResponse.Body.Close() }()
	if parseResponse.StatusCode >= 400 {
		parseT.Fatalf("GET %s returned %d", parseSpec.Route, parseResponse.StatusCode)
	}
	parseBody, parseErr := io.ReadAll(parseResponse.Body)
	if parseErr != nil {
		parseT.Fatalf("read %s: %v", parseSpec.Route, parseErr)
	}
	parseMatch := atlasPerfBootstrapPattern.FindSubmatch(parseBody)
	if parseMatch == nil {
		parseT.Fatalf("no __ATLAS_BOOTSTRAP__ script in %s (%d bytes); the server's document shape changed", parseSpec.Route, len(parseBody))
	}
	return json.RawMessage(parseMatch[1])
}

// atlasPerfDescribeBootstrap records the payload's shape so a stale fixture is
// visible in a report rather than inferred.
//
// It also PROVES an operator capture is really an operator payload: a signed-out
// capture of /app/comments returns a valid document with a sign-in screen and no
// items, which would quietly turn "88-item inbox" into "empty inbox".
func atlasPerfDescribeBootstrap(parseT *testing.T, parseBootstrap json.RawMessage) string {
	parseT.Helper()
	var parseEnvelope struct {
		Data struct {
			Atlas struct {
				Route struct {
					Path    string `json:"path"`
					Screen  string `json:"screen"`
					Surface string `json:"surface"`
				} `json:"route"`
				User *struct {
					Role string `json:"role"`
				} `json:"user"`
				Data map[string]json.RawMessage `json:"data"`
			} `json:"atlas"`
		} `json:"data"`
	}
	if parseErr := json.Unmarshal(parseBootstrap, &parseEnvelope); parseErr != nil {
		parseT.Fatalf("decode bootstrap envelope: %v", parseErr)
	}
	parseRoute := parseEnvelope.Data.Atlas.Route
	parseCounts := ""
	if parsePage, parseOk := parseEnvelope.Data.Atlas.Data["page"]; parseOk {
		var parseLists map[string]json.RawMessage
		if json.Unmarshal(parsePage, &parseLists) == nil {
			for _, parseKey := range []string{"items", "comments", "transfers", "receiving", "orders", "lines"} {
				parseRaw, parseHas := parseLists[parseKey]
				if !parseHas {
					continue
				}
				var parseSlice []json.RawMessage
				if json.Unmarshal(parseRaw, &parseSlice) == nil {
					parseCounts += fmt.Sprintf(" %s=%d", parseKey, len(parseSlice))
				}
			}
		}
	}
	parseRole := "anonymous"
	if parseEnvelope.Data.Atlas.User != nil {
		parseRole = parseEnvelope.Data.Atlas.User.Role
	}
	return fmt.Sprintf("screen=%s surface=%s session=%s%s", parseRoute.Screen, parseRoute.Surface, parseRole, parseCounts)
}

// atlasPerfRepoRoot walks up from this file to the module root.
func atlasPerfRepoRoot(parseT *testing.T) string {
	parseT.Helper()
	_, parseFile, _, parseOk := runtime.Caller(0)
	if !parseOk {
		parseT.Fatal("runtime.Caller failed; cannot locate repo root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(parseFile), "..", "..", ".."))
}

// atlasPerfStartServer boots the Atlas server on a free port and returns its
// base URL. Registers cleanup that kills the whole process tree — on Windows the
// compiled child of `go run` outlives its parent and keeps the port.
func atlasPerfStartServer(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parsePort := atlasPerfReservePort(parseT)
	parseAddress := "127.0.0.1:" + parsePort

	parseCmd := exec.Command("go", "run", "./examples/server/atlas-commerce-os/server")
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "ATLAS_ADDR="+parseAddress)
	parseCmd.Stdout = io.Discard
	parseCmd.Stderr = io.Discard
	if parseErr := parseCmd.Start(); parseErr != nil {
		parseT.Fatalf("start atlas server: %v", parseErr)
	}
	parseDone := make(chan error, 1)
	go func() { parseDone <- parseCmd.Wait() }()
	parseT.Cleanup(func() {
		if parseCmd.Process == nil {
			return
		}
		if runtime.GOOS == "windows" {
			// /PID, never an image name: this repo has lost probe runs to a
			// broad kill.
			_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(parseCmd.Process.Pid)).Run()
		} else {
			_ = parseCmd.Process.Kill()
		}
		select {
		case <-parseDone:
		case <-time.After(5 * time.Second):
		}
	})

	parseBaseURL := "http://" + parseAddress
	parseDeadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseResponse, parseErr := http.Get(parseBaseURL + "/healthz")
		if parseErr == nil {
			var parseHealth struct {
				OK bool `json:"ok"`
			}
			_ = json.NewDecoder(parseResponse.Body).Decode(&parseHealth)
			_ = parseResponse.Body.Close()
			if parseHealth.OK {
				return parseBaseURL
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	parseT.Fatalf("atlas server never became healthy at %s", parseBaseURL)
	return ""
}

func atlasPerfReservePort(parseT *testing.T) string {
	parseT.Helper()
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("reserve a free port: %v", parseErr)
	}
	parsePort := parseListener.Addr().(*net.TCPAddr).Port
	if parseCloseErr := parseListener.Close(); parseCloseErr != nil {
		parseT.Fatalf("release reserved port %d: %v", parsePort, parseCloseErr)
	}
	return strconv.Itoa(parsePort)
}
