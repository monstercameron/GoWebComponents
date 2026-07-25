//go:build js && wasm

package interop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall/js"
)

func (parseS *goWASMWorkerState) start(parseCtx context.Context) error {
	parseWorkerOptions, parseBootstrapURL, parseErr := buildGoWASMWorkerOptions(parseS.options)
	if parseErr != nil {
		return parseErr
	}
	parseWorker, parseErr := OpenWorker(parseCtx, parseWorkerOptions)
	if parseErr != nil {
		revokeObjectURL(parseBootstrapURL)
		return parseErr
	}
	parseS.mu.Lock()
	parseS.worker = parseWorker
	parseS.bootstrapURL = parseBootstrapURL
	parseS.active = true
	parseS.mu.Unlock()
	return nil
}

func (parseS *goWASMWorkerState) current(parseOp string, parseTarget string) (Worker, error) {
	parseS.mu.RLock()
	defer parseS.mu.RUnlock()
	if !parseS.active {
		return Worker{}, wrapError(parseOp, parseTarget, CodeDisposed, errors.New("worker is not active"))
	}
	return parseS.worker, nil
}

func (parseS *goWASMWorkerState) post(parseMessage any) error {
	parseWorker, parseErr := parseS.current("Worker.Post", parseS.options.WASMURL)
	if parseErr != nil {
		return parseErr
	}
	return parseWorker.Post(parseMessage)
}

func (parseS *goWASMWorkerState) postPorts(parseMessage any, parsePorts ...MessagePort) error {
	parseWorker, parseErr := parseS.current("Worker.PostPorts", parseS.options.WASMURL)
	if parseErr != nil {
		return parseErr
	}
	return parseWorker.PostPorts(parseMessage, parsePorts...)
}

func (parseS *goWASMWorkerState) postTransferable(parseMessage any, parseBuffers []Transferable) error {
	parseWorker, parseErr := parseS.current("Worker.PostTransferable", parseS.options.WASMURL)
	if parseErr != nil {
		return parseErr
	}
	return parseWorker.PostTransferable(parseMessage, parseBuffers...)
}

func (parseS *goWASMWorkerState) subscribe(parseHandler func(WorkerMessage, error)) (Subscription, error) {
	parseWorker, parseErr := parseS.current("Worker.Subscribe", parseS.options.WASMURL)
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	return parseWorker.Subscribe(parseHandler)
}

func (parseS *goWASMWorkerState) request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	parseWorker, parseErr := parseS.current("Worker.Request", parseName)
	if parseErr != nil {
		return WorkerMessage{}, parseErr
	}
	return parseWorker.Request(parseCtx, parseName, parsePayload, parseOnProgress)
}

func (parseS *goWASMWorkerState) terminate() error {
	parseS.mu.Lock()
	parseWorker := parseS.worker
	parseBootstrapURL := parseS.bootstrapURL
	parseActive := parseS.active
	parseS.worker = Worker{}
	parseS.bootstrapURL = ""
	parseS.active = false
	parseS.mu.Unlock()
	if !parseActive {
		return wrapError("Worker.Terminate", parseS.options.WASMURL, CodeDisposed, errors.New("worker is not active"))
	}
	parseErr := parseWorker.Terminate()
	revokeObjectURL(parseBootstrapURL)
	return parseErr
}

func (parseS *goWASMWorkerState) restart(parseCtx context.Context) error {
	if parseErr := parseS.terminate(); parseErr != nil && !IsCode(parseErr, CodeDisposed) {
		return parseErr
	}
	return parseS.start(parseCtx)
}

