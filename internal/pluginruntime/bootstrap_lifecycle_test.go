package pluginruntime

import (
	"strings"
	"testing"
)

// TestBootstrapServiceLifecycle exercises the global bootstrap registry:
// registration validation, replace-on-rekey semantics, clone isolation of the
// registered views, live propagation into a booted kernel, and reset.
func TestBootstrapServiceLifecycle(parseT *testing.T) {
	if parseErr := ResetGlobalKernelForTesting(); parseErr != nil {
		parseT.Fatalf("reset: %v", parseErr)
	}
	parseT.Cleanup(func() { _ = ResetGlobalKernelForTesting() })

	// Validation: missing key and missing value must be rejected.
	if parseErr := RegisterBuiltinService(ServiceRegistration{Value: 1}); parseErr == nil || !strings.Contains(parseErr.Error(), "key") {
		parseT.Fatalf("empty key accepted: %v", parseErr)
	}
	if parseErr := RegisterBuiltinService(ServiceRegistration{Key: "svc"}); parseErr == nil || !strings.Contains(parseErr.Error(), "value") {
		parseT.Fatalf("nil value accepted: %v", parseErr)
	}

	// Register, then re-register the same key: must replace, not duplicate.
	if parseErr := RegisterBuiltinService(ServiceRegistration{Key: "svc", Value: "v1"}); parseErr != nil {
		parseT.Fatalf("register v1: %v", parseErr)
	}
	if parseErr := RegisterBuiltinService(ServiceRegistration{Key: "svc", Value: "v2"}); parseErr != nil {
		parseT.Fatalf("register v2: %v", parseErr)
	}
	parseServices := RegisteredBuiltinServices()
	parseCount := 0
	for _, parseSvc := range parseServices {
		if parseSvc.Key == "svc" {
			parseCount++
			if parseSvc.Value != "v2" {
				parseT.Fatalf("re-registration did not replace: %v", parseSvc.Value)
			}
		}
	}
	if parseCount != 1 {
		parseT.Fatalf("expected exactly one svc registration, got %d", parseCount)
	}

	// The returned slice must be a clone: mutating it must not corrupt state.
	if len(parseServices) > 0 {
		parseServices[0] = ServiceRegistration{Key: "corrupted", Value: "x"}
		parseAgain := RegisteredBuiltinServices()
		for _, parseSvc := range parseAgain {
			if parseSvc.Key == "corrupted" {
				parseT.Fatal("RegisteredBuiltinServices returned aliased storage")
			}
		}
	}

	// Boot the kernel: pre-registered services must resolve; post-boot
	// registrations must propagate into the live kernel.
	parseKernel, parseErr := BootGlobalKernel(BootstrapOptions{})
	if parseErr != nil {
		parseT.Fatalf("boot: %v", parseErr)
	}
	if GetGlobalKernel() != parseKernel {
		parseT.Fatal("GetGlobalKernel did not return the booted kernel")
	}
	if parseVal, parseOk := parseKernel.ResolveService("svc"); !parseOk || parseVal != "v2" {
		parseT.Fatalf("pre-boot service did not resolve: %v %v", parseVal, parseOk)
	}
	if parseErr := RegisterBuiltinService(ServiceRegistration{Key: "late", Value: 42}); parseErr != nil {
		parseT.Fatalf("late register: %v", parseErr)
	}
	if parseVal, parseOk := parseKernel.ResolveService("late"); !parseOk || parseVal != 42 {
		parseT.Fatalf("late service did not propagate into live kernel: %v %v", parseVal, parseOk)
	}

	// Boot is idempotent while the kernel is open.
	parseSecond, parseErr := BootGlobalKernel(BootstrapOptions{})
	if parseErr != nil || parseSecond != parseKernel {
		parseT.Fatalf("second boot must return the same kernel: %v", parseErr)
	}

	// Reset closes and clears.
	if parseErr := ResetGlobalKernelForTesting(); parseErr != nil {
		parseT.Fatalf("final reset: %v", parseErr)
	}
	if GetGlobalKernel() != nil {
		parseT.Fatal("kernel survived reset")
	}
	if len(RegisteredBuiltinServices()) != 0 || len(RegisteredBuiltinPlugins()) != 0 {
		parseT.Fatal("registrations survived reset")
	}
}

// TestNormalizeDisabledPluginIDs pins trimming, dedup, and blank-dropping.
func TestNormalizeDisabledPluginIDs(parseT *testing.T) {
	parseGot := normalizeDisabledPluginIDs([]string{" a ", "b", "a", "  ", "", "b "})
	if len(parseGot) != 2 || parseGot[0] != "a" || parseGot[1] != "b" {
		parseT.Fatalf("normalize = %v", parseGot)
	}
	if normalizeDisabledPluginIDs(nil) != nil {
		parseT.Fatal("nil input must stay nil")
	}
}

// TestCloneStringMapIsolation verifies clones never alias their source.
func TestCloneStringMapIsolation(parseT *testing.T) {
	if cloneStringMap(nil) != nil {
		parseT.Fatal("nil map should clone to nil")
	}
	parseSrc := map[string]string{"k": "v"}
	parseClone := cloneStringMap(parseSrc)
	parseClone["k"] = "mutated"
	if parseSrc["k"] != "v" {
		parseT.Fatal("clone aliased source map")
	}
}
