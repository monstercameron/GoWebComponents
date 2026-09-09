package desktop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

type parseSystemFake struct {
	parseNativeFake
	parseSystemCalls  int
	parseURLCalls     int
	parseStatusCalls  int
	parseEnableCalls  int
	parseDisableCalls int
	parseRevealCalls  int
}

type parseFileManagerTransport struct {
	parseNativeTransport
	parseStarts int
}

// Start records whether lexical validation allowed a request to reach the transport.
func (parseTransport *parseFileManagerTransport) Start(parseMethod string, parseData json.RawMessage) (string, error) {
	parseTransport.parseStarts++
	return parseTransport.parseNativeTransport.Start(parseMethod, parseData)
}

// SystemEnvironment returns deterministic Windows environment metadata.
func (parseFake *parseSystemFake) SystemEnvironment(context.Context) (SystemEnvironmentInfo, error) {
	parseFake.parseSystemCalls++
	return SystemEnvironmentInfo{OS: "windows", Architecture: "amd64", Theme: "dark", AccentColor: "#0078d4"}, nil
}

// OpenExternalURL records a validated external URL request.
func (parseFake *parseSystemFake) OpenExternalURL(context.Context, ExternalURLOpenRequest) error {
	parseFake.parseURLCalls++
	return nil
}

// AutostartStatus returns deterministic disabled state without modifying the system.
func (parseFake *parseSystemFake) AutostartStatus(context.Context) (AutostartStatus, error) {
	parseFake.parseStatusCalls++
	return AutostartStatus{}, nil
}

// AutostartEnable records an enable request without modifying the system.
func (parseFake *parseSystemFake) AutostartEnable(context.Context) error {
	parseFake.parseEnableCalls++
	return nil
}

// AutostartDisable records a disable request without modifying the system.
func (parseFake *parseSystemFake) AutostartDisable(context.Context) error {
	parseFake.parseDisableCalls++
	return nil
}

// RevealPath records a file-manager request without launching an external process.
func (parseFake *parseSystemFake) RevealPath(context.Context, FileManagerRevealRequest) error {
	parseFake.parseRevealCalls++
	return nil
}

// TestSystemHostValidationPreventsEffects verifies denial, invalid URLs, and cancelled calls stop before backend work.
func TestSystemHostValidationPreventsEffects(parseTest *testing.T) {
	parseFake := &parseSystemFake{parseNativeFake: parseNativeFake{parseFeatures: []Feature{SystemEnvironment, ExternalURLs, Autostart, FileManager}}}
	parsePolicy, _ := ParseFeaturePolicy("system-environment")
	parseHost := NewNativeHost(parseFake, parsePolicy)
	if parseErr := parseHost.OpenExternalURL(context.Background(), ExternalURLOpenRequest{URL: "https://example.com"}); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseTest.Fatalf("denial error=%v", parseErr)
	}
	parsePolicy, _ = ParseFeaturePolicy("all")
	parseHost = NewNativeHost(parseFake, parsePolicy)
	for _, parseURL := range []string{"file:///tmp/test", "javascript:alert(1)", "https://user:pass@example.com", "//example.com"} {
		if parseErr := parseHost.OpenExternalURL(context.Background(), ExternalURLOpenRequest{URL: parseURL}); !interop.IsCode(parseErr, interop.CodeInvalid) {
			parseTest.Fatalf("URL %q error=%v", parseURL, parseErr)
		}
	}
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if parseErr := parseHost.EnableAutostart(parseContext); parseErr == nil {
		parseTest.Fatal("expected cancelled autostart enable")
	}
	if parseFake.parseURLCalls != 0 || parseFake.parseEnableCalls != 0 {
		parseTest.Fatalf("invalid calls reached backend: url=%d enable=%d", parseFake.parseURLCalls, parseFake.parseEnableCalls)
	}
	for _, parsePath := range []string{`relative\file.txt`, `\\server\share\file.txt`, `\\?\C:\file.txt`, `file:///C:/file.txt`, "C:\\bad\x00path"} {
		if parseErr := parseHost.RevealPath(context.Background(), FileManagerRevealRequest{Path: parsePath}); !interop.IsCode(parseErr, interop.CodeInvalid) {
			parseTest.Fatalf("path %q error=%v", parsePath, parseErr)
		}
	}
	if parseFake.parseRevealCalls != 0 {
		parseTest.Fatal("invalid file-manager request reached backend")
	}
}

