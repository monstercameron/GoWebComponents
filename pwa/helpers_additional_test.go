package pwa

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

func TestDiagnosticsHelpersSummariesAndPressure(parseT *testing.T) {
	parseEntries := []OfflineQueueEntry{
		{State: "queued", CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 3, 20, 10, 5, 0, 0, time.UTC)},
		{State: "retrying", CreatedAt: time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 3, 20, 10, 10, 0, 0, time.UTC)},
		{State: "dead", CreatedAt: time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 3, 20, 10, 15, 0, 0, time.UTC)},
	}
	parseSummary := summarizeOfflineQueue(parseEntries)
	if parseSummary.TotalEntries != 3 || parseSummary.QueuedEntries != 1 || parseSummary.RetryingEntries != 1 || parseSummary.DeadEntries != 1 {
		parseT.Fatalf("summarizeOfflineQueue() = %+v, want 1 queued/retrying/dead", parseSummary)
	}
	if !parseSummary.OldestCreatedAt.Equal(parseEntries[2].CreatedAt) || !parseSummary.LatestUpdatedAt.Equal(parseEntries[2].UpdatedAt) {
		parseT.Fatalf("summarizeOfflineQueue() timestamps = %+v, want oldest=%v latest=%v", parseSummary, parseEntries[2].CreatedAt, parseEntries[2].UpdatedAt)
	}

	if pressureLabel(0) != "unknown" || pressureLabel(0.5) != "normal" || pressureLabel(0.8) != "elevated" || pressureLabel(0.95) != "critical" {
		parseT.Fatal("pressureLabel() returned unexpected values")
	}

	parseNormalized := (StoragePressureDiagnostics{QuotaBytes: 100, UsageBytes: 80}).normalized()
	if parseNormalized.UsageRatio != 0.8 || parseNormalized.Pressure != "elevated" {
		parseT.Fatalf("normalized() = %+v, want ratio=0.8 pressure=elevated", parseNormalized)
	}
	_diagnosticsContext(context.Background())
}

func TestInstallabilityManagerHelpers(parseT *testing.T) {
	parseManager := InstallabilityManager{}
	if parseState := parseManager.State(); parseState.ManifestValid || parseState.ManifestError != "" || parseState.PromptAvailable || parseState.Installed || len(parseState.Reasons) != 0 {
		parseT.Fatalf("State() = %+v, want zero value", parseState)
	}
	if _, parseErr := parseManager.Prompt(context.Background()); parseErr == nil || !strings.Contains(parseErr.Error(), "InstallabilityManager.Prompt") {
		parseT.Fatalf("Prompt() error = %v, want unavailable error", parseErr)
	}
	if _, parseErr2 := parseManager.Subscribe(func(InstallabilityState) {}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "InstallabilityManager.Subscribe") {
		parseT.Fatalf("Subscribe() error = %v, want unavailable error", parseErr2)
	}

	isParseCalled := false
	parseSubscription := InstallabilitySubscription{cancel: func() { isParseCalled = true }}
	parseSubscription.Cancel()
	if !isParseCalled {
		parseT.Fatal("InstallabilitySubscription.Cancel() did not run callback")
	}
}

