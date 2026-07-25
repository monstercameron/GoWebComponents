package projection_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/projection"
)

// v5 P3.7 criterion (d) — a misspelled command does not compile.

// ------------------------------------------------------- the compile check

// buildTestdataProgram compiles one testdata program and reports its output.
//
// The binary goes to a temp path rather than the package directory, so a
// successful build does not leave an executable behind in the source tree.
func buildTestdataProgram(parseT *testing.T, parseProgram string) (string, error) {
	parseT.Helper()

	parseOutput := filepath.Join(parseT.TempDir(), "out")
	parseCommand := exec.Command("go", "build", "-o", parseOutput, "./testdata/"+parseProgram)
	parseCombined, parseErr := parseCommand.CombinedOutput()
	return string(parseCombined), parseErr
}

// TestCorrectCommandUsageCompiles is the positive half, and it is not optional.
//
// Without it the negative test could pass because of an unrelated build error —
// a missing import, a renamed symbol — and would be asserting nothing at all.
func TestCorrectCommandUsageCompiles(parseT *testing.T) {
	if _, parseErr := exec.LookPath("go"); parseErr != nil {
		parseT.Skip("the go toolchain is not available")
	}

	parseOutput, parseErr := buildTestdataProgram(parseT, "goodcommand")
	if parseErr != nil {
		parseT.Fatalf("correct command usage must compile, but it failed:\n%s", parseOutput)
	}
}

// TestMisspelledCommandDoesNotCompile is P3.7 criterion (d).
func TestMisspelledCommandDoesNotCompile(parseT *testing.T) {
	if _, parseErr := exec.LookPath("go"); parseErr != nil {
		parseT.Skip("the go toolchain is not available")
	}

	parseOutput, parseErr := buildTestdataProgram(parseT, "badcommand")
	if parseErr == nil {
		parseT.Fatal("a misspelled command compiled — with a string-keyed API this typo would fail in the worker after a user clicked something")
	}
	// The failure must be the typo, not something incidental. A test that
	// accepts any build failure would keep passing after the API changed shape.
	if !strings.Contains(parseOutput, "undefined: creatItem") {
		parseT.Errorf("build failed for the wrong reason:\n%s", parseOutput)
	}
}

// ------------------------------------------------------------ invocation

type recordedSend struct {
	name    string
	request []byte
}

// countingClient records every send, so tests can assert message COUNTS rather
// than trusting that no round trip happened.
type countingClient struct {
	sends    []recordedSend
	response []byte
	failWith error
}

func (parseClient *countingClient) Send(parseCtx context.Context, parseName string, parseRequest []byte) ([]byte, error) {
	parseClient.sends = append(parseClient.sends, recordedSend{name: parseName, request: parseRequest})
	if parseClient.failWith != nil {
		return nil, parseClient.failWith
	}
	return parseClient.response, nil
}

// jsonCodec is a Codec over encoding/json.
type jsonCodec struct{}

func (jsonCodec) Encode(parseValue any) ([]byte, error) { return json.Marshal(parseValue) }
func (jsonCodec) Decode(parseData []byte, parseTarget any) error {
	return json.Unmarshal(parseData, parseTarget)
}

type addItemArgs struct {
	Name string `json:"name"`
}

type addItemResult struct {
	ID int64 `json:"id"`
}

var addItem = projection.Define[addItemArgs, addItemResult]("addItem")

func TestInvokeSendsTheDeclaredNameAndDecodesTheResult(parseT *testing.T) {
	parseClient := &countingClient{response: []byte(`{"id":42}`)}

	parseResult, parseErr := addItem.Invoke(context.Background(), parseClient, jsonCodec{}, addItemArgs{Name: "widget"})
	if parseErr != nil {
		parseT.Fatalf("Invoke: %v", parseErr)
	}
	if parseResult.ID != 42 {
		parseT.Errorf("result = %+v, want id 42", parseResult)
	}

	if len(parseClient.sends) != 1 {
		parseT.Fatalf("sends = %d, want 1", len(parseClient.sends))
	}
	if parseClient.sends[0].name != "addItem" {
		parseT.Errorf("name = %q, want addItem", parseClient.sends[0].name)
	}
	if !strings.Contains(string(parseClient.sends[0].request), "widget") {
		parseT.Errorf("request = %q, want the argument encoded", parseClient.sends[0].request)
	}
}

