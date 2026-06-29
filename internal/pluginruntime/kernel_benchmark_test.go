package pluginruntime

import "testing"

// BenchmarkKernelResolveService benchmarks repeated service lookup from the kernel registry.
func BenchmarkKernelResolveService(parseB *testing.B) {
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Services: []ServiceRegistration{{Key: ServiceKeyDiagnostics, Value: "diag"}},
	})
	if parseErr != nil {
		parseB.Fatalf("NewKernel() error = %v", parseErr)
	}
	parseB.ReportAllocs()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, hasService := buildKernel.ResolveService(ServiceKeyDiagnostics); !hasService {
			parseB.Fatal("expected diagnostics service")
		}
	}
}

// BenchmarkKernelListDevtoolsSections benchmarks ordered section resolution.
func BenchmarkKernelListDevtoolsSections(parseB *testing.B) {
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: []PluginRegistration{{
			Factory: func() Plugin {
				return buildTestPlugin{
					buildManifest: Manifest{ID: "bench", Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot},
					buildStart: func(parseContext Context) (Handle, error) {
						return nil, parseContext.RegisterContribution(ContributionRegistration{
							Metadata: ContributionMetadata{
								ID:               "bench-sections",
								Kind:             ContributionKindDevtoolsSection,
								ExecutionClass:   ExecutionClassWarm,
								ActivationPolicy: ActivationPolicyBoot,
							},
							Value: DevtoolsSectionProviderFunc(func() ([]DevtoolsSection, error) {
								return []DevtoolsSection{{Name: "bench"}}, nil
							}),
						})
					},
				}
			},
		}},
	})
	if parseErr != nil {
		parseB.Fatalf("NewKernel() error = %v", parseErr)
	}
	parseB.ReportAllocs()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if len(buildKernel.ListDevtoolsSections()) != 1 {
			parseB.Fatal("expected one devtools section")
		}
	}
}