func TestServiceWorkerRegistrationHelpers(parseT *testing.T) {
	parseRegistration := ServiceWorkerRegistration{}
	if parseSnapshot := parseRegistration.Snapshot(); parseSnapshot != (ServiceWorkerSnapshot{}) {
		parseT.Fatalf("Snapshot() = %+v, want zero value", parseSnapshot)
	}
	if parseCaps := parseRegistration.BackgroundSyncCapabilities(); parseCaps.Available() {
		parseT.Fatalf("BackgroundSyncCapabilities() = %+v, want unavailable", parseCaps)
	}
	if _, parseErr := parseRegistration.Unregister(context.Background()); parseErr == nil || !strings.Contains(parseErr.Error(), "ServiceWorkerRegistration.Unregister") {
		parseT.Fatalf("Unregister() error = %v, want unavailable error", parseErr)
	}
	if parseErr2 := parseRegistration.RegisterSync(context.Background(), "sync"); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "ServiceWorkerRegistration.RegisterSync") {
		parseT.Fatalf("RegisterSync() error = %v, want unavailable error", parseErr2)
	}
	if parseErr3 := parseRegistration.Update(context.Background()); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "ServiceWorkerRegistration.Update") {
		parseT.Fatalf("Update() error = %v, want unavailable error", parseErr3)
	}
	if parseErr4 := parseRegistration.SkipWaiting(context.Background()); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "ServiceWorkerRegistration.SkipWaiting") {
		parseT.Fatalf("SkipWaiting() error = %v, want unavailable error", parseErr4)
	}
	if _, parseErr5 := parseRegistration.SubscribeLifecycle(func(ServiceWorkerSnapshot) {}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "ServiceWorkerRegistration.SubscribeLifecycle") {
		parseT.Fatalf("SubscribeLifecycle() error = %v, want unavailable error", parseErr5)
	}
	if _, parseErr6 := parseRegistration.ReloadOnControllerChange(); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "ServiceWorkerRegistration.ReloadOnControllerChange") {
		parseT.Fatalf("ReloadOnControllerChange() error = %v, want unavailable error", parseErr6)
	}

	if !(BackgroundSyncCapabilities{OneShot: true}).Available() {
		parseT.Fatal("BackgroundSyncCapabilities.Available() should be true when one-shot sync exists")
	}
	isParseCalled := false
	parseSubscription := ServiceWorkerSubscription{cancel: func() { isParseCalled = true }}
	parseSubscription.Cancel()
	if !isParseCalled {
		parseT.Fatal("ServiceWorkerSubscription.Cancel() did not run callback")
	}
}

func TestServiceWorkerAssetPlanURLHelpers(parseT *testing.T) {
	if parseGot := resolveServiceWorkerAssetURL("", " app.wasm "); parseGot != "app.wasm" {
		parseT.Fatalf("resolveServiceWorkerAssetURL() = %q, want app.wasm", parseGot)
	}
	if parseGot2 := resolveServiceWorkerAssetURL("/static/", "/app.wasm"); parseGot2 != "/static/app.wasm" {
		parseT.Fatalf("resolveServiceWorkerAssetURL(base) = %q, want /static/app.wasm", parseGot2)
	}
	parseUrls := normalizeServiceWorkerURLs("/static", []string{" app.wasm ", "/app.wasm", "", "styles.css"})
	if len(parseUrls) != 2 || parseUrls[0] != "/static/app.wasm" || parseUrls[1] != "/static/styles.css" {
		parseT.Fatalf("normalizeServiceWorkerURLs() = %+v, want deduped normalized URLs", parseUrls)
	}
	if dedupeServiceWorkerURLs([]string{"", "a", "a", " b "})[1] != "b" {
		parseT.Fatalf("dedupeServiceWorkerURLs() returned unexpected values")
	}
}