func (parseS *browserWorkerState) start(parseCtx context.Context) error {
	parseRaw, parseErr := createBrowserWorker(parseS.options)
	if parseErr != nil {
		return parseErr
	}
	if parseS.options.Ready {
		parseWaitCtx := parseCtx
		if _, parseOk := parseWaitCtx.Deadline(); !parseOk {
			parseTimeout := parseS.options.ReadyTimeout
			if parseTimeout <= 0 {
				parseTimeout = defaultWorkerReadyTimeout
			}
			var parseCancel context.CancelFunc
			parseWaitCtx, parseCancel = context.WithTimeout(parseWaitCtx, parseTimeout)
			defer parseCancel()
		}
		if parseErr2 := waitWorkerReady(parseWaitCtx, parseRaw, parseS.options.URL); parseErr2 != nil {
			parseRaw.Call("terminate")
			return parseErr2
		}
	}
	parseS.mu.Lock()
	parseS.raw = parseRaw
	parseS.active = true
	parseS.mu.Unlock()
	return nil
}

func (parseS *browserWorkerState) current(parseOp string, parseTarget string) (js.Value, error) {
	parseS.mu.RLock()
	defer parseS.mu.RUnlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return js.Undefined(), wrapError(parseOp, parseTarget, CodeDisposed, errors.New("worker is not active"))
	}
	return parseS.raw, nil
}

func (parseS *browserWorkerState) post(parseMessage any) error {
	parseRaw, parseErr := parseS.current("Worker.Post", parseS.options.URL)
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("Worker.Post", parseS.options.URL, parseRaw, parseMessage)
}

func (parseS *browserWorkerState) postPorts(parseMessage any, parsePorts ...MessagePort) error {
	parseRaw, parseErr := parseS.current("Worker.PostPorts", parseS.options.URL)
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("Worker.PostPorts", parseS.options.URL, parseRaw, parseMessage, parsePorts...)
}

func (parseS *browserWorkerState) postTransferable(parseMessage any, parseBuffers []Transferable) error {
	parseRaw, parseErr := parseS.current("Worker.PostTransferable", parseS.options.URL)
	if parseErr != nil {
		return parseErr
	}
	return postTransferableMessageJS("Worker.PostTransferable", parseS.options.URL, parseRaw, parseMessage, parseBuffers)
}

func (parseS *browserWorkerState) subscribe(parseHandler func(WorkerMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("Worker.Subscribe", parseS.options.URL, CodeInvalid, errors.New("handler is nil"))
	}
	parseRaw, parseErr := parseS.current("Worker.Subscribe", parseS.options.URL)
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("subscribe callback")
		parseMessage, parseErr2 := workerMessageFromEvent("Worker.Subscribe", parseS.options.URL, parseArgs)
		parseHandler(parseMessage, parseErr2)
		return nil
	})
	parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("subscribe callback")
		parseHandler(WorkerMessage{}, wrapError("Worker.Subscribe", parseS.options.URL, CodeRemote, errors.New(workerRemoteErrorSummary(parseArgs2))))
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		defer RecoverContainedPanic("subscribe callback")
		parseHandler(WorkerMessage{}, wrapError("Worker.Subscribe", parseS.options.URL, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs3))))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "error", parseErrorFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	return Subscription{cancel: func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "error", parseErrorFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseErrorFn.Release()
		parseMessageErrorFn.Release()
	}}, nil
}

