package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestHostNilValidationAndCloneHelpers(parseT *testing.T) {
	var parseNilHost *Host
	if parseErr := parseNilHost.Register(nil); parseErr == nil || !strings.Contains(parseErr.Error(), "host is nil") {
		parseT.Fatalf("nil host Register() error = %v, want host is nil", parseErr)
	}
	if parseErr2 := parseNilHost.Close(); parseErr2 != nil {
		parseT.Fatalf("nil host Close() error = %v", parseErr2)
	}
	if parseNilHost.Capabilities() != nil || parseNilHost.Plugins() != nil || parseNilHost.BootstrapData() != nil || parseNilHost.ValidateForm(FormSubmission{}) != nil || parseNilHost.DevtoolsSections() != nil || parseNilHost.DevtoolsActions() != nil {
		parseT.Fatal("nil host helper methods should return nil slices/maps")
	}
	if parseDecision := parseNilHost.EvaluateRoute(RouteRequest{Path: "/"}); parseDecision.Outcome != GuardAllow {
		parseT.Fatalf("nil host EvaluateRoute() = %+v, want allow", parseDecision)
	}
	parseNilHost.NotifyNavigation(NavigationEvent{Path: "/pricing"})
	parseNilHost.NotifyRequest(RequestEvent{Key: "pricing"})
	parseNilHost.NotifySubmit(FormSubmission{ID: "submit"})

	parseHost := NewHost(HostOptions{Capabilities: []Capability{"", CapabilitySSR, CapabilityRouter, CapabilitySSR}})
	parseCaps := parseHost.Capabilities()
	if len(parseCaps) != 2 || parseCaps[0] != CapabilityRouter || parseCaps[1] != CapabilitySSR {
		parseT.Fatalf("Capabilities() = %+v, want sorted unique capabilities", parseCaps)
	}

	parseHost.SetValue("  ", "ignored")
	if _, parseOk := parseHost.Value("  "); parseOk {
		parseT.Fatal("blank key should not be stored")
	}
	parseHost.SetValue("theme", "dark")
	if parseValue, parseOk2 := parseHost.Value("theme"); !parseOk2 || parseValue.(string) != "dark" {
		parseT.Fatalf("Value(theme) = %#v, %t; want dark, true", parseValue, parseOk2)
	}

	parseManifest := Manifest{
		ID:          "seo",
		Version:     "1.0.0",
		Description: "server helpers",
		Tier:        TierStable,
		Requires:    []Capability{CapabilitySSR},
	}
	if parseErr3 := parseHost.Register(Define(parseManifest, nil)); parseErr3 != nil {
		parseT.Fatalf("Register() error = %v", parseErr3)
	}
	if parseErr4 := parseHost.Register(Define(parseManifest, nil)); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "already registered") {
		parseT.Fatalf("duplicate Register() error = %v, want duplicate error", parseErr4)
	}

	parsePlugins := parseHost.Plugins()
	parsePlugins[0].Requires[0] = CapabilityRouter
	if parseCurrent := parseHost.Plugins()[0].Requires[0]; parseCurrent != CapabilitySSR {
		parseT.Fatalf("Plugins() should return a clone, got Requires[0]=%q", parseCurrent)
	}

	parseClone := cloneManifest(parseManifest)
	parseClone.Requires[0] = CapabilityRouter
	if parseManifest.Requires[0] != CapabilitySSR {
		parseT.Fatalf("cloneManifest() should not mutate source manifest, got %#v", parseManifest.Requires)
	}

	if parseMissing := parseHost.missingCapabilities([]Capability{CapabilityForms, CapabilityAsyncData, ""}); len(parseMissing) != 2 || parseMissing[0] != "async-data" || parseMissing[1] != "forms" {
		parseT.Fatalf("missingCapabilities() = %+v, want async-data/forms", parseMissing)
	}
}

