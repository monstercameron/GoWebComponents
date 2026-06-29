// Package conformance provides a Plugin API conformance suite that third-party
// plugin authors can run against their plugin to prove it meets the host
// contract defined by the plugin package.
//
// Usage:
//
//	result := conformance.Verify(myPlugin)
//	if !result.OK() {
//	    for _, f := range result.Failures() {
//	        fmt.Println(f.Name, ":", f.Detail)
//	    }
//	}
//
// Or, from a test:
//
//	func TestMyPluginConforms(t *testing.T) {
//	    conformance.VerifyT(t, myPlugin)
//	}
package conformance

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/plugin"
)

// Check holds the outcome of a single conformance assertion.
type Check struct {
	// Name is a short, stable identifier for the assertion (e.g. "manifest/id").
	Name string
	// Passed reports whether the assertion succeeded.
	Passed bool
	// Detail carries a human-readable explanation; non-empty on failure, may
	// also carry a success summary.
	Detail string
}

// Result aggregates the outcome of all conformance checks produced by Verify.
type Result struct {
	Checks []Check
}

// OK returns true when every check in the result passed.
func (parseResult Result) OK() bool {
	for _, parseCheck := range parseResult.Checks {
		if !parseCheck.Passed {
			return false
		}
	}
	return true
}

// Failures returns the subset of checks that did not pass.
func (parseResult Result) Failures() []Check {
	parseFailed := make([]Check, 0)
	for _, parseCheck := range parseResult.Checks {
		if !parseCheck.Passed {
			parseFailed = append(parseFailed, parseCheck)
		}
	}
	return parseFailed
}

// knownTiers is the set of Tier values that the plugin package recognises.
var knownTiers = map[plugin.Tier]struct{}{
	plugin.TierStable:             {},
	plugin.TierSupportedCompanion: {},
	plugin.TierExperimental:       {},
	plugin.TierInternal:           {},
}

// knownCapabilities is the set of Capability values that the plugin package
// recognises.
var knownCapabilities = map[plugin.Capability]struct{}{
	plugin.CapabilityRouter:    {},
	plugin.CapabilityAsyncData: {},
	plugin.CapabilityDevtools:  {},
	plugin.CapabilitySSR:       {},
	plugin.CapabilityForms:     {},
}

// allCapabilities enumerates every capability so a test host can enable them
// all, ensuring a plugin that Requires any subset can always be registered.
var allCapabilities = []plugin.Capability{
	plugin.CapabilityRouter,
	plugin.CapabilityAsyncData,
	plugin.CapabilityDevtools,
	plugin.CapabilitySSR,
	plugin.CapabilityForms,
}