func (parseS *browserWorkerState) request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if strings.TrimSpace(parseName) == "" {
		return WorkerMessage{}, wrapError("Worker.Request", parseName, CodeInvalid, errors.New("request name is empty"))
	}
	parseRaw, parseErr := parseS.current("Worker.Request", parseName)
	if parseErr != nil {
		return WorkerMessage{}, parseErr
	}

	parseS.mu.Lock()
	parseS.nextRequestID++
	parseRequestID := fmt.Sprintf("worker-%d", parseS.nextRequestID)
	parseS.mu.Unlock()

	parseResultCh := make(chan WorkerMessage, 1)
	parseErrCh := make(chan error, 1)
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("request callback")
		parseMessage, parseDecodeErr := workerMessageFromEvent("Worker.Request", parseName, parseArgs)
		if parseDecodeErr != nil {
			if parseOnProgress != nil {
				parseOnProgress(WorkerMessage{}, parseDecodeErr)
			}
			parseErrCh <- parseDecodeErr
			return nil
		}
		if strings.TrimSpace(parseMessage.ID) != parseRequestID {
			return nil
		}
		switch strings.TrimSpace(parseMessage.Phase) {
		case "progress":
			if parseOnProgress != nil {
				parseOnProgress(parseMessage, nil)
			}
		case "error":
			parseErrCh <- wrapError("Worker.Request", parseName, CodeRemote, errors.New(workerRemoteEnvelopeError(parseMessage)))
		case "result", "message", "":
			parseResultCh <- parseMessage
		}
		return nil
	})
	parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("request callback")
		parseErrCh <- wrapError("Worker.Request", parseName, CodeRemote, errors.New(workerRemoteErrorSummary(parseArgs2)))
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		defer RecoverContainedPanic("request callback")
		parseErrCh <- wrapError("Worker.Request", parseName, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs3)))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "error", parseErrorFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	defer func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "error", parseErrorFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseErrorFn.Release()
		parseMessageErrorFn.Release()
	}()

	if parseErr2 := parseS.post(WorkerMessage{
		ID:      parseRequestID,
		Phase:   "request",
		Name:    parseName,
		Payload: parsePayload,
	}); parseErr2 != nil {
		return WorkerMessage{}, parseErr2
	}

	select {
	case parseMessage2 := <-parseResultCh:
		return parseMessage2, nil
	case parseErr3 := <-parseErrCh:
		return WorkerMessage{}, parseErr3
	case <-parseCtx.Done():
		return WorkerMessage{}, workerContextError("Worker.Request", parseName, parseCtx.Err())
	}
}

func (parseS *browserWorkerState) terminate() error {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return wrapError("Worker.Terminate", parseS.options.URL, CodeDisposed, errors.New("worker is not active"))
	}
	parseS.raw.Call("terminate")
	parseS.raw = js.Undefined()
	parseS.active = false
	return nil
}

func (parseS *browserWorkerState) restart(parseCtx context.Context) error {
	parseS.mu.Lock()
	parseRaw := parseS.raw
	parseActive := parseS.active
	parseS.raw = js.Undefined()
	parseS.active = false
	parseS.mu.Unlock()
	if parseActive && !parseRaw.IsUndefined() && !parseRaw.IsNull() {
		parseRaw.Call("terminate")
	}
	return parseS.start(parseCtx)
}

func createBrowserWorker(parseOptions WorkerOptions) (js.Value, error) {
	parseWorkerType := strings.TrimSpace(parseOptions.Type)
	if parseWorkerType != "" && parseWorkerType != "classic" && parseWorkerType != "module" {
		return js.Undefined(), wrapError("NewWorker", parseOptions.URL, CodeInvalid, errors.New("worker type must be classic or module"))
	}
	parseCtor, parseErr := globalProperty("Worker", "Worker")
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	if parseWorkerType == "" && strings.HasSuffix(strings.ToLower(strings.TrimSpace(parseOptions.URL)), ".mjs") {
		parseWorkerType = "module"
	}
	if strings.TrimSpace(parseOptions.Name) == "" && parseWorkerType == "" {
		return parseCtor.New(parseOptions.URL), nil
	}
	parseInit := js.Global().Get("Object").New()
	if strings.TrimSpace(parseOptions.Name) != "" {
		parseInit.Set("name", parseOptions.Name)
	}
	if parseWorkerType != "" {
		parseInit.Set("type", parseWorkerType)
	}
	return parseCtor.New(parseOptions.URL, parseInit), nil
}

func newMessagePort(parseTarget string, parseRaw js.Value) MessagePort {
	parseState := &browserMessagePortState{
		raw:    parseRaw,
		active: true,
		target: parseTarget,
	}
	startMessagePort(parseRaw)
	return MessagePort{
		raw: parseRaw,
		// MessagePort.PostTransferable reaches the raw port directly, so it
		// needs no closure here — unlike Worker, whose raw handle is owned by a
		// restartable state object.
		post:      parseState.post,
		postPorts: parseState.postPorts,
		subscribe: parseState.subscribe,
		close:     parseState.close,
	}
}

