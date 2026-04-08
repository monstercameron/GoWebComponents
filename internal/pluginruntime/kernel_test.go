package pluginruntime

import (
	"errors"
	"testing"
)

type buildTestPlugin struct {
	buildManifest Manifest
	buildStart    func(Context) (Handle, error)
}

// Manifest returns the configured test manifest.
func (parsePlugin buildTestPlugin) Manifest() Manifest {
	return parsePlugin.buildManifest
}

// StartPlugin starts the configured test plugin.
func (parsePlugin buildTestPlugin) StartPlugin(parseContext Context) (Handle, error) {
	if parsePlugin.buildStart == nil {
		return nil, nil
	}
	return parsePlugin.buildStart(parseContext)
}

type buildTestHandle struct {
	buildStop func() error
}

// StopPlugin stops the configured test handle.
func (parseHandle buildTestHandle) StopPlugin() error {
	if parseHandle.buildStop == nil {
		return nil
	}
	return parseHandle.buildStop()
}

// TestNewKernelRejectsDuplicatePluginIDs verifies duplicate registrations fail deterministically.
func TestNewKernelRejectsDuplicatePluginIDs(parseT *testing.T) {
	_, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{
			{
				Factory: func() Plugin {
					return buildTestPlugin{buildManifest: Manifest{ID: "dup", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot}}
				},
			},
			{
				Factory: func() Plugin {
					return buildTestPlugin{buildManifest: Manifest{ID: "dup", Version: "1.0.1", ActivationPolicy: ActivationPolicyBoot}}
				},
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected duplicate registration error")
	}
}

// TestNewKernelRejectsMissingRequiredServices verifies required services gate startup.
func TestNewKernelRejectsMissingRequiredServices(parseT *testing.T) {
	_, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{buildManifest: Manifest{
					ID:               "needs-route",
					Version:          "1.0.0",
					RequiredServices: []ServiceKey{ServiceKeyRoute},
					ActivationPolicy: ActivationPolicyBoot,
				}}
			},
		}},
	})
	if parseErr == nil {
		parseT.Fatal("expected missing service error")
	}
}

// TestNewKernelStartsPluginsAndResolvesServices verifies startup and service resolution.
func TestNewKernelStartsPluginsAndResolvesServices(parseT *testing.T) {
	buildStarted := false
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Services: []ServiceRegistration{{Key: ServiceKeyDiagnostics, Value: "diag"}},
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "alpha", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
					buildStart: func(parseContext Context) (Handle, error) {
						buildStarted = true
						getService, hasService := parseContext.ResolveService(ServiceKeyDiagnostics)
						if !hasService || getService.(string) != "diag" {
							parseT.Fatalf("unexpected diagnostics service: value=%v ok=%t", getService, hasService)
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
	if !buildStarted {
		parseT.Fatal("expected plugin to start")
	}
	getReports := buildKernel.HealthReports()
	if len(getReports) != 1 || getReports[0].State != HealthStateHealthy {
		parseT.Fatalf("unexpected health reports: %+v", getReports)
	}
}

// TestKernelRegistersAndOrdersContributions verifies contribution ordering is deterministic.
func TestKernelRegistersAndOrdersContributions(parseT *testing.T) {
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{
			{
				Factory: func() Plugin {
					return buildTestPlugin{
						buildManifest: Manifest{ID: "b", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
						buildStart: func(parseContext Context) (Handle, error) {
							return nil, parseContext.RegisterContribution(ContributionRegistration{
								Metadata: ContributionMetadata{
									ID:               "two",
									Kind:             ContributionKindDevtoolsSection,
									ExecutionClass:   ExecutionClassWarm,
									ActivationPolicy: ActivationPolicyBoot,
									Order:            1,
								},
								Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
									return []DevtoolsSection{{Name: "second"}}, nil
								}),
							})
						},
					}
				},
			},
			{
				Factory: func() Plugin {
					return buildTestPlugin{
						buildManifest: Manifest{ID: "a", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
						buildStart: func(parseContext Context) (Handle, error) {
							return nil, parseContext.RegisterContribution(ContributionRegistration{
								Metadata: ContributionMetadata{
									ID:               "one",
									Kind:             ContributionKindDevtoolsSection,
									ExecutionClass:   ExecutionClassWarm,
									ActivationPolicy: ActivationPolicyBoot,
									Order:            5,
								},
								Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
									return []DevtoolsSection{{Name: "first"}}, nil
								}),
							})
						},
					}
				},
			},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("NewKernel() error = %v", parseErr)
	}
	getSections := buildKernel.ListDevtoolsSections()
	if len(getSections) != 2 || getSections[0].Name != "first" || getSections[1].Name != "second" {
		parseT.Fatalf("unexpected contribution ordering: %+v", getSections)
	}
}

