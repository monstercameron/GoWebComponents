package desktop

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// TestNativeClientRejectsInvalidInputBeforeEncoding prevents JSON from silently repairing invalid UTF-8.
func TestNativeClientRejectsInvalidInputBeforeEncoding(parseTest *testing.T) {
	parseTransport := getTestTransport()
	parseTransport.parseCapabilities.Methods = []string{ClipboardReadMethod, ClipboardWriteMethod, MessageMethod, WindowMethod}
	parseTransport.parseStart = func(string, json.RawMessage) (string, error) {
		parseTest.Error("invalid native input reached transport")
		return "unused", nil
	}
	parseClient := NewClient(parseTransport)
	if parseErr := parseClient.WriteClipboard(context.Background(), string([]byte{0xff})); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseTest.Errorf("invalid clipboard UTF-8: %v", parseErr)
	}
	for _, parseRequest := range []MessageRequest{{Kind: "unknown"}, {Kind: "info", Title: string([]byte{0xff})}, {Kind: "question", Message: "bad\x00message"}} {
		if _, parseErr := parseClient.ShowMessage(context.Background(), parseRequest); !interop.IsCode(parseErr, interop.CodeInvalid) {
			parseTest.Errorf("invalid message: %v", parseErr)
		}
	}
	if _, parseErr := parseClient.ControlWindow(context.Background(), WindowRequest{Action: "resize", Width: 100000, Height: 1}); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseTest.Errorf("invalid window: %v", parseErr)
	}
}

// TestNativeMenuCapabilityDoesNotInventRPC distinguishes installed menus from callable methods.
func TestNativeMenuCapabilityDoesNotInventRPC(parseTest *testing.T) {
	parseTransport := getTestTransport()
	parseClient := NewClient(parseTransport)
	if parseClient.Supports(NativeMenus) {
		parseTest.Fatal("unadvertised native menus accepted")
	}
	parseTransport.parseCapabilities.Features = []Feature{NativeMenus, Clipboard}
	if !parseClient.Supports(NativeMenus) {
		parseTest.Fatal("installed native menus not discovered")
	}
	if parseClient.Supports(Clipboard) {
		parseTest.Fatal("feature advertisement bypassed missing clipboard methods")
	}
	if _, parseErr := Call[any](context.Background(), parseClient, "desktop.menu.install"); !interop.IsCode(parseErr, interop.CodeMissingExport) {
		parseTest.Fatalf("invented menu RPC accepted: %v", parseErr)
	}
}

// TestNativeMessageReplyMatchesKind rejects buttons from the wrong native dialog family.
func TestNativeMessageReplyMatchesKind(parseTest *testing.T) {
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(&parseNativeFake{parseFeatures: []Feature{MessageDialogs}}, parsePolicy)
	if _, parseErr := parseHost.ShowMessage(context.Background(), MessageRequest{Kind: "question"}); !interop.IsCode(parseErr, interop.CodeDecode) {
		parseTest.Errorf("host accepted question Ok: %v", parseErr)
	}
	parseData, _ := json.Marshal(NativeReply{Version: NativeContractVersion, Data: json.RawMessage(`{"button":"Yes"}`)})
	parseClient := NewClient(&parseNativeTransport{parseMethods: []string{MessageMethod}, parseReplies: map[string]Reply{MessageMethod: {Done: true, Data: parseData}}})
	if _, parseErr := parseClient.ShowMessage(context.Background(), MessageRequest{Kind: "info"}); !interop.IsCode(parseErr, interop.CodeDecode) {
		parseTest.Errorf("client accepted information Yes: %v", parseErr)
	}
}

// TestNativeMalformedRequestIsInvalid preserves caller-error classification without backend effects.
func TestNativeMalformedRequestIsInvalid(parseTest *testing.T) {
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseFake := &parseNativeFake{parseFeatures: []Feature{Clipboard, MessageDialogs, WindowControls, Screens}}
	parseHost := NewNativeHost(parseFake, parsePolicy)
	for _, parseMethod := range []string{ClipboardWriteMethod, ClipboardReadMethod, MessageMethod, WindowMethod, ScreensMethod} {
		parseReply := parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: parseMethod, Args: json.RawMessage(`{"broken"`)})
		if parseReply.Code != interop.CodeInvalid {
			parseTest.Errorf("%s malformed request: %+v", parseMethod, parseReply)
		}
	}
	if parseFake.parseWrites != 0 || parseFake.parseReads != 0 {
		parseTest.Fatal("malformed request reached backend")
	}
}

// TestNativeClipboardClientRejectsMalformedResult checks the same text bounds as the host.
func TestNativeClipboardClientRejectsMalformedResult(parseTest *testing.T) {
	for _, parseText := range []string{strings.Repeat("x", 4097), "bad\x00text"} {
		parseValue, _ := json.Marshal(parseText)
		parseData, _ := json.Marshal(NativeReply{Version: NativeContractVersion, Data: parseValue})
		parseClient := NewClient(&parseNativeTransport{parseMethods: []string{ClipboardReadMethod, ClipboardWriteMethod}, parseReplies: map[string]Reply{ClipboardReadMethod: {Done: true, Data: parseData}}})
		if _, parseErr := parseClient.ReadClipboard(context.Background()); !interop.IsCode(parseErr, interop.CodeDecode) {
			parseTest.Errorf("malformed clipboard result accepted: %v", parseErr)
		}
	}
}