func startMessagePort(parseRaw js.Value) {
	parseStart := parseRaw.Get("start")
	if parseStart.Type() == js.TypeFunction {
		parseRaw.Call("start")
	}
}

func (parseS *browserMessagePortState) current(parseOp string) (js.Value, error) {
	parseS.mu.RLock()
	defer parseS.mu.RUnlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return js.Undefined(), wrapError(parseOp, parseS.target, CodeDisposed, errors.New("message port is not active"))
	}
	return parseS.raw, nil
}

func (parseS *browserMessagePortState) post(parsePayload any) error {
	parseRaw, parseErr := parseS.current("MessagePort.Post")
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("MessagePort.Post", parseS.target, parseRaw, parsePayload)
}

func (parseS *browserMessagePortState) postPorts(parsePayload any, parsePorts ...MessagePort) error {
	parseRaw, parseErr := parseS.current("MessagePort.PostPorts")
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("MessagePort.PostPorts", parseS.target, parseRaw, parsePayload, parsePorts...)
}

func (parseS *browserMessagePortState) subscribe(parseHandler func(MessagePortMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("MessagePort.Subscribe", parseS.target, CodeInvalid, errors.New("handler is nil"))
	}
	parseRaw, parseErr := parseS.current("MessagePort.Subscribe")
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	startMessagePort(parseRaw)
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("subscribe callback")
		parseMessage, parseMessageErr := messagePortMessageFromEvent("MessagePort.Subscribe", parseS.target, parseArgs)
		parseHandler(parseMessage, parseMessageErr)
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("subscribe callback")
		parseHandler(MessagePortMessage{}, wrapError("MessagePort.Subscribe", parseS.target, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs2))))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	return Subscription{cancel: func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseMessageErrorFn.Release()
	}}, nil
}

func (parseS *browserMessagePortState) close() error {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return wrapError("MessagePort.Close", parseS.target, CodeDisposed, errors.New("message port is not active"))
	}
	parseS.raw.Call("close")
	parseS.raw = js.Undefined()
	parseS.active = false
	return nil
}

func waitWorkerReady(parseCtx context.Context, parseRaw js.Value, parseTarget string) error {
	parseReadyCh := make(chan struct{}, 1)
	parseErrCh := make(chan error, 1)
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("waitWorkerReady callback")
		parseMessage, parseErr := workerMessageFromEvent("NewWorker", parseTarget, parseArgs)
		if parseErr != nil {
			parseErrCh <- parseErr
			return nil
		}
		if strings.TrimSpace(parseMessage.Phase) == "ready" {
			parseReadyCh <- struct{}{}
			return nil
		}
		if strings.TrimSpace(parseMessage.Phase) == "error" {
			parseErrCh <- wrapError("NewWorker", parseTarget, CodeRemote, errors.New(workerRemoteEnvelopeError(parseMessage)))
		}
		return nil
	})
	parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("waitWorkerReady callback")
		parseErrCh <- wrapError("NewWorker", parseTarget, CodeRemote, errors.New(workerRemoteErrorSummary(parseArgs2)))
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		defer RecoverContainedPanic("waitWorkerReady callback")
		parseErrCh <- wrapError("NewWorker", parseTarget, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs3)))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "error", parseErrorFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	defer func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "error", parseErrorFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseErrorFn.Release()
		parseMessageErrorFn.Release()
	}()
	select {
	case <-parseReadyCh:
		return nil
	case parseErr2 := <-parseErrCh:
		consoleError(fmt.Sprintf("[interop/NewWorker] worker %q reported an error during startup: %v", parseTarget, parseErr2))
		return parseErr2
	case <-parseCtx.Done():
		consoleError(fmt.Sprintf("[interop/NewWorker] worker %q startup timed out - if this runs on the main goroutine inside a synchronous effect, the JS event loop is starved and the worker ready message can never arrive; wrap the call in a goroutine", parseTarget))
		return workerContextError("NewWorker", parseTarget, parseCtx.Err())
	}
}