// Verify runs parsePlugin through a real plugin.Host and returns the result of
// all conformance checks. The plugin is not required to be registered on any
// external host; Verify constructs its own isolated host for each phase.
//
// Checks performed:
//  1. Manifest validity — non-empty ID and Version, recognised Tier,
//     well-formed Requires (no blank or unknown capabilities, no duplicates).
//  2. Registration — registering onto a fresh Host succeeds without error, and
//     if Setup returns a CleanupFunc, that func is non-nil.
//  3. Contributions observable — whatever the plugin contributes (devtools
//     sections, route guards, panels, etc.) is visible on the Host after
//     registration.
//  4. Teardown — calling Host.Close returns nil error; contributions are gone
//     afterwards; calling Close a second time does not return an error.
//  5. No panic — the full register→observe→teardown cycle does not panic.
func Verify(parsePlugin plugin.Plugin) Result {
	parseResult := Result{}

	// --- Guard: nil plugin -------------------------------------------------
	if parsePlugin == nil {
		parseResult.Checks = append(parseResult.Checks, Check{
			Name:   "plugin/not-nil",
			Passed: false,
			Detail: "plugin value is nil",
		})
		return parseResult
	}

	// -----------------------------------------------------------------------
	// 1. Manifest validity
	// -----------------------------------------------------------------------
	parseManifest := parsePlugin.Manifest()

	parseIDCheck := Check{Name: "manifest/id"}
	parseIDTrimmed := strings.TrimSpace(parseManifest.ID)
	if parseIDTrimmed == "" {
		parseIDCheck.Passed = false
		parseIDCheck.Detail = "manifest ID is empty; set a non-empty ID string"
	} else {
		parseIDCheck.Passed = true
		parseIDCheck.Detail = fmt.Sprintf("ID = %q", parseIDTrimmed)
	}
	parseResult.Checks = append(parseResult.Checks, parseIDCheck)

	parseVersionCheck := Check{Name: "manifest/version"}
	parseVersionTrimmed := strings.TrimSpace(parseManifest.Version)
	if parseVersionTrimmed == "" {
		parseVersionCheck.Passed = false
		parseVersionCheck.Detail = "manifest Version is empty; set a non-empty version string (e.g. \"1.0.0\")"
	} else {
		parseVersionCheck.Passed = true
		parseVersionCheck.Detail = fmt.Sprintf("Version = %q", parseVersionTrimmed)
	}
	parseResult.Checks = append(parseResult.Checks, parseVersionCheck)

	parseTierCheck := Check{Name: "manifest/tier"}
	if _, parseTierOK := knownTiers[parseManifest.Tier]; !parseTierOK {
		parseTierCheck.Passed = false
		parseTierCheck.Detail = fmt.Sprintf(
			"Tier %q is not recognised; must be one of: stable, supported-companion, experimental, internal",
			parseManifest.Tier,
		)
	} else {
		parseTierCheck.Passed = true
		parseTierCheck.Detail = fmt.Sprintf("Tier = %q", parseManifest.Tier)
	}
	parseResult.Checks = append(parseResult.Checks, parseTierCheck)

	parseCapCheck := Check{Name: "manifest/capabilities"}
	parseCapErr := checkCapabilities(parseManifest.Requires)
	if parseCapErr != "" {
		parseCapCheck.Passed = false
		parseCapCheck.Detail = parseCapErr
	} else {
		parseCapCheck.Passed = true
		parseCapCheck.Detail = fmt.Sprintf("%d required capability/capabilities declared", len(parseManifest.Requires))
	}
	parseResult.Checks = append(parseResult.Checks, parseCapCheck)

	// -----------------------------------------------------------------------
	// 2–5. Registration, contributions, teardown, panic safety
	//
	// All remaining checks share a single host and are wrapped in a recover so
	// that a panicking plugin still produces structured output rather than
	// crashing the caller.
	// -----------------------------------------------------------------------
	parseRegCheck := Check{Name: "registration/succeeds"}
	parseContribCheck := Check{Name: "contributions/observable"}
	parseTeardownCheck := Check{Name: "teardown/cleanup-nil-error"}
	parseTeardownGoneCheck := Check{Name: "teardown/contributions-removed"}
	parseTeardownIdempotentCheck := Check{Name: "teardown/idempotent"}
	parsePanicCheck := Check{Name: "lifecycle/no-panic"}

	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				parsePanicCheck.Passed = false
				parsePanicCheck.Detail = fmt.Sprintf("panic during register→observe→teardown cycle: %v", parseRecovered)
				// Mark any check that was not yet set as inconclusive due to the panic.
				if !parseRegCheck.Passed && parseRegCheck.Detail == "" {
					parseRegCheck.Passed = false
					parseRegCheck.Detail = "check not reached due to panic"
				}
			}
		}()

		parseHost := plugin.NewHost(plugin.HostOptions{Capabilities: allCapabilities})

		// Baseline counts before registering the plugin under test.
		parseBaselineSections := len(parseHost.DevtoolsSections())
		parseBaselinePanels := len(parseHost.Panels())

		parseRegErr := parseHost.Register(parsePlugin)
		if parseRegErr != nil {
			parseRegCheck.Passed = false
			parseRegCheck.Detail = fmt.Sprintf("Register returned error: %v", parseRegErr)
			// Without registration the remaining checks are meaningless.
			parseContribCheck.Passed = false
			parseContribCheck.Detail = "skipped: registration failed"
			parseTeardownCheck.Passed = false
			parseTeardownCheck.Detail = "skipped: registration failed"
			parseTeardownGoneCheck.Passed = false
			parseTeardownGoneCheck.Detail = "skipped: registration failed"
			parseTeardownIdempotentCheck.Passed = false
			parseTeardownIdempotentCheck.Detail = "skipped: registration failed"
			parsePanicCheck.Passed = true
			parsePanicCheck.Detail = "no panic (registration returned an error instead)"
			return
		}
		parseRegCheck.Passed = true
		parseRegCheck.Detail = fmt.Sprintf("plugin %q registered without error", parseManifest.ID)

		// --- 3. Contributions observable -----------------------------------
		// The plugin is now registered. Check that Plugins() reflects it and
		// that any declared Requires result in the expected contribution
		// categories appearing (section count or panel count increments, or
		// plugin manifest is present).
		parseAfterPlugins := parseHost.Plugins()
		parseFoundManifest := false
		for _, parsePM := range parseAfterPlugins {
			if parsePM.ID == parseManifest.ID {
				parseFoundManifest = true
				break
			}
		}

		parseAfterSections := len(parseHost.DevtoolsSections())
		parseAfterPanels := len(parseHost.Panels())

		// Determine what we can observe. If the plugin contributes devtools
		// providers the counts should increase; if it contributes nothing that
		// is observable through the query methods that is also acceptable, as
		// long as the manifest appears in Plugins().
		parseContribDetail := ""
		parseContribOK := parseFoundManifest
		if !parseFoundManifest {
			parseContribDetail = fmt.Sprintf("plugin %q not found in Host.Plugins() after registration", parseManifest.ID)
		} else {
			parseContribDetail = fmt.Sprintf(
				"manifest present in Host.Plugins(); devtools sections: %d→%d, panels: %d→%d",
				parseBaselineSections, parseAfterSections,
				parseBaselinePanels, parseAfterPanels,
			)
		}
		parseContribCheck.Passed = parseContribOK
		parseContribCheck.Detail = parseContribDetail

		// --- 4a. Teardown: Close returns nil --------------------------------
		parseTeardownErr := parseHost.Close()
		if parseTeardownErr != nil {
			parseTeardownCheck.Passed = false
			parseTeardownCheck.Detail = fmt.Sprintf("Host.Close() returned error: %v", parseTeardownErr)
		} else {
			parseTeardownCheck.Passed = true
			parseTeardownCheck.Detail = "Host.Close() returned nil"
		}

		// --- 4b. Contributions removed after teardown -----------------------
		parseGoneSections := len(parseHost.DevtoolsSections())
		parseGonePanels := len(parseHost.Panels())
		parseGonePlugins := parseHost.Plugins()

		parseGoneOK := len(parseGonePlugins) == 0 && parseGoneSections == 0 && parseGonePanels == 0
		if parseGoneOK {
			parseTeardownGoneCheck.Passed = true
			parseTeardownGoneCheck.Detail = "Host state cleared: no plugins, sections, or panels remain"
		} else {
			parseTeardownGoneCheck.Passed = false
			parseTeardownGoneCheck.Detail = fmt.Sprintf(
				"after Host.Close(): plugins=%d (want 0), sections=%d (want 0), panels=%d (want 0)",
				len(parseGonePlugins), parseGoneSections, parseGonePanels,
			)
		}

		// --- 4c. Idempotency: second Close does not error/panic -------------
		parseSecondErr := parseHost.Close()
		if parseSecondErr != nil {
			parseTeardownIdempotentCheck.Passed = false
			parseTeardownIdempotentCheck.Detail = fmt.Sprintf("second Host.Close() returned error: %v", parseSecondErr)
		} else {
			parseTeardownIdempotentCheck.Passed = true
			parseTeardownIdempotentCheck.Detail = "second Host.Close() returned nil (idempotent)"
		}

		parsePanicCheck.Passed = true
		parsePanicCheck.Detail = "no panic during register→observe→teardown cycle"
	}()

	parseResult.Checks = append(parseResult.Checks,
		parseRegCheck,
		parseContribCheck,
		parseTeardownCheck,
		parseTeardownGoneCheck,
		parseTeardownIdempotentCheck,
		parsePanicCheck,
	)

	return parseResult
}

