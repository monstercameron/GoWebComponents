package pluginruntime

import (
	"errors"
	"strings"
	"testing"
)

func TestPluginContextNilKernelIsSafe(parseT *testing.T) {
	parseContext := pluginContext{}

	if parseService, parseOK := parseContext.ResolveService(ServiceKeyDiagnostics); parseService != nil || parseOK {
		parseT.Fatalf("nil context ResolveService = %v, %t; want nil, false", parseService, parseOK)
	}
	if parseErr := parseContext.RegisterContribution(ContributionRegistration{}); parseErr == nil || !strings.Contains(parseErr.Error(), "kernel is unavailable") {
		parseT.Fatalf("nil context RegisterContribution error = %v", parseErr)
	}
	parseContext.RegisterCleanup(func() error {
		parseT.Fatal("nil kernel context must ignore cleanup callbacks")
		return nil
	})
	if parseInfo := parseContext.KernelInfo(); parseInfo != (KernelInfo{}) {
		parseT.Fatalf("nil context KernelInfo = %#v, want zero value", parseInfo)
	}
}

func TestPluginContextExposesKernelInfoAndCleanup(parseT *testing.T) {
	buildCleanups := 0
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Services: []ServiceRegistration{{Key: ServiceKeyDiagnostics, Value: "diag"}},
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "context-plugin", Version: "1.0.0"},
					buildStart: func(parseContext Context) (Handle, error) {
						if parseInfo := parseContext.KernelInfo(); parseInfo.APIVersion != "v1alpha1" {
							parseT.Fatalf("KernelInfo() = %#v", parseInfo)
						}
						parseContext.RegisterCleanup(nil)
						parseContext.RegisterCleanup(func() error {
							buildCleanups++
							return nil
						})
						parseService, parseOK := parseContext.ResolveService(ServiceKeyDiagnostics)
						if !parseOK || parseService != "diag" {
							parseT.Fatalf("ResolveService() = %v, %t", parseService, parseOK)
						}
						return nil, nil
					},
				}
			},
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("NewKernel() error = %v", parseErr)
	}
	if parseErr := buildKernel.Close(); parseErr != nil {
		parseT.Fatalf("Close() error = %v", parseErr)
	}
	if buildCleanups != 1 {
		parseT.Fatalf("cleanup callbacks ran %d times, want 1", buildCleanups)
	}
}

func TestManifestValidationAndRegistrationOverrides(parseT *testing.T) {
	parseBaseRequired := []ServiceKey{ServiceKeyDiagnostics}
	parseBaseOptional := []ServiceKey{ServiceKeyRoute}
	buildRegistration := PluginRegistration{
		Manifest: PluginManifestOverride{
			ID:               " override ",
			Version:          " 2.0.0 ",
			Description:      " replacement ",
			RequiredServices: []ServiceKey{ServiceKeyDOM},
			OptionalServices: []ServiceKey{ServiceKeyStyle},
			ActivationPolicy: ActivationPolicySession,
		},
		Factory: func() Plugin {
			return buildTestPlugin{buildManifest: Manifest{
				ID:               "base",
				Version:          "1.0.0",
				Description:      "base description",
				RequiredServices: parseBaseRequired,
				OptionalServices: parseBaseOptional,
			}}
		},
	}

	parseManifest, parseErr := mergeRegistrationManifest(buildRegistration)
	if parseErr != nil {
		parseT.Fatalf("mergeRegistrationManifest() error = %v", parseErr)
	}
	if parseManifest.ID != "override" || parseManifest.Version != "2.0.0" || parseManifest.Description != "replacement" {
		parseT.Fatalf("override fields not applied: %#v", parseManifest)
	}
	if parseManifest.ActivationPolicy != ActivationPolicySession {
		parseT.Fatalf("ActivationPolicy = %q", parseManifest.ActivationPolicy)
	}
	if len(parseManifest.RequiredServices) != 1 || parseManifest.RequiredServices[0] != ServiceKeyDOM {
		parseT.Fatalf("RequiredServices = %#v", parseManifest.RequiredServices)
	}
	if len(parseManifest.OptionalServices) != 1 || parseManifest.OptionalServices[0] != ServiceKeyStyle {
		parseT.Fatalf("OptionalServices = %#v", parseManifest.OptionalServices)
	}
	parseManifest.RequiredServices[0] = ServiceKeyFetch
	parseManifest.OptionalServices[0] = ServiceKeyFetch
	if parseBaseRequired[0] != ServiceKeyDiagnostics || parseBaseOptional[0] != ServiceKeyRoute {
		parseT.Fatal("merged manifest aliased source service slices")
	}
}

