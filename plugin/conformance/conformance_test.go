package conformance_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/plugin"
	"github.com/monstercameron/GoWebComponents/plugin/conformance"
)

// ---------------------------------------------------------------------------
// Fixture helpers
// ---------------------------------------------------------------------------

// minimalConformingPlugin builds the smallest valid plugin that does not
// require any host capability.
func minimalConformingPlugin() plugin.Plugin {
	return plugin.Define(plugin.Manifest{
		ID:      "conformance.minimal",
		Version: "1.0.0",
		Tier:    plugin.TierExperimental,
	}, nil)
}

// devtoolsConformingPlugin builds a conforming plugin that contributes a
// devtools section and correctly removes it via its cleanup function. This
// exercises the contributions-observable and teardown-contributions-removed
// checks.
func devtoolsConformingPlugin() plugin.Plugin {
	return plugin.Define(plugin.Manifest{
		ID:      "conformance.devtools",
		Version: "1.0.0",
		Tier:    plugin.TierSupportedCompanion,
		Requires: []plugin.Capability{
			plugin.CapabilityDevtools,
		},
	}, func(parseHost *plugin.Host) (plugin.CleanupFunc, error) {
		parseProvider := func() plugin.DevtoolsSection {
			return plugin.DevtoolsSection{
				Name:  "Conformance Suite",
				Lines: []string{"all checks passing"},
			}
		}
		if parseErr := parseHost.AddDevtoolsSectionProvider(parseProvider); parseErr != nil {
			return nil, parseErr
		}
		// Cleanup: Host.Close() clears all providers, so returning nil here
		// delegates teardown entirely to the host — which is the standard
		// pattern. The teardown check verifies Host.Close() zeros out all
		// contributions.
		return func() error { return nil }, nil
	})
}

// ---------------------------------------------------------------------------
// 1. Built-in / kernel-style plugin conforms
// ---------------------------------------------------------------------------

