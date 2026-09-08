//go:build js && wasm

package desktop

import (
	"context"
	"encoding/base64"
	"strings"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/events"
	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// getWasmFixture imports the actual embedded ES module and gives tests pure-JS controlled promises/events.
func getWasmFixture(parseT *testing.T) (Client, js.Value) {
	parseT.Helper()
	parseResult := make(chan js.Value, 1)
	parseFailure := make(chan string, 1)
	parseResolve := js.FuncOf(func(_ js.Value, parseArgs []js.Value) any { parseResult <- parseArgs[0]; return nil })
	parseReject := js.FuncOf(func(_ js.Value, parseArgs []js.Value) any { parseFailure <- parseArgs[0].String(); return nil })
	parseURL := "data:text/javascript;base64," + base64.StdEncoding.EncodeToString([]byte(BootstrapSource))
	js.Global().Get("Function").New("src", "return import(src)").Invoke(parseURL).Call("then", parseResolve, parseReject)
	var parseModule js.Value
	select {
	case parseModule = <-parseResult:
	case parseMessage := <-parseFailure:
		parseResolve.Release()
		parseReject.Release()
		parseT.Fatal(parseMessage)
	case <-time.After(3 * time.Second):
		// A failed import is a test failure; do not release callbacks that could still settle.
		parseT.Fatal("desktop.js import timed out")
	}
	parseResolve.Release()
	parseReject.Release()
	parseFixture := js.Global().Get("Function").New("create", getFixtureSource).Invoke(parseModule.Get("createDesktopTransport"))
	parsePrevious := js.Global().Get("__gwcDesktop")
	js.Global().Set("__gwcDesktop", parseFixture.Get("transport"))
	parseT.Cleanup(func() { parseFixture.Get("transport").Call("close"); js.Global().Set("__gwcDesktop", parsePrevious) })
	parseClient, parseErr := Connect()
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	return parseClient, parseFixture
}

// getFixtureSource deliberately retains controllable promises to test cancellation after the JS registry forgets them.
const getFixtureSource = `
const state = { cancelled: 0, stopped: 0, handlers: new Set(), resolve: null, reject: null, closeOnListen: false, throwOnStop: false };
let transport;
const methods = {
  "desktop.files.select": request => ({selection: {paths: ["C:/fixture/雪.txt"], cancelled: false}}),
  "desktop.clipboard.write": request => request.args && request.args.text === "error" ? ({version: 1, code: "unavailable", message: "clipboard disabled"}) : ({version: 1, data: {}}),
  "desktop.clipboard.read": request => ({version: 1, data: "fixture"}),
  "desktop.message.show": request => ({version: 1, data: {button: "Yes"}}),
  "desktop.window.control": request => ({version: 1, data: {id: "wasm-window", width: 800, height: 600}}),
  "desktop.screens.list": request => ({version: 1, data: [{id: "wasm-screen", width: 1920, height: 1080}]}),
  echo: value => value,
  reject: () => Promise.reject(new Error("native rejected")),
  throwing: () => { throw new Error("native threw"); },
  unsafe: () => 9007199254740992,
  cyclic: () => { const value = {}; value.self = value; return value; },
  badError: () => Promise.reject({ get message() { throw new Error("message getter failed"); } }),
  never: () => { const promise = new Promise((resolve,reject) => {state.resolve=resolve;state.reject=reject;}); promise.cancel=()=>{state.cancelled++;}; return promise; },
  cancelThrows: () => Object.defineProperty(new Promise(()=>{}), "cancel", {get(){throw new Error("cancel getter threw");}}),
  closing: () => {transport.close(); return 7; }
};
const events = { On: (topic,handler) => {
  state.handlers.add(handler);
  if (state.closeOnListen) transport.close();
  return () => {state.handlers.delete(handler);state.stopped++;if(state.throwOnStop)throw new Error("stop failed");};
}};
transport = create({methods, topics:["counter.progress"],features:["native-menus"],events,platform:"test",hostVersion:"test",limit:2});
return {transport,state,emit: value=>{for(const handler of state.handlers)handler({data:value});}};
`

// TestWasmFileWorkflow uses the real JS transport without native backend dependencies.
func TestWasmFileWorkflow(parseT *testing.T) {
	parseClient, _ := getWasmFixture(parseT)
	if !parseClient.Supports(FileDialogs) {
		parseT.Fatal("missing file capability")
	}
	parseSelection, parseErr := OpenFile(context.Background(), FileDialogOptions{Title: "Synthetic"})
	if parseErr != nil || len(parseSelection.Paths) != 1 || parseSelection.Paths[0] != "C:/fixture/雪.txt" {
		parseT.Fatalf("selection=%+v error=%v", parseSelection, parseErr)
	}
}

// TestWasmNativeEnvelopeWorkflows uses the embedded desktop.js transport and envelope-shaped native methods.
func TestWasmNativeEnvelopeWorkflows(parseT *testing.T) {
	parseClient, _ := getWasmFixture(parseT)
	if parseErr := parseClient.WriteClipboard(context.Background(), "fixture"); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseValue, parseErr := parseClient.ReadClipboard(context.Background()); parseErr != nil || parseValue != "fixture" {
		parseT.Fatalf("clipboard=%q err=%v", parseValue, parseErr)
	}
	if parseValue, parseErr := parseClient.ShowMessage(context.Background(), MessageRequest{Kind: "question", Title: "Question", Message: "Continue?"}); parseErr != nil || parseValue.Button != "Yes" {
		parseT.Fatalf("message=%+v err=%v", parseValue, parseErr)
	}
	if parseValue, parseErr := parseClient.ControlWindow(context.Background(), WindowRequest{Action: "info"}); parseErr != nil || parseValue.ID != "wasm-window" {
		parseT.Fatalf("window=%+v err=%v", parseValue, parseErr)
	}
	if parseValue, parseErr := parseClient.ListScreens(context.Background()); parseErr != nil || len(parseValue) != 1 || parseValue[0].ID != "wasm-screen" {
		parseT.Fatalf("screens=%+v err=%v", parseValue, parseErr)
	}
}

// TestWasmNativeEnvelopeErrorCode verifies a structured host code survives the real JS transport.
func TestWasmNativeEnvelopeErrorCode(parseT *testing.T) {
	parseClient, _ := getWasmFixture(parseT)
	if parseErr := parseClient.WriteClipboard(context.Background(), "error"); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("error=%v", parseErr)
	}
}