func TestManifestValidationFailures(parseT *testing.T) {
	parseCases := []struct {
		parseName         string
		parseRegistration PluginRegistration
		parseWant         string
	}{
		{
			parseName: "nil factory",
			parseRegistration: PluginRegistration{
				Factory: nil,
			},
			parseWant: "plugin factory is required",
		},
		{
			parseName: "nil plugin",
			parseRegistration: PluginRegistration{
				Factory: func() Plugin { return nil },
			},
			parseWant: "returned nil plugin",
		},
		{
			parseName: "blank id",
			parseRegistration: PluginRegistration{
				Factory: func() Plugin {
					return buildTestPlugin{buildManifest: Manifest{Version: "1.0.0"}}
				},
			},
			parseWant: "manifest ID is required",
		},
		{
			parseName: "blank version",
			parseRegistration: PluginRegistration{
				Factory: func() Plugin {
					return buildTestPlugin{buildManifest: Manifest{ID: "missing-version"}}
				},
			},
			parseWant: "manifest version is required",
		},
		{
			parseName: "bad activation",
			parseRegistration: PluginRegistration{
				Factory: func() Plugin {
					return buildTestPlugin{buildManifest: Manifest{ID: "bad-policy", Version: "1.0.0", ActivationPolicy: "later"}}
				},
			},
			parseWant: "invalid activation policy",
		},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.parseName, func(parseT *testing.T) {
			_, parseErr := mergeRegistrationManifest(parseCase.parseRegistration)
			if parseErr == nil || !strings.Contains(parseErr.Error(), parseCase.parseWant) {
				parseT.Fatalf("mergeRegistrationManifest() error = %v, want containing %q", parseErr, parseCase.parseWant)
			}
		})
	}
}

func TestContributionValidationFailures(parseT *testing.T) {
	parseCases := []struct {
		parseName         string
		parseRegistration ContributionRegistration
		parseWant         string
	}{
		{
			parseName: "blank id",
			parseRegistration: ContributionRegistration{
				Metadata: ContributionMetadata{Kind: ContributionKindDevtoolsSection},
				Value:    DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) { return nil, nil }),
			},
			parseWant: "contribution ID is required",
		},
		{
			parseName: "unsupported kind",
			parseRegistration: ContributionRegistration{
				Metadata: ContributionMetadata{ID: "bad-kind", Kind: "unknown"},
				Value:    "value",
			},
			parseWant: "unsupported contribution kind",
		},
		{
			parseName: "unsupported activation",
			parseRegistration: ContributionRegistration{
				Metadata: ContributionMetadata{ID: "bad-activation", Kind: ContributionKindDevtoolsSection, ActivationPolicy: "later"},
				Value:    "value",
			},
			parseWant: "activation policy",
		},
		{
			parseName: "unsupported execution class",
			parseRegistration: ContributionRegistration{
				Metadata: ContributionMetadata{ID: "bad-exec", Kind: ContributionKindDevtoolsSection, ExecutionClass: "instant"},
				Value:    "value",
			},
			parseWant: "unsupported execution class",
		},
		{
			parseName: "nil value",
			parseRegistration: ContributionRegistration{
				Metadata: ContributionMetadata{ID: "nil-value", Kind: ContributionKindDevtoolsSection},
			},
			parseWant: "value is required",
		},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.parseName, func(parseT *testing.T) {
			parseErr := validateContributionRegistration(parseCase.parseRegistration)
			if parseErr == nil || !strings.Contains(parseErr.Error(), parseCase.parseWant) {
				parseT.Fatalf("validateContributionRegistration() error = %v, want containing %q", parseErr, parseCase.parseWant)
			}
		})
	}
}

func TestContributionProvidersAndActionClonesAreIsolated(parseT *testing.T) {
	if parseSections, parseErr := (DevtoolsSectionProviderFunc(nil)).GetDevtoolsSections(); parseErr != nil || parseSections != nil {
		parseT.Fatalf("nil section provider = %#v, %v", parseSections, parseErr)
	}
	if parseActions, parseErr := (DevtoolsActionProviderFunc(nil)).GetDevtoolsActions(); parseErr != nil || parseActions != nil {
		parseT.Fatalf("nil action provider = %#v, %v", parseActions, parseErr)
	}

	parseActions := []DevtoolsAction{{
		Label:        "Fix",
		MatchCodes:   []string{"GWC001"},
		MatchSources: []string{"runtime"},
	}}
	parseClone := cloneDevtoolsActions(parseActions)
	parseClone[0].MatchCodes[0] = "changed"
	parseClone[0].MatchSources[0] = "changed"
	if parseActions[0].MatchCodes[0] != "GWC001" || parseActions[0].MatchSources[0] != "runtime" {
		parseT.Fatalf("cloneDevtoolsActions aliased source: %#v", parseActions)
	}
}

