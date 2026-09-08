package desktop

import (
	"context"
	"strings"
	"testing"
)

type parseNativeFake struct {
	parseFeatures []Feature
	parseWrites   int
	parseReads    int
}

// TestNativeHostValidationPreventsBackendEffects verifies invalid and cancelled calls stop before backend work.
func TestNativeHostValidationPreventsBackendEffects(parseTest *testing.T) {
	parseFake := &parseNativeFake{parseFeatures: []Feature{Clipboard, MessageDialogs, WindowControls, Screens}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(parseFake, parsePolicy)
	if parseErr := parseHost.ClipboardWrite(context.Background(), ClipboardWriteRequest{Text: strings.Repeat("x", 4097)}); parseErr == nil {
		parseTest.Fatal("expected clipboard bound")
	}
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if parseErr := parseHost.ClipboardWrite(parseContext, ClipboardWriteRequest{Text: "ok"}); parseErr == nil {
		parseTest.Fatal("expected cancelled clipboard")
	}
	if parseFake.parseWrites != 0 {
		parseTest.Fatal("cancelled/invalid clipboard reached backend")
	}
	if _, parseErr := parseHost.WindowControl(context.Background(), WindowRequest{Action: "resize", Width: 0, Height: 1}); parseErr == nil {
		parseTest.Fatal("expected invalid resize")
	}
	if _, parseErr := parseHost.ShowMessage(context.Background(), MessageRequest{Kind: "info", Title: "ok\x00", Message: "x"}); parseErr == nil {
		parseTest.Fatal("expected invalid message")
	}
}

// Features returns fake backend capabilities.
func (parseFake *parseNativeFake) Features() []Feature {
	return append([]Feature(nil), parseFake.parseFeatures...)
}

// ClipboardWrite records one fake clipboard write.
func (parseFake *parseNativeFake) ClipboardWrite(context.Context, ClipboardWriteRequest) error {
	parseFake.parseWrites++
	return nil
}

// ClipboardRead records one fake clipboard read.
func (parseFake *parseNativeFake) ClipboardRead(context.Context) (string, error) {
	parseFake.parseReads++
	return "fixture", nil
}

// ShowMessage returns a deterministic fake button.
func (parseFake *parseNativeFake) ShowMessage(context.Context, MessageRequest) (MessageReply, error) {
	return MessageReply{Button: "Ok"}, nil
}

// Window returns deterministic fake window information.
func (parseFake *parseNativeFake) Window(context.Context, WindowRequest) (WindowInfo, error) {
	return WindowInfo{ID: "fake"}, nil
}

// Screens returns one deterministic fake display.
func (parseFake *parseNativeFake) Screens(context.Context) ([]ScreenInfo, error) {
	return []ScreenInfo{{ID: "screen"}}, nil
}

// TestParseFeaturePolicy verifies all, none, comma lists, and unknown names.
func TestParseFeaturePolicy(parseTest *testing.T) {
	for parseValue, parseExpected := range map[string]struct {
		parseAll   bool
		parseCount int
	}{"all": {true, len(nativeFeatures)}, "none": {false, 0}, "clipboard,message-dialogs": {false, 2}} {
		parsePolicy, parseErr := ParseFeaturePolicy(parseValue)
		if parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		if parsePolicy.parseAll != parseExpected.parseAll || len(parsePolicy.FeatureNames()) != parseExpected.parseCount {
			parseTest.Fatalf("policy %q: %#v", parseValue, parsePolicy.FeatureNames())
		}
	}
	if _, parseErr := ParseFeaturePolicy("clipboard,unknown"); parseErr == nil {
		parseTest.Fatal("expected unknown feature error")
	}
}

// TestNativeHostIntersectionAndDenial verifies policy and backend ceilings.
func TestNativeHostIntersectionAndDenial(parseTest *testing.T) {
	parseFake := &parseNativeFake{parseFeatures: []Feature{Clipboard, Screens}}
	parsePolicy, parseErr := ParseFeaturePolicy("all")
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseHost := NewNativeHost(parseFake, parsePolicy)
	if parseHost.Require(Clipboard) != nil {
		parseTest.Fatal("clipboard should be allowed")
	}
	if parseHost.Require(MessageDialogs) == nil {
		parseTest.Fatal("message dialog should be denied by backend")
	}
	if _, parseErr = parseHost.ClipboardRead(context.Background()); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseFake.parseReads != 1 {
		parseTest.Fatal("fake backend was not called")
	}
	if _, parseErr = parseHost.ListScreens(nil); parseErr == nil { //nolint:staticcheck // Deliberately exercise the invalid-context contract.
		parseTest.Fatal("nil context should be denied")
	}
}

// TestNativeHostInvalidMessage verifies typed request validation.
func TestNativeHostInvalidMessage(parseTest *testing.T) {
	parseFake := &parseNativeFake{parseFeatures: []Feature{MessageDialogs}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	if _, parseErr := NewNativeHost(parseFake, parsePolicy).ShowMessage(context.Background(), MessageRequest{Kind: "bad"}); parseErr == nil {
		parseTest.Fatal("expected invalid message kind")
	}
}