func currentWorkerGlobal(parseOp string) (js.Value, error) {
	parseGlobal := js.Global()
	if parseDoc := parseGlobal.Get("document"); !parseDoc.IsUndefined() && !parseDoc.IsNull() {
		return js.Undefined(), unavailable(parseOp, "worker")
	}
	parsePostMessage := parseGlobal.Get("postMessage")
	if parsePostMessage.IsUndefined() || parsePostMessage.IsNull() {
		return js.Undefined(), unavailable(parseOp, "worker")
	}
	return parseGlobal, nil
}

func buildGoWASMWorkerOptions(parseOptions GoWASMWorkerOptions) (WorkerOptions, string, error) {
	parseRuntimeURL, parseErr := resolveURL("NewGoWASMWorker", parseOptions.RuntimeURL)
	if parseErr != nil {
		return WorkerOptions{}, "", parseErr
	}
	parseWasmURL, parseErr := resolveURL("NewGoWASMWorker", parseOptions.WASMURL)
	if parseErr != nil {
		return WorkerOptions{}, "", parseErr
	}
	parseBootstrapURL, parseErr := createObjectURL(goWASMWorkerBootstrapSource(parseRuntimeURL, parseWasmURL))
	if parseErr != nil {
		return WorkerOptions{}, "", parseErr
	}
	return WorkerOptions{
		URL:          parseBootstrapURL,
		Name:         parseOptions.Name,
		Type:         "classic",
		Ready:        parseOptions.Ready,
		ReadyTimeout: parseOptions.ReadyTimeout,
	}, parseBootstrapURL, nil
}

func resolveURL(parseOp string, parseInput string) (string, error) {
	parseTrimmed := strings.TrimSpace(parseInput)
	if parseTrimmed == "" {
		return "", wrapError(parseOp, parseInput, CodeInvalid, errors.New("URL is empty"))
	}
	if strings.HasPrefix(parseTrimmed, "http://") || strings.HasPrefix(parseTrimmed, "https://") || strings.HasPrefix(parseTrimmed, "blob:") || strings.HasPrefix(parseTrimmed, "data:") {
		return parseTrimmed, nil
	}
	parseUrlCtor, parseErr := globalProperty(parseOp, "URL")
	if parseErr != nil {
		return "", parseErr
	}
	parseBase := js.Undefined()
	if parseDocument := js.Global().Get("document"); !parseDocument.IsUndefined() && !parseDocument.IsNull() {
		parseBase = parseDocument.Get("baseURI")
	}
	if parseBase.IsUndefined() || parseBase.IsNull() || strings.TrimSpace(parseBase.String()) == "" {
		parseLocation, parseLocationErr := globalProperty(parseOp, "location")
		if parseLocationErr != nil {
			return "", parseLocationErr
		}
		parseBase = parseLocation.Get("href")
	}
	parseResolved := parseUrlCtor.New(parseTrimmed, parseBase).Get("href").String()
	if looksLikeJSTypeDescriptor(parseResolved) {
		consoleError(fmt.Sprintf("[interop/%s] resolveURL produced a JS type descriptor %q for input %q - this usually means js.Value.String() was called on a non-string JS value", parseOp, parseResolved, parseInput))
		return "", wrapError(parseOp, parseInput, CodeInvalid, fmt.Errorf("resolved URL is a JS type descriptor %q, not a valid URL - check that the input is a string value", parseResolved))
	}
	return parseResolved, nil
}

