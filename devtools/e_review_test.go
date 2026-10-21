package devtools

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/logging"
	"github.com/monstercameron/GoWebComponents/v7/plugin"
)

// TestRedactSupportStringCatchesQuotedAndSchemeSecrets verifies secrets that
// the first-generation patterns missed are redacted: JSON-quoted keys, an
// Authorization scheme followed by a credential, whole Cookie headers, and URL
// userinfo or fragment credentials.
func TestRedactSupportStringCatchesQuotedAndSchemeSecrets(parseT *testing.T) {
	parseCases := []struct {
		name   string
		input  string
		secret string
	}{
		{"json quoted token", `request body {"token":"s3cr3tvalue"}`, "s3cr3tvalue"},
		{"json quoted password spaced", `{"password" : "hunter2hunter2"}`, "hunter2hunter2"},
		{"basic authorization", `Authorization: Basic dXNlcjpwYXNzd29yZA==`, "dXNlcjpwYXNzd29yZA"},
		{"cookie header second cookie", `Cookie: theme=dark; sid=abcdef123456`, "abcdef123456"},
		{"url userinfo", `https://alice:topsecretpw@example.com/path`, "topsecretpw"},
		{"protocol-relative url userinfo", `//alice:relativesecret@example.com/path`, "relativesecret"},
		{"url fragment token", `https://example.com/cb#access_token=frag-secret-1&state=x`, "frag-secret-1"},
		{"passphrase key", `passphrase=correct-horse-battery`, "correct-horse-battery"},
	}
	for _, parseCase := range parseCases {
		parseGot := redactSupportString(parseCase.input)
		if strings.Contains(parseGot, parseCase.secret) {
			parseT.Errorf("%s: secret survived: %q", parseCase.name, parseGot)
		}
	}
}

// TestRedactSupportStringKeepsPlainText verifies ordinary text is not mangled.
func TestRedactSupportStringKeepsPlainText(parseT *testing.T) {
	for _, parseInput := range []string{"component failed to render", "https://example.com/a?page=2#top", "/users/42"} {
		if parseGot := redactSupportString(parseInput); parseGot != parseInput {
			parseT.Errorf("plain text changed: %q -> %q", parseInput, parseGot)
		}
	}
}

// TestSanitizeSnapshotForSupportCoversRoutePathAndDiagnosticProse verifies
// route paths and diagnostic prose fields pass through redaction.
func TestSanitizeSnapshotForSupportCoversRoutePathAndDiagnosticProse(parseT *testing.T) {
	parseSnapshot := Snapshot{
		Route: Route{
			Path:  "/reset?token=routesecret1",
			Stack: []RouteStack{{ID: "r", Path: "/reset?token=stacksecret2"}},
		},
		Diagnostics: []Diagnostic{{
			Consequence: "retry with password=consecret3",
			Remediation: "set apikey=remedsecret4",
		}},
		MultiClient: MultiClient{LocalPeerID: "peer?token=peersecret5"},
	}
	parseOut := sanitizeSnapshotForSupport(parseSnapshot)
	parseText := parseOut.Route.Path + parseOut.Route.Stack[0].Path + parseOut.Diagnostics[0].Consequence +
		parseOut.Diagnostics[0].Remediation + parseOut.MultiClient.LocalPeerID
	for _, parseSecret := range []string{"routesecret1", "stacksecret2", "consecret3", "remedsecret4", "peersecret5"} {
		if strings.Contains(parseText, parseSecret) {
			parseT.Errorf("secret %q survived support sanitization: %q", parseSecret, parseText)
		}
	}
}

// TestExtensionPanelPayloadJSONIsRedacted verifies the extension bridge export
// applies the same redaction as ExportSnapshotJSON.
func TestExtensionPanelPayloadJSONIsRedacted(parseT *testing.T) {
	parseSnapshot := Snapshot{
		Logs: []Log{{Message: "login", Fields: map[string]string{"password": "extpwsecret"}}},
	}
	logging.ConfigureTelemetryRedaction(logging.RedactionPolicy{Fields: []string{"password"}})
	parseT.Cleanup(func() { logging.ConfigureTelemetryRedaction(logging.RedactionPolicy{}) })
	parseBase, parseErr := ExportSnapshotJSON(parseSnapshot)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if strings.Contains(string(parseBase), "extpwsecret") {
		parseT.Fatalf("baseline export no longer redacts under policy: %s", parseBase)
	}
	parseData, parseErr := ExportExtensionPanelPayloadJSON(parseSnapshot)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if strings.Contains(string(parseData), "extpwsecret") {
		parseT.Fatalf("extension payload leaked a secret that ExportSnapshotJSON redacts: %s", parseData)
	}
}

// TestApplyHostExtensionsCleanupIsIdempotent verifies calling one cleanup twice
// does not drop a second holder's registration of the same host.
func TestApplyHostExtensionsCleanupIsIdempotent(parseT *testing.T) {
	parseT.Cleanup(ResetExtensionSections)
	parseHost := plugin.NewHost(plugin.HostOptions{Capabilities: []plugin.Capability{plugin.CapabilityDevtools}})
	if parseErr := parseHost.AddDevtoolsSectionProvider(func() plugin.DevtoolsSection {
		return plugin.DevtoolsSection{Name: "Shared", Lines: []string{"x"}}
	}); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseFirst := ApplyHostExtensions(parseHost)
	parseSecond := ApplyHostExtensions(parseHost)
	parseT.Cleanup(parseSecond)
	parseFirst()
	parseFirst()
	parseFound := false
	for _, parseSection := range InspectComposedExtensionSections() {
		if parseSection.Name == "Shared" {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Fatal("second holder's registration was dropped by a repeated first cleanup")
	}
}