// TestKernelQuarantinesPanickingProviders verifies panicking providers cannot unwind into callers.
func TestKernelQuarantinesPanickingProviders(parseT *testing.T) {
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "panic-plugin", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
					buildStart: func(parseContext Context) (Handle, error) {
						return nil, parseContext.RegisterContribution(ContributionRegistration{
							Metadata: ContributionMetadata{
								ID:               "panic-section",
								Kind:             ContributionKindDevtoolsSection,
								ExecutionClass:   ExecutionClassWarm,
								ActivationPolicy: ActivationPolicyBoot,
							},
							Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
								panic("boom")
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
	getSections := buildKernel.ListDevtoolsSections()
	if len(getSections) != 0 {
		parseT.Fatalf("expected no sections after quarantine, got %+v", getSections)
	}
	getReports := buildKernel.HealthReports()
	if len(getReports) != 1 || getReports[0].State != HealthStateQuarantined {
		parseT.Fatalf("expected quarantined health report, got %+v", getReports)
	}
	if len(buildKernel.Diagnostics()) == 0 {
		parseT.Fatal("expected diagnostic event for panicking provider")
	}
}

// TestKernelCloseRunsCleanupInReverseOrder verifies stop and cleanup ordering.
func TestKernelCloseRunsCleanupInReverseOrder(parseT *testing.T) {
	buildOrder := make([]string, 0, 4)
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "cleanup", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
					buildStart: func(parseContext Context) (Handle, error) {
						parseContext.RegisterCleanup(func() error {
							buildOrder = append(buildOrder, "cleanup-one")
							return nil
						})
						parseContext.RegisterCleanup(func() error {
							buildOrder = append(buildOrder, "cleanup-two")
							return nil
						})
						return buildTestHandle{buildStop: func() error {
							buildOrder = append(buildOrder, "stop")
							return nil
						}}, nil
					},
				}
			},
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("NewKernel() error = %v", parseErr)
	}
	if parseErr2 := buildKernel.Close(); parseErr2 != nil {
		parseT.Fatalf("Close() error = %v", parseErr2)
	}
	if len(buildOrder) != 3 || buildOrder[0] != "stop" || buildOrder[1] != "cleanup-two" || buildOrder[2] != "cleanup-one" {
		parseT.Fatalf("unexpected cleanup order: %+v", buildOrder)
	}
}