func createObjectURL(parseSource string) (string, error) {
	parseBlobCtor, parseErr := globalProperty("NewGoWASMWorker", "Blob")
	if parseErr != nil {
		return "", parseErr
	}
	parseUrlAPI, parseErr := globalProperty("NewGoWASMWorker", "URL")
	if parseErr != nil {
		return "", parseErr
	}
	parseParts := js.Global().Get("Array").New()
	parseParts.Call("push", parseSource)
	parseOptions := js.Global().Get("Object").New()
	parseOptions.Set("type", "text/javascript")
	parseBlob := parseBlobCtor.New(parseParts, parseOptions)
	parseResult := parseUrlAPI.Call("createObjectURL", parseBlob).String()
	if looksLikeJSTypeDescriptor(parseResult) {
		consoleError(fmt.Sprintf("[interop/NewGoWASMWorker] createObjectURL returned JS type descriptor %q instead of a blob: URL", parseResult))
		return "", wrapError("NewGoWASMWorker", "createObjectURL", CodeInvalid, fmt.Errorf("createObjectURL returned %q, not a valid blob: URL", parseResult))
	}
	return parseResult, nil
}

func revokeObjectURL(parseObjectURL string) {
	parseTrimmed := strings.TrimSpace(parseObjectURL)
	if parseTrimmed == "" {
		return
	}
	parseUrlAPI := js.Global().Get("URL")
	if parseUrlAPI.IsUndefined() || parseUrlAPI.IsNull() {
		return
	}
	parseUrlAPI.Call("revokeObjectURL", parseTrimmed)
}

func goWASMWorkerBootstrapSource(parseRuntimeURL string, parseWasmURL string) string {
	parseRuntimeJSON, _ := json.Marshal(parseRuntimeURL)
	parseWasmJSON, _ := json.Marshal(parseWasmURL)
	return `(function(){
const runtimeURL=` + string(parseRuntimeJSON) + `;
const wasmURL=` + string(parseWasmJSON) + `;
const postBootstrapError = (error) => {
  const message = error && error.message ? error.message : String(error);
  try {
    self.postMessage({ phase: "error", name: "bootstrap", error: message });
  } catch (_) {}
};
const looksInvalid = (url, label) => {
  if (!url || /^<\w+>$/.test(url)) {
    const msg = "[interop/worker-bootstrap] " + label + " is invalid: " + JSON.stringify(url) + " - this usually means a Go js.Value.String() was called on a non-string JS value";
    console.error(msg);
    postBootstrapError(new Error(msg));
    return true;
  }
  return false;
};
if (looksInvalid(runtimeURL, "runtimeURL") || looksInvalid(wasmURL, "wasmURL")) { return; }
const instantiate = async (go) => {
  if (WebAssembly.instantiateStreaming) {
    try {
      return await WebAssembly.instantiateStreaming(fetch(wasmURL), go.importObject);
    } catch (_) {}
  }
  const response = await fetch(wasmURL);
  if (!response.ok) {
    throw new Error("failed to fetch worker wasm: " + response.status + " " + response.statusText);
  }
  const bytes = await response.arrayBuffer();
  return await WebAssembly.instantiate(bytes, go.importObject);
};
(async () => {
  try {
    self.importScripts(runtimeURL);
    if (typeof Go !== "function") {
      throw new Error("Go runtime was not registered by wasm_exec.js");
    }
    const go = new Go();
    const result = await instantiate(go);
    await go.run(result.instance);
  } catch (error) {
    postBootstrapError(error);
  }
})();
})();`
}

func workerMessageFromEvent(parseOp string, parseTarget string, parseArgs []js.Value) (WorkerMessage, error) {
	if len(parseArgs) == 0 {
		return WorkerMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("worker event payload is missing"))
	}
	parsePayload := parseArgs[0]
	if parsePayload.IsUndefined() || parsePayload.IsNull() {
		return WorkerMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("worker event payload is missing"))
	}
	parseData := parsePayload.Get("data")
	if parseData.IsUndefined() || parseData.IsNull() {
		return WorkerMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("worker message is missing data"))
	}
	parseValue, parseErr := jsValueToGo(parseOp, parseTarget, parseData)
	if parseErr != nil {
		return WorkerMessage{}, parseErr
	}
	parseMessage := workerMessageFromGo(parseValue)
	parseMessage.Ports = messagePortsFromValue(parseTarget, parsePayload.Get("ports"))
	return parseMessage, nil
}

