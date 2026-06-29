//go:build !js || !wasm

package devtools

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/pluginruntime"
)

type devtoolsBridgeTestContext struct {
	services       map[pluginruntime.ServiceKey]any
	contributions  []pluginruntime.ContributionRegistration
	failOnRegister int
	registerCalls  int
}

func (parseContext *devtoolsBridgeTestContext) ResolveService(parseKey pluginruntime.ServiceKey) (any, bool) {
	parseValue, parseOK := parseContext.services[parseKey]
	return parseValue, parseOK
}

func (parseContext *devtoolsBridgeTestContext) RegisterContribution(parseRegistration pluginruntime.ContributionRegistration) error {
	parseContext.registerCalls++
	if parseContext.failOnRegister == parseContext.registerCalls {
		return errors.New("register failed")
	}
	parseContext.contributions = append(parseContext.contributions, parseRegistration)
	return nil
}

func (parseContext *devtoolsBridgeTestContext) RegisterCleanup(pluginruntime.CleanupFunc) {}

func (parseContext *devtoolsBridgeTestContext) KernelInfo() pluginruntime.KernelInfo {
	return pluginruntime.KernelInfo{APIVersion: "test"}
}

type devtoolsBridgeRuntime2Service struct {
	snapshot pluginruntime.Runtime2MetaSnapshot
	err      error
}

func (parseService devtoolsBridgeRuntime2Service) GetRuntime2MetaSnapshot(pluginruntime.QueryBudget) (pluginruntime.Runtime2MetaSnapshot, error) {
	return parseService.snapshot, parseService.err
}

func TestBuiltInDevtoolsPluginRegistersSectionsAndRuntime2States(parseT *testing.T) {
	parsePlugin := builtInDevtoolsPlugin{}
	parseManifest := parsePlugin.Manifest()
	if parseManifest.ID != "devtools.kernel" || parseManifest.ActivationPolicy != pluginruntime.ActivationPolicyBoot {
		parseT.Fatalf("Manifest() = %#v", parseManifest)
	}

	parseContext := &devtoolsBridgeTestContext{services: map[pluginruntime.ServiceKey]any{}}
	if parseHandle, parseErr := parsePlugin.StartPlugin(parseContext); parseErr != nil || parseHandle != nil {
		parseT.Fatalf("StartPlugin() handle=%#v err=%v", parseHandle, parseErr)
	}
	if len(parseContext.contributions) != 2 {
		parseT.Fatalf("registered contributions = %d, want 2", len(parseContext.contributions))
	}
	if parseContext.contributions[0].Metadata.ID != "kernel-health" || parseContext.contributions[1].Metadata.ID != "runtime2-summary" {
		parseT.Fatalf("unexpected contributions: %#v", parseContext.contributions)
	}

	parseRuntime2Provider := parseContext.contributions[1].Value.(pluginruntime.DevtoolsSectionProvider)
	parseSections, parseErr := parseRuntime2Provider.GetDevtoolsSections()
	if parseErr != nil {
		parseT.Fatalf("runtime2 provider without service error = %v", parseErr)
	}
	if len(parseSections) != 0 {
		parseT.Fatalf("runtime2 provider without service = %#v, want no sections", parseSections)
	}

	parseContext.services[pluginruntime.ServiceKeyRuntime2Meta] = "wrong-type"
	parseSections, parseErr = parseRuntime2Provider.GetDevtoolsSections()
	if parseErr != nil || len(parseSections) != 0 {
		parseT.Fatalf("runtime2 provider with wrong service sections=%#v err=%v", parseSections, parseErr)
	}

	parseContext.services[pluginruntime.ServiceKeyRuntime2Meta] = devtoolsBridgeRuntime2Service{err: errors.New("metadata failed")}
	parseSections, parseErr = parseRuntime2Provider.GetDevtoolsSections()
	if parseErr != nil || len(parseSections) != 1 || parseSections[0].Summary["error"] != "metadata failed" {
		parseT.Fatalf("runtime2 provider error section=%#v err=%v", parseSections, parseErr)
	}

	parseContext.services[pluginruntime.ServiceKeyRuntime2Meta] = devtoolsBridgeRuntime2Service{}
	parseSections, parseErr = parseRuntime2Provider.GetDevtoolsSections()
	if parseErr != nil || len(parseSections) != 0 {
		parseT.Fatalf("runtime2 empty snapshot sections=%#v err=%v", parseSections, parseErr)
	}

	parseContext.services[pluginruntime.ServiceKeyRuntime2Meta] = devtoolsBridgeRuntime2Service{
		snapshot: pluginruntime.Runtime2MetaSnapshot{
			Meta: pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime2, true),
			Capabilities: map[string]bool{
				"patch":    true,
				"disabled": false,
				"stream":   true,
			},
			Regions: []pluginruntime.Runtime2RegionSnapshot{{
				RegionInstanceID: "region-1",
				RegionMode:       "island",
				TransportTier:    "stream",
				DiagnosticCount:  2,
			}},
			Diagnostics: []pluginruntime.Runtime2Diagnostic{{Type: "stale-patch"}},
		},
	}
	parseSections, parseErr = parseRuntime2Provider.GetDevtoolsSections()
	if parseErr != nil || len(parseSections) != 1 {
		parseT.Fatalf("runtime2 populated sections=%#v err=%v", parseSections, parseErr)
	}
	parseSection := parseSections[0]
	if parseSection.Name != "Runtime2" || parseSection.Summary["backend"] != string(pluginruntime.BackendIDRuntime2) {
		parseT.Fatalf("unexpected runtime2 section: %#v", parseSection)
	}
	parseJoinedLines := strings.Join(parseSection.Lines, "\n")
	if !strings.Contains(parseJoinedLines, "capabilities: patch, stream") || !strings.Contains(parseJoinedLines, "region-1 => island") {
		parseT.Fatalf("runtime2 section lines missing sorted capability or region detail: %#v", parseSection.Lines)
	}
}

