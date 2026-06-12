package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestRegisterRejectsMissingCapability(parseT *testing.T) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	parseErr := parseHost.Register(Define(Manifest{
		ID:          "seo",
		Version:     "0.1.0",
		Description: "needs SSR",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilitySSR},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		return nil, parseHost2.AddHeadProvider(func() ui.Node {
			return html.Meta(html.Props{Raw: map[string]any{
				"name":    "robots",
				"content": "index,follow",
			}})
		})
	}))
	if parseErr == nil {
		parseT.Fatal("expected missing capability registration to fail")
	}
	if !strings.Contains(parseErr.Error(), "missing capabilities") {
		parseT.Fatalf("expected missing capability error, got %v", parseErr)
	}
}

func TestRegisterRollsBackOnSetupFailure(parseT *testing.T) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	parseErr := parseHost.Register(Define(Manifest{
		ID:          "broken",
		Version:     "0.1.0",
		Description: "fails after adding a hook",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		if parseErr2 := parseHost2.AddRouteGuard(func(RouteRequest) GuardDecision {
			return Block("should never survive rollback")
		}); parseErr2 != nil {
			return nil, parseErr2
		}
		return nil, errors.New("setup failed after adding a guard")
	}))
	if parseErr == nil {
		parseT.Fatal("expected setup failure")
	}
	parseDecision := parseHost.EvaluateRoute(RouteRequest{Path: "/"})
	if parseDecision.Outcome != GuardAllow {
		parseT.Fatalf("expected rollback to remove guards, got %+v", parseDecision)
	}
}

