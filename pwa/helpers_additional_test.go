package pwa

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestDiagnosticsHelpersSummariesAndPressure(t *testing.T) {
	entries := []OfflineQueueEntry{
		{State: "queued", CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 3, 20, 10, 5, 0, 0, time.UTC)},
		{State: "retrying", CreatedAt: time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 3, 20, 10, 10, 0, 0, time.UTC)},
		{State: "dead", CreatedAt: time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 3, 20, 10, 15, 0, 0, time.UTC)},
	}
	summary := summarizeOfflineQueue(entries)
	if summary.TotalEntries != 3 || summary.QueuedEntries != 1 || summary.RetryingEntries != 1 || summary.DeadEntries != 1 {
		t.Fatalf("summarizeOfflineQueue() = %+v, want 1 queued/retrying/dead", summary)
	}
	if !summary.OldestCreatedAt.Equal(entries[2].CreatedAt) || !summary.LatestUpdatedAt.Equal(entries[2].UpdatedAt) {
		t.Fatalf("summarizeOfflineQueue() timestamps = %+v, want oldest=%v latest=%v", summary, entries[2].CreatedAt, entries[2].UpdatedAt)
	}

	if pressureLabel(0) != "unknown" || pressureLabel(0.5) != "normal" || pressureLabel(0.8) != "elevated" || pressureLabel(0.95) != "critical" {
		t.Fatal("pressureLabel() returned unexpected values")
	}

	normalized := (StoragePressureDiagnostics{QuotaBytes: 100, UsageBytes: 80}).normalized()
	if normalized.UsageRatio != 0.8 || normalized.Pressure != "elevated" {
		t.Fatalf("normalized() = %+v, want ratio=0.8 pressure=elevated", normalized)
	}
	_diagnosticsContext(context.Background())
}

func TestInstallabilityManagerHelpers(t *testing.T) {
	manager := InstallabilityManager{}
	if state := manager.State(); state.ManifestValid || state.ManifestError != "" || state.PromptAvailable || state.Installed || len(state.Reasons) != 0 {
		t.Fatalf("State() = %+v, want zero value", state)
	}
	if _, err := manager.Prompt(context.Background()); err == nil || !strings.Contains(err.Error(), "InstallabilityManager.Prompt") {
		t.Fatalf("Prompt() error = %v, want unavailable error", err)
	}
	if _, err := manager.Subscribe(func(InstallabilityState) {}); err == nil || !strings.Contains(err.Error(), "InstallabilityManager.Subscribe") {
		t.Fatalf("Subscribe() error = %v, want unavailable error", err)
	}

	called := false
	subscription := InstallabilitySubscription{cancel: func() { called = true }}
	subscription.Cancel()
	if !called {
		t.Fatal("InstallabilitySubscription.Cancel() did not run callback")
	}
}

func TestServiceWorkerRegistrationHelpers(t *testing.T) {
	registration := ServiceWorkerRegistration{}
	if snapshot := registration.Snapshot(); snapshot != (ServiceWorkerSnapshot{}) {
		t.Fatalf("Snapshot() = %+v, want zero value", snapshot)
	}
	if caps := registration.BackgroundSyncCapabilities(); caps.Available() {
		t.Fatalf("BackgroundSyncCapabilities() = %+v, want unavailable", caps)
	}
	if _, err := registration.Unregister(context.Background()); err == nil || !strings.Contains(err.Error(), "ServiceWorkerRegistration.Unregister") {
		t.Fatalf("Unregister() error = %v, want unavailable error", err)
	}
	if err := registration.RegisterSync(context.Background(), "sync"); err == nil || !strings.Contains(err.Error(), "ServiceWorkerRegistration.RegisterSync") {
		t.Fatalf("RegisterSync() error = %v, want unavailable error", err)
	}
	if err := registration.Update(context.Background()); err == nil || !strings.Contains(err.Error(), "ServiceWorkerRegistration.Update") {
		t.Fatalf("Update() error = %v, want unavailable error", err)
	}
	if err := registration.SkipWaiting(context.Background()); err == nil || !strings.Contains(err.Error(), "ServiceWorkerRegistration.SkipWaiting") {
		t.Fatalf("SkipWaiting() error = %v, want unavailable error", err)
	}
	if _, err := registration.SubscribeLifecycle(func(ServiceWorkerSnapshot) {}); err == nil || !strings.Contains(err.Error(), "ServiceWorkerRegistration.SubscribeLifecycle") {
		t.Fatalf("SubscribeLifecycle() error = %v, want unavailable error", err)
	}
	if _, err := registration.ReloadOnControllerChange(); err == nil || !strings.Contains(err.Error(), "ServiceWorkerRegistration.ReloadOnControllerChange") {
		t.Fatalf("ReloadOnControllerChange() error = %v, want unavailable error", err)
	}

	if !(BackgroundSyncCapabilities{OneShot: true}).Available() {
		t.Fatal("BackgroundSyncCapabilities.Available() should be true when one-shot sync exists")
	}
	called := false
	subscription := ServiceWorkerSubscription{cancel: func() { called = true }}
	subscription.Cancel()
	if !called {
		t.Fatal("ServiceWorkerSubscription.Cancel() did not run callback")
	}
}