func TestManagerSuccessCallbacksAndNativeContextNoops(parseT *testing.T) {
	testNativeNoopHelpers(context.Background(), interop.CodeUnavailable)
	_diagnosticsContext(context.Background())

	parsePlan := CacheStoragePlan{CacheName: "release-cache"}
	parseManager := CacheStorageManager{
		sync: func(parseCtx context.Context, parseP CacheStoragePlan) (CacheStorageSnapshot, error) {
			if parseCtx == nil || parseP.CacheName != "release-cache" {
				parseT.Fatalf("unexpected sync input: ctx=%v plan=%+v", parseCtx, parseP)
			}
			return CacheStorageSnapshot{CacheName: parseP.CacheName, EntryCount: 2}, nil
		},
		inspect: func(parseCtx2 context.Context, parseP2 CacheStoragePlan) (CacheStorageSnapshot, error) {
			if parseCtx2 == nil || parseP2.CacheName != "release-cache" {
				parseT.Fatalf("unexpected inspect input: ctx=%v plan=%+v", parseCtx2, parseP2)
			}
			return CacheStorageSnapshot{CacheName: parseP2.CacheName, CacheNames: []string{"release-cache"}}, nil
		},
	}
	parseSyncSnapshot, parseErr := parseManager.Sync(context.Background(), parsePlan)
	if parseErr != nil || parseSyncSnapshot.EntryCount != 2 {
		parseT.Fatalf("expected cache sync callback snapshot, got %+v err=%v", parseSyncSnapshot, parseErr)
	}
	parseInspectSnapshot, parseErr := parseManager.Inspect(context.Background(), parsePlan)
	if parseErr != nil || len(parseInspectSnapshot.CacheNames) != 1 {
		parseT.Fatalf("expected cache inspect callback snapshot, got %+v err=%v", parseInspectSnapshot, parseErr)
	}

	parseInstallability := InstallabilityManager{
		state: func() InstallabilityState {
			return InstallabilityState{ManifestValid: true, PromptAvailable: true}
		},
		prompt: func(context.Context) (InstallPromptResult, error) {
			return InstallPromptResult{Outcome: "accepted", Platform: "web"}, nil
		},
		subscribe: func(handler func(InstallabilityState)) (InstallabilitySubscription, error) {
			if handler != nil {
				handler(InstallabilityState{Installed: true})
			}
			return InstallabilitySubscription{cancel: func() {}}, nil
		},
	}
	if parseState := parseInstallability.State(); !parseState.ManifestValid || !parseState.PromptAvailable {
		parseT.Fatalf("expected installability state callback values, got %+v", parseState)
	}
	if parsePrompt, parseErr2 := parseInstallability.Prompt(context.Background()); parseErr2 != nil || parsePrompt.Outcome != "accepted" {
		parseT.Fatalf("expected installability prompt callback result, got %+v err=%v", parsePrompt, parseErr2)
	}
	isParseInstalled := false
	parseSubscription, parseErr := parseInstallability.Subscribe(func(parseState2 InstallabilityState) {
		isParseInstalled = parseState2.Installed
	})
	if parseErr != nil {
		parseT.Fatalf("expected installability subscribe callback to succeed, got %v", parseErr)
	}
	parseSubscription.Cancel()
	if !isParseInstalled {
		parseT.Fatal("expected installability subscribe callback to receive state")
	}
	InstallabilitySubscription{}.Cancel()

	parseServiceWorker := ServiceWorkerRegistration{
		snapshot: func() ServiceWorkerSnapshot {
			return ServiceWorkerSnapshot{Scope: "/app", HasController: true}
		},
		backgroundSync: func() BackgroundSyncCapabilities {
			return BackgroundSyncCapabilities{Periodic: true}
		},
		registerSync: func(context.Context, string) error { return nil },
		update:       func(context.Context) error { return nil },
		unregister: func(context.Context) (bool, error) {
			return true, nil
		},
		skipWaiting: func(context.Context) error { return nil },
		subscribeLifecycle: func(handler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
			if handler != nil {
				handler(ServiceWorkerSnapshot{Active: ServiceWorkerVersion{State: ServiceWorkerStateActivated}})
			}
			return ServiceWorkerSubscription{cancel: func() {}}, nil
		},
		reloadOnControllerChange: func() (ServiceWorkerSubscription, error) {
			return ServiceWorkerSubscription{cancel: func() {}}, nil
		},
	}
	if parseSnapshot := parseServiceWorker.Snapshot(); parseSnapshot.Scope != "/app" || !parseSnapshot.HasController {
		parseT.Fatalf("expected service worker snapshot callback values, got %+v", parseSnapshot)
	}
	if !parseServiceWorker.BackgroundSyncCapabilities().Available() {
		parseT.Fatal("expected periodic background sync capability to be available")
	}
	if parseErr3 := parseServiceWorker.RegisterSync(context.Background(), "refresh-cache"); parseErr3 != nil {
		parseT.Fatalf("expected register sync callback to succeed, got %v", parseErr3)
	}
	if parseErr4 := parseServiceWorker.Update(context.Background()); parseErr4 != nil {
		parseT.Fatalf("expected update callback to succeed, got %v", parseErr4)
	}
	if parseOk, parseErr5 := parseServiceWorker.Unregister(context.Background()); parseErr5 != nil || !parseOk {
		parseT.Fatalf("expected unregister callback to succeed, got ok=%t err=%v", parseOk, parseErr5)
	}
	if parseErr6 := parseServiceWorker.SkipWaiting(context.Background()); parseErr6 != nil {
		parseT.Fatalf("expected skip waiting callback to succeed, got %v", parseErr6)
	}
	isParseLifecycleSeen := false
	parseLifecycleSub, parseErr := parseServiceWorker.SubscribeLifecycle(func(parseSnapshot2 ServiceWorkerSnapshot) {
		isParseLifecycleSeen = parseSnapshot2.Active.State == ServiceWorkerStateActivated
	})
	if parseErr != nil {
		parseT.Fatalf("expected lifecycle subscribe callback to succeed, got %v", parseErr)
	}
	parseLifecycleSub.Cancel()
	if !isParseLifecycleSeen {
		parseT.Fatal("expected service worker lifecycle callback to receive snapshot")
	}
	parseReloadSub, parseErr := parseServiceWorker.ReloadOnControllerChange()
	if parseErr != nil {
		parseT.Fatalf("expected controller-change callback to succeed, got %v", parseErr)
	}
	parseReloadSub.Cancel()
	ServiceWorkerSubscription{}.Cancel()
}