func TestBuiltInDevtoolsPluginPropagatesContributionRegistrationErrors(parseT *testing.T) {
	parsePlugin := builtInDevtoolsPlugin{}
	for _, parseFailOn := range []int{1, 2} {
		parseContext := &devtoolsBridgeTestContext{services: map[pluginruntime.ServiceKey]any{}, failOnRegister: parseFailOn}
		if _, parseErr := parsePlugin.StartPlugin(parseContext); parseErr == nil {
			parseT.Fatalf("StartPlugin with failure on register call %d returned nil error", parseFailOn)
		}
	}
}

type devtoolsBridgeActionPlugin struct {
	ran *pluginruntime.DevtoolsIssue
}

func (parsePlugin devtoolsBridgeActionPlugin) Manifest() pluginruntime.Manifest {
	return pluginruntime.Manifest{
		ID:               "devtools.bridge.action",
		Version:          "1.0.0",
		ActivationPolicy: pluginruntime.ActivationPolicyBoot,
	}
}

func (parsePlugin devtoolsBridgeActionPlugin) StartPlugin(parseContext pluginruntime.Context) (pluginruntime.Handle, error) {
	return nil, parseContext.RegisterContribution(pluginruntime.ContributionRegistration{
		Metadata: pluginruntime.ContributionMetadata{
			ID:               "actions",
			Kind:             pluginruntime.ContributionKindDevtoolsAction,
			ExecutionClass:   pluginruntime.ExecutionClassWarm,
			ActivationPolicy: pluginruntime.ActivationPolicyBoot,
		},
		Value: pluginruntime.DevtoolsActionProviderFunc(func() ([]pluginruntime.DevtoolsAction, error) {
			return []pluginruntime.DevtoolsAction{
				{Label: "   "},
				{
					Label:        "Retry",
					MatchCodes:   []string{"GWC001"},
					MatchSources: []string{"runtime"},
					Run: func(parseContext pluginruntime.DevtoolsActionContext) error {
						if parsePlugin.ran != nil {
							*parsePlugin.ran = parseContext.Issue
						}
						return nil
					},
				},
			}, nil
		}),
	})
}

func resetDevtoolsPluginRuntimeForTest(parseT *testing.T) {
	parseT.Helper()
	if parseErr := pluginruntime.ResetGlobalKernelForTesting(); parseErr != nil {
		parseT.Fatalf("ResetGlobalKernelForTesting() error = %v", parseErr)
	}
	parseT.Cleanup(func() {
		ClearTraceReplay()
		_ = pluginruntime.ResetGlobalKernelForTesting()
		_ = pluginruntime.RegisterBuiltinPlugin(pluginruntime.PluginRegistration{
			Factory: func() pluginruntime.Plugin { return builtInDevtoolsPlugin{} },
		})
		_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
			Key:   pluginruntime.ServiceKeyCapture,
			Value: buildCaptureService{},
		})
	})
}

