package desktop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v8/interop"
)

type parseChildWindowBackend struct {
	parseNativeFake
	parseCreated int
}

// Features advertises the optional child-window extension for the fake backend.
func (parseBackend *parseChildWindowBackend) Features() []Feature {
	return []Feature{ChildWindows}
}

// CreateChildWindow records validated creation.
func (parseBackend *parseChildWindowBackend) CreateChildWindow(_ context.Context, parseRequest ChildWindowCreateRequest) (ChildWindowInfo, error) {
	parseBackend.parseCreated++
	return ChildWindowInfo{ID: parseRequest.ID, TemplateID: parseRequest.TemplateID, Title: parseRequest.Title, Width: parseRequest.Width, Height: parseRequest.Height, Visible: true}, nil
}

// ListChildWindows returns a deterministic owned child.
func (*parseChildWindowBackend) ListChildWindows(context.Context) ([]ChildWindowInfo, error) {
	return []ChildWindowInfo{{ID: "child", TemplateID: "counter", Title: "Counter", Width: 640, Height: 480, Visible: true}}, nil
}

// InspectChildWindow returns a deterministic owned child.
func (*parseChildWindowBackend) InspectChildWindow(_ context.Context, parseRequest ChildWindowRequest) (ChildWindowInfo, error) {
	return ChildWindowInfo{ID: parseRequest.ID, TemplateID: "counter", Title: "Counter", Width: 640, Height: 480, Visible: true}, nil
}

// ControlChildWindow returns a deterministic controlled child.
func (*parseChildWindowBackend) ControlChildWindow(_ context.Context, parseRequest ChildWindowControlRequest) (ChildWindowInfo, error) {
	return ChildWindowInfo{ID: parseRequest.ID, TemplateID: "counter", Title: "Counter", Width: 640, Height: 480, Visible: parseRequest.Action == "show"}, nil
}

// TestChildWindowHostAdvertisesOptionalMethods verifies feature and method wiring.
func TestChildWindowHostAdvertisesOptionalMethods(parseTest *testing.T) {
	parsePolicy, parseErr := ParseFeaturePolicy("all")
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseHost := NewNativeHost(&parseChildWindowBackend{}, parsePolicy)
	for _, parseMethod := range []string{ChildWindowCreateMethod, ChildWindowListMethod, ChildWindowInspectMethod, ChildWindowControlMethod} {
		if !hasName(parseHost.GetMethods(), parseMethod) {
			parseTest.Fatalf("missing child-window method %q", parseMethod)
		}
	}
}

// TestChildWindowCreateRejectsURLAndBoundsBeforeBackend verifies the route is host-only.
func TestChildWindowCreateRejectsURLAndBoundsBeforeBackend(parseTest *testing.T) {
	parseBackend := &parseChildWindowBackend{}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(parseBackend, parsePolicy)
	parseReply := parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: ChildWindowCreateMethod, Args: json.RawMessage(`{"id":"child","templateId":"counter","title":"Counter","width":640,"height":480,"url":"https://example.com"}`)})
	if parseReply.Code != interop.CodeInvalid || parseBackend.parseCreated != 0 {
		parseTest.Fatalf("arbitrary URL must be rejected before backend: %#v", parseReply)
	}
	if _, parseErr := parseHost.CreateChildWindow(context.Background(), ChildWindowCreateRequest{ID: "child", TemplateID: "counter", Width: 4097, Height: 480}); parseErr == nil || parseBackend.parseCreated != 0 {
		parseTest.Fatal("oversized child window reached backend")
	}
	parseInfo, parseErr := parseHost.CreateChildWindow(context.Background(), ChildWindowCreateRequest{ID: "child", TemplateID: "counter", Title: "Counter", Width: 640, Height: 480})
	if parseErr != nil || parseInfo.ID != "child" || parseBackend.parseCreated != 1 {
		parseTest.Fatalf("valid child create failed: %#v %v", parseInfo, parseErr)
	}
}

