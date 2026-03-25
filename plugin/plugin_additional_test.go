package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestHostNilValidationAndCloneHelpers(t *testing.T) {
	var nilHost *Host
	if err := nilHost.Register(nil); err == nil || !strings.Contains(err.Error(), "host is nil") {
		t.Fatalf("nil host Register() error = %v, want host is nil", err)
	}
	if err := nilHost.Close(); err != nil {
		t.Fatalf("nil host Close() error = %v", err)
	}
	if nilHost.Capabilities() != nil || nilHost.Plugins() != nil || nilHost.BootstrapData() != nil || nilHost.ValidateForm(FormSubmission{}) != nil {
		t.Fatal("nil host helper methods should return nil slices/maps")
	}
	if decision := nilHost.EvaluateRoute(RouteRequest{Path: "/"}); decision.Outcome != GuardAllow {
		t.Fatalf("nil host EvaluateRoute() = %+v, want allow", decision)
	}
	nilHost.NotifyNavigation(NavigationEvent{Path: "/pricing"})
	nilHost.NotifyRequest(RequestEvent{Key: "pricing"})
	nilHost.NotifySubmit(FormSubmission{ID: "submit"})

	host := NewHost(HostOptions{Capabilities: []Capability{"", CapabilitySSR, CapabilityRouter, CapabilitySSR}})
	caps := host.Capabilities()
	if len(caps) != 2 || caps[0] != CapabilityRouter || caps[1] != CapabilitySSR {
		t.Fatalf("Capabilities() = %+v, want sorted unique capabilities", caps)
	}

	host.SetValue("  ", "ignored")
	if _, ok := host.Value("  "); ok {
		t.Fatal("blank key should not be stored")
	}
	host.SetValue("theme", "dark")
	if value, ok := host.Value("theme"); !ok || value.(string) != "dark" {
		t.Fatalf("Value(theme) = %#v, %t; want dark, true", value, ok)
	}

	manifest := Manifest{
		ID:          "seo",
		Version:     "1.0.0",
		Description: "server helpers",
		Tier:        TierStable,
		Requires:    []Capability{CapabilitySSR},
	}
	if err := host.Register(Define(manifest, nil)); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := host.Register(Define(manifest, nil)); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("duplicate Register() error = %v, want duplicate error", err)
	}

	plugins := host.Plugins()
	plugins[0].Requires[0] = CapabilityRouter
	if current := host.Plugins()[0].Requires[0]; current != CapabilitySSR {
		t.Fatalf("Plugins() should return a clone, got Requires[0]=%q", current)
	}

	clone := cloneManifest(manifest)
	clone.Requires[0] = CapabilityRouter
	if manifest.Requires[0] != CapabilitySSR {
		t.Fatalf("cloneManifest() should not mutate source manifest, got %#v", manifest.Requires)
	}

	if missing := host.missingCapabilities([]Capability{CapabilityForms, CapabilityAsyncData, ""}); len(missing) != 2 || missing[0] != "async-data" || missing[1] != "forms" {
		t.Fatalf("missingCapabilities() = %+v, want async-data/forms", missing)
	}
}