// getWasmCount reads registry sizes without reaching into retained payloads.
func getWasmCount(parseFixture js.Value, parseName string) int {
	return parseFixture.Get("transport").Call("stats").Get(parseName).Int()
}

// TestWasmNativeMenuCapability reads the actual JS feature advertisement without a synthetic RPC.
func TestWasmNativeMenuCapability(parseT *testing.T) {
	parseClient, _ := getWasmFixture(parseT)
	if !parseClient.Supports(NativeMenus) {
		parseT.Fatal("native menu feature lost across actual JS transport")
	}
	if _, parseErr := Call[any](context.Background(), parseClient, "desktop.menu.install"); !interop.IsCode(parseErr, interop.CodeMissingExport) {
		parseT.Fatalf("fake menu RPC accepted: %v", parseErr)
	}
}

// TestWasmUnavailableAndMalformedTransport checks ordinary browser and malformed host boundaries.
func TestWasmUnavailableAndMalformedTransport(parseT *testing.T) {
	parsePrevious := js.Global().Get("__gwcDesktop")
	defer js.Global().Set("__gwcDesktop", parsePrevious)
	js.Global().Set("__gwcDesktop", js.Null())
	if _, parseErr := Connect(); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("unavailable = %v", parseErr)
	}
	for _, parseSource := range []string{`return {capabilities(){throw new Error("transport threw")}}`, `return {capabilities(){return "not-json"}}`, `return {capabilities(){return '{"data":{"protocol":999}}'}}`} {
		js.Global().Set("__gwcDesktop", js.Global().Get("Function").New(parseSource).Invoke())
		if _, parseErr := Connect(); parseErr == nil {
			parseT.Fatal("malformed bootstrap accepted")
		}
	}
}