func TestHostValidationPanelsBootstrapAndCleanup(parseT *testing.T) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityDevtools, CapabilitySSR, CapabilityForms, CapabilityAsyncData, CapabilityRouter}})

	if parseErr := parseHost.AddRouteGuard(nil); parseErr != nil {
		parseT.Fatalf("AddRouteGuard(nil) error = %v", parseErr)
	}
	if parseErr2 := parseHost.AddNavigationObserver(nil); parseErr2 != nil {
		parseT.Fatalf("AddNavigationObserver(nil) error = %v", parseErr2)
	}
	if parseErr3 := parseHost.AddCacheKeyDecorator(nil); parseErr3 != nil {
		parseT.Fatalf("AddCacheKeyDecorator(nil) error = %v", parseErr3)
	}
	if parseErr4 := parseHost.AddRequestObserver(nil); parseErr4 != nil {
		parseT.Fatalf("AddRequestObserver(nil) error = %v", parseErr4)
	}
	if parseErr5 := parseHost.AddPanelProvider(nil); parseErr5 != nil {
		parseT.Fatalf("AddPanelProvider(nil) error = %v", parseErr5)
	}
	if parseErr5b := parseHost.AddDevtoolsSectionProvider(nil); parseErr5b != nil {
		parseT.Fatalf("AddDevtoolsSectionProvider(nil) error = %v", parseErr5b)
	}
	if parseErr5c := parseHost.AddDevtoolsActionProvider(nil); parseErr5c != nil {
		parseT.Fatalf("AddDevtoolsActionProvider(nil) error = %v", parseErr5c)
	}
	if parseErr6 := parseHost.AddHeadProvider(nil); parseErr6 != nil {
		parseT.Fatalf("AddHeadProvider(nil) error = %v", parseErr6)
	}
	if parseErr7 := parseHost.AddBootstrapProvider(nil); parseErr7 != nil {
		parseT.Fatalf("AddBootstrapProvider(nil) error = %v", parseErr7)
	}
	if parseErr8 := parseHost.AddFormValidator(nil); parseErr8 != nil {
		parseT.Fatalf("AddFormValidator(nil) error = %v", parseErr8)
	}
	if parseErr9 := parseHost.AddSubmitObserver(nil); parseErr9 != nil {
		parseT.Fatalf("AddSubmitObserver(nil) error = %v", parseErr9)
	}

	_ = parseHost.AddPanelProvider(func() Panel { return Panel{} })
	_ = parseHost.AddPanelProvider(func() Panel { return Panel{ID: "inspect", Title: "Inspect"} })
	parsePanels := parseHost.Panels()
	if len(parsePanels) != 1 || parsePanels[0].ID != "inspect" {
		parseT.Fatalf("Panels() = %+v, want one valid panel", parsePanels)
	}

	_ = parseHost.AddDevtoolsSectionProvider(func() DevtoolsSection {
		return DevtoolsSection{Name: "Companion", Summary: map[string]string{"state": "ready"}, Lines: []string{"line one"}}
	})
	parseSections := parseHost.DevtoolsSections()
	parseSections[0].Summary["state"] = "mutated"
	parseSections[0].Lines[0] = "mutated"
	parseSections[0].Name = "mutated"
	if parseNextSections := parseHost.DevtoolsSections(); len(parseNextSections) != 1 || parseNextSections[0].Name != "Companion" || parseNextSections[0].Summary["state"] != "ready" || parseNextSections[0].Lines[0] != "line one" {
		parseT.Fatalf("DevtoolsSections() = %+v, want cloned section values", parseNextSections)
	}

	_ = parseHost.AddDevtoolsActionProvider(func() []DevtoolsAction {
		return []DevtoolsAction{{
			Label:        "Retry loader",
			MatchCodes:   []string{"GWC-ROUTER-LOADER-FAILED"},
			MatchSources: []string{"router"},
			Run:          func(DevtoolsActionContext) {},
		}}
	})
	parseActions := parseHost.DevtoolsActions()
	parseActions[0].Label = "mutated"
	parseActions[0].MatchCodes[0] = "mutated"
	if parseNextActions := parseHost.DevtoolsActions(); len(parseNextActions) != 1 || parseNextActions[0].Label != "Retry loader" || parseNextActions[0].MatchCodes[0] != "GWC-ROUTER-LOADER-FAILED" {
		parseT.Fatalf("DevtoolsActions() = %+v, want cloned action values", parseNextActions)
	}

	_ = parseHost.AddHeadProvider(func() ui.Node {
		return html.Meta(html.Props{Raw: map[string]any{"name": "robots", "content": "index,follow"}})
	})
	if parseNodes := parseHost.HeadNodes(); len(parseNodes) != 1 {
		parseT.Fatalf("HeadNodes() len = %d, want 1", len(parseNodes))
	}

	_ = parseHost.AddBootstrapProvider(func() BootstrapPayload {
		return BootstrapPayload{Namespace: "shell", Data: map[string]any{"theme": "dark"}}
	})
	parseData := parseHost.BootstrapData()
	parseData["shell"]["theme"] = "mutated"
	if parseNext := parseHost.BootstrapData()["shell"]["theme"]; parseNext != "dark" {
		parseT.Fatalf("BootstrapData() should clone nested maps, got %v", parseNext)
	}

	_ = parseHost.AddFormValidator(func(FormSubmission) []ValidationIssue {
		return []ValidationIssue{{Field: "email", Message: "required"}}
	})
	if parseIssues := parseHost.ValidateForm(FormSubmission{}); len(parseIssues) != 1 || parseIssues[0].Field != "email" {
		parseT.Fatalf("ValidateForm() = %+v, want one email issue", parseIssues)
	}

	parseCloser := NewHost(HostOptions{})
	parseCloser.cleanups = []CleanupFunc{
		func() error { return errors.New("cleanup one") },
		func() error { return errors.New("cleanup two") },
	}
	if parseErr10 := parseCloser.Close(); parseErr10 == nil || !strings.Contains(parseErr10.Error(), "cleanup one") || !strings.Contains(parseErr10.Error(), "cleanup two") {
		parseT.Fatalf("Close() error = %v, want joined cleanup errors", parseErr10)
	}
}

