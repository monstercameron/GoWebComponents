package desktop

import (
	"context"
	"testing"
)

// TestScreenGeometryValidationBounds verifies bounded integer geometry and operation shapes.
func TestScreenGeometryValidationBounds(parseTest *testing.T) {
	parseValid := ScreenGeometryRequest{Operation: ScreenDipToPhysicalPoint, Point: ScreenPoint{X: -1920, Y: 1080}}
	if parseErr := validateScreenGeometryRequest(parseValid); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	for _, parseInvalid := range []ScreenGeometryRequest{
		{Operation: ScreenDipToPhysicalPoint, Point: ScreenPoint{X: screenGeometryCoordinateLimit + 1}},
		{Operation: ScreenDipToPhysicalPoint, Point: ScreenPoint{X: 1}, Rect: ScreenRect{Width: 1, Height: 1}},
		{Operation: ScreenDipToPhysicalRect, Rect: ScreenRect{Width: 0, Height: 1}},
		{Operation: ScreenDipToPhysicalRect, Point: ScreenPoint{X: 1}, Rect: ScreenRect{Width: 1, Height: 1}},
		{Operation: ScreenDipToPhysicalRect, Rect: ScreenRect{Width: screenGeometryCoordinateLimit + 1, Height: 1}},
		{Operation: ScreenGeometryOperation("unknown"), Point: ScreenPoint{}},
	} {
		if validateScreenGeometryRequest(parseInvalid) == nil {
			parseTest.Fatalf("accepted invalid request: %+v", parseInvalid)
		}
	}
}

// TestScreenGeometryMixedInputsDoNotReachBackend verifies validation precedes optional backend work.
func TestScreenGeometryMixedInputsDoNotReachBackend(parseTest *testing.T) {
	parseBackend := &parseScreenGeometryBackendFake{}
	parsePolicy, _ := ParseFeaturePolicy("screen-geometry")
	parseHost := NewNativeHost(parseBackend, parsePolicy)
	_, parseErr := parseHost.ScreenGeometry(context.Background(), ScreenGeometryRequest{Operation: ScreenDipToPhysicalPoint, Point: ScreenPoint{X: 1}, Rect: ScreenRect{Width: 1, Height: 1}})
	if parseErr == nil || parseBackend.parseCalled {
		parseTest.Fatalf("mixed geometry input reached backend: err=%v called=%v", parseErr, parseBackend.parseCalled)
	}
}

type parseScreenGeometryBackendFake struct{ parseCalled bool }

func (parseBackend *parseScreenGeometryBackendFake) Features() []Feature {
	return []Feature{ScreenGeometry}
}
func (parseBackend *parseScreenGeometryBackendFake) ClipboardWrite(context.Context, ClipboardWriteRequest) error {
	return nil
}
func (parseBackend *parseScreenGeometryBackendFake) ClipboardRead(context.Context) (string, error) {
	return "", nil
}
func (parseBackend *parseScreenGeometryBackendFake) ShowMessage(context.Context, MessageRequest) (MessageReply, error) {
	return MessageReply{}, nil
}
func (parseBackend *parseScreenGeometryBackendFake) Window(context.Context, WindowRequest) (WindowInfo, error) {
	return WindowInfo{}, nil
}
func (parseBackend *parseScreenGeometryBackendFake) Screens(context.Context) ([]ScreenInfo, error) {
	return nil, nil
}
func (parseBackend *parseScreenGeometryBackendFake) ScreenGeometry(context.Context, ScreenGeometryRequest) (ScreenGeometryReply, error) {
	parseBackend.parseCalled = true
	return ScreenGeometryReply{Point: ScreenPoint{X: 1, Y: 1}}, nil
}

// TestScreenGeometryReplyValidationPreservesUnits verifies operation-specific reply shapes.
func TestScreenGeometryReplyValidationPreservesUnits(parseTest *testing.T) {
	parseRequest := ScreenGeometryRequest{Operation: ScreenDipToPhysicalRect, Rect: ScreenRect{Width: 20, Height: 10}}
	if parseErr := validateScreenGeometryReply(parseRequest, ScreenGeometryReply{Rect: ScreenRect{X: -5, Y: 3, Width: 40, Height: 20}}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr := validateScreenGeometryReply(parseRequest, ScreenGeometryReply{Point: ScreenPoint{X: 1}}); parseErr == nil {
		parseTest.Fatal("accepted point for rectangle transform")
	}
	if parseErr := validateScreenGeometryReply(ScreenGeometryRequest{Operation: ScreenNearestPhysicalPoint, Point: ScreenPoint{}}, ScreenGeometryReply{ScreenID: "display-1"}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
}

// TestScreenGeometryReplyRejectsMalformedNumericValues documents integer-only finite geometry.
func TestScreenGeometryReplyRejectsMalformedNumericValues(parseTest *testing.T) {
	parseRequest := ScreenGeometryRequest{Operation: ScreenNearestDipPoint}
	if parseErr := validateScreenGeometryReply(parseRequest, ScreenGeometryReply{ScreenID: ""}); parseErr == nil {
		parseTest.Fatal("accepted empty nearest-screen ID")
	}
}

// TestScreenGeometryFeatureWiring verifies policy discovery exposes the complete typed method.
func TestScreenGeometryFeatureWiring(parseTest *testing.T) {
	parsePolicy, parseErr := ParseFeaturePolicy("screen-geometry")
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if !parsePolicy.Allows(ScreenGeometry) {
		parseTest.Fatal("screen geometry feature was not recognized")
	}
	parseMethods := featureMethods(ScreenGeometry)
	if len(parseMethods) != 1 || parseMethods[0] != ScreenGeometryMethod {
		parseTest.Fatalf("unexpected screen geometry methods: %#v", parseMethods)
	}
}
