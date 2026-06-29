package pwa

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseWasmReleaseManifestJSONValidatesAndNormalizes(parseT *testing.T) {
	parseManifest, parseErr := ParseWasmReleaseManifestJSON([]byte(`
{
  "package": " ./examples/server/atlas-commerce-os/client ",
  "profile": " production ",
  "goos": " js ",
  "goarch": " wasm ",
  "flags": {"trimpath": true, "ldflags": " -s -w ", "gcflags": " all=-N -l ", "buildvcs": " false ", "compression": true},
  "artifacts": {
    "wasm": {"path": " dist/app.1234.wasm ", "bytes": 42, "sha256": " ABCD "}
  }
}`))
	if parseErr != nil {
		parseT.Fatalf("expected release manifest parse to succeed, got %v", parseErr)
	}
	if parseManifest.Package != "./examples/server/atlas-commerce-os/client" {
		parseT.Fatalf("unexpected normalized package: %q", parseManifest.Package)
	}
	if parseManifest.Artifacts["wasm"].Path != "dist/app.1234.wasm" || parseManifest.Artifacts["wasm"].SHA256 != "abcd" {
		parseT.Fatalf("unexpected normalized artifact: %#v", parseManifest.Artifacts["wasm"])
	}
	if !parseManifest.Flags.Trimpath || parseManifest.Flags.LDFlags != "-s -w" || parseManifest.Flags.GCFlags != "all=-N -l" || parseManifest.Flags.BuildVCS != "false" || !parseManifest.Flags.Compression {
		parseT.Fatalf("unexpected normalized flags: %#v", parseManifest.Flags)
	}
}

func TestChooseWasmRolloutIsDeterministicAndRollbackForcesStable(parseT *testing.T) {
	parseStable := WasmReleaseManifest{
		Package: "./examples/app",
		Profile: "production",
		GOOS:    "js",
		GOARCH:  "wasm",
		BuildID: "stable-build",
		Artifacts: map[string]WasmReleaseArtifact{
			"wasm": {Path: "stable.wasm", SHA256: "stable-sha"},
		},
	}
	parseCanary := WasmReleaseManifest{
		Package: "./examples/app",
		Profile: "production",
		GOOS:    "js",
		GOARCH:  "wasm",
		BuildID: "canary-build",
		Artifacts: map[string]WasmReleaseArtifact{
			"wasm": {Path: "canary.wasm", SHA256: "canary-sha"},
		},
	}
	parseConfig := WasmRolloutConfig{Stable: parseStable, Canary: parseCanary, CanaryPercent: 25, Salt: "deploy-1", CacheTTLSeconds: 60}
	parseFirst, parseErr := ChooseWasmRollout(parseConfig, "client-123")
	if parseErr != nil {
		parseT.Fatalf("ChooseWasmRollout: %v", parseErr)
	}
	parseSecond, parseErr := ChooseWasmRollout(parseConfig, "client-123")
	if parseErr != nil {
		parseT.Fatalf("ChooseWasmRollout second: %v", parseErr)
	}
	if parseFirst != parseSecond {
		parseT.Fatalf("expected stable cohort assignment across reloads, got %+v then %+v", parseFirst, parseSecond)
	}
	if parseFirst.BuildID == "" || parseFirst.SHA256 == "" {
		parseT.Fatalf("expected decision to expose build metadata, got %+v", parseFirst)
	}

	parseRollback, parseErr := ChooseWasmRollout(WasmRolloutConfig{Stable: parseStable, Canary: parseCanary, CanaryPercent: 100, Rollback: true, CacheTTLSeconds: 300}, "client-123")
	if parseErr != nil {
		parseT.Fatalf("ChooseWasmRollout rollback: %v", parseErr)
	}
	if parseRollback.Cohort != "stable" || parseRollback.BuildID != "stable-build" || !parseRollback.RolledBack || parseRollback.CacheTTLSeconds != 1 {
		parseT.Fatalf("expected rollback to force stable within one short TTL, got %+v", parseRollback)
	}
}

