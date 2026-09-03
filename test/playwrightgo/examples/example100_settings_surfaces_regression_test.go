//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

type example100SettingsSurfacesRegressionArtifact struct {
	HasProfilePanelDirect  bool
	HasProfileNameInput    bool
	HasProfileSaveSuccess  bool
	HasBillingPanelDirect  bool
	HasBillingPlanLabel    bool
	HasMemoriesPanelDirect bool
	HasMemoriesHeading     bool
	HasSecurityPanelDirect bool
	HasSecurityLinkedBadge bool
	GetConsoleErrorCount   int
	GetPageErrorCount      int
	GetConsoleSampleText   string
	GetPageErrorSampleText string
}

// formatExample100SettingsSurfacesRegressionSummary formats one settings-surfaces regression artifact for concise logs.
func formatExample100SettingsSurfacesRegressionSummary(parseArtifact example100SettingsSurfacesRegressionArtifact) string {
	return fmt.Sprintf(
		"profile-panel=%t name-input=%t save-success=%t billing-panel=%t billing-plan-label=%t memories-panel=%t memories-heading=%t security-panel=%t security-linked=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.HasProfilePanelDirect,
		parseArtifact.HasProfileNameInput,
		parseArtifact.HasProfileSaveSuccess,
		parseArtifact.HasBillingPanelDirect,
		parseArtifact.HasBillingPlanLabel,
		parseArtifact.HasMemoriesPanelDirect,
		parseArtifact.HasMemoriesHeading,
		parseArtifact.HasSecurityPanelDirect,
		parseArtifact.HasSecurityLinkedBadge,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100SettingsSurfacesRegressionArtifact exercises the settings modal panels via direct URL entry
// for one authenticated seeded user and records whether each surface loads its expected content.
func captureExample100SettingsSurfacesRegressionArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100SettingsSurfacesRegressionArtifact {
	parseT.Helper()
	parseArtifact := example100SettingsSurfacesRegressionArtifact{}
	parseConsoleSamples := make([]string, 0, 120)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseMu.Lock()
		defer parseMu.Unlock()
		if len(parseConsoleSamples) < 120 {
			parseConsoleSamples = append(parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
		}
		if parseMessage.Type() == "error" {
			parseArtifact.GetConsoleErrorCount++
		}
	})
	parsePage.OnPageError(func(parseErr error) {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseArtifact.GetPageErrorCount++
		if parseErr != nil && len(parsePageErrorSamples) < 12 {
			parsePageErrorSamples = append(parsePageErrorSamples, strings.TrimSpace(parseErr.Error()))
		}
	})

	// ── Login as the seeded user ──────────────────────────────────────────────────
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("settings-surfaces: goto /login: %v", parseErr)
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "settings-surfaces /login")
	if parseErr := parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("settings-surfaces: fill login email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", "password"); parseErr != nil {
		parseT.Fatalf("settings-surfaces: fill login password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("settings-surfaces: submit login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({path: window.location.pathname + window.location.search, body: ((document.body && document.body.innerText) || "").slice(0, 600)})`)
		parseT.Fatalf("settings-surfaces: wait for chat input after login: %v debug=%#v", parseErr, parseDebugValue)
	}

	// ── Panel 1: Profile — direct URL entry with default panel ───────────────────
	// Navigating to /app/settings without a panel query falls back to the profile section.
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/settings", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("settings-surfaces: goto /app/settings (profile): %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector(
		"#settings-profile",
		playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(14000)},
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({path: window.location.pathname + window.location.search, body: ((document.body && document.body.innerText) || "").slice(0, 600)})`)
		parseT.Fatalf("settings-surfaces: wait for #settings-profile: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasProfilePanelDirect = true

	// The profile pane must contain the name input field.
	if _, parseErr := parsePage.WaitForSelector("#name-input"); parseErr != nil {
		parseT.Fatalf("settings-surfaces: wait for #name-input in profile panel: %v", parseErr)
	}
	parseArtifact.HasProfileNameInput = true

	// Fill and save a display name to exercise the save path.
	// Using a value different from the seeded name ensures the input is dirty.
	if parseErr := parsePage.Fill("#name-input", "Settings Regression User"); parseErr != nil {
		parseT.Fatalf("settings-surfaces: fill name input: %v", parseErr)
	}
	if parseErr := parsePage.Click(`button:has-text("Save")`); parseErr != nil {
		parseT.Fatalf("settings-surfaces: click Save on profile panel: %v", parseErr)
	}
	// After save, the modal must remain open with no in-modal error element in the footer.
	// The settings-save-error element only appears when the gRPC client is unavailable.
	parseErrorCountValue, parseErr := parsePage.Evaluate(
		`() => document.querySelectorAll('#settings-save-error').length`,
	)
	if parseErr != nil {
		parseT.Fatalf("settings-surfaces: count #settings-save-error after profile save: %v", parseErr)
	}
	parseErrorCount, _ := parseErrorCountValue.(float64)
	if int(parseErrorCount) == 0 {
		// No settings-save-error element → save succeeded without a store-unavailable failure.
		parseArtifact.HasProfileSaveSuccess = true
	}

	// Close the settings modal via the Cancel button so the workspace shell is visible again.
	if parseErr = parsePage.Click(`button:has-text("Cancel")`); parseErr != nil {
		parseT.Fatalf("settings-surfaces: click Cancel to close settings modal: %v", parseErr)
	}
	// After cancel the profile panel pane must be absent from the DOM.
	if _, parseWaitErr := parsePage.WaitForSelector("#settings-profile", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(8000),
	}); parseWaitErr != nil {
		parseT.Fatalf("settings-surfaces: #settings-profile not hidden after Cancel: %v", parseWaitErr)
	}

	// ── Panel 2: Billing — direct URL with panel query param ─────────────────────
	if _, parseErr = parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-billing", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("settings-surfaces: goto billing panel: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForSelector(
		"#settings-billing",
		playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(14000)},
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({path: window.location.pathname + window.location.search, body: ((document.body && document.body.innerText) || "").slice(0, 600)})`)
		parseT.Fatalf("settings-surfaces: wait for #settings-billing: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasBillingPanelDirect = true

	// The billing pane must render the plan label regardless of whether exact usage data is present.
	parseHasPlanLabelValue, parseErr := parsePage.Evaluate(
		`() => document.body && document.body.innerText.includes("Current plan")`,
	)
	if parseErr != nil {
		parseT.Fatalf("settings-surfaces: check billing plan label: %v", parseErr)
	}
	if parseHasPlanLabel, _ := parseHasPlanLabelValue.(bool); parseHasPlanLabel {
		parseArtifact.HasBillingPlanLabel = true
	}

	// ── Panel 3: Remembered preferences — direct URL with panel query param ──────
	if _, parseErr = parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-memories", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("settings-surfaces: goto memories panel: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForSelector(
		"#settings-memories",
		playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(14000)},
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({path: window.location.pathname + window.location.search, body: ((document.body && document.body.innerText) || "").slice(0, 600)})`)
		parseT.Fatalf("settings-surfaces: wait for #settings-memories: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasMemoriesPanelDirect = true

	// The heading "Remembered preferences" must appear in the panel body text.
	parseHasMemHeadingValue, parseErr := parsePage.Evaluate(
		`() => document.body && document.body.innerText.includes("Remembered preferences")`,
	)
	if parseErr != nil {
		parseT.Fatalf("settings-surfaces: check memories heading: %v", parseErr)
	}
	if parseHasMemHeading, _ := parseHasMemHeadingValue.(bool); parseHasMemHeading {
		parseArtifact.HasMemoriesHeading = true
	}

	// ── Panel 4: Security — direct URL with panel query param ────────────────────
	if _, parseErr = parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-security", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("settings-surfaces: goto security panel: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForSelector(
		"#settings-security",
		playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(14000)},
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({path: window.location.pathname + window.location.search, body: ((document.body && document.body.innerText) || "").slice(0, 600)})`)
		parseT.Fatalf("settings-surfaces: wait for #settings-security: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasSecurityPanelDirect = true

	// The seeded user authenticated with a password, so the password method row must show
	// the Linked badge (an emerald-styled span) rather than the Not Linked badge.
	parseLinkedCountValue, parseErr := parsePage.Evaluate(
		`() => { var spans = document.querySelectorAll('#settings-security span'); var linked = 0; spans.forEach(function(s) { if (s.textContent.trim() === 'Linked') linked++; }); return linked; }`,
	)
	if parseErr != nil {
		parseT.Fatalf("settings-surfaces: count Linked badges in security panel: %v", parseErr)
	}
	parseLinkedCount, _ := parseLinkedCountValue.(float64)
	if int(parseLinkedCount) > 0 {
		parseArtifact.HasSecurityLinkedBadge = true
	}

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100SettingsSurfacesRegression verifies the settings modal panels load the correct content when
// navigated to directly via URL for an authenticated user. Covers: profile panel default fallback and name-input
// presence, profile save success (no store-unavailable error), billing panel with plan label, remembered-preferences
// (memories) panel heading, and security panel linked-password badge for a password-authenticated user.
func TestExample100SettingsSurfacesRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18127")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100SettingsSurfacesRegressionArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 settings-surfaces regression: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))

		if !parseArtifact.HasProfilePanelDirect {
			parseT.Fatalf("profile-panel-direct leg missing: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasProfileNameInput {
			parseT.Fatalf("profile-name-input leg missing: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasProfileSaveSuccess {
			parseT.Fatalf("profile-save-success leg missing (store-unavailable error was present after save): %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasBillingPanelDirect {
			parseT.Fatalf("billing-panel-direct leg missing: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasBillingPlanLabel {
			parseT.Fatalf("billing-plan-label leg missing: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasMemoriesPanelDirect {
			parseT.Fatalf("memories-panel-direct leg missing: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasMemoriesHeading {
			parseT.Fatalf("memories-heading leg missing: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasSecurityPanelDirect {
			parseT.Fatalf("security-panel-direct leg missing: %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasSecurityLinkedBadge {
			parseT.Fatalf("security-linked-badge leg missing (password method not showing Linked for seeded user): %s", formatExample100SettingsSurfacesRegressionSummary(parseArtifact))
		}
	})
}
