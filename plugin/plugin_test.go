package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestRegisterRejectsMissingCapability(t *testing.T) {
	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	err := host.Register(Define(Manifest{
		ID:          "seo",
		Version:     "0.1.0",
		Description: "needs SSR",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilitySSR},
	}, func(host *Host) (CleanupFunc, error) {
		return nil, host.AddHeadProvider(func() ui.Node {
			return html.Tag("meta", html.Props{Raw: map[string]interface{}{
				"name":    "robots",
				"content": "index,follow",
			}})
		})
	}))
	if err == nil {
		t.Fatal("expected missing capability registration to fail")
	}
	if !strings.Contains(err.Error(), "missing capabilities") {
		t.Fatalf("expected missing capability error, got %v", err)
	}
}

func TestRegisterRollsBackOnSetupFailure(t *testing.T) {
	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	err := host.Register(Define(Manifest{
		ID:          "broken",
		Version:     "0.1.0",
		Description: "fails after adding a hook",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter},
	}, func(host *Host) (CleanupFunc, error) {
		if err := host.AddRouteGuard(func(RouteRequest) GuardDecision {
			return Block("should never survive rollback")
		}); err != nil {
			return nil, err
		}
		return nil, errors.New("setup failed after adding a guard")
	}))
	if err == nil {
		t.Fatal("expected setup failure")
	}
	decision := host.EvaluateRoute(RouteRequest{Path: "/"})
	if decision.Outcome != GuardAllow {
		t.Fatalf("expected rollback to remove guards, got %+v", decision)
	}
}

func TestHostAggregatesContributionsAndCleanup(t *testing.T) {
	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter, CapabilityAsyncData, CapabilityDevtools, CapabilitySSR, CapabilityForms}})
	order := make([]string, 0, 2)

	first := Define(Manifest{
		ID:          "first",
		Version:     "0.1.0",
		Description: "adds route, cache, ssr, forms",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter, CapabilityAsyncData, CapabilitySSR, CapabilityForms},
	}, func(host *Host) (CleanupFunc, error) {
		host.SetValue("prefix", "gwc")
		_ = host.AddRouteGuard(func(request RouteRequest) GuardDecision {
			if request.Path == "/admin" {
				return Redirect("/signin", "auth required")
			}
			return Allow("public route")
		})
		_ = host.AddCacheKeyDecorator(func(key string) string { return "gwc:" + key })
		_ = host.AddHeadProvider(func() ui.Node {
			return html.Tag("meta", html.Props{Raw: map[string]interface{}{"name": "robots", "content": "index,follow"}})
		})
		_ = host.AddBootstrapProvider(func() BootstrapPayload {
			return BootstrapPayload{Namespace: "first", Data: map[string]interface{}{"owner": "plugins"}}
		})
		_ = host.AddFormValidator(func(submission FormSubmission) []ValidationIssue {
			if submission.Values["quantity"] == "0" {
				return []ValidationIssue{{Field: "quantity", Message: "Quantity must be greater than zero."}}
			}
			return nil
		})
		return func() error {
			order = append(order, "first")
			return nil
		}, nil
	})

	second := Define(Manifest{
		ID:          "second",
		Version:     "0.1.0",
		Description: "adds observers and a panel",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter, CapabilityAsyncData, CapabilityDevtools, CapabilityForms},
	}, func(host *Host) (CleanupFunc, error) {
		_ = host.AddNavigationObserver(func(event NavigationEvent) { host.SetValue("last-route", event.Path) })
		_ = host.AddRequestObserver(func(event RequestEvent) { host.SetValue("last-request", event.Key) })
		_ = host.AddPanelProvider(func() Panel {
			return Panel{ID: "plugin-panel", Title: "Plugin Panel", Summary: "Shows plugin status."}
		})
		_ = host.AddSubmitObserver(func(submission FormSubmission) { host.SetValue("last-form", submission.ID) })
		return func() error {
			order = append(order, "second")
			return nil
		}, nil
	})

	if err := host.Register(first); err != nil {
		t.Fatalf("unexpected first plugin error: %v", err)
	}
	if err := host.Register(second); err != nil {
		t.Fatalf("unexpected second plugin error: %v", err)
	}

	decision := host.EvaluateRoute(RouteRequest{Path: "/admin"})
	if decision.Outcome != GuardRedirect || decision.Redirect != "/signin" {
		t.Fatalf("expected redirect decision, got %+v", decision)
	}
	if decorated := host.DecorateCacheKey("inventory"); decorated != "gwc:inventory" {
		t.Fatalf("expected decorated cache key, got %q", decorated)
	}
	host.NotifyNavigation(NavigationEvent{Path: "/pricing", Source: "test"})
	if value, ok := host.Value("last-route"); !ok || value.(string) != "/pricing" {
		t.Fatalf("expected navigation observer to store last route, got %v %v", value, ok)
	}
	host.NotifyRequest(RequestEvent{Key: "gwc:inventory", Phase: "ready", Source: "test"})
	if value, ok := host.Value("last-request"); !ok || value.(string) != "gwc:inventory" {
		t.Fatalf("expected request observer to store last request, got %v %v", value, ok)
	}
	issues := host.ValidateForm(FormSubmission{ID: "purchase", Values: map[string]string{"quantity": "0"}})
	if len(issues) != 1 || issues[0].Field != "quantity" {
		t.Fatalf("expected one quantity validation issue, got %+v", issues)
	}
	host.NotifySubmit(FormSubmission{ID: "purchase", Values: map[string]string{"quantity": "2"}})
	if value, ok := host.Value("last-form"); !ok || value.(string) != "purchase" {
		t.Fatalf("expected submit observer to store last form, got %v %v", value, ok)
	}
	panels := host.Panels()
	if len(panels) != 1 || panels[0].ID != "plugin-panel" {
		t.Fatalf("expected one panel, got %+v", panels)
	}
	headMarkup, err := ui.RenderToString(ui.Fragment(host.HeadNodes()...))
	if err != nil {
		t.Fatalf("unexpected head render error: %v", err)
	}
	if !strings.Contains(headMarkup, `name="robots"`) {
		t.Fatalf("expected head provider output, got %q", headMarkup)
	}
	bootstrap := host.BootstrapData()
	if bootstrap["first"]["owner"] != "plugins" {
		t.Fatalf("expected bootstrap payload, got %+v", bootstrap)
	}
	if err := host.Close(); err != nil {
		t.Fatalf("unexpected cleanup error: %v", err)
	}
	if strings.Join(order, ",") != "second,first" {
		t.Fatalf("expected reverse cleanup order, got %v", order)
	}
}