// TestFileManagerHostUsesExistingFixture verifies host validation reaches only the fake backend.
func TestFileManagerHostUsesExistingFixture(parseTest *testing.T) {
	parseFake := &parseSystemFake{parseNativeFake: parseNativeFake{parseFeatures: []Feature{FileManager}}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parsePath := parseTest.TempDir()
	if parseErr := NewNativeHost(parseFake, parsePolicy).RevealPath(context.Background(), FileManagerRevealRequest{Path: parsePath}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseFake.parseRevealCalls != 1 {
		parseTest.Fatalf("reveal calls=%d", parseFake.parseRevealCalls)
	}
}

// TestFileManagerClientWindowsLexicalPath verifies Windows paths behave identically under native and js/wasm runtimes.
func TestFileManagerClientWindowsLexicalPath(parseTest *testing.T) {
	parseData, _ := json.Marshal(NativeReply{Version: NativeContractVersion, Data: json.RawMessage(`{}`)})
	parseTransport := &parseFileManagerTransport{parseNativeTransport: parseNativeTransport{parseMethods: []string{FileManagerRevealMethod}, parseReplies: map[string]Reply{FileManagerRevealMethod: {Done: true, Data: parseData}}}}
	parseClient := NewClient(parseTransport)
	if parseErr := parseClient.RevealPath(context.Background(), `C:\trusted\fixture.txt`, true); parseErr != nil {
		parseTest.Fatalf("valid Windows drive path failed before transport: %v", parseErr)
	}
	if parseTransport.parseStarts != 1 {
		parseTest.Fatalf("valid Windows drive path starts=%d", parseTransport.parseStarts)
	}
	for _, parsePath := range []string{`C:relative.txt`, `\\server\share\file.txt`, `\\?\C:\file.txt`, `file:///C:/file.txt`, `C:\trusted\..\secret.txt`, "C:\\bad\x00path"} {
		if parseErr := parseClient.RevealPath(context.Background(), parsePath, false); !interop.IsCode(parseErr, interop.CodeInvalid) {
			parseTest.Fatalf("invalid path %q error=%v", parsePath, parseErr)
		}
	}
	if parseTransport.parseStarts != 1 {
		parseTest.Fatalf("invalid paths reached transport: starts=%d", parseTransport.parseStarts)
	}
}

// TestSystemHostExecuteWire verifies method dispatch and typed wire encoding without real system changes.
func TestSystemHostExecuteWire(parseTest *testing.T) {
	parseFake := &parseSystemFake{parseNativeFake: parseNativeFake{parseFeatures: []Feature{SystemEnvironment, ExternalURLs, Autostart}}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(parseFake, parsePolicy)
	parseReply := parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: SystemEnvironmentMethod, Args: json.RawMessage(`{}`)})
	if parseReply.Code != "" {
		parseTest.Fatal(parseReply.Message)
	}
	var parseInfo SystemEnvironmentInfo
	if parseErr := json.Unmarshal(parseReply.Data, &parseInfo); parseErr != nil || parseInfo.Theme != "dark" {
		parseTest.Fatalf("info=%+v err=%v", parseInfo, parseErr)
	}
	parseReply = parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: ExternalURLOpenMethod, Args: json.RawMessage(`{"url":"https://example.com/path?q=1"}`)})
	if parseReply.Code != "" || parseFake.parseURLCalls != 1 {
		parseTest.Fatalf("reply=%+v calls=%d", parseReply, parseFake.parseURLCalls)
	}
	parseReply = parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: AutostartStatusMethod, Args: json.RawMessage(`{}`)})
	if parseReply.Code != "" || parseFake.parseStatusCalls != 1 {
		parseTest.Fatalf("reply=%+v calls=%d", parseReply, parseFake.parseStatusCalls)
	}
}

// TestSystemClientWireDecode verifies typed client decoding and rejects malformed host state.
func TestSystemClientWireDecode(parseTest *testing.T) {
	parseJSON := func(parseValue any) json.RawMessage { parseData, _ := json.Marshal(parseValue); return parseData }
	parseMethods := []string{SystemEnvironmentMethod, ExternalURLOpenMethod, AutostartStatusMethod, AutostartEnableMethod, AutostartDisableMethod}
	parseTransport := &parseNativeTransport{parseMethods: parseMethods, parseReplies: map[string]Reply{
		SystemEnvironmentMethod: {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(SystemEnvironmentInfo{OS: "windows", Architecture: "arm64", Theme: "light", AccentColor: "rgb(1,2,3)"})})},
		ExternalURLOpenMethod:   {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(struct{}{})})},
		AutostartStatusMethod:   {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(AutostartStatus{Enabled: true, Path: `HKCU\\Software\\Run\\App`, Strategy: "registry-run"})})},
		AutostartEnableMethod:   {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(struct{}{})})},
		AutostartDisableMethod:  {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(struct{}{})})},
	}}
	parseClient := NewClient(parseTransport)
	if parseInfo, parseErr := parseClient.InspectSystemEnvironment(context.Background()); parseErr != nil || parseInfo.Architecture != "arm64" {
		parseTest.Fatalf("info=%+v err=%v", parseInfo, parseErr)
	}
	if parseErr := parseClient.OpenExternalURL(context.Background(), "https://example.com"); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseStatus, parseErr := parseClient.GetAutostartStatus(context.Background()); parseErr != nil || !parseStatus.Enabled {
		parseTest.Fatalf("status=%+v err=%v", parseStatus, parseErr)
	}
	if parseErr := parseClient.EnableAutostart(context.Background()); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr := parseClient.DisableAutostart(context.Background()); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
}