func TestValidateManifestErrorsAndGuardHelpers(parseT *testing.T) {
	parseTests := []struct {
		name     string
		manifest Manifest
		want     string
	}{
		{name: "missing id", manifest: Manifest{Version: "1.0.0", Tier: TierStable}, want: "manifest ID is required"},
		{name: "missing version", manifest: Manifest{ID: "demo", Tier: TierStable}, want: "manifest version is required"},
		{name: "invalid tier", manifest: Manifest{ID: "demo", Version: "1.0.0", Tier: "beta"}, want: "manifest tier"},
	}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseErr := validateManifest(parseTest.manifest); parseErr == nil || !strings.Contains(parseErr.Error(), parseTest.want) {
				parseT2.Fatalf("validateManifest() error = %v, want substring %q", parseErr, parseTest.want)
			}
		})
	}

	if parseGot := Allow(" ok "); parseGot.Reason != "ok" || parseGot.Outcome != GuardAllow {
		parseT.Fatalf("Allow() = %+v, want trimmed allow decision", parseGot)
	}
	if parseGot2 := Block(" denied "); parseGot2.Reason != "denied" || parseGot2.Outcome != GuardBlock {
		parseT.Fatalf("Block() = %+v, want trimmed block decision", parseGot2)
	}
	if parseGot3 := Redirect(" /signin ", " auth "); parseGot3.Redirect != "/signin" || parseGot3.Reason != "auth" || parseGot3.Outcome != GuardRedirect {
		parseT.Fatalf("Redirect() = %+v, want trimmed redirect decision", parseGot3)
	}
}

func TestRegisterNilAndInvalidPluginInputsAndCloseSkipsNilCleanup(parseT *testing.T) {
	parseHost := NewHost(HostOptions{})
	if parseErr := parseHost.Register(nil); parseErr == nil || !strings.Contains(parseErr.Error(), "plugin is nil") {
		parseT.Fatalf("Register(nil) error = %v, want plugin is nil", parseErr)
	}
	if parseErr2 := parseHost.Register(Define(Manifest{Version: "1.0.0", Tier: TierStable}, nil)); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "manifest ID is required") {
		parseT.Fatalf("Register(invalid manifest) error = %v, want manifest ID error", parseErr2)
	}

	isParseCalled := false
	parseHost.cleanups = []CleanupFunc{
		nil,
		func() error {
			isParseCalled = true
			return nil
		},
	}
	if parseErr3 := parseHost.Close(); parseErr3 != nil {
		parseT.Fatalf("Close() error = %v", parseErr3)
	}
	if !isParseCalled {
		parseT.Fatal("expected non-nil cleanup to run even when nil cleanups are present")
	}
	if parseHost.cleanups != nil {
		parseT.Fatalf("expected Close() to clear cleanups, got %+v", parseHost.cleanups)
	}
}