func TestManifestValidationAdditionalBranchesAndIndentedMarshal(parseT *testing.T) {
	parseValid := Manifest{
		Name:     "Atlas",
		StartURL: "/",
		Display:  ManifestDisplayStandalone,
	}
	if _, parseErr := MarshalManifestJSONIndented(parseValid, "", "  "); parseErr != nil {
		parseT.Fatalf("expected indented manifest marshal success, got %v", parseErr)
	}

	if parseErr2 := (Manifest{
		Name:            "Atlas",
		StartURL:        "/",
		DisplayOverride: []ManifestDisplay{"invalid"},
	}).Validate(); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "display_override") {
		parseT.Fatalf("expected display_override validation failure, got %v", parseErr2)
	}
	if parseErr3 := (Manifest{
		Name:        "Atlas",
		StartURL:    "/",
		Shortcuts:   []ManifestShortcut{{URL: "/orders"}},
		Icons:       []ManifestImage{{Src: "/icon.png"}},
		Display:     ManifestDisplayStandalone,
		Orientation: ManifestOrientationPortrait,
	}).Validate(); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "shortcuts require a non-empty name") {
		parseT.Fatalf("expected shortcut name validation failure, got %v", parseErr3)
	}
	if parseErr4 := (Manifest{
		Name:      "Atlas",
		StartURL:  "/",
		Shortcuts: []ManifestShortcut{{Name: "Orders"}},
	}).Validate(); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "shortcuts require a non-empty url") {
		parseT.Fatalf("expected shortcut url validation failure, got %v", parseErr4)
	}

	parseApps := normalizeRelatedApplications([]RelatedApplication{
		{Platform: " ios ", URL: " ", ID: " "},
		{Platform: " ", URL: " ", ID: " "},
	})
	if len(parseApps) != 1 || parseApps[0].Platform != "ios" {
		parseT.Fatalf("expected normalized related apps to retain only non-empty entry, got %+v", parseApps)
	}
	if parseApps2 := normalizeRelatedApplications(nil); parseApps2 != nil {
		parseT.Fatalf("expected nil related apps normalization for nil input, got %+v", parseApps2)
	}

	if _, parseErr5 := MarshalManifestJSON(Manifest{}); parseErr5 == nil {
		parseT.Fatal("expected manifest marshal to fail when required fields are missing")
	}
}