func TestKernelClosedStateBlocksServicesAndContributions(parseT *testing.T) {
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Services: []ServiceRegistration{{Key: ServiceKeyDiagnostics, Value: "diag"}},
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{buildManifest: Manifest{ID: "closed-plugin", Version: "1.0.0"}}
			},
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("NewKernel() error = %v", parseErr)
	}
	if parseErr := buildKernel.Close(); parseErr != nil {
		parseT.Fatalf("Close() error = %v", parseErr)
	}
	if parseErr := buildKernel.Close(); parseErr != nil {
		parseT.Fatalf("second Close() error = %v", parseErr)
	}
	if parseValue, parseOK := buildKernel.ResolveService(ServiceKeyDiagnostics); parseValue != nil || parseOK {
		parseT.Fatalf("closed ResolveService() = %v, %t; want nil, false", parseValue, parseOK)
	}
	buildKernel.SetService(ServiceKeyRoute, "route")
	if parseValue, parseOK := buildKernel.ResolveService(ServiceKeyRoute); parseValue != nil || parseOK {
		parseT.Fatalf("closed SetService should be ignored, got %v, %t", parseValue, parseOK)
	}
	if parseReports := (*Kernel)(nil).HealthReports(); parseReports != nil {
		parseT.Fatalf("nil HealthReports() = %#v", parseReports)
	}
	if parseDiagnostics := (*Kernel)(nil).Diagnostics(); parseDiagnostics != nil {
		parseT.Fatalf("nil Diagnostics() = %#v", parseDiagnostics)
	}
}

func TestKernelProviderErrorsProgressToQuarantine(parseT *testing.T) {
	buildErr := errors.New("provider failed")
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "flaky-plugin", Version: "1.0.0"},
					buildStart: func(parseContext Context) (Handle, error) {
						return nil, parseContext.RegisterContribution(ContributionRegistration{
							Metadata: ContributionMetadata{
								ID:               "flaky-section",
								Kind:             ContributionKindDevtoolsSection,
								ActivationPolicy: ActivationPolicyBoot,
							},
							Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
								return nil, buildErr
							}),
						})
					},
				}
			},
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("NewKernel() error = %v", parseErr)
	}

	if parseSections := buildKernel.ListDevtoolsSections(); len(parseSections) != 0 {
		parseT.Fatalf("first ListDevtoolsSections() = %#v", parseSections)
	}
	parseReports := buildKernel.HealthReports()
	if len(parseReports) != 1 || parseReports[0].State != HealthStateDegraded || parseReports[0].ErrorCount != 1 {
		parseT.Fatalf("after first provider failure health = %#v", parseReports)
	}

	if parseSections := buildKernel.ListDevtoolsSections(); len(parseSections) != 0 {
		parseT.Fatalf("second ListDevtoolsSections() = %#v", parseSections)
	}
	parseReports = buildKernel.HealthReports()
	if len(parseReports) != 1 || parseReports[0].State != HealthStateQuarantined || parseReports[0].ErrorCount != 2 {
		parseT.Fatalf("after repeated provider failure health = %#v", parseReports)
	}
	if parseDiagnostics := buildKernel.Diagnostics(); len(parseDiagnostics) != 2 {
		parseT.Fatalf("Diagnostics() count = %d, want 2: %#v", len(parseDiagnostics), parseDiagnostics)
	}
}

func TestKernelStartPluginFailuresRecordHealthAndDiagnostics(parseT *testing.T) {
	parseT.Run("start error", func(parseT *testing.T) {
		buildKernel, parseErr := NewKernel(BootstrapOptions{
			Registrations: []PluginRegistration{{
				Factory: func() Plugin {
					return buildTestPlugin{
						buildManifest: Manifest{ID: "start-error", Version: "1.0.0"},
						buildStart: func(Context) (Handle, error) {
							return nil, errors.New("start failed")
						},
					}
				},
			}},
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "start failed") {
			parseT.Fatalf("NewKernel() error = %v", parseErr)
		}
		if buildKernel != nil {
			parseT.Fatalf("NewKernel() returned kernel on startup failure: %#v", buildKernel)
		}
	})

	parseT.Run("nil runtime plugin", func(parseT *testing.T) {
		parseKernel := &Kernel{
			getInfo:     KernelInfo{APIVersion: "test"},
			getServices: map[ServiceKey]any{},
			getPlugins:  map[string]*pluginState{},
			getHealth:   newHealthStateStore(),
		}
		parseErr := parseKernel.startPlugin(mergedRegistration{
			getManifest: Manifest{ID: "nil-runtime", Version: "1.0.0"},
			getFactory:  func() Plugin { return nil },
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "returned nil plugin") {
			parseT.Fatalf("startPlugin() error = %v", parseErr)
		}
		parseReports := parseKernel.HealthReports()
		if len(parseReports) != 1 || parseReports[0].State != HealthStateQuarantined || parseReports[0].Reason != HealthReasonStartupError {
			parseT.Fatalf("health after nil runtime plugin = %#v", parseReports)
		}
		if parseDiagnostics := parseKernel.Diagnostics(); len(parseDiagnostics) != 1 {
			parseT.Fatalf("Diagnostics() = %#v", parseDiagnostics)
		}
	})
}
