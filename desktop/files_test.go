package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"testing"
)

type fileDialogFake struct {
	parseCalls   int
	parseResult  FileSelection
	parseErr     error
	parseRequest FileDialogRequest
}

// SelectPaths records backend work without opening an OS dialog.
func (parseFake *fileDialogFake) SelectPaths(_ context.Context, parseRequest FileDialogRequest) (FileSelection, error) {
	parseFake.parseCalls++
	parseFake.parseRequest = parseRequest
	return parseFake.parseResult, parseFake.parseErr
}

// TestFileHostDeniesBeforeSideEffects verifies direct-binding calls cannot bypass host policy.
func TestFileHostDeniesBeforeSideEffects(parseT *testing.T) {
	parseFake := &fileDialogFake{}
	parseRequest := FileDialogRequest{Version: 1, Kind: "open-file"}
	for _, parseHost := range []*FileDialogHost{nil, {}, NewFileDialogHost(parseFake, false)} {
		if parseReply := parseHost.SelectPaths(context.Background(), parseRequest); parseReply.Code != interop.CodeUnavailable {
			parseT.Fatalf("not denied: %+v", parseReply)
		}
	}
	parseHost := NewFileDialogHost(parseFake, true)
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if parseReply := parseHost.SelectPaths(parseContext, parseRequest); parseReply.Code != interop.CodeCancelled {
		parseT.Fatalf("context: %+v", parseReply)
	}
	parseRequest.Version = 2
	if parseReply := parseHost.SelectPaths(context.Background(), parseRequest); parseReply.Code != interop.CodeInvalid {
		parseT.Fatalf("version: %+v", parseReply)
	}
	if parseFake.parseCalls != 0 {
		parseT.Fatal("denied request reached backend")
	}
}

// TestFileClientWorkflows verifies every public operation through the same replaceable contract.
func TestFileClientWorkflows(parseT *testing.T) {
	parseFake := &fileDialogFake{parseResult: FileSelection{Paths: []string{"C:/fixture/雪.txt"}}}
	parseHost := NewFileDialogHost(parseFake, true)
	parseTransport := getTestTransport()
	parseTransport.parseCapabilities.Methods = parseHost.GetMethods()
	var parseReply FileDialogReply
	parseTransport.parseStart = func(parseMethod string, parseArgs json.RawMessage) (string, error) {
		if parseMethod != FileDialogMethod {
			parseT.Fatalf("method=%s", parseMethod)
		}
		var parseRequests []FileDialogRequest
		if parseErr := json.Unmarshal(parseArgs, &parseRequests); parseErr != nil {
			return "", parseErr
		}
		parseReply = parseHost.SelectPaths(context.Background(), parseRequests[0])
		return "files", nil
	}
	parseTransport.parsePoll = func(string) (Reply, error) {
		parseData, parseErr := json.Marshal(parseReply)
		return Reply{Done: true, Data: parseData}, parseErr
	}
	parseClient := NewClient(parseTransport)
	if !parseClient.Supports(FileDialogs) || parseClient.Supports(Feature("typo")) {
		parseT.Fatal("capabilities")
	}
	for _, parseCase := range []struct {
		parseKind string
		parseCall func(context.Context, FileDialogOptions) (FileSelection, error)
	}{
		{"open-file", parseClient.OpenFile}, {"open-files", parseClient.OpenFiles}, {"open-directory", parseClient.OpenDirectory}, {"save-file", parseClient.SaveFile},
	} {
		parseValue, parseErr := parseCase.parseCall(context.Background(), FileDialogOptions{Title: "Synthetic"})
		if parseErr != nil || len(parseValue.Paths) != 1 || parseFake.parseRequest.Kind != parseCase.parseKind {
			parseT.Fatalf("%s: %+v %v", parseCase.parseKind, parseValue, parseErr)
		}
	}
	parseFake.parseResult = FileSelection{Cancelled: true}
	if parseValue, parseErr := parseClient.OpenFile(context.Background(), FileDialogOptions{}); parseErr != nil || !parseValue.Cancelled {
		parseT.Fatalf("cancel: %+v %v", parseValue, parseErr)
	}
	parseFake.parseErr = errors.New("native failed")
	if _, parseErr := parseClient.OpenFile(context.Background(), FileDialogOptions{}); !interop.IsCode(parseErr, interop.CodeRemote) {
		parseT.Fatalf("error: %v", parseErr)
	}
	parseFake.parseErr = nil
	parseFake.parseResult = FileSelection{}
	if _, parseErr := parseClient.OpenFile(context.Background(), FileDialogOptions{}); !interop.IsCode(parseErr, interop.CodeDecode) {
		parseT.Fatalf("invalid result: %v", parseErr)
	}
	parseTransport.parseCapabilities.Methods = nil
	parseBefore := parseFake.parseCalls
	if _, parseErr := parseClient.OpenFile(context.Background(), FileDialogOptions{}); !interop.IsCode(parseErr, interop.CodeUnavailable) || parseBefore != parseFake.parseCalls {
		parseT.Fatalf("disabled: %v", parseErr)
	}
}

// TestFileContractBounds checks malformed options and ambiguous selection fail closed.
func TestFileContractBounds(parseT *testing.T) {
	for _, parseRequest := range []FileDialogRequest{
		{Version: 1, Kind: "open-file", Options: FileDialogOptions{Filename: "unexpected.txt"}},
		{Version: 1, Kind: "open-directory", Options: FileDialogOptions{Filters: []FileFilter{{Name: "Text", Pattern: "*.txt"}}}},
	} {
		if validateFileDialogRequest(parseRequest) == nil {
			parseT.Fatalf("unsupported option accepted: %+v", parseRequest)
		}
	}
	for _, parseRequest := range []FileDialogRequest{{Version: 1, Kind: "typo"}, {Version: 1, Kind: "open-file", Options: FileDialogOptions{Title: "bad\x00"}}, {Version: 1, Kind: "open-file", Options: FileDialogOptions{Filters: []FileFilter{{}}}}} {
		if validateFileDialogRequest(parseRequest) == nil {
			parseT.Fatalf("accepted %+v", parseRequest)
		}
	}
	for _, parseSelection := range []FileSelection{{Cancelled: true, Paths: []string{"x"}}, {Paths: []string{""}}, {Paths: []string{"a", "b"}}} {
		if validateFileSelection("open-file", parseSelection) == nil {
			parseT.Fatalf("accepted %+v", parseSelection)
		}
	}
	if (Client{}).Supports() {
		parseT.Fatal("zero client available")
	}
}