func workerMessageFromGo(parseValue any) WorkerMessage {
	parseMessage := WorkerMessage{
		Phase:   "message",
		Payload: parseValue,
	}
	parseData, parseOk := parseValue.(map[string]any)
	if !parseOk {
		return parseMessage
	}
	if parseId := workerStringField(parseData, "id"); parseId != "" {
		parseMessage.ID = parseId
	}
	if parsePhase := workerStringField(parseData, "phase"); parsePhase != "" {
		parseMessage.Phase = parsePhase
	}
	if parseName := workerStringField(parseData, "name"); parseName != "" {
		parseMessage.Name = parseName
	} else if parseName2 := workerStringField(parseData, "type"); parseName2 != "" {
		parseMessage.Name = parseName2
	}
	if parsePayload, parseOk2 := parseData["payload"]; parseOk2 {
		parseMessage.Payload = parsePayload
	}
	if parseRemoteErr := workerStringField(parseData, "error"); parseRemoteErr != "" {
		parseMessage.Error = parseRemoteErr
	}
	return parseMessage
}

func messagePortMessageFromEvent(parseOp string, parseTarget string, parseArgs []js.Value) (MessagePortMessage, error) {
	if len(parseArgs) == 0 {
		return MessagePortMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("message port event payload is missing"))
	}
	parseEvent := parseArgs[0]
	if parseEvent.IsUndefined() || parseEvent.IsNull() {
		return MessagePortMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("message port event payload is missing"))
	}
	parseData := parseEvent.Get("data")
	if parseData.IsUndefined() || parseData.IsNull() {
		return MessagePortMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("message port message is missing data"))
	}
	parseValue, parseErr := jsValueToGo(parseOp, parseTarget, parseData)
	if parseErr != nil {
		return MessagePortMessage{}, parseErr
	}
	return MessagePortMessage{
		Payload: parseValue,
		Ports:   messagePortsFromValue(parseTarget, parseEvent.Get("ports")),
	}, nil
}

func messagePortsFromValue(parseTarget string, parsePorts js.Value) []MessagePort {
	if parsePorts.IsUndefined() || parsePorts.IsNull() {
		return nil
	}
	parseCount := parsePorts.Length()
	if parseCount == 0 {
		return nil
	}
	parseResolved := make([]MessagePort, 0, parseCount)
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parseRawPort := parsePorts.Index(parseIndex)
		if parseRawPort.IsUndefined() || parseRawPort.IsNull() {
			continue
		}
		parseResolved = append(parseResolved, newMessagePort(fmt.Sprintf("%s.port[%d]", parseTarget, parseIndex), parseRawPort))
	}
	return parseResolved
}

func workerStringField(parseData map[string]any, parseKey string) string {
	parseValue, parseOk := parseData[parseKey]
	if !parseOk {
		return ""
	}
	switch parseTyped := parseValue.(type) {
	case string:
		return parseTyped
	default:
		return fmt.Sprint(parseTyped)
	}
}

func workerRemoteEnvelopeError(parseMessage WorkerMessage) string {
	if strings.TrimSpace(parseMessage.Error) != "" {
		return parseMessage.Error
	}
	if parseSummary := strings.TrimSpace(fmt.Sprint(parseMessage.Payload)); parseSummary != "" && parseSummary != "<nil>" {
		return parseSummary
	}
	return "worker reported an error"
}

func workerRemoteErrorSummary(parseArgs []js.Value) string {
	if len(parseArgs) == 0 {
		return "worker reported an error"
	}
	if parseMessage := parseArgs[0].Get("message"); !parseMessage.IsUndefined() && !parseMessage.IsNull() {
		return strings.TrimSpace(parseMessage.String())
	}
	return strings.TrimSpace(jsValueSummary(parseArgs[0]))
}

func workerContextError(parseOp string, parseTarget string, parseErr error) error {
	if errors.Is(parseErr, context.DeadlineExceeded) {
		return wrapError(parseOp, parseTarget, CodeTimeout, parseErr)
	}
	return wrapError(parseOp, parseTarget, CodeCancelled, parseErr)
}