func TestHostValidationPanelsBootstrapAndCleanup(t *testing.T) {
	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityDevtools, CapabilitySSR, CapabilityForms, CapabilityAsyncData, CapabilityRouter}})

	if err := host.AddRouteGuard(nil); err != nil {
		t.Fatalf("AddRouteGuard(nil) error = %v", err)
	}
	if err := host.AddNavigationObserver(nil); err != nil {
		t.Fatalf("AddNavigationObserver(nil) error = %v", err)
	}
	if err := host.AddCacheKeyDecorator(nil); err != nil {
		t.Fatalf("AddCacheKeyDecorator(nil) error = %v", err)
	}
	if err := host.AddRequestObserver(nil); err != nil {
		t.Fatalf("AddRequestObserver(nil) error = %v", err)
	}
	if err := host.AddPanelProvider(nil); err != nil {
		t.Fatalf("AddPanelProvider(nil) error = %v", err)
	}
	if err := host.AddHeadProvider(nil); err != nil {
		t.Fatalf("AddHeadProvider(nil) error = %v", err)
	}
	if err := host.AddBootstrapProvider(nil); err != nil {
		t.Fatalf("AddBootstrapProvider(nil) error = %v", err)
	}
	if err := host.AddFormValidator(nil); err != nil {
		t.Fatalf("AddFormValidator(nil) error = %v", err)
	}
	if err := host.AddSubmitObserver(nil); err != nil {
		t.Fatalf("AddSubmitObserver(nil) error = %v", err)
	}

	_ = host.AddPanelProvider(func() Panel { return Panel{} })
	_ = host.AddPanelProvider(func() Panel { return Panel{ID: "inspect", Title: "Inspect"} })
	panels := host.Panels()
	if len(panels) != 1 || panels[0].ID != "inspect" {
		t.Fatalf("Panels() = %+v, want one valid panel", panels)
	}

	_ = host.AddHeadProvider(func() ui.Node {
		return html.Meta(html.Props{Raw: map[string]interface{}{"name": "robots", "content": "index,follow"}})
	})
	if nodes := host.HeadNodes(); len(nodes) != 1 {
		t.Fatalf("HeadNodes() len = %d, want 1", len(nodes))
	}

	_ = host.AddBootstrapProvider(func() BootstrapPayload {
		return BootstrapPayload{Namespace: "shell", Data: map[string]interface{}{"theme": "dark"}}
	})
	data := host.BootstrapData()
	data["shell"]["theme"] = "mutated"
	if next := host.BootstrapData()["shell"]["theme"]; next != "dark" {
		t.Fatalf("BootstrapData() should clone nested maps, got %v", next)
	}

	_ = host.AddFormValidator(func(FormSubmission) []ValidationIssue {
		return []ValidationIssue{{Field: "email", Message: "required"}}
	})
	if issues := host.ValidateForm(FormSubmission{}); len(issues) != 1 || issues[0].Field != "email" {
		t.Fatalf("ValidateForm() = %+v, want one email issue", issues)
	}

	closer := NewHost(HostOptions{})
	closer.cleanups = []CleanupFunc{
		func() error { return errors.New("cleanup one") },
		func() error { return errors.New("cleanup two") },
	}
	if err := closer.Close(); err == nil || !strings.Contains(err.Error(), "cleanup one") || !strings.Contains(err.Error(), "cleanup two") {
		t.Fatalf("Close() error = %v, want joined cleanup errors", err)
	}
}

func TestValidateManifestErrorsAndGuardHelpers(t *testing.T) {
	tests := []struct {
		name     string
		manifest Manifest
		want     string
	}{
		{name: "missing id", manifest: Manifest{Version: "1.0.0", Tier: TierStable}, want: "manifest ID is required"},
		{name: "missing version", manifest: Manifest{ID: "demo", Tier: TierStable}, want: "manifest version is required"},
		{name: "invalid tier", manifest: Manifest{ID: "demo", Version: "1.0.0", Tier: "beta"}, want: "manifest tier"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateManifest(test.manifest); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateManifest() error = %v, want substring %q", err, test.want)
			}
		})
	}

	if got := Allow(" ok "); got.Reason != "ok" || got.Outcome != GuardAllow {
		t.Fatalf("Allow() = %+v, want trimmed allow decision", got)
	}
	if got := Block(" denied "); got.Reason != "denied" || got.Outcome != GuardBlock {
		t.Fatalf("Block() = %+v, want trimmed block decision", got)
	}
	if got := Redirect(" /signin ", " auth "); got.Redirect != "/signin" || got.Reason != "auth" || got.Outcome != GuardRedirect {
		t.Fatalf("Redirect() = %+v, want trimmed redirect decision", got)
	}
}

func TestRegisterNilAndInvalidPluginInputsAndCloseSkipsNilCleanup(t *testing.T) {
	host := NewHost(HostOptions{})
	if err := host.Register(nil); err == nil || !strings.Contains(err.Error(), "plugin is nil") {
		t.Fatalf("Register(nil) error = %v, want plugin is nil", err)
	}
	if err := host.Register(Define(Manifest{Version: "1.0.0", Tier: TierStable}, nil)); err == nil || !strings.Contains(err.Error(), "manifest ID is required") {
		t.Fatalf("Register(invalid manifest) error = %v, want manifest ID error", err)
	}

	called := false
	host.cleanups = []CleanupFunc{
		nil,
		func() error {
			called = true
			return nil
		},
	}
	if err := host.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !called {
		t.Fatal("expected non-nil cleanup to run even when nil cleanups are present")
	}
	if host.cleanups != nil {
		t.Fatalf("expected Close() to clear cleanups, got %+v", host.cleanups)
	}
}