// TestWasmWireAndRejections verifies typed JSON round trips and safe exception/error serialization.
func TestWasmWireAndRejections(parseT *testing.T) {
	parseClient, parseFixture := getWasmFixture(parseT)
	type getWire struct {
		ID    string    `json:"id"`
		Bytes []byte    `json:"bytes"`
		At    time.Time `json:"at"`
		Empty *string   `json:"empty"`
	}
	parseInput := getWire{ID: "9007199254740993", Bytes: []byte{0, 1, 255}, At: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)}
	parseOutput, parseErr := Call[getWire](context.Background(), parseClient, "echo", parseInput)
	if parseErr != nil || parseOutput.ID != parseInput.ID || string(parseOutput.Bytes) != string(parseInput.Bytes) || !parseOutput.At.Equal(parseInput.At) || parseOutput.Empty != nil {
		parseT.Fatalf("wire=%+v error=%v", parseOutput, parseErr)
	}
	for _, parseCase := range []struct {
		parseMethod  string
		parseCode    interop.ErrorCode
		parseMessage string
	}{
		{"reject", interop.CodeRemote, "native rejected"}, {"throwing", interop.CodeRemote, "native threw"},
		{"unsafe", interop.CodeDecode, ""}, {"cyclic", interop.CodeDecode, ""}, {"badError", interop.CodeRemote, ""},
	} {
		parseContext, parseCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		_, parseErr = Call[any](parseContext, parseClient, parseCase.parseMethod)
		parseCancel()
		if !interop.IsCode(parseErr, parseCase.parseCode) || !strings.Contains(parseErr.Error(), parseCase.parseMessage) {
			parseT.Errorf("%s = %v, want %s", parseCase.parseMethod, parseErr, parseCase.parseCode)
		}
		if parseCount := getWasmCount(parseFixture, "requests"); parseCount != 0 {
			parseT.Errorf("%s leaked %d requests", parseCase.parseMethod, parseCount)
		}
	}
}

// TestWasmInteractiveTimeout checks override validation and earlier deadlines against real JS promises.
func TestWasmInteractiveTimeout(parseT *testing.T) {
	parseClient, parseFixture := getWasmFixture(parseT)
	if _, parseErr := CallWithTimeout[any](context.Background(), parseClient, MaximumRequestTimeout+time.Second, "never"); !interop.IsCode(parseErr, interop.CodeInvalid) || getWasmCount(parseFixture, "requests") != 0 {
		parseT.Fatalf("invalid timeout: %v", parseErr)
	}
	parseContext, parseCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer parseCancel()
	if _, parseErr := CallWithTimeout[any](parseContext, parseClient, 5*time.Minute, "never"); !interop.IsCode(parseErr, interop.CodeTimeout) {
		parseT.Fatalf("earlier deadline: %v", parseErr)
	}
	if getWasmCount(parseFixture, "requests") != 0 || parseFixture.Get("state").Get("cancelled").Int() != 1 {
		parseT.Fatal("interactive timeout failed native cancellation/registry cleanup")
	}
}

