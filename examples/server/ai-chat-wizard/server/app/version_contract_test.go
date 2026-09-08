package app

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/internal/buildinfo"
)

// versionPattern is the semver-like format used by the shared version constant:
// "v<year>.<month>.<day>.<seq>" — e.g. "v2026.03.27.2".
var versionPattern = regexp.MustCompile(`^v\d{4}\.\d{2}\.\d{2}\.\d+$`)

// parseVersionContractExampleRoot resolves the example root from this source
// file's path (this file lives at <root>/server/app/).
func parseVersionContractExampleRoot(t *testing.T) string {
	t.Helper()
	_, parseFile, _, parseOk := runtime.Caller(0)
	if !parseOk {
		t.Fatal("runtime.Caller(0) failed — cannot determine test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(parseFile), "..", ".."))
}

// TestVersionContractFormatValid confirms that GetBuildAppVersion returns a
// non-empty string and matches the project's versioning convention so a stale
// or accidentally cleared constant is caught before it ships.
func TestVersionContractFormatValid(t *testing.T) {
	parseVersion := buildinfo.GetBuildAppVersion()
	if parseVersion == "" {
		t.Fatal("buildinfo.GetBuildAppVersion() returned an empty string")
	}
	if !versionPattern.MatchString(parseVersion) {
		t.Errorf("buildinfo.GetBuildAppVersion() = %q; want format matching %s", parseVersion, versionPattern)
	}
}

// TestVersionContractBootShellRendersVersion confirms that the server-side boot
// shell template contains exactly one {{APP_VERSION}} placeholder, and that the
// replacement used during production request handling injects the shared version
// string and leaves no unresolved tokens behind.
func TestVersionContractBootShellRendersVersion(t *testing.T) {
	parsePlaceholder := "{{APP_VERSION}}"
	parseVersion := buildinfo.GetBuildAppVersion()

	// Verify the template carries exactly one token.
	parseCount := strings.Count(chatShellHTML, parsePlaceholder)
	if parseCount != 1 {
		t.Errorf("chatShellHTML contains %d occurrences of %q; want exactly 1", parseCount, parsePlaceholder)
	}

	// Mirror the production replacement path from bootstrap.go line 969.
	parseRendered := strings.Replace(chatShellHTML, parsePlaceholder, parseVersion, 1)

	if strings.Contains(parseRendered, parsePlaceholder) {
		t.Errorf("chatShellHTML still contains %q after replacement — template substitution is broken", parsePlaceholder)
	}
	if !strings.Contains(parseRendered, parseVersion) {
		t.Errorf("rendered shell HTML does not contain version %q after substitution", parseVersion)
	}
}

// TestVersionContractClientSourceUsesBuildinfo reads the WASM-only
// client/app/constants.go source file at the text level (bypassing js+wasm
// build tags that prevent importing it in a native test) and asserts that it
// imports the shared buildinfo package and calls GetBuildAppVersion().  This
// catches any accidental in-lining of the version as a hard-coded literal on
// the client side.
func TestVersionContractClientSourceUsesBuildinfo(t *testing.T) {
	parseRoot := parseVersionContractExampleRoot(t)
	parseConstantsPath := filepath.Join(parseRoot, "client", "app", "constants.go")

	parseSource, parseReadErr := os.ReadFile(parseConstantsPath)
	if parseReadErr != nil {
		t.Fatalf("read client/app/constants.go: %v — check that the file exists at %s", parseReadErr, parseConstantsPath)
	}
	parseText := string(parseSource)

	parseBuildinfoPkg := "examples/server/ai-chat-wizard/internal/buildinfo"
	if !strings.Contains(parseText, parseBuildinfoPkg) {
		t.Errorf("client/app/constants.go does not import %q; the client version badge may be using a hard-coded string", parseBuildinfoPkg)
	}
	if !strings.Contains(parseText, "GetBuildAppVersion()") {
		t.Errorf("client/app/constants.go does not call buildinfo.GetBuildAppVersion(); the client version badge may not use the shared constant")
	}
}

// TestVersionContractServerBootstrapSourceUsesBuildinfo reads
// server/app/bootstrap.go at the text level and asserts that the boot shell
// substitution continues to pull the version from buildinfo rather than a
// literal copy.  This prevents a future refactor from accidentally baking in a
// stale version string.
func TestVersionContractServerBootstrapSourceUsesBuildinfo(t *testing.T) {
	parseRoot := parseVersionContractExampleRoot(t)
	parseBootstrapPath := filepath.Join(parseRoot, "server", "app", "bootstrap.go")

	parseSource, parseReadErr := os.ReadFile(parseBootstrapPath)
	if parseReadErr != nil {
		t.Fatalf("read server/app/bootstrap.go: %v", parseReadErr)
	}
	parseText := string(parseSource)

	if !strings.Contains(parseText, "buildinfo.GetBuildAppVersion()") {
		t.Errorf("server/app/bootstrap.go does not call buildinfo.GetBuildAppVersion() in the shell substitution; the boot shell version label may be stale or hard-coded")
	}
	if !strings.Contains(parseText, `"{{APP_VERSION}}"`) {
		t.Errorf("server/app/bootstrap.go does not replace {{APP_VERSION}} via buildinfo; the boot shell template contract may be broken")
	}
}