func TestHostSkipBranchesForNilProvidersAndEmptyOutputs(parseT *testing.T) {
	var parseNilHost *Host
	parseNilHost.SetValue("ignored", "value")
	if parseValue, parseOk := parseNilHost.Value("ignored"); parseOk || parseValue != nil {
		parseT.Fatalf("nil host Value() = %#v, %t; want nil, false", parseValue, parseOk)
	}
	if parseDecorated := parseNilHost.DecorateCacheKey("cache-key"); parseDecorated != "cache-key" {
		parseT.Fatalf("nil host DecorateCacheKey() = %q, want original key", parseDecorated)
	}
	if parseNilHost.Panels() != nil {
		parseT.Fatal("nil host Panels() should return nil")
	}
	if parseNilHost.DevtoolsSections() != nil {
		parseT.Fatal("nil host DevtoolsSections() should return nil")
	}
	if parseNilHost.DevtoolsActions() != nil {
		parseT.Fatal("nil host DevtoolsActions() should return nil")
	}
	if parseNilHost.HeadNodes() != nil {
		parseT.Fatal("nil host HeadNodes() should return nil")
	}

	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter, CapabilityAsyncData, CapabilityDevtools, CapabilitySSR, CapabilityForms}})
	parseHost.routeGuards = []RouteGuard{
		nil,
		func(RouteRequest) GuardDecision { return GuardDecision{} },
		func(RouteRequest) GuardDecision { return Allow("ok") },
	}
	if parseDecision := parseHost.EvaluateRoute(RouteRequest{Path: "/pricing"}); parseDecision.Outcome != GuardAllow || !strings.Contains(parseDecision.Reason, "all registered route guards") {
		parseT.Fatalf("EvaluateRoute() = %+v, want final allow decision after skipped guards", parseDecision)
	}

	parseHost.cacheDecorators = []CacheKeyDecorator{
		nil,
		func(parseKey string) string { return parseKey + ":decorated" },
	}
	if parseDecorated2 := parseHost.DecorateCacheKey("cache-key"); parseDecorated2 != "cache-key:decorated" {
		parseT.Fatalf("DecorateCacheKey() = %q, want decorated key", parseDecorated2)
	}

	parseHost.panelProviders = []PanelProvider{
		nil,
		func() Panel { return Panel{ID: "panel-only"} },
		func() Panel { return Panel{ID: "valid", Title: "Valid"} },
	}
	if parsePanels := parseHost.Panels(); len(parsePanels) != 1 || parsePanels[0].ID != "valid" {
		parseT.Fatalf("Panels() = %+v, want one valid panel", parsePanels)
	}

	parseHost.devtoolsSectionProviders = []DevtoolsSectionProvider{
		nil,
		func() DevtoolsSection { return DevtoolsSection{} },
		func() DevtoolsSection {
			return DevtoolsSection{Name: "Companion", Summary: map[string]string{"state": "ready"}}
		},
	}
	if parseSections := parseHost.DevtoolsSections(); len(parseSections) != 1 || parseSections[0].Name != "Companion" {
		parseT.Fatalf("DevtoolsSections() = %+v, want one valid section", parseSections)
	}

	parseHost.devtoolsActionProviders = []DevtoolsActionProvider{
		nil,
		func() []DevtoolsAction { return []DevtoolsAction{{Label: "ignored"}} },
		func() []DevtoolsAction {
			return []DevtoolsAction{{Label: "Retry", MatchCodes: []string{"GWC-ROUTER-FAILED"}, Run: func(DevtoolsActionContext) {}}}
		},
	}
	if parseActions := parseHost.DevtoolsActions(); len(parseActions) != 1 || parseActions[0].Label != "Retry" {
		parseT.Fatalf("DevtoolsActions() = %+v, want one valid action", parseActions)
	}

	parseHost.headProviders = []HeadProvider{
		nil,
		func() ui.Node { return nil },
		func() ui.Node {
			return html.Meta(html.Props{Raw: map[string]any{"name": "robots", "content": "index,follow"}})
		},
	}
	if parseNodes := parseHost.HeadNodes(); len(parseNodes) != 1 {
		parseT.Fatalf("HeadNodes() len = %d, want 1", len(parseNodes))
	}

	parseHost.bootstrapProviders = []BootstrapProvider{
		nil,
		func() BootstrapPayload {
			return BootstrapPayload{Namespace: " ", Data: map[string]any{"ignored": true}}
		},
		func() BootstrapPayload { return BootstrapPayload{Namespace: "empty", Data: nil} },
		func() BootstrapPayload {
			return BootstrapPayload{Namespace: "ok", Data: map[string]any{"theme": "dark"}}
		},
	}
	parseData := parseHost.BootstrapData()
	if len(parseData) != 1 || parseData["ok"]["theme"] != "dark" {
		parseT.Fatalf("BootstrapData() = %+v, want only the valid payload", parseData)
	}

	parseHost.formValidators = []FormValidator{
		nil,
		func(FormSubmission) []ValidationIssue { return nil },
		func(FormSubmission) []ValidationIssue {
			return []ValidationIssue{{Field: "email", Message: "required"}}
		},
	}
	if parseIssues := parseHost.ValidateForm(FormSubmission{}); len(parseIssues) != 1 || parseIssues[0].Field != "email" {
		parseT.Fatalf("ValidateForm() = %+v, want one issue from the non-nil validator", parseIssues)
	}
}