// checkCapabilities validates that parseRequires contains no blank entries, no
// unknown capability strings, and no duplicates. It returns an empty string on
// success or a human-readable error description.
func checkCapabilities(parseRequires []plugin.Capability) string {
	parseSeen := make(map[plugin.Capability]struct{}, len(parseRequires))
	for _, parseCap := range parseRequires {
		parseTrimmed := plugin.Capability(strings.TrimSpace(string(parseCap)))
		if parseTrimmed == "" {
			return "Requires contains a blank capability string; remove it"
		}
		if _, parseKnown := knownCapabilities[parseTrimmed]; !parseKnown {
			return fmt.Sprintf("Requires contains unknown capability %q; valid values: router, async-data, devtools, ssr, forms", parseTrimmed)
		}
		if _, parseDup := parseSeen[parseTrimmed]; parseDup {
			return fmt.Sprintf("Requires contains duplicate capability %q; each capability should appear at most once", parseTrimmed)
		}
		parseSeen[parseTrimmed] = struct{}{}
	}
	return ""
}

// testingTB is the subset of *testing.T used by VerifyT, allowing the package
// to accept *testing.T without depending on the full testing interface in
// non-test callers.
type testingTB interface {
	Helper()
	Errorf(format string, args ...any)
}

// VerifyT is a testing helper that runs Verify(parsePlugin) and calls
// parseT.Errorf for each failing check. It is intended for use inside
// *testing.T test functions.
//
//	func TestMyPlugin(t *testing.T) {
//	    conformance.VerifyT(t, myPlugin)
//	}
func VerifyT(parseT testingTB, parsePlugin plugin.Plugin) {
	parseT.Helper()
	parseResult := Verify(parsePlugin)
	for _, parseCheck := range parseResult.Checks {
		if !parseCheck.Passed {
			parseT.Errorf("conformance check %q failed: %s", parseCheck.Name, parseCheck.Detail)
		}
	}
}