func TestHostSkipBranchesForNilProvidersAndEmptyOutputs(t *testing.T) {
	var nilHost *Host
	nilHost.SetValue("ignored", "value")
	if value, ok := nilHost.Value("ignored"); ok || value != nil {
		t.Fatalf("nil host Value() = %#v, %t; want nil, false", value, ok)
	}
	if decorated := nilHost.DecorateCacheKey("cache-key"); decorated != "cache-key" {
		t.Fatalf("nil host DecorateCacheKey() = %q, want original key", decorated)
	}
	if nilHost.Panels() != nil {
		t.Fatal("nil host Panels() should return nil")
	}
	if nilHost.HeadNodes() != nil {
		t.Fatal("nil host HeadNodes() should return nil")
	}

	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter, CapabilityAsyncData, CapabilityDevtools, CapabilitySSR, CapabilityForms}})
	host.routeGuards = []RouteGuard{
		nil,
		func(RouteRequest) GuardDecision { return GuardDecision{} },
		func(RouteRequest) GuardDecision { return Allow("ok") },
	}
	if decision := host.EvaluateRoute(RouteRequest{Path: "/pricing"}); decision.Outcome != GuardAllow || !strings.Contains(decision.Reason, "all registered route guards") {
		t.Fatalf("EvaluateRoute() = %+v, want final allow decision after skipped guards", decision)
	}

	host.cacheDecorators = []CacheKeyDecorator{
		nil,
		func(key string) string { return key + ":decorated" },
	}
	if decorated := host.DecorateCacheKey("cache-key"); decorated != "cache-key:decorated" {
		t.Fatalf("DecorateCacheKey() = %q, want decorated key", decorated)
	}

	host.panelProviders = []PanelProvider{
		nil,
		func() Panel { return Panel{ID: "panel-only"} },
		func() Panel { return Panel{ID: "valid", Title: "Valid"} },
	}
	if panels := host.Panels(); len(panels) != 1 || panels[0].ID != "valid" {
		t.Fatalf("Panels() = %+v, want one valid panel", panels)
	}

	host.headProviders = []HeadProvider{
		nil,
		func() ui.Node { return nil },
		func() ui.Node {
			return html.Meta(html.Props{Raw: map[string]interface{}{"name": "robots", "content": "index,follow"}})
		},
	}
	if nodes := host.HeadNodes(); len(nodes) != 1 {
		t.Fatalf("HeadNodes() len = %d, want 1", len(nodes))
	}

	host.bootstrapProviders = []BootstrapProvider{
		nil,
		func() BootstrapPayload {
			return BootstrapPayload{Namespace: " ", Data: map[string]interface{}{"ignored": true}}
		},
		func() BootstrapPayload { return BootstrapPayload{Namespace: "empty", Data: nil} },
		func() BootstrapPayload {
			return BootstrapPayload{Namespace: "ok", Data: map[string]interface{}{"theme": "dark"}}
		},
	}
	data := host.BootstrapData()
	if len(data) != 1 || data["ok"]["theme"] != "dark" {
		t.Fatalf("BootstrapData() = %+v, want only the valid payload", data)
	}

	host.formValidators = []FormValidator{
		nil,
		func(FormSubmission) []ValidationIssue { return nil },
		func(FormSubmission) []ValidationIssue {
			return []ValidationIssue{{Field: "email", Message: "required"}}
		},
	}
	if issues := host.ValidateForm(FormSubmission{}); len(issues) != 1 || issues[0].Field != "email" {
		t.Fatalf("ValidateForm() = %+v, want one issue from the non-nil validator", issues)
	}
}

func TestCapabilityGatedAddersReturnMissingCapabilityErrors(t *testing.T) {
	host := NewHost(HostOptions{})

	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "navigation observer",
			call: func() error {
				return host.AddNavigationObserver(func(NavigationEvent) {})
			},
			want: `capability "router" is not enabled`,
		},
		{
			name: "cache decorator",
			call: func() error {
				return host.AddCacheKeyDecorator(func(key string) string { return key })
			},
			want: `capability "async-data" is not enabled`,
		},
		{
			name: "request observer",
			call: func() error {
				return host.AddRequestObserver(func(RequestEvent) {})
			},
			want: `capability "async-data" is not enabled`,
		},
		{
			name: "panel provider",
			call: func() error {
				return host.AddPanelProvider(func() Panel { return Panel{ID: "p", Title: "Panel"} })
			},
			want: `capability "devtools" is not enabled`,
		},
		{
			name: "bootstrap provider",
			call: func() error {
				return host.AddBootstrapProvider(func() BootstrapPayload {
					return BootstrapPayload{Namespace: "ns", Data: map[string]interface{}{"ok": true}}
				})
			},
			want: `capability "ssr" is not enabled`,
		},
		{
			name: "form validator",
			call: func() error {
				return host.AddFormValidator(func(FormSubmission) []ValidationIssue { return nil })
			},
			want: `capability "forms" is not enabled`,
		},
		{
			name: "submit observer",
			call: func() error {
				return host.AddSubmitObserver(func(FormSubmission) {})
			},
			want: `capability "forms" is not enabled`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("%s error = %v, want substring %q", test.name, err, test.want)
			}
		})
	}
}