// TestChildWindowControlAllowsBoundedActions verifies application quit and arbitrary URLs remain absent.
func TestChildWindowControlAllowsBoundedActions(t *testing.T) {
	for _, req := range []ChildWindowControlRequest{
		{ID: "child", Action: "show"},
		{ID: "child", Action: "hide"},
		{ID: "child", Action: "close"},
		{ID: "child", Action: "focus"},
		{ID: "child", Action: "minimize"},
		{ID: "child", Action: "unminimize"},
		{ID: "child", Action: "maximize"},
		{ID: "child", Action: "unmaximize"},
		{ID: "child", Action: "restore"},
		{ID: "child", Action: "center"},
		{ID: "child", Action: "set-title", Title: "Inspector"},
		{ID: "child", Action: "resize", Width: 640, Height: 480},
		{ID: "child", Action: "set-bounds", X: 20, Y: 30, Width: 640, Height: 480},
	} {
		if err := validateChildWindowControlRequest(req); err != nil {
			t.Fatalf("request %+v rejected: %v", req, err)
		}
	}
	for _, req := range []ChildWindowControlRequest{
		{ID: "child", Action: "quit"},
		{ID: "child", Action: "set-url"},
		{ID: "child", Action: "destroy-parent"},
		{ID: "child", Action: "show", Title: "extra"},
		{ID: "child", Action: "resize", Width: 0, Height: 480},
		{ID: "child", Action: "resize", Width: 640, Height: 480, X: 1},
		{ID: "child", Action: "set-title", Title: "bad\x00title"},
		{ID: "child", Action: "set-bounds", X: 100001, Y: 0, Width: 640, Height: 480},
	} {
		if err := validateChildWindowControlRequest(req); err == nil {
			t.Fatalf("request %+v must be rejected", req)
		}
	}
}

// TestChildWindowClientWireMethods verifies every typed client method reaches the native envelope.
func TestChildWindowClientWireMethods(parseTest *testing.T) {
	parseJSON := func(parseValue any) json.RawMessage { parseData, _ := json.Marshal(parseValue); return parseData }
	parseInfo := ChildWindowInfo{ID: "child", TemplateID: "counter", Title: "Counter", Width: 640, Height: 480, Visible: true}
	parseNativeReply := func(parseValue any) Reply {
		return Reply{Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(parseValue)})}
	}
	parseTransport := &parseNativeTransport{
		parseMethods: []string{ChildWindowCreateMethod, ChildWindowListMethod, ChildWindowInspectMethod, ChildWindowControlMethod},
		parseReplies: map[string]Reply{
			ChildWindowCreateMethod:  parseNativeReply(parseInfo),
			ChildWindowListMethod:    parseNativeReply([]ChildWindowInfo{parseInfo}),
			ChildWindowInspectMethod: parseNativeReply(parseInfo),
			ChildWindowControlMethod: parseNativeReply(parseInfo),
		},
	}
	parseClient := NewClient(parseTransport)
	if _, parseErr := parseClient.CreateChildWindow(context.Background(), ChildWindowCreateRequest{ID: "child", TemplateID: "counter", Title: "Counter", Width: 640, Height: 480}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseItems, parseErr := parseClient.ListChildWindows(context.Background()); parseErr != nil || len(parseItems) != 1 {
		parseTest.Fatalf("list=%#v err=%v", parseItems, parseErr)
	}
	if _, parseErr := parseClient.InspectChildWindow(context.Background(), "child"); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	for _, parseCall := range []func(context.Context, string) (ChildWindowInfo, error){parseClient.ShowChildWindow, parseClient.HideChildWindow, parseClient.CloseChildWindow, parseClient.FocusChildWindow} {
		if _, parseErr := parseCall(context.Background(), "child"); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
	}
	if _, err := parseClient.ResizeChildWindow(context.Background(), "child", 800, 600); err != nil {
		parseTest.Fatal(err)
	}
	if _, err := parseClient.SetChildWindowTitle(context.Background(), "child", "Updated"); err != nil {
		parseTest.Fatal(err)
	}
}