func TestInvokeSurfacesTransportFailure(parseT *testing.T) {
	parseClient := &countingClient{failWith: errors.New("worker died")}
	if _, parseErr := addItem.Invoke(context.Background(), parseClient, jsonCodec{}, addItemArgs{}); parseErr == nil {
		parseT.Error("a transport failure must surface")
	}
}

func TestInvokeSurfacesADecodeFailure(parseT *testing.T) {
	parseClient := &countingClient{response: []byte("not json")}
	if _, parseErr := addItem.Invoke(context.Background(), parseClient, jsonCodec{}, addItemArgs{}); parseErr == nil {
		parseT.Error("an undecodable result must surface rather than yielding a zero value")
	}
}

func TestInvokeRejectsMissingDependencies(parseT *testing.T) {
	if _, parseErr := addItem.Invoke(context.Background(), nil, jsonCodec{}, addItemArgs{}); parseErr == nil {
		parseT.Error("a nil client must be rejected")
	}
	if _, parseErr := addItem.Invoke(context.Background(), &countingClient{}, nil, addItemArgs{}); parseErr == nil {
		parseT.Error("a nil codec must be rejected")
	}

	var parseUndeclared projection.Command[addItemArgs, addItemResult]
	if _, parseErr := parseUndeclared.Invoke(context.Background(), &countingClient{}, jsonCodec{}, addItemArgs{}); parseErr == nil {
		parseT.Error("a command that was never given a name must be rejected")
	}
}

// -------------------------------------------------------------- registry

// TestRegistryCatchesTheOneThingTypesCannot: the compiler checks that callers
// spelled the VARIABLE correctly, not that the name string inside the
// declaration matches a handler the worker has.
func TestRegistryCatchesTheOneThingTypesCannot(parseT *testing.T) {
	parseRegistry := projection.NewRegistry([]string{"addItem", "removeItem"})

	if parseErr := parseRegistry.Verify(addItem); parseErr != nil {
		parseT.Errorf("a declared command the worker handles must verify: %v", parseErr)
	}

	parseTypoed := projection.Define[addItemArgs, addItemResult]("addItm")
	parseErr := parseRegistry.Verify(parseTypoed)
	if parseErr == nil {
		parseT.Fatal("a declaration naming a handler the worker lacks must be caught at startup")
	}
	if !strings.Contains(parseErr.Error(), "addItm") {
		parseT.Errorf("err = %v, want it to name the offending command", parseErr)
	}
}

// TestRegistryReportsEveryMissingCommand: a version skew usually breaks several
// commands at once, and fixing them one boot at a time is needless.
func TestRegistryReportsEveryMissingCommand(parseT *testing.T) {
	parseRegistry := projection.NewRegistry([]string{"addItem"})

	parseErr := parseRegistry.Verify(
		addItem,
		projection.Define[addItemArgs, addItemResult]("gone1"),
		projection.Define[addItemArgs, addItemResult]("gone2"),
	)
	if parseErr == nil {
		parseT.Fatal("expected missing commands to be reported")
	}
	for _, parseWanted := range []string{"gone1", "gone2"} {
		if !strings.Contains(parseErr.Error(), parseWanted) {
			parseT.Errorf("err = %v, want it to mention %q", parseErr, parseWanted)
		}
	}
}

func TestRegistryRejectsBadInput(parseT *testing.T) {
	parseRegistry := projection.NewRegistry(nil)
	if parseErr := parseRegistry.Verify(projection.Command[addItemArgs, addItemResult]{}); parseErr == nil {
		parseT.Error("a command without a name must be rejected")
	}

	var parseNil *projection.Registry
	if parseErr := parseNil.Verify(addItem); parseErr == nil {
		parseT.Error("a nil registry must error rather than panic")
	}
}

func TestCommandNameIsReadable(parseT *testing.T) {
	if addItem.Name() != "addItem" {
		parseT.Errorf("Name() = %q, want addItem", addItem.Name())
	}
}

// TestTestdataIsNotBuiltNormally guards the compile-check harness itself: if the
// bad program were part of the ordinary build, this package would never compile.
func TestTestdataIsNotBuiltNormally(parseT *testing.T) {
	if _, parseErr := os.Stat(filepath.Join("testdata", "badcommand", "main.go")); parseErr != nil {
		parseT.Fatalf("the negative compile-check program is missing: %v", parseErr)
	}
}
