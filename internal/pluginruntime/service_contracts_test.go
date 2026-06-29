package pluginruntime

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultServicesReturnNativeEmptySnapshotsAndNoopCommands(parseT *testing.T) {
	parseDOM := buildDefaultDOMService{}
	parseDOMSnapshot, parseErr := parseDOM.GetDOMSnapshot(QueryBudget{MaxItems: 1})
	if parseErr != nil {
		parseT.Fatalf("GetDOMSnapshot: %v", parseErr)
	}
	if parseDOMSnapshot.Meta != BuildSnapshotMeta(BackendIDNative, false) || parseDOMSnapshot.Root != nil {
		parseT.Fatalf("unexpected DOM snapshot: %#v", parseDOMSnapshot)
	}
	if parseErr := parseDOM.HighlightNode("node", CommandOptions{}); parseErr != nil {
		parseT.Fatalf("HighlightNode: %v", parseErr)
	}
	if parseErr := parseDOM.ScrollNodeIntoView("node", CommandOptions{}); parseErr != nil {
		parseT.Fatalf("ScrollNodeIntoView: %v", parseErr)
	}

	parseStyle := buildDefaultStyleService{}
	parseStyleSnapshot, parseErr := parseStyle.GetStyleSnapshot(QueryBudget{})
	if parseErr != nil {
		parseT.Fatalf("GetStyleSnapshot: %v", parseErr)
	}
	if parseStyleSnapshot.Meta != BuildSnapshotMeta(BackendIDNative, false) || parseStyleSnapshot.Variables != nil || parseStyleSnapshot.Stylesheet != nil {
		parseT.Fatalf("unexpected style snapshot: %#v", parseStyleSnapshot)
	}
	parsePatch, parseErr := parseStyle.ApplyStyleVariables(map[string]string{"--accent": "blue"}, CommandOptions{})
	if parseErr != nil {
		parseT.Fatalf("ApplyStyleVariables: %v", parseErr)
	}
	if parseErr := parsePatch.RemoveStylePatch(); parseErr != nil {
		parseT.Fatalf("RemoveStylePatch: %v", parseErr)
	}

	parseEventSnapshot, parseErr := (buildDefaultEventService{}).GetEventSnapshot(QueryBudget{})
	if parseErr != nil || parseEventSnapshot.Meta != BuildSnapshotMeta(BackendIDNative, false) || parseEventSnapshot.Events != nil {
		parseT.Fatalf("unexpected event snapshot/err: %#v %v", parseEventSnapshot, parseErr)
	}
	parseAsset := buildDefaultAssetService{}
	parseAssetSnapshot, parseErr := parseAsset.GetAssetSnapshot(QueryBudget{})
	if parseErr != nil || parseAssetSnapshot.Meta != BuildSnapshotMeta(BackendIDNative, false) || parseAssetSnapshot.Entries != nil {
		parseT.Fatalf("unexpected asset snapshot/err: %#v %v", parseAssetSnapshot, parseErr)
	}
	if parseErr := parseAsset.RefreshAssets(CommandOptions{}); parseErr != nil {
		parseT.Fatalf("RefreshAssets: %v", parseErr)
	}
	parseSecuritySnapshot, parseErr := (buildDefaultSecurityService{}).GetSecuritySnapshot(QueryBudget{})
	if parseErr != nil || parseSecuritySnapshot.Meta != BuildSnapshotMeta(BackendIDNative, false) || parseSecuritySnapshot.Findings != nil {
		parseT.Fatalf("unexpected security snapshot/err: %#v %v", parseSecuritySnapshot, parseErr)
	}
	parseCaptureSnapshot, parseErr := (buildDefaultCaptureService{}).GetCaptureSnapshot(QueryBudget{})
	if parseErr != nil || parseCaptureSnapshot.Meta != BuildSnapshotMeta(BackendIDNative, false) || parseCaptureSnapshot.Sessions != nil {
		parseT.Fatalf("unexpected capture snapshot/err: %#v %v", parseCaptureSnapshot, parseErr)
	}
}

func TestResolveServiceAsReportsNilMissingAndWrongType(parseT *testing.T) {
	if _, parseErr := ResolveServiceAs[DOMService](nil, ServiceKeyDOM); parseErr == nil || !strings.Contains(parseErr.Error(), "kernel is unavailable") {
		parseT.Fatalf("nil kernel error = %v", parseErr)
	}

	parseKernel, parseErr := NewKernel(BootstrapOptions{Services: []ServiceRegistration{{Key: ServiceKeyDOM, Value: buildDefaultDOMService{}}}})
	if parseErr != nil {
		parseT.Fatalf("NewKernel: %v", parseErr)
	}
	parseT.Cleanup(func() {
		_ = parseKernel.Close()
	})

	parseDOM, parseErr := ResolveServiceAs[DOMService](parseKernel, ServiceKeyDOM)
	if parseErr != nil {
		parseT.Fatalf("ResolveServiceAs DOMService: %v", parseErr)
	}
	parseSnapshot, parseErr := parseDOM.GetDOMSnapshot(QueryBudget{})
	if parseErr != nil || parseSnapshot.Meta.BackendID != string(BackendIDNative) {
		parseT.Fatalf("resolved DOM service returned %#v %v", parseSnapshot, parseErr)
	}

	if _, parseErr := ResolveServiceAs[RouteService](parseKernel, ServiceKeyRoute); parseErr == nil || !strings.Contains(parseErr.Error(), "unavailable") {
		parseT.Fatalf("missing service error = %v", parseErr)
	}
	parseKernel.SetService(ServiceKeyRoute, "not-route-service")
	if _, parseErr := ResolveServiceAs[RouteService](parseKernel, ServiceKeyRoute); parseErr == nil || !strings.Contains(parseErr.Error(), "unexpected type") {
		parseT.Fatalf("wrong type error = %v", parseErr)
	}
}

func TestSubscriptionHandleStopIsNilSafeAndIdempotent(parseT *testing.T) {
	var parseNil *SubscriptionHandle
	if parseErr := parseNil.Stop(); parseErr != nil {
		parseT.Fatalf("nil Stop returned error: %v", parseErr)
	}

	parseCalls := 0
	parseHandle := &SubscriptionHandle{getStop: func() error {
		parseCalls++
		return errors.New("stop failed")
	}}
	if parseErr := parseHandle.Stop(); parseErr == nil || parseErr.Error() != "stop failed" {
		parseT.Fatalf("first Stop error = %v", parseErr)
	}
	if parseErr := parseHandle.Stop(); parseErr != nil {
		parseT.Fatalf("second Stop should be idempotent nil, got %v", parseErr)
	}
	if parseCalls != 1 {
		parseT.Fatalf("cleanup called %d times, want 1", parseCalls)
	}
}

func TestBuildSnapshotMetaPreservesBackendAndTruncation(parseT *testing.T) {
	parseMeta := BuildSnapshotMeta(BackendIDRuntime2, true)
	if parseMeta.BackendID != "runtime2" || !parseMeta.Truncated {
		parseT.Fatalf("unexpected snapshot meta: %#v", parseMeta)
	}
}
