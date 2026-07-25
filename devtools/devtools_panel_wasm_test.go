//go:build js && wasm

package devtools

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type panelNoOpScheduler struct{}

func (panelNoOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

func (panelNoOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

// installPanelHookContext installs one fresh wasm hook context for direct hook-using panel helpers.
func installPanelHookContext(parseT *testing.T) {
	parseT.Helper()
	runtime.SetCurrentFiber(nil)
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: mockdom.NewMockDOMAdapter(), Scheduler: panelNoOpScheduler{}, Reset: true})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

// getPanelHookEffects returns the queued effect slice from the active panel test fiber.
func getPanelHookEffects(parseT *testing.T) []runtime.Effect {
	parseT.Helper()
	parseFiber := runtime.GetCurrentFiber()
	if parseFiber == nil {
		parseT.Fatal("expected active panel test fiber")
	}
	parseFiberValue := reflect.ValueOf(parseFiber).Elem()
	parseEffectsField := parseFiberValue.FieldByName("effects")
	return reflect.NewAt(parseEffectsField.Type(), unsafe.Pointer(parseEffectsField.UnsafeAddr())).Elem().Interface().([]runtime.Effect)
}

// buildDevtoolsTestReplaySnapshot creates one rich replay snapshot that exercises the devtools wasm panel and overlay sections.
func buildDevtoolsTestReplaySnapshot() Snapshot {
	return Snapshot{
		Route: Route{
			Path:    "/dashboard/reports/7",
			Loading: true,
			Query: map[string][]string{
				"tab": {"activity"},
			},
			Params: map[string]string{
				"reportID": "7",
			},
			Stack: []RouteStack{
				{ID: "dashboard", Path: "/dashboard", HasBeforeLeave: true},
				{ID: "report", Path: "/dashboard/reports/7", HasLoader: true, HasBeforeEnter: true},
			},
			Loaders: []RouteLoader{{
				Key:     "report@/dashboard/reports/7",
				Path:    "/dashboard/reports/7",
				Pending: true,
				HasData: false,
				Error:   "route loader failed",
			}},
			LastRedirect: RouteRedirect{Cause: "before-enter", From: "/secure", To: "/login"},
			Metadata: RouteMetadata{
				Title:        "Report detail",
				Description:  "Daily metrics",
				CanonicalURL: "/dashboard/reports/7",
			},
		},
		Cache: []CacheEntry{
			{
				Key:             "report:7",
				Loading:         false,
				Ready:           true,
				Stale:           true,
				LastError:       "stale while revalidate",
				UpdatedAt:       time.Date(2026, 4, 6, 15, 0, 0, 0, time.UTC),
				LastLoaded:      time.Date(2026, 4, 6, 14, 59, 0, 0, time.UTC),
				SubscriberCount: 2,
				OwnerPaths:      []string{"App > Sidebar", "App"},
				ResumePolicy:    "resume-latest",
			},
		},
		MultiClient: MultiClient{
			Enabled:           true,
			LocalPeerID:       "ops-1",
			ResolvedTransport: "broadcast-channel",
			AuthorityView: map[string]string{
				"report:7": "ops-1",
			},
			Peers: []MultiClientPeer{{
				ID:            "ops-1",
				Surface:       "tab",
				Role:          "operator",
				State:         "ready",
				Compatible:    true,
				LeaseDeadline: time.Date(2026, 4, 6, 15, 1, 0, 0, time.UTC),
			}},
			RecentTraffic: []MultiClientTraffic{{
				Direction:     "outbound",
				Kind:          "query",
				Topic:         "reports",
				PeerID:        "ops-1",
				CorrelationID: "req-77",
				LatencyMs:     14,
			}},
			FailedPublishes: []MultiClientFailure{{
				Op:      "PublishClientIntent",
				Topic:   "reports",
				Target:  "ops-1",
				Code:    "unauthorized",
				Message: "policy rejected publish",
			}},
		},
		Boundaries: BoundaryInspection{Entries: []Boundary{{
			Name:          "ssr.bootstrap",
			Kind:          "ssr-bootstrap",
			Direction:     "server-to-client",
			Status:        "downgraded",
			Encoding:      "json",
			Transport:     "inline",
			Scope:         "route",
			Target:        "/dashboard/reports/7",
			CorrelationID: "boot-1",
			SizeBytes:     512,
			InlineBytes:   256,
			BinaryBytes:   128,
			Notes:         []string{"route payload"},
			Downgraded:    []string{"blob"},
			Rejected:      []string{"oversized"},
		}}},
		Coordination: Coordination{
			Workers: []WorkerJob{{
				Name:      "worker.search",
				Status:    "running",
				RequestID: "req-22",
				Running:   true,
				Ready:     true,
			}},
			SyncEvents: []SyncEvent{{
				Transport: "broadcast-channel",
				Direction: "outbound",
				Topic:     "reports",
				Target:    "ops-1",
				Status:    "sent",
			}},
			Replay: []ReplayEntry{{
				ID:          "mut-1",
				Kind:        "order.submit",
				Owner:       "sync-engine",
				State:       "retrying",
				Attempts:    1,
				MaxAttempts: 3,
			}},
			QueueEntries: []SyncQueueEntry{{
				ID:        "queue-1",
				Entity:    "order:42",
				Operation: "submit",
				Owner:     "sync-engine",
				State:     "queued",
			}},
			SyncHealth: []SyncHealthEntry{{
				Entity:     "order:42",
				Owner:      "sync-engine",
				Status:     "healthy",
				Version:    "v12",
				PendingOps: 1,
			}},
			Reconnect: ReconnectStatus{
				State:       "reconnecting",
				Transport:   "broadcast-channel",
				Attempts:    2,
				MaxAttempts: 5,
				NextRetryAt: time.Date(2026, 4, 6, 15, 2, 0, 0, time.UTC),
				LastChange:  time.Date(2026, 4, 6, 15, 1, 0, 0, time.UTC),
			},
			Conflict: ConflictState{
				Entity:     "order:42",
				Owner:      "sync-engine",
				Status:     "pending",
				Strategy:   "server-wins",
				DetectedAt: time.Date(2026, 4, 6, 15, 1, 0, 0, time.UTC),
			},
			LastReplayError: "HTTP 409 conflict",
		},
		Extensions: []ExtensionSection{{
			Name: "Companion",
			Summary: map[string]string{
				"state": "ready",
			},
			Lines: []string{"extra diagnostics enabled"},
		}},
		Tree: &Node{
			Name:              "App",
			Path:              "App",
			Kind:              "component",
			HookCount:         2,
			EffectCount:       1,
			Signature:         "App() hooks=2",
			FineGrained:       true,
			ReactiveSource:    "reportID",
			UpdateOrigin:      "fine-grained",
			RenderDurationNs:  18,
			DiffDurationNs:    7,
			CommitDurationNs:  6,
			EffectDurationNs:  4,
			CleanupDurationNs: 2,
			SelfDurationNs:    22,
			SubtreeDurationNs: 64,
			Hooks: []Hook{{
				Slot:         1,
				Kind:         "state",
				Value:        "open=true",
				Dependencies: "reportID",
				Status:       "stable",
			}},
			Children: []Node{{
				Name:              "Sidebar",
				Path:              "App > Sidebar",
				Kind:              "component",
				Dirty:             true,
				NeedsUpdate:       true,
				HookCount:         1,
				EffectCount:       1,
				RenderDurationNs:  9,
				DiffDurationNs:    3,
				CommitDurationNs:  2,
				SelfDurationNs:    10,
				SubtreeDurationNs: 26,
				Children: []Node{{
					Name:             "NavLink",
					Path:             "App > Sidebar > NavLink",
					Kind:             "host",
					RenderDurationNs: 4,
					DiffDurationNs:   1,
					CommitDurationNs: 1,
					SelfDurationNs:   5,
				}},
			}},
		},
		Stats: Stats{
			TotalFibers:       3,
			DirtyFibers:       1,
			ComponentFibers:   2,
			HostFibers:        1,
			TextFibers:        0,
			FineGrainedFibers: 1,
			HookEntries:       2,
			Effects:           2,
		},
		Profiling: Profiling{
			RenderCalls:                      4,
			ScheduledRootUpdates:             2,
			ScheduledFiberMarks:              3,
			ScheduledGranularMarks:           1,
			WorkLoopPasses:                   5,
			ProcessedUnits:                   12,
			CommitCount:                      2,
			FineGrainedCommits:               1,
			FineGrainedDescendantHostCommits: 2,
			FineGrainedDescendantTextCommits: 1,
			EffectExecutions:                 2,
			CleanupExecutions:                1,
			LastRenderDurationNs:             18,
			LastCommitDurationNs:             6,
			LastEffectDurationNs:             4,
			LastCleanupDurationNs:            2,
			PhaseTotals: ProfilingPhaseTotals{
				RenderDurationNs:  48,
				DiffDurationNs:    17,
				CommitDurationNs:  14,
				EffectDurationNs:  9,
				CleanupDurationNs: 3,
			},
			RecentEvents: []ProfilingEvent{{
				Domain:     "router",
				Name:       "navigation",
				Phase:      "finish",
				Target:     "/dashboard/reports/7",
				DurationNs: 14,
				Fields: map[string]string{
					"mode": "push",
				},
			}},
			ComponentRenders: []ComponentRenderTrace{{
				Name:                    "Dashboard",
				Path:                    "App > Dashboard",
				RenderCount:             4,
				RerenderCount:           3,
				LastTrigger:             "hook",
				LastRenderDurationNs:    9,
				AverageRenderDurationNs: 6,
				TriggerCounts: map[string]int{
					"hook": 3,
				},
			}},
			FlamegraphFrames: []FlamegraphFrame{{
				Name:              "Dashboard",
				Path:              "App > Dashboard",
				Depth:             1,
				StartNs:           2,
				DurationNs:        40,
				SelfDurationNs:    23,
				RenderDurationNs:  10,
				DiffDurationNs:    4,
				CommitDurationNs:  6,
				EffectDurationNs:  2,
				CleanupDurationNs: 1,
			}},
			HotBranches: []Branch{{
				Name:              "Dashboard",
				Path:              "App > Dashboard",
				RenderDurationNs:  10,
				DiffDurationNs:    4,
				CommitDurationNs:  6,
				EffectDurationNs:  2,
				CleanupDurationNs: 1,
				SelfDurationNs:    23,
				SubtreeDurationNs: 40,
			}},
			Startup: StartupProfiling{
				Mode:                       "hydrate",
				StartedAt:                  "2026-04-06T15:00:00.000Z",
				BootstrapReadDurationNs:    9,
				WASMTransferBytes:          1024,
				WASMDecodedBytes:           2048,
				BootstrapDecodedBytes:      512,
				CacheWarmupDurationNs:      3,
				ServiceWorkerOverheadNs:    2,
				InitialRouteDataBytes:      144,
				HydrationDurationNs:        14,
				StartupCommitDurationNs:    6,
				FirstInteractionDurationNs: 42,
				FirstInteractionCaptured:   true,
				FirstInteractionEvent:      "click",
				RouteBudgets: []RouteStartupBudget{{
					RouteFamily:                       "/reports/*",
					LastRoutePath:                     "/reports/7",
					SampleCount:                       3,
					AverageBootstrapReadDurationNs:    8,
					AverageWASMTransferBytes:          1000,
					AverageWASMDecodedBytes:           2100,
					AverageBootstrapDecodedBytes:      500,
					AverageCacheWarmupDurationNs:      4,
					AverageServiceWorkerOverheadNs:    3,
					AverageInitialRouteDataBytes:      140,
					AverageHydrationDurationNs:        13,
					AverageStartupCommitDurationNs:    5,
					AverageFirstInteractionDurationNs: 37,
				}},
			},
		},
		Hydration: HydrationDebug{
			CorrelationID:        "hydrate-77",
			StartedAt:            "2026-04-06T15:00:00.000Z",
			FinishedAt:           "2026-04-06T15:00:00.014Z",
			DurationNs:           14,
			ExistingDOMNodeCount: 8,
			FallbackCount:        1,
			MismatchCount:        2,
			DiscardedNodeCount:   4,
			Strict:               true,
			Failed:               true,
			Failure:              "hydration mismatch forced replacement",
			RecentMessages: []string{
				"hydration text mismatch at App > Hero",
				"hydration fallback at App > Sidebar",
			},
		},
		Logs: []Log{{
			Domain:         "router",
			Level:          LogLevel(runtime.LogInfo),
			Classification: Classification(runtime.DiagnosticInformational),
			Code:           "GWC-ROUTER-NAVIGATION",
			Message:        "route loader started",
			Timestamp:      "2026-04-06T15:00:00.010Z",
			CorrelationID:  "nav-7",
			Fields: map[string]string{
				"target": "/dashboard/reports/7",
			},
		}},
		Diagnostics: []Diagnostic{{
			Source:         "router",
			Severity:       SeverityError,
			Classification: Classification(runtime.DiagnosticCorrectness),
			Code:           "GWC-ROUTER-LOADER-FAILED",
			Docs:           "docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md#actionable-errors-and-troubleshooting",
			Remediation:    "Retry the failing loader",
			Recoverable:    true,
			TopFrame:       "router/loader.go:42",
			Consequence:    "data missing",
			Message:        "route loader failed",
			Count:          1,
			Path:           "/dashboard/reports/7",
			ComponentStack: []string{"App", "Reports"},
			Fields: map[string]string{
				"route": "/dashboard/reports/7",
			},
		}},
	}
}

// TestRenderPanelWasmWithReplaySnapshot verifies the devtools panel renders both closed and expanded states from replay data.
func TestRenderPanelWasmWithReplaySnapshot(parseT *testing.T) {
	ClearTraceReplay()
	defer ClearTraceReplay()

	SetTraceReplay(TraceCapture{Label: "panel", Snapshot: buildDevtoolsTestReplaySnapshot()})
	installPanelHookContext(parseT)

	parseClosedMarkup, parseErr := ui.RenderToString(Panel(PanelProps{
		Title:         "Replay Panel",
		InitiallyOpen: false,
	}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(closed Panel) error = %v", parseErr)
	}
	if !strings.Contains(parseClosedMarkup, "Replay Panel") {
		parseT.Fatalf("expected closed panel button title, got %q", parseClosedMarkup)
	}

	parseOpenMarkup, parseErr := ui.RenderToString(Panel(PanelProps{
		Title:           "Replay Panel",
		InitiallyOpen:   true,
		RefreshInterval: time.Second,
		MaxDepth:        1,
	}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(open Panel) error = %v", parseErr)
	}
	for _, parseExpected := range []string{
		"Replay Panel",
		"Route",
		"Cache",
		"Multi-Client",
		"Boundaries",
		"Coordination",
		"Profiling",
		"Hydration",
		"Logs",
		"Diagnostics",
		"child nodes hidden at max depth",
		"extra diagnostics enabled",
		"route loader started",
		"route loader failed",
		"Dashboard",
	} {
		if !strings.Contains(parseOpenMarkup, parseExpected) {
			parseT.Fatalf("expected expanded panel markup to include %q\n%s", parseExpected, parseOpenMarkup)
		}
	}
}

// TestRenderErrorOverlayWasmWithReplaySnapshot verifies the overlay renders replayed issues, hydration fallback state, and matching actions.
func TestRenderErrorOverlayWasmWithReplaySnapshot(parseT *testing.T) {
	ClearTraceReplay()
	ResetErrorOverlayActions()
	defer ClearTraceReplay()
	defer ResetErrorOverlayActions()

	parseSnapshot := buildDevtoolsTestReplaySnapshot()
	parseSnapshot.Diagnostics = []Diagnostic{{
		Source:      "router",
		Severity:    SeverityError,
		Code:        "GWC-ROUTER-LOADER-FAILED",
		Message:     "route loader failed",
		Docs:        "docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md#actionable-errors-and-troubleshooting",
		TopFrame:    "router/loader.go:42",
		Path:        "/dashboard/reports/7",
		Remediation: "Retry the failing loader",
	}}
	SetTraceReplay(TraceCapture{Label: "overlay", Snapshot: parseSnapshot})
	SetErrorOverlayActions([]ErrorOverlayAction{
		{Label: "Retry loader", MatchCodes: []string{"GWC-ROUTER-LOADER-FAILED"}},
		{Label: "Inspect hydration", MatchSources: []string{"hydration"}},
	})
	installPanelHookContext(parseT)

	parseMarkup, parseErr := ui.RenderToString(ErrorOverlay(ErrorOverlayProps{
		Title:           "Replay Overlay",
		RefreshInterval: time.Second,
		MaxItems:        3,
	}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ErrorOverlay) error = %v", parseErr)
	}
	for _, parseExpected := range []string{
		"Replay Overlay",
		"route loader failed",
		"Retry loader",
		"hydration mismatch forced replacement",
		"Inspect hydration",
		"docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md#actionable-errors-and-troubleshooting",
	} {
		if !strings.Contains(parseMarkup, parseExpected) {
			parseT.Fatalf("expected overlay markup to include %q\n%s", parseExpected, parseMarkup)
		}
	}
}

// TestUseSnapshotWasmRunsDefaultRefreshEffect verifies the snapshot hook seeds replay state and installs one cleanup-producing refresh effect.
func TestUseSnapshotWasmRunsDefaultRefreshEffect(parseT *testing.T) {
	ClearTraceReplay()
	defer ClearTraceReplay()

	SetTraceReplay(TraceCapture{Label: "snapshot", Snapshot: buildDevtoolsTestReplaySnapshot()})
	installPanelHookContext(parseT)

	parseSnapshot := UseSnapshot(0)
	if parseSnapshot.Route.Path != "/dashboard/reports/7" {
		parseT.Fatalf("expected UseSnapshot to seed replay route state, got %+v", parseSnapshot.Route)
	}

	parseEffects := getPanelHookEffects(parseT)
	if len(parseEffects) == 0 || parseEffects[0].Fn == nil {
		parseT.Fatalf("expected UseSnapshot to queue one refresh effect, got %+v", parseEffects)
	}
	parseCleanup := parseEffects[0].Fn()
	if parseCleanup == nil {
		parseT.Fatal("expected UseSnapshot effect to return a cleanup")
	}
	parseCleanup()
}

// TestRenderPanelAndOverlayWasmUseDefaultTitles verifies the devtools overlays fall back to their built-in titles and overlay helper branches.
func TestRenderPanelAndOverlayWasmUseDefaultTitles(parseT *testing.T) {
	ClearTraceReplay()
	ResetErrorOverlayActions()
	defer ClearTraceReplay()
	defer ResetErrorOverlayActions()

	parseSnapshot := buildDevtoolsTestReplaySnapshot()
	SetTraceReplay(TraceCapture{Label: "defaults", Snapshot: parseSnapshot})
	installPanelHookContext(parseT)

	parsePanelMarkup, parseErr := ui.RenderToString(Panel(PanelProps{InitiallyOpen: false}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(default Panel) error = %v", parseErr)
	}
	if !strings.Contains(parsePanelMarkup, "GWC Devtools") {
		parseT.Fatalf("expected default panel title, got %q", parsePanelMarkup)
	}

	parseOverlayMarkup, parseErr := ui.RenderToString(ErrorOverlay(ErrorOverlayProps{}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(default ErrorOverlay) error = %v", parseErr)
	}
	if !strings.Contains(parseOverlayMarkup, "GWC Error Overlay") {
		parseT.Fatalf("expected default overlay title, got %q", parseOverlayMarkup)
	}

	if !shouldShowOverlayDiagnostic(Diagnostic{Severity: SeverityWarning, Code: "GWC-HYDRATION-MISMATCH"}) {
		parseT.Fatal("expected hydration warnings to stay visible in the overlay")
	}
	if !shouldShowOverlayDiagnostic(Diagnostic{Severity: SeverityWarning, Code: "GWC-ROUTER-LOADER-FAILED"}) {
		parseT.Fatal("expected loader failures to stay visible in the overlay")
	}
	if shouldShowOverlayDiagnostic(Diagnostic{Severity: SeverityWarning, Code: "GWC-ROUTER-NAVIGATION"}) {
		parseT.Fatal("expected informational warnings to stay hidden in the overlay")
	}
	if !overlayHasHydrationIssue([]ErrorOverlayIssue{{Code: "GWC-HYDRATION-FAILED"}}) {
		parseT.Fatal("expected hydration overlay issue detection to match hydration codes")
	}
	if overlayHasHydrationIssue([]ErrorOverlayIssue{{Code: "GWC-ROUTER-LOADER-FAILED"}}) {
		parseT.Fatal("expected non-hydration overlay issues to stay unmatched")
	}
}

// TestTreeHelpersWasmUseSelectedPath verifies selected-node and formatting helpers can target nested nodes from replay snapshots.
func TestTreeHelpersWasmUseSelectedPath(parseT *testing.T) {
	parseSnapshot := buildDevtoolsTestReplaySnapshot()
	parseSelectedPath := "App > Sidebar"

	if parseFound := findNodeByPath(parseSnapshot.Tree, parseSelectedPath); parseFound == nil || parseFound.Path != parseSelectedPath {
		parseT.Fatalf("expected selected node lookup to find %q, got %+v", parseSelectedPath, parseFound)
	}
	parseMatches := cacheEntriesForNode(findNodeByPath(parseSnapshot.Tree, parseSelectedPath), parseSnapshot.Cache)
	if len(parseMatches) != 1 || parseMatches[0].Key != "report:7" {
		parseT.Fatalf("expected one cache match for selected node, got %+v", parseMatches)
	}
	if selectedNodeSummary(parseSnapshot, parseSelectedPath) == nil {
		parseT.Fatal("expected selected node summary")
	}
	if treeSummary(parseSnapshot.Tree, 0, 2, parseSelectedPath, nil) == nil {
		parseT.Fatal("expected tree summary")
	}
	if parseFormatted := formatQuery(map[string][]string{"b": {"2"}, "a": {"1"}}); parseFormatted != "a=1 & b=2" {
		parseT.Fatalf("expected stable query formatting, got %q", parseFormatted)
	}
	if parseFormatted := formatParams(map[string]string{"b": "2", "a": "1"}); parseFormatted != "a=1 & b=2" {
		parseT.Fatalf("expected stable param formatting, got %q", parseFormatted)
	}
	if parseFormatted := formatStringMap(map[string]string{"b": "2", "a": "1"}); parseFormatted != "a=1 & b=2" {
		parseT.Fatalf("expected stable string-map formatting, got %q", parseFormatted)
	}
	if parseFormatted := formatByteCount(2048); parseFormatted != "2.00 KiB" {
		parseT.Fatalf("expected kibibyte formatting, got %q", parseFormatted)
	}
}