func TestHostAggregatesContributionsAndCleanup(parseT *testing.T) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter, CapabilityAsyncData, CapabilityDevtools, CapabilitySSR, CapabilityForms}})
	parseOrder := make([]string, 0, 2)

	parseFirst := Define(Manifest{
		ID:          "first",
		Version:     "0.1.0",
		Description: "adds route, cache, ssr, forms",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter, CapabilityAsyncData, CapabilitySSR, CapabilityForms},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		parseHost2.SetValue("prefix", "gwc")
		_ = parseHost2.AddRouteGuard(func(parseRequest RouteRequest) GuardDecision {
			if parseRequest.Path == "/admin" {
				return Redirect("/signin", "auth required")
			}
			return Allow("public route")
		})
		_ = parseHost2.AddCacheKeyDecorator(func(parseKey string) string { return "gwc:" + parseKey })
		_ = parseHost2.AddHeadProvider(func() ui.Node {
			return html.Meta(html.Props{Raw: map[string]any{"name": "robots", "content": "index,follow"}})
		})
		_ = parseHost2.AddBootstrapProvider(func() BootstrapPayload {
			return BootstrapPayload{Namespace: "first", Data: map[string]any{"owner": "plugins"}}
		})
		_ = parseHost2.AddFormValidator(func(parseSubmission FormSubmission) []ValidationIssue {
			if parseSubmission.Values["quantity"] == "0" {
				return []ValidationIssue{{Field: "quantity", Message: "Quantity must be greater than zero."}}
			}
			return nil
		})
		return func() error {
			parseOrder = append(parseOrder, "first")
			return nil
		}, nil
	})

	parseSecond := Define(Manifest{
		ID:          "second",
		Version:     "0.1.0",
		Description: "adds observers and a panel",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter, CapabilityAsyncData, CapabilityDevtools, CapabilityForms},
	}, func(parseHost3 *Host) (CleanupFunc, error) {
		_ = parseHost3.AddNavigationObserver(func(parseEvent NavigationEvent) { parseHost3.SetValue("last-route", parseEvent.Path) })
		_ = parseHost3.AddRequestObserver(func(parseEvent2 RequestEvent) { parseHost3.SetValue("last-request", parseEvent2.Key) })
		_ = parseHost3.AddPanelProvider(func() Panel {
			return Panel{ID: "plugin-panel", Title: "Plugin Panel", Summary: "Shows plugin status."}
		})
		_ = parseHost3.AddSubmitObserver(func(parseSubmission2 FormSubmission) { parseHost3.SetValue("last-form", parseSubmission2.ID) })
		return func() error {
			parseOrder = append(parseOrder, "second")
			return nil
		}, nil
	})

	if parseErr := parseHost.Register(parseFirst); parseErr != nil {
		parseT.Fatalf("unexpected first plugin error: %v", parseErr)
	}
	if parseErr2 := parseHost.Register(parseSecond); parseErr2 != nil {
		parseT.Fatalf("unexpected second plugin error: %v", parseErr2)
	}

	parseDecision := parseHost.EvaluateRoute(RouteRequest{Path: "/admin"})
	if parseDecision.Outcome != GuardRedirect || parseDecision.Redirect != "/signin" {
		parseT.Fatalf("expected redirect decision, got %+v", parseDecision)
	}
	if parseDecorated := parseHost.DecorateCacheKey("inventory"); parseDecorated != "gwc:inventory" {
		parseT.Fatalf("expected decorated cache key, got %q", parseDecorated)
	}
	parseHost.NotifyNavigation(NavigationEvent{Path: "/pricing", Source: "test"})
	if parseValue, parseOk := parseHost.Value("last-route"); !parseOk || parseValue.(string) != "/pricing" {
		parseT.Fatalf("expected navigation observer to store last route, got %v %v", parseValue, parseOk)
	}
	parseHost.NotifyRequest(RequestEvent{Key: "gwc:inventory", Phase: "ready", Source: "test"})
	if parseValue2, parseOk2 := parseHost.Value("last-request"); !parseOk2 || parseValue2.(string) != "gwc:inventory" {
		parseT.Fatalf("expected request observer to store last request, got %v %v", parseValue2, parseOk2)
	}
	parseIssues := parseHost.ValidateForm(FormSubmission{ID: "purchase", Values: map[string]string{"quantity": "0"}})
	if len(parseIssues) != 1 || parseIssues[0].Field != "quantity" {
		parseT.Fatalf("expected one quantity validation issue, got %+v", parseIssues)
	}
	parseHost.NotifySubmit(FormSubmission{ID: "purchase", Values: map[string]string{"quantity": "2"}})
	if parseValue3, parseOk3 := parseHost.Value("last-form"); !parseOk3 || parseValue3.(string) != "purchase" {
		parseT.Fatalf("expected submit observer to store last form, got %v %v", parseValue3, parseOk3)
	}
	parsePanels := parseHost.Panels()
	if len(parsePanels) != 1 || parsePanels[0].ID != "plugin-panel" {
		parseT.Fatalf("expected one panel, got %+v", parsePanels)
	}
	parseHeadMarkup, parseErr3 := ui.RenderToString(ui.Fragment(parseHost.HeadNodes()...))
	if parseErr3 != nil {
		parseT.Fatalf("unexpected head render error: %v", parseErr3)
	}
	if !strings.Contains(parseHeadMarkup, `name="robots"`) {
		parseT.Fatalf("expected head provider output, got %q", parseHeadMarkup)
	}
	parseBootstrap := parseHost.BootstrapData()
	if parseBootstrap["first"]["owner"] != "plugins" {
		parseT.Fatalf("expected bootstrap payload, got %+v", parseBootstrap)
	}
	if parseErr4 := parseHost.Close(); parseErr4 != nil {
		parseT.Fatalf("unexpected cleanup error: %v", parseErr4)
	}
	if strings.Join(parseOrder, ",") != "second,first" {
		parseT.Fatalf("expected reverse cleanup order, got %v", parseOrder)
	}
	if parseDecisionAfterClose := parseHost.EvaluateRoute(RouteRequest{Path: "/admin"}); parseDecisionAfterClose.Outcome != GuardAllow {
		parseT.Fatalf("expected close to clear route guards, got %+v", parseDecisionAfterClose)
	}
	if parsePanelsAfterClose := parseHost.Panels(); len(parsePanelsAfterClose) != 0 {
		parseT.Fatalf("expected close to clear panels, got %+v", parsePanelsAfterClose)
	}
	if parsePluginsAfterClose := parseHost.Plugins(); len(parsePluginsAfterClose) != 0 {
		parseT.Fatalf("expected close to clear plugin manifests, got %+v", parsePluginsAfterClose)
	}
	if parseBootstrapAfterClose := parseHost.BootstrapData(); len(parseBootstrapAfterClose) != 0 {
		parseT.Fatalf("expected close to clear bootstrap providers, got %+v", parseBootstrapAfterClose)
	}
	if _, parseOk := parseHost.Value("last-route"); parseOk {
		parseT.Fatal("expected close to clear host values")
	}
}