// TestKernelReportsActionErrorsWithoutPanicking verifies action callbacks stay guarded.
func TestKernelReportsActionErrorsWithoutPanicking(parseT *testing.T) {
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "action-plugin", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
					buildStart: func(parseContext Context) (Handle, error) {
						return nil, parseContext.RegisterContribution(ContributionRegistration{
							Metadata: ContributionMetadata{
								ID:               "action",
								Kind:             ContributionKindDevtoolsAction,
								ExecutionClass:   ExecutionClassWarm,
								ActivationPolicy: ActivationPolicyBoot,
							},
							Value: DevtoolsActionProviderFunc(func() ([]DevtoolsAction, error) {
								return []DevtoolsAction{{
									Label: "Retry",
									Run: func(parseContext DevtoolsActionContext) error {
										return errors.New("action failed")
									},
								}}, nil
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
	getActions := buildKernel.ListDevtoolsActions()
	if len(getActions) != 1 {
		parseT.Fatalf("expected one action, got %+v", getActions)
	}
	if parseErr2 := getActions[0].Run(DevtoolsActionContext{}); parseErr2 == nil {
		parseT.Fatal("expected guarded action error")
	}
	if len(buildKernel.Diagnostics()) == 0 {
		parseT.Fatal("expected diagnostic event for action failure")
	}
}

func TestKernelCloseRemovesStoppedPluginContributions(parseT *testing.T) {
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "close-plugin", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
					buildStart: func(parseContext Context) (Handle, error) {
						return nil, parseContext.RegisterContribution(ContributionRegistration{
							Metadata: ContributionMetadata{
								ID:               "close-section",
								Kind:             ContributionKindDevtoolsSection,
								ExecutionClass:   ExecutionClassWarm,
								ActivationPolicy: ActivationPolicyBoot,
							},
							Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
								return []DevtoolsSection{{Name: "close"}}, nil
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
	if getSections := buildKernel.ListDevtoolsSections(); len(getSections) != 1 || getSections[0].Name != "close" {
		parseT.Fatalf("expected one section before close, got %+v", getSections)
	}
	if parseErr2 := buildKernel.Close(); parseErr2 != nil {
		parseT.Fatalf("Close() error = %v", parseErr2)
	}
	if getSections := buildKernel.ListDevtoolsSections(); len(getSections) != 0 {
		parseT.Fatalf("expected no sections after close, got %+v", getSections)
	}
}

// TestBootGlobalKernelMergesBuiltinServices verifies built-in services participate in global boot.
func TestBootGlobalKernelMergesBuiltinServices(parseT *testing.T) {
	parseT.Cleanup(func() {
		if parseErr := ResetGlobalKernelForTesting(); parseErr != nil {
			parseT.Fatalf("ResetGlobalKernelForTesting() error = %v", parseErr)
		}
	})
	if parseErr := RegisterBuiltinService(ServiceRegistration{Key: ServiceKeyDiagnostics, Value: "builtin"}); parseErr != nil {
		parseT.Fatalf("RegisterBuiltinService() error = %v", parseErr)
	}
	buildKernel, parseErr := BootGlobalKernel(BootstrapOptions{})
	if parseErr != nil {
		parseT.Fatalf("BootGlobalKernel() error = %v", parseErr)
	}
	getService, hasService := buildKernel.ResolveService(ServiceKeyDiagnostics)
	if !hasService || getService.(string) != "builtin" {
		parseT.Fatalf("expected builtin diagnostics service, got value=%v ok=%t", getService, hasService)
	}
}

// TestRegisterBuiltinPluginAfterBootStartsImmediately verifies late built-ins attach to an already booted kernel.
func TestRegisterBuiltinPluginAfterBootStartsImmediately(parseT *testing.T) {
	parseT.Cleanup(func() {
		if parseErr := ResetGlobalKernelForTesting(); parseErr != nil {
			parseT.Fatalf("ResetGlobalKernelForTesting() error = %v", parseErr)
		}
	})
	buildKernel, parseErr := BootGlobalKernel(BootstrapOptions{})
	if parseErr != nil {
		parseT.Fatalf("BootGlobalKernel() error = %v", parseErr)
	}
	if parseErr2 := RegisterBuiltinPlugin(PluginRegistration{
		Factory: func() Plugin {
			return buildTestPlugin{
				buildManifest: Manifest{ID: "late-plugin", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
				buildStart: func(parseContext Context) (Handle, error) {
					return nil, parseContext.RegisterContribution(ContributionRegistration{
						Metadata: ContributionMetadata{
							ID:               "late-section",
							Kind:             ContributionKindDevtoolsSection,
							ExecutionClass:   ExecutionClassWarm,
							ActivationPolicy: ActivationPolicyBoot,
						},
						Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
							return []DevtoolsSection{{Name: "late"}}, nil
						}),
					})
				},
			}
		},
	}); parseErr2 != nil {
		parseT.Fatalf("RegisterBuiltinPlugin() error = %v", parseErr2)
	}
	getSections := buildKernel.ListDevtoolsSections()
	if len(getSections) != 1 || getSections[0].Name != "late" {
		parseT.Fatalf("expected late plugin section after boot, got %+v", getSections)
	}
}

func TestBootGlobalKernelRebuildsAfterClose(parseT *testing.T) {
	parseT.Cleanup(func() {
		if parseErr := ResetGlobalKernelForTesting(); parseErr != nil {
			parseT.Fatalf("ResetGlobalKernelForTesting() error = %v", parseErr)
		}
	})

	parseStarts := 0
	if parseErr := RegisterBuiltinPlugin(PluginRegistration{
		Factory: func() Plugin {
			return buildTestPlugin{
				buildManifest: Manifest{ID: "reboot-plugin", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
				buildStart: func(parseContext Context) (Handle, error) {
					parseStarts++
					return nil, parseContext.RegisterContribution(ContributionRegistration{
						Metadata: ContributionMetadata{
							ID:               "reboot-section",
							Kind:             ContributionKindDevtoolsSection,
							ExecutionClass:   ExecutionClassWarm,
							ActivationPolicy: ActivationPolicyBoot,
						},
						Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
							return []DevtoolsSection{{Name: "reboot"}}, nil
						}),
					})
				},
			}
		},
	}); parseErr != nil {
		parseT.Fatalf("RegisterBuiltinPlugin() error = %v", parseErr)
	}

	buildKernel, parseErr := BootGlobalKernel(BootstrapOptions{})
	if parseErr != nil {
		parseT.Fatalf("BootGlobalKernel() error = %v", parseErr)
	}
	if parseStarts != 1 {
		parseT.Fatalf("expected one initial plugin start, got %d", parseStarts)
	}
	if parseErr2 := buildKernel.Close(); parseErr2 != nil {
		parseT.Fatalf("Close() error = %v", parseErr2)
	}

	buildRebootedKernel, parseErr3 := BootGlobalKernel(BootstrapOptions{})
	if parseErr3 != nil {
		parseT.Fatalf("BootGlobalKernel() reboot error = %v", parseErr3)
	}
	if buildRebootedKernel == buildKernel {
		parseT.Fatal("expected BootGlobalKernel() to replace a closed kernel instance")
	}
	if parseStarts != 2 {
		parseT.Fatalf("expected closed global kernel to reboot built-ins, got %d starts", parseStarts)
	}
	if getSections := buildRebootedKernel.ListDevtoolsSections(); len(getSections) != 1 || getSections[0].Name != "reboot" {
		parseT.Fatalf("expected rebooted kernel section, got %+v", getSections)
	}
}

// TestDefaultExtendedServicesResolve verifies the kernel exposes baseline extended service families.
func TestDefaultExtendedServicesResolve(parseT *testing.T) {
	parseT.Cleanup(func() {
		if parseErr := ResetGlobalKernelForTesting(); parseErr != nil {
			parseT.Fatalf("ResetGlobalKernelForTesting() error = %v", parseErr)
		}
	})
	for _, parseRegistration := range []ServiceRegistration{
		{Key: ServiceKeyDOM, Value: buildDefaultDOMService{}},
		{Key: ServiceKeyStyle, Value: buildDefaultStyleService{}},
		{Key: ServiceKeyEvents, Value: buildDefaultEventService{}},
		{Key: ServiceKeyAssets, Value: buildDefaultAssetService{}},
		{Key: ServiceKeySecurity, Value: buildDefaultSecurityService{}},
		{Key: ServiceKeyCapture, Value: buildDefaultCaptureService{}},
	} {
		if parseErr := RegisterBuiltinService(parseRegistration); parseErr != nil {
			parseT.Fatalf("RegisterBuiltinService(%q) error = %v", parseRegistration.Key, parseErr)
		}
	}
	buildKernel, parseErr := BootGlobalKernel(BootstrapOptions{})
	if parseErr != nil {
		parseT.Fatalf("BootGlobalKernel() error = %v", parseErr)
	}
	for _, parseKey := range []ServiceKey{
		ServiceKeyDOM,
		ServiceKeyStyle,
		ServiceKeyEvents,
		ServiceKeyAssets,
		ServiceKeySecurity,
		ServiceKeyCapture,
	} {
		if _, hasService := buildKernel.ResolveService(parseKey); !hasService {
			parseT.Fatalf("expected default service %q to resolve", parseKey)
		}
	}
}