// TestBuiltinKernelPluginConforms verifies that the minimal plugin produced
// by plugin.Define — the same pattern used by the framework's own internal
// plugins — passes the full conformance suite. Because the framework does not
// expose an internal kernel singleton, we use the closest faithful equivalent:
// a plugin constructed exactly as the existing plugin_test.go constructs its
// first plugin, but without test-local state that cannot be replicated here.
func TestBuiltinKernelPluginConforms(t *testing.T) {
	// This replicates the "first" plugin from plugin_test.go, minus the
	// closure-captured parseOrder slice (which is test-local state). All
	// manifest fields, capability declarations, and contributions match.
	parseKernelStyle := plugin.Define(plugin.Manifest{
		ID:          "kernel-style",
		Version:     "0.1.0",
		Description: "faithful kernel-style plugin as used in the framework tests",
		Tier:        plugin.TierExperimental,
		Requires: []plugin.Capability{
			plugin.CapabilityRouter,
			plugin.CapabilityAsyncData,
			plugin.CapabilitySSR,
			plugin.CapabilityForms,
		},
	}, func(parseHost *plugin.Host) (plugin.CleanupFunc, error) {
		parseHost.SetValue("prefix", "gwc")

		if parseErr := parseHost.AddRouteGuard(func(parseReq plugin.RouteRequest) plugin.GuardDecision {
			if parseReq.Path == "/admin" {
				return plugin.Redirect("/signin", "auth required")
			}
			return plugin.Allow("public route")
		}); parseErr != nil {
			return nil, parseErr
		}

		if parseErr := parseHost.AddCacheKeyDecorator(func(parseKey string) string {
			return "gwc:" + parseKey
		}); parseErr != nil {
			return nil, parseErr
		}

		if parseErr := parseHost.AddBootstrapProvider(func() plugin.BootstrapPayload {
			return plugin.BootstrapPayload{
				Namespace: "kernel-style",
				Data:      map[string]any{"ready": true},
			}
		}); parseErr != nil {
			return nil, parseErr
		}

		if parseErr := parseHost.AddFormValidator(func(parseSub plugin.FormSubmission) []plugin.ValidationIssue {
			if parseSub.Values["qty"] == "0" {
				return []plugin.ValidationIssue{{Field: "qty", Message: "must be > 0"}}
			}
			return nil
		}); parseErr != nil {
			return nil, parseErr
		}

		return func() error { return nil }, nil
	})

	parseResult := conformance.Verify(parseKernelStyle)
	if !parseResult.OK() {
		for _, parseFailure := range parseResult.Failures() {
			t.Errorf("kernel-style plugin conformance check %q failed: %s",
				parseFailure.Name, parseFailure.Detail)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. Conforming fixture passes
// ---------------------------------------------------------------------------

// TestConformingFixturePasses verifies that hand-written conforming plugins
// (both a no-op minimal plugin and one that contributes a devtools section)
// receive a fully passing result from Verify.
func TestConformingFixturePasses(t *testing.T) {
	parseCases := []struct {
		parseName   string
		parsePlugin plugin.Plugin
	}{
		{"minimal-no-op", minimalConformingPlugin()},
		{"devtools-section", devtoolsConformingPlugin()},
	}

	for _, parseCase := range parseCases {
		t.Run(parseCase.parseName, func(t *testing.T) {
			parseResult := conformance.Verify(parseCase.parsePlugin)
			if !parseResult.OK() {
				for _, parseFailure := range parseResult.Failures() {
					t.Errorf("check %q failed: %s", parseFailure.Name, parseFailure.Detail)
				}
			}
			// All individual checks must be present.
			if len(parseResult.Checks) == 0 {
				t.Error("Verify returned no checks at all")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. Non-conforming fixtures fail with actionable messages
// ---------------------------------------------------------------------------

// TestNonConformingFixtureFails asserts that deliberately broken plugins fail
// Verify and that the SPECIFIC check name for each violation is reported.
func TestNonConformingFixtureFails(t *testing.T) {
	// 3a. Empty manifest ID.
	t.Run("empty-id", func(t *testing.T) {
		parsePlugin := plugin.Define(plugin.Manifest{
			ID:      "", // violation
			Version: "1.0.0",
			Tier:    plugin.TierStable,
		}, nil)

		parseResult := conformance.Verify(parsePlugin)

		if parseResult.OK() {
			t.Fatal("expected Verify to fail for a plugin with empty ID, but OK() returned true")
		}
		parseFailures := parseResult.Failures()
		if !containsCheckName(parseFailures, "manifest/id") {
			t.Errorf("expected failure named \"manifest/id\"; got failures: %v", failureNames(parseFailures))
		}
	})

	// 3b. CleanupFunc that does not remove the plugin's contribution.
	//
	// This plugin's CleanupFunc is intentionally a no-op (the section provider
	// is never de-registered individually). After Host.Close() the Host itself
	// zeros everything out, so the teardown/contributions-removed check should
	// still pass. To exercise a genuine contribution-leak we test a plugin
	// whose Setup returns an error, which triggers rollback — let us instead
	// test a setup that returns an error so registration/succeeds fails.
	//
	// Actually, the Host.Close() design always clears all providers regardless
	// of the plugin's own cleanup func, so a "leaking" CleanupFunc cannot
	// cause teardown/contributions-removed to fail in this architecture. We
	// instead verify that a setup returning an error is correctly reported.
	t.Run("setup-returns-error", func(t *testing.T) {
		parsePlugin := plugin.Define(plugin.Manifest{
			ID:      "broken.setup-error",
			Version: "1.0.0",
			Tier:    plugin.TierExperimental,
		}, func(_ *plugin.Host) (plugin.CleanupFunc, error) {
			return nil, errSetupFailed
		})

		parseResult := conformance.Verify(parsePlugin)

		if parseResult.OK() {
			t.Fatal("expected Verify to fail for a plugin whose Setup returns an error, but OK() returned true")
		}
		parseFailures := parseResult.Failures()
		if !containsCheckName(parseFailures, "registration/succeeds") {
			t.Errorf("expected failure named \"registration/succeeds\"; got failures: %v", failureNames(parseFailures))
		}
	})

	// 3c. Setup function that panics.
	t.Run("setup-panics", func(t *testing.T) {
		parsePlugin := plugin.Define(plugin.Manifest{
			ID:      "broken.panics",
			Version: "1.0.0",
			Tier:    plugin.TierExperimental,
		}, func(_ *plugin.Host) (plugin.CleanupFunc, error) {
			panic("intentional panic from plugin setup")
		})

		parseResult := conformance.Verify(parsePlugin)

		if parseResult.OK() {
			t.Fatal("expected Verify to fail for a plugin whose Setup panics, but OK() returned true")
		}
		parseFailures := parseResult.Failures()
		if !containsCheckName(parseFailures, "lifecycle/no-panic") {
			t.Errorf("expected failure named \"lifecycle/no-panic\"; got failures: %v", failureNames(parseFailures))
		}
	})

	// 3d. Invalid (unrecognised) Tier.
	t.Run("invalid-tier", func(t *testing.T) {
		parsePlugin := plugin.Define(plugin.Manifest{
			ID:      "broken.bad-tier",
			Version: "1.0.0",
			Tier:    plugin.Tier("unknown-tier"), // violation
		}, nil)

		parseResult := conformance.Verify(parsePlugin)

		if parseResult.OK() {
			t.Fatal("expected Verify to fail for a plugin with an unrecognised Tier, but OK() returned true")
		}
		parseFailures := parseResult.Failures()
		if !containsCheckName(parseFailures, "manifest/tier") {
			t.Errorf("expected failure named \"manifest/tier\"; got failures: %v", failureNames(parseFailures))
		}
	})

	// 3e. Duplicate capability in Requires.
	t.Run("duplicate-capability", func(t *testing.T) {
		parsePlugin := plugin.Define(plugin.Manifest{
			ID:      "broken.dup-cap",
			Version: "1.0.0",
			Tier:    plugin.TierExperimental,
			Requires: []plugin.Capability{
				plugin.CapabilityDevtools,
				plugin.CapabilityDevtools, // duplicate
			},
		}, nil)

		parseResult := conformance.Verify(parsePlugin)

		if parseResult.OK() {
			t.Fatal("expected Verify to fail for a plugin with duplicate capabilities, but OK() returned true")
		}
		parseFailures := parseResult.Failures()
		if !containsCheckName(parseFailures, "manifest/capabilities") {
			t.Errorf("expected failure named \"manifest/capabilities\"; got failures: %v", failureNames(parseFailures))
		}
	})
}

// ---------------------------------------------------------------------------
// 4. VerifyT helper smoke test
// ---------------------------------------------------------------------------

// TestVerifyTHelperPasses confirms that VerifyT does not call t.Errorf for a
// conforming plugin (i.e. the helper itself is wired correctly).
func TestVerifyTHelperPasses(t *testing.T) {
	conformance.VerifyT(t, minimalConformingPlugin())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// errSetupFailed is a sentinel error returned by the broken setup fixture.
var errSetupFailed = fmt.Errorf("intentional setup failure")

// containsCheckName reports whether any failure in parseFailures has the given
// check name.
func containsCheckName(parseFailures []conformance.Check, parseName string) bool {
	for _, parseF := range parseFailures {
		if parseF.Name == parseName {
			return true
		}
	}
	return false
}

// failureNames returns the Names of all failed checks for use in error
// messages.
func failureNames(parseFailures []conformance.Check) []string {
	parseNames := make([]string, 0, len(parseFailures))
	for _, parseF := range parseFailures {
		parseNames = append(parseNames, parseF.Name)
	}
	return parseNames
}

// fmt is used by the errSetupFailed sentinel and must be imported.
var _ = fmt.Sprintf