// TestWasmCancellationDeadlineAndLateSettlement checks real registry deletion and native cancel invocation.
func TestWasmCancellationDeadlineAndLateSettlement(parseT *testing.T) {
	parseClient, parseFixture := getWasmFixture(parseT)
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if _, parseErr := Call[any](parseContext, parseClient, "never"); !interop.IsCode(parseErr, interop.CodeCancelled) {
		parseT.Fatal(parseErr)
	}
	if parseFixture.Get("state").Get("cancelled").Int() != 0 {
		parseT.Fatal("cancel-before-call started native work")
	}
	for parseIndex := 0; parseIndex < 4; parseIndex++ {
		parseContext, parseCancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
		_, parseErr := Call[any](parseContext, parseClient, "never")
		parseCancel()
		if !interop.IsCode(parseErr, interop.CodeTimeout) {
			parseT.Fatalf("deadline=%v", parseErr)
		}
		if getWasmCount(parseFixture, "requests") != 0 {
			parseT.Fatal("never-settling request retained")
		}
		if parseIndex%2 == 0 {
			parseFixture.Get("state").Call("resolve", "late")
		} else {
			parseFixture.Get("state").Call("reject", "late rejection")
		}
		time.Sleep(time.Millisecond)
		if getWasmCount(parseFixture, "requests") != 0 {
			parseT.Fatal("late settlement restored a request")
		}
	}
	if parseFixture.Get("state").Get("cancelled").Int() != 4 {
		parseT.Fatal("deadline did not forward cancel exactly once")
	}
	parseContext, parseCancel = context.WithCancel(context.Background())
	go func() { time.Sleep(10 * time.Millisecond); parseCancel() }()
	if _, parseErr := Call[any](parseContext, parseClient, "never"); !interop.IsCode(parseErr, interop.CodeCancelled) {
		parseT.Fatalf("in-flight cancel=%v", parseErr)
	}
}

// TestWasmCloseAndQuota bounds outstanding resources and rejects reentrant starts after close.
func TestWasmCloseAndQuota(parseT *testing.T) {
	parseClient, parseFixture := getWasmFixture(parseT)
	parseTransport := parseClient.parseTransport
	parseFirst, parseErr := parseTransport.Start("never", []byte("[]"))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	_, parseErr = parseTransport.Start("never", []byte("[]"))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if _, parseErr = parseTransport.Start("never", []byte("[]")); !interop.IsCode(parseErr, interop.CodeQuotaExceeded) {
		parseT.Fatalf("quota=%v", parseErr)
	}
	parseFixture.Get("transport").Call("close")
	if getWasmCount(parseFixture, "requests") != 0 || parseFixture.Get("state").Get("cancelled").Int() != 2 {
		parseT.Fatal("close leaked pending requests")
	}
	if _, parseErr = parseTransport.Poll(parseFirst); !interop.IsCode(parseErr, interop.CodeDisposed) {
		parseT.Fatalf("closed poll=%v", parseErr)
	}
	parseClient, parseFixture = getWasmFixture(parseT)
	if _, parseErr = Call[int](context.Background(), parseClient, "closing"); !interop.IsCode(parseErr, interop.CodeDisposed) {
		parseT.Fatalf("reentrant close=%v", parseErr)
	}
	if getWasmCount(parseFixture, "requests") != 0 {
		parseT.Fatal("method closed the transport but start added a request afterward")
	}
}

// TestWasmEventsCoalesceDecodeAndRelease proves burst semantics and idempotent native unsubscribe.
func TestWasmEventsCoalesceDecodeAndRelease(parseT *testing.T) {
	parseClient, parseFixture := getWasmFixture(parseT)
	parseReceived := make(chan struct {
		parseValue int
		parseError error
	}, 8)
	parseStop, parseErr := Subscribe[int](context.Background(), parseClient, "counter.progress", func(parseValue int, parseErr error) {
		parseReceived <- struct {
			parseValue int
			parseError error
		}{parseValue, parseErr}
	})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	defer parseStop()
	events.Publish("counter.progress", 500)
	select {
	case <-parseReceived:
		parseT.Fatal("local GWC event was implicitly forwarded to the desktop subscription")
	case <-time.After(25 * time.Millisecond):
	}
	for parseIndex := 0; parseIndex < 100; parseIndex++ {
		parseFixture.Call("emit", parseIndex)
	}
	select {
	case parseEvent := <-parseReceived:
		if parseEvent.parseError != nil || parseEvent.parseValue != 99 {
			parseT.Fatalf("burst=%+v", parseEvent)
		}
	case <-time.After(time.Second):
		parseT.Fatal("event not delivered")
	}
	parseFixture.Call("emit", "invalid typed integer")
	select {
	case parseEvent := <-parseReceived:
		if !interop.IsCode(parseEvent.parseError, interop.CodeDecode) {
			parseT.Fatalf("decode=%v", parseEvent.parseError)
		}
	case <-time.After(time.Second):
		parseT.Fatal("decode error not delivered")
	}
	parseStop()
	parseStop()
	if getWasmCount(parseFixture, "subscriptions") != 0 || parseFixture.Get("state").Get("stopped").Int() != 1 {
		parseT.Fatal("subscription not released exactly once")
	}
	parseFixture.Call("emit", 200)
	select {
	case <-parseReceived:
		parseT.Fatal("event after unsubscribe")
	case <-time.After(25 * time.Millisecond):
	}
}