func TestCapabilityGatedAddersReturnMissingCapabilityErrors(parseT *testing.T) {
	parseHost := NewHost(HostOptions{})

	parseTests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "navigation observer",
			call: func() error {
				return parseHost.AddNavigationObserver(func(NavigationEvent) {})
			},
			want: `capability "router" is not enabled`,
		},
		{
			name: "cache decorator",
			call: func() error {
				return parseHost.AddCacheKeyDecorator(func(parseKey string) string { return parseKey })
			},
			want: `capability "async-data" is not enabled`,
		},
		{
			name: "request observer",
			call: func() error {
				return parseHost.AddRequestObserver(func(RequestEvent) {})
			},
			want: `capability "async-data" is not enabled`,
		},
		{
			name: "panel provider",
			call: func() error {
				return parseHost.AddPanelProvider(func() Panel { return Panel{ID: "p", Title: "Panel"} })
			},
			want: `capability "devtools" is not enabled`,
		},
		{
			name: "devtools section provider",
			call: func() error {
				return parseHost.AddDevtoolsSectionProvider(func() DevtoolsSection { return DevtoolsSection{Name: "Companion"} })
			},
			want: `capability "devtools" is not enabled`,
		},
		{
			name: "devtools action provider",
			call: func() error {
				return parseHost.AddDevtoolsActionProvider(func() []DevtoolsAction {
					return []DevtoolsAction{{Label: "Retry", Run: func(DevtoolsActionContext) {}}}
				})
			},
			want: `capability "devtools" is not enabled`,
		},
		{
			name: "bootstrap provider",
			call: func() error {
				return parseHost.AddBootstrapProvider(func() BootstrapPayload {
					return BootstrapPayload{Namespace: "ns", Data: map[string]any{"ok": true}}
				})
			},
			want: `capability "ssr" is not enabled`,
		},
		{
			name: "form validator",
			call: func() error {
				return parseHost.AddFormValidator(func(FormSubmission) []ValidationIssue { return nil })
			},
			want: `capability "forms" is not enabled`,
		},
		{
			name: "submit observer",
			call: func() error {
				return parseHost.AddSubmitObserver(func(FormSubmission) {})
			},
			want: `capability "forms" is not enabled`,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseErr := parseTest.call(); parseErr == nil || !strings.Contains(parseErr.Error(), parseTest.want) {
				parseT2.Fatalf("%s error = %v, want substring %q", parseTest.name, parseErr, parseTest.want)
			}
		})
	}
}