func TestEvaluateVersionSkewRefreshOneShotAndSnapshotRestore(parseT *testing.T) {
	parseDecision := EvaluateVersionSkewRefresh(VersionSkewRefreshInput{
		ClientBuildID:      "old",
		ServerBuildID:      "new",
		StateSnapshotJSON:  []byte(`{"state":{"theme":"dark"}}`),
		VersionsCanMigrate: true,
	})
	if !parseDecision.Mismatch || !parseDecision.ShouldReload || !parseDecision.BypassCache || !parseDecision.RestoreAfterReload {
		parseT.Fatalf("expected one forced reload with snapshot restore, got %+v", parseDecision)
	}
	if string(parseDecision.SnapshotBeforeReload) != `{"state":{"theme":"dark"}}` {
		parseT.Fatalf("expected snapshot to be carried across refresh, got %q", parseDecision.SnapshotBeforeReload)
	}

	parseLoopGuard := EvaluateVersionSkewRefresh(VersionSkewRefreshInput{
		ClientBuildID:      "old",
		ServerBuildID:      "new",
		ReloadAlreadyTried: true,
		StateSnapshotJSON:  []byte(`{"state":{"theme":"dark"}}`),
		VersionsCanMigrate: true,
	})
	if !parseLoopGuard.Mismatch || parseLoopGuard.ShouldReload || !parseLoopGuard.LoopGuarded || parseLoopGuard.RestoreAfterReload {
		parseT.Fatalf("expected persistent mismatch to be loop-guarded, got %+v", parseLoopGuard)
	}
}

func TestWasmReleaseManifestJSONWireShapePreservesFlagsObject(parseT *testing.T) {
	parseEncoded, parseErr := json.Marshal(WasmReleaseManifest{})
	if parseErr != nil {
		parseT.Fatalf("marshal release manifest: %v", parseErr)
	}
	if parseText := string(parseEncoded); !strings.Contains(parseText, `"flags":{}`) {
		parseT.Fatalf("expected zero-value release manifest to preserve flags object, got %s", parseText)
	}
}

func TestBuildServiceWorkerAssetPlanUsesManifestRevisionAndURLs(parseT *testing.T) {
	parseManifest := WasmReleaseManifest{
		Package: "./examples/app",
		Profile: "production",
		GOOS:    "js",
		GOARCH:  "wasm",
		Artifacts: map[string]WasmReleaseArtifact{
			"wasm": {Path: "assets/app.1234.wasm", SHA256: "deadbeef"},
		},
	}
	parsePlan, parseErr := BuildServiceWorkerAssetPlan(parseManifest, ServiceWorkerAssetPlanOptions{
		BaseURL:       "/static",
		CachePrefix:   "atlas-release",
		ShellURLs:     []string{"/index.html", "/offline.html"},
		ImmutableURLs: []string{"/wasm_exec.js", "/assets/app.css", "/wasm_exec.js"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected asset plan build to succeed, got %v", parseErr)
	}
	if parsePlan.WasmURL != "/static/assets/app.1234.wasm" {
		parseT.Fatalf("unexpected wasm URL: %q", parsePlan.WasmURL)
	}
	if len(parsePlan.PrecacheURLs) != 5 {
		parseT.Fatalf("unexpected precache urls: %#v", parsePlan.PrecacheURLs)
	}
	if parsePlan.CacheName == "" || parsePlan.ManifestRevision == "" {
		parseT.Fatalf("expected derived cache name and revision, got %+v", parsePlan)
	}
	if parsePlan.CacheName[:14] != "atlas-release-" {
		parseT.Fatalf("expected cache prefix, got %q", parsePlan.CacheName)
	}
}

func TestWasmReleaseManifestValidateRequiresWasmArtifact(parseT *testing.T) {
	parseErr := (WasmReleaseManifest{Package: "./examples/app", GOOS: "js", GOARCH: "wasm"}).Validate()
	if parseErr == nil {
		parseT.Fatal("expected missing wasm artifact validation error")
	}
}