// TestWasmEventCloseDuringRegistration releases an unsubscribe returned after reentrant close.
func TestWasmEventCloseDuringRegistration(parseT *testing.T) {
	parseClient, parseFixture := getWasmFixture(parseT)
	parseFixture.Get("state").Set("closeOnListen", true)
	if _, parseErr := parseClient.parseTransport.Listen("counter.progress"); !interop.IsCode(parseErr, interop.CodeDisposed) {
		parseT.Errorf("reentrant listen=%v", parseErr)
	}
	if parseFixture.Get("state").Get("handlers").Get("size").Int() != 0 {
		parseT.Fatal("event listener retained after registration closed the transport")
	}
}

// TestWasmTransportRejectsInvalidPayloadAndReleasesThrowingCleanup checks direct bridge boundaries and failure cleanup.
func TestWasmTransportRejectsInvalidPayloadAndReleasesThrowingCleanup(parseT *testing.T) {
	parseClient, parseFixture := getWasmFixture(parseT)
	parseTransport := parseClient.parseTransport
	if _, parseErr := parseTransport.Start("echo", []byte("{}")); !interop.IsCode(parseErr, interop.CodeEncode) {
		parseT.Fatalf("non-array payload=%v", parseErr)
	}
	if _, parseErr := parseTransport.Start("missing", []byte("[]")); !interop.IsCode(parseErr, interop.CodeMissingExport) {
		parseT.Fatalf("missing method=%v", parseErr)
	}
	if _, parseErr := parseTransport.Listen("other.topic"); !interop.IsCode(parseErr, interop.CodeMissingExport) {
		parseT.Fatalf("missing topic=%v", parseErr)
	}
	parseFirst, parseErr := parseTransport.Listen("counter.progress")
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if _, parseErr = parseTransport.Listen("counter.progress"); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if _, parseErr = parseTransport.Listen("counter.progress"); !interop.IsCode(parseErr, interop.CodeQuotaExceeded) {
		parseT.Fatalf("event quota=%v", parseErr)
	}
	parseFixture.Get("state").Set("throwOnStop", true)
	if parseErr = parseTransport.Unlisten(parseFirst); !interop.IsCode(parseErr, interop.CodeRemote) {
		parseT.Fatalf("unsubscribe error=%v", parseErr)
	}
	parseFixture.Get("state").Set("throwOnStop", false)
	if getWasmCount(parseFixture, "subscriptions") != 1 {
		parseT.Fatal("throwing unsubscribe retained registry entry")
	}
	if _, parseErr = parseTransport.Start("cancelThrows", []byte("[]")); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if _, parseErr = parseTransport.Start("never", []byte("[]")); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseFixture.Get("transport").Call("close")
	if getWasmCount(parseFixture, "subscriptions") != 0 || getWasmCount(parseFixture, "requests") != 0 || parseFixture.Get("state").Get("handlers").Get("size").Int() != 0 {
		parseT.Fatal("throwing cancel interrupted remaining close cleanup")
	}
}