func TestKernelInspectionAdaptersFilterCloneAndMapActions(parseT *testing.T) {
	resetDevtoolsPluginRuntimeForTest(parseT)

	var parseRanIssue pluginruntime.DevtoolsIssue
	_, parseErr := pluginruntime.BootGlobalKernel(pluginruntime.BootstrapOptions{
		Registrations: []pluginruntime.PluginRegistration{
			{Factory: func() pluginruntime.Plugin { return builtInDevtoolsPlugin{} }},
			{Factory: func() pluginruntime.Plugin { return devtoolsBridgeActionPlugin{ran: &parseRanIssue} }},
		},
		Services: []pluginruntime.ServiceRegistration{{
			Key: pluginruntime.ServiceKeyRuntime2Meta,
			Value: devtoolsBridgeRuntime2Service{snapshot: pluginruntime.Runtime2MetaSnapshot{
				Meta:         pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime2, false),
				Capabilities: map[string]bool{"patch": true},
			}},
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("BootGlobalKernel() error = %v", parseErr)
	}

	parseSections := inspectKernelExtensionSections()
	if len(parseSections) != 2 {
		parseT.Fatalf("inspectKernelExtensionSections() = %#v", parseSections)
	}
	for parseIndex := range parseSections {
		if strings.TrimSpace(parseSections[parseIndex].Name) == "" {
			parseT.Fatalf("blank section should be filtered: %#v", parseSections)
		}
	}
	parseSections[0].Summary["mutated"] = "yes"
	parseSections[0].Lines[0] = "mutated"
	parseAgain := inspectKernelExtensionSections()
	if _, parseOK := parseAgain[0].Summary["mutated"]; parseOK || parseAgain[0].Lines[0] == "mutated" {
		parseT.Fatalf("sections should be defensively cloned: %#v", parseAgain[0])
	}

	parseActions := inspectKernelOverlayActions()
	if len(parseActions) != 1 || parseActions[0].Label != "Retry" {
		parseT.Fatalf("inspectKernelOverlayActions() = %#v", parseActions)
	}
	parseActions[0].MatchCodes[0] = "mutated"
	if parseFresh := inspectKernelOverlayActions(); parseFresh[0].MatchCodes[0] != "GWC001" {
		parseT.Fatalf("actions should be defensively cloned: %#v", parseFresh[0])
	}
	parseActions[0].Run(ErrorOverlayActionContext{Issue: ErrorOverlayIssue{
		Source:   "runtime",
		Code:     "GWC001",
		Message:  "failed",
		Path:     "App",
		Docs:     "docs",
		TopFrame: "frame",
	}})
	if parseRanIssue.Code != "GWC001" || parseRanIssue.Path != "App" || parseRanIssue.TopFrame != "frame" {
		parseT.Fatalf("overlay action context was not mapped: %#v", parseRanIssue)
	}

	parseSnapshot := snapshotKernelState()
	if parseSnapshot.APIVersion == "" || len(parseSnapshot.Plugins) != 2 {
		parseT.Fatalf("snapshotKernelState() = %#v", parseSnapshot)
	}
}

func TestCaptureServiceReportsReplaySessions(parseT *testing.T) {
	ClearTraceReplay()
	parseT.Cleanup(ClearTraceReplay)

	parseService := buildCaptureService{}
	parseSnapshot, parseErr := parseService.GetCaptureSnapshot(pluginruntime.QueryBudget{})
	if parseErr != nil {
		parseT.Fatalf("GetCaptureSnapshot(empty) error = %v", parseErr)
	}
	if parseSnapshot.Meta.BackendID != string(pluginruntime.BackendIDRuntime1) || len(parseSnapshot.Sessions) != 0 {
		parseT.Fatalf("empty capture snapshot = %#v", parseSnapshot)
	}

	SetTraceReplay(TraceCapture{Label: " checkout ", Snapshot: Snapshot{Route: Route{Path: "/checkout"}}})
	parseSnapshot, parseErr = parseService.GetCaptureSnapshot(pluginruntime.QueryBudget{MaxItems: 1})
	if parseErr != nil {
		parseT.Fatalf("GetCaptureSnapshot(replay) error = %v", parseErr)
	}
	if len(parseSnapshot.Sessions) != 1 || parseSnapshot.Sessions[0] != " checkout " {
		parseT.Fatalf("replay capture snapshot = %#v", parseSnapshot)
	}
}