func TestServiceWorkerAssetPlanURLHelpers(t *testing.T) {
	if got := resolveServiceWorkerAssetURL("", " app.wasm "); got != "app.wasm" {
		t.Fatalf("resolveServiceWorkerAssetURL() = %q, want app.wasm", got)
	}
	if got := resolveServiceWorkerAssetURL("/static/", "/app.wasm"); got != "/static/app.wasm" {
		t.Fatalf("resolveServiceWorkerAssetURL(base) = %q, want /static/app.wasm", got)
	}
	urls := normalizeServiceWorkerURLs("/static", []string{" app.wasm ", "/app.wasm", "", "styles.css"})
	if len(urls) != 2 || urls[0] != "/static/app.wasm" || urls[1] != "/static/styles.css" {
		t.Fatalf("normalizeServiceWorkerURLs() = %+v, want deduped normalized URLs", urls)
	}
	if dedupeServiceWorkerURLs([]string{"", "a", "a", " b "})[1] != "b" {
		t.Fatalf("dedupeServiceWorkerURLs() returned unexpected values")
	}
}

func TestManagerSuccessCallbacksAndNativeContextNoops(t *testing.T) {
	_cacheStorageContext(context.Background())
	_diagnosticsNativeInterop(interop.CodeUnavailable)
	_diagnosticsContext(context.Background())

	plan := CacheStoragePlan{CacheName: "release-cache"}
	manager := CacheStorageManager{
		sync: func(ctx context.Context, p CacheStoragePlan) (CacheStorageSnapshot, error) {
			if ctx == nil || p.CacheName != "release-cache" {
				t.Fatalf("unexpected sync input: ctx=%v plan=%+v", ctx, p)
			}
			return CacheStorageSnapshot{CacheName: p.CacheName, EntryCount: 2}, nil
		},
		inspect: func(ctx context.Context, p CacheStoragePlan) (CacheStorageSnapshot, error) {
			if ctx == nil || p.CacheName != "release-cache" {
				t.Fatalf("unexpected inspect input: ctx=%v plan=%+v", ctx, p)
			}
			return CacheStorageSnapshot{CacheName: p.CacheName, CacheNames: []string{"release-cache"}}, nil
		},
	}
	syncSnapshot, err := manager.Sync(context.Background(), plan)
	if err != nil || syncSnapshot.EntryCount != 2 {
		t.Fatalf("expected cache sync callback snapshot, got %+v err=%v", syncSnapshot, err)
	}
	inspectSnapshot, err := manager.Inspect(context.Background(), plan)
	if err != nil || len(inspectSnapshot.CacheNames) != 1 {
		t.Fatalf("expected cache inspect callback snapshot, got %+v err=%v", inspectSnapshot, err)
	}

	installability := InstallabilityManager{
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
	if state := installability.State(); !state.ManifestValid || !state.PromptAvailable {
		t.Fatalf("expected installability state callback values, got %+v", state)
	}
	if prompt, err := installability.Prompt(context.Background()); err != nil || prompt.Outcome != "accepted" {
		t.Fatalf("expected installability prompt callback result, got %+v err=%v", prompt, err)
	}
	installed := false
	subscription, err := installability.Subscribe(func(state InstallabilityState) {
		installed = state.Installed
	})
	if err != nil {
		t.Fatalf("expected installability subscribe callback to succeed, got %v", err)
	}
	subscription.Cancel()
	if !installed {
		t.Fatal("expected installability subscribe callback to receive state")
	}
	InstallabilitySubscription{}.Cancel()

	serviceWorker := ServiceWorkerRegistration{
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
	if snapshot := serviceWorker.Snapshot(); snapshot.Scope != "/app" || !snapshot.HasController {
		t.Fatalf("expected service worker snapshot callback values, got %+v", snapshot)
	}
	if !serviceWorker.BackgroundSyncCapabilities().Available() {
		t.Fatal("expected periodic background sync capability to be available")
	}
	if err := serviceWorker.RegisterSync(context.Background(), "refresh-cache"); err != nil {
		t.Fatalf("expected register sync callback to succeed, got %v", err)
	}
	if err := serviceWorker.Update(context.Background()); err != nil {
		t.Fatalf("expected update callback to succeed, got %v", err)
	}
	if ok, err := serviceWorker.Unregister(context.Background()); err != nil || !ok {
		t.Fatalf("expected unregister callback to succeed, got ok=%t err=%v", ok, err)
	}
	if err := serviceWorker.SkipWaiting(context.Background()); err != nil {
		t.Fatalf("expected skip waiting callback to succeed, got %v", err)
	}
	lifecycleSeen := false
	lifecycleSub, err := serviceWorker.SubscribeLifecycle(func(snapshot ServiceWorkerSnapshot) {
		lifecycleSeen = snapshot.Active.State == ServiceWorkerStateActivated
	})
	if err != nil {
		t.Fatalf("expected lifecycle subscribe callback to succeed, got %v", err)
	}
	lifecycleSub.Cancel()
	if !lifecycleSeen {
		t.Fatal("expected service worker lifecycle callback to receive snapshot")
	}
	reloadSub, err := serviceWorker.ReloadOnControllerChange()
	if err != nil {
		t.Fatalf("expected controller-change callback to succeed, got %v", err)
	}
	reloadSub.Cancel()
	ServiceWorkerSubscription{}.Cancel()
}

func TestManifestValidationAdditionalBranchesAndIndentedMarshal(t *testing.T) {
	valid := Manifest{
		Name:     "Atlas",
		StartURL: "/",
		Display:  ManifestDisplayStandalone,
	}
	if _, err := MarshalManifestJSONIndented(valid, "", "  "); err != nil {
		t.Fatalf("expected indented manifest marshal success, got %v", err)
	}

	if err := (Manifest{
		Name:            "Atlas",
		StartURL:        "/",
		DisplayOverride: []ManifestDisplay{"invalid"},
	}).Validate(); err == nil || !strings.Contains(err.Error(), "display_override") {
		t.Fatalf("expected display_override validation failure, got %v", err)
	}
	if err := (Manifest{
		Name:       "Atlas",
		StartURL:   "/",
		Shortcuts:  []ManifestShortcut{{URL: "/orders"}},
		Icons:      []ManifestImage{{Src: "/icon.png"}},
		Display:    ManifestDisplayStandalone,
		Orientation: ManifestOrientationPortrait,
	}).Validate(); err == nil || !strings.Contains(err.Error(), "shortcuts require a non-empty name") {
		t.Fatalf("expected shortcut name validation failure, got %v", err)
	}
	if err := (Manifest{
		Name:      "Atlas",
		StartURL:  "/",
		Shortcuts: []ManifestShortcut{{Name: "Orders"}},
	}).Validate(); err == nil || !strings.Contains(err.Error(), "shortcuts require a non-empty url") {
		t.Fatalf("expected shortcut url validation failure, got %v", err)
	}

	apps := normalizeRelatedApplications([]RelatedApplication{
		{Platform: " ios ", URL: " ", ID: " "},
		{Platform: " ", URL: " ", ID: " "},
	})
	if len(apps) != 1 || apps[0].Platform != "ios" {
		t.Fatalf("expected normalized related apps to retain only non-empty entry, got %+v", apps)
	}
	if apps := normalizeRelatedApplications(nil); apps != nil {
		t.Fatalf("expected nil related apps normalization for nil input, got %+v", apps)
	}

	if _, err := MarshalManifestJSON(Manifest{}); err == nil {
		t.Fatal("expected manifest marshal to fail when required fields are missing")
	}
}
