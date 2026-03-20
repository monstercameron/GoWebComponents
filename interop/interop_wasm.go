//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// LocalStorage returns the browser localStorage wrapper.
func LocalStorage() (Storage, error) {
	return resolveStorage("localStorage")
}

// SessionStorage returns the browser sessionStorage wrapper.
func SessionStorage() (Storage, error) {
	return resolveStorage("sessionStorage")
}

func resolveStorage(name string) (Storage, error) {
	raw, err := globalProperty("Storage", name)
	if err != nil {
		return Storage{}, err
	}
	return Storage{
		getItem: func(key string) (string, bool, error) {
			value := raw.Call("getItem", key)
			if value.IsUndefined() || value.IsNull() {
				return "", false, nil
			}
			return value.String(), true, nil
		},
		getMany: func(keys []string) (map[string]string, error) {
			return storageGetMany(raw, keys), nil
		},
		setItem: func(key string, value string) error {
			raw.Call("setItem", key, value)
			return nil
		},
		removeItem: func(key string) error {
			raw.Call("removeItem", key)
			return nil
		},
		clear: func() error {
			raw.Call("clear")
			return nil
		},
		length: func() (int, error) {
			return raw.Get("length").Int(), nil
		},
		key: func(index int) (string, bool, error) {
			value := raw.Call("key", index)
			if value.IsUndefined() || value.IsNull() {
				return "", false, nil
			}
			return value.String(), true, nil
		},
	}, nil
}

func WindowLocation() (Location, error) {
	raw, err := globalPath("Location", "window", "location")
	if err != nil {
		return Location{}, err
	}
	return Location{
		href:     func() string { return raw.Get("href").String() },
		pathname: func() string { return raw.Get("pathname").String() },
		search:   func() string { return raw.Get("search").String() },
		hash:     func() string { return raw.Get("hash").String() },
		origin:   func() string { return raw.Get("origin").String() },
		assign: func(rawURL string) error {
			raw.Call("assign", rawURL)
			return nil
		},
		replace: func(rawURL string) error {
			raw.Call("replace", rawURL)
			return nil
		},
		reload: func() error {
			raw.Call("reload")
			return nil
		},
	}, nil
}

func WindowHistory() (History, error) {
	raw, err := globalPath("History", "window", "history")
	if err != nil {
		return History{}, err
	}
	return History{
		length: func() (int, error) {
			return raw.Get("length").Int(), nil
		},
		state: func() (any, error) {
			return jsValueToGo("History.State", "history.state", raw.Get("state"))
		},
		back: func() error {
			raw.Call("back")
			return nil
		},
		forward: func() error {
			raw.Call("forward")
			return nil
		},
		goDelta: func(delta int) error {
			raw.Call("go", delta)
			return nil
		},
		pushState: func(state any, title string, rawURL string) error {
			value, err := goValueToJS("History.PushState", "state", state)
			if err != nil {
				return err
			}
			raw.Call("pushState", value, title, rawURL)
			return nil
		},
		replaceState: func(state any, title string, rawURL string) error {
			value, err := goValueToJS("History.ReplaceState", "state", state)
			if err != nil {
				return err
			}
			raw.Call("replaceState", value, title, rawURL)
			return nil
		},
	}, nil
}

func NavigatorClipboard() (Clipboard, error) {
	raw, err := globalPath("Clipboard", "navigator", "clipboard")
	if err != nil {
		return Clipboard{}, err
	}
	return Clipboard{
		writeText: func(ctx context.Context, text string) error {
			_, err := awaitValue(ctx, "Clipboard.WriteText", "navigator.clipboard.writeText", raw.Call("writeText", text))
			return err
		},
		readText: func(ctx context.Context) (string, error) {
			value, err := awaitValue(ctx, "Clipboard.ReadText", "navigator.clipboard.readText", raw.Call("readText"))
			if err != nil {
				return "", err
			}
			if value.IsUndefined() || value.IsNull() {
				return "", nil
			}
			return value.String(), nil
		},
	}, nil
}

func SetTimeout(delay time.Duration, fn func()) (Timer, error) {
	if fn == nil {
		return Timer{}, wrapError("SetTimeout", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		callback js.Func
		once     sync.Once
		id       js.Value
	)
	callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		once.Do(func() {
			callback.Release()
		})
		fn()
		return nil
	})
	id = js.Global().Call("setTimeout", callback, durationMS(delay))
	return Timer{
		cancel: func() error {
			once.Do(func() {
				js.Global().Call("clearTimeout", id)
				callback.Release()
			})
			return nil
		},
	}, nil
}

func SetInterval(interval time.Duration, fn func()) (Timer, error) {
	if fn == nil {
		return Timer{}, wrapError("SetInterval", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		callback js.Func
		once     sync.Once
		id       js.Value
	)
	callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fn()
		return nil
	})
	id = js.Global().Call("setInterval", callback, durationMS(interval))
	return Timer{
		cancel: func() error {
			once.Do(func() {
				js.Global().Call("clearInterval", id)
				callback.Release()
			})
			return nil
		},
	}, nil
}

func WindowEvents() (EventTarget, error) {
	raw, err := globalProperty("EventTarget", "window")
	if err != nil {
		return EventTarget{}, err
	}
	return newEventTarget("window", raw), nil
}

func DocumentEvents() (EventTarget, error) {
	raw, err := globalProperty("EventTarget", "document")
	if err != nil {
		return EventTarget{}, err
	}
	return newEventTarget("document", raw), nil
}

func CurrentDocument() (Document, error) {
	raw, err := globalProperty("Document", "document")
	if err != nil {
		return Document{}, err
	}
	return Document{
		elementByID: func(id string) (Element, bool, error) {
			value := raw.Call("getElementById", id)
			if value.IsUndefined() || value.IsNull() {
				return Element{}, false, nil
			}
			return newElement("document.getElementById", value), true, nil
		},
		elementsByID: func(ids []string) (map[string]Element, error) {
			resolved := make(map[string]Element, len(ids))
			for id, value := range documentElementsByID(raw, ids) {
				resolved[id] = newElement("document.getElementById", value)
			}
			return resolved, nil
		},
		querySelector: func(selector string) (Element, bool, error) {
			value := raw.Call("querySelector", selector)
			if value.IsUndefined() || value.IsNull() {
				return Element{}, false, nil
			}
			return newElement("document.querySelector", value), true, nil
		},
	}, nil
}

var (
	storageGetManyHelperOnce sync.Once
	storageGetManyHelper     js.Value
	documentByIDHelperOnce   sync.Once
	documentByIDHelper       js.Value
)

func storageGetMany(raw js.Value, keys []string) map[string]string {
	storageGetManyHelperOnce.Do(func() {
		storageGetManyHelper = js.Global().Get("Function").New("storage", "keys", `
			const result = {};
			for (const key of keys) {
				const value = storage.getItem(key);
				if (value !== null && value !== undefined) {
					result[key] = value;
				}
			}
			return result;
		`)
	})
	values := storageGetManyHelper.Invoke(raw, stringArrayValue(keys))
	if values.IsUndefined() || values.IsNull() {
		return map[string]string{}
	}
	result := make(map[string]string, len(keys))
	objectKeys := js.Global().Get("Object").Call("keys", values)
	for index := 0; index < objectKeys.Get("length").Int(); index++ {
		key := objectKeys.Index(index).String()
		result[key] = values.Get(key).String()
	}
	return result
}

func documentElementsByID(raw js.Value, ids []string) map[string]js.Value {
	documentByIDHelperOnce.Do(func() {
		documentByIDHelper = js.Global().Get("Function").New("documentRef", "ids", `
			const result = {};
			for (const id of ids) {
				const value = documentRef.getElementById(id);
				if (value !== null && value !== undefined) {
					result[id] = value;
				}
			}
			return result;
		`)
	})
	values := documentByIDHelper.Invoke(raw, stringArrayValue(ids))
	if values.IsUndefined() || values.IsNull() {
		return map[string]js.Value{}
	}
	result := make(map[string]js.Value, len(ids))
	objectKeys := js.Global().Get("Object").Call("keys", values)
	for index := 0; index < objectKeys.Get("length").Int(); index++ {
		key := objectKeys.Index(index).String()
		result[key] = values.Get(key)
	}
	return result
}

func stringArrayValue(values []string) js.Value {
	array := js.Global().Get("Array").New()
	for _, value := range values {
		array.Call("push", value)
	}
	return array
}

func MatchMedia(query string) (MediaQueryList, error) {
	window, err := globalProperty("MatchMedia", "window")
	if err != nil {
		return MediaQueryList{}, err
	}
	raw := window.Call("matchMedia", query)
	if raw.IsUndefined() || raw.IsNull() {
		return MediaQueryList{}, unavailable("MatchMedia", query)
	}
	return MediaQueryList{
		matches: func() bool { return raw.Get("matches").Bool() },
		media:   func() string { return raw.Get("media").String() },
		subscribe: func(handler func(MediaQueryEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("MatchMedia.Subscribe", query, CodeInvalid, errors.New("handler is nil"))
			}
			listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				event := raw
				if len(args) > 0 && !args[0].IsUndefined() && !args[0].IsNull() {
					event = args[0]
				}
				handler(MediaQueryEvent{
					Matches: event.Get("matches").Bool(),
					Media:   event.Get("media").String(),
				})
				return nil
			})
			if add := raw.Get("addEventListener"); add.Type() == js.TypeFunction {
				raw.Call("addEventListener", "change", listener)
				return Subscription{cancel: func() {
					raw.Call("removeEventListener", "change", listener)
					listener.Release()
				}}, nil
			}
			raw.Call("addListener", listener)
			return Subscription{cancel: func() {
				raw.Call("removeListener", listener)
				listener.Release()
			}}, nil
		},
	}, nil
}

func ImportModule(ctx context.Context, specifier string) (Module, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(specifier) == "" {
		return Module{}, wrapError("ImportModule", specifier, CodeInvalid, errors.New("specifier is empty"))
	}
	importFn := js.Global().Get("__gwcImportModule")
	if importFn.IsUndefined() || importFn.IsNull() {
		constructor := js.Global().Get("Function")
		if constructor.IsUndefined() || constructor.IsNull() {
			return Module{}, unavailable("ImportModule", specifier)
		}
		importFn = constructor.New("specifier", "return import(specifier);")
	}
	rawModule, err := awaitValue(ctx, "ImportModule", specifier, importFn.Invoke(specifier))
	if err != nil {
		return Module{}, err
	}
	state := &moduleState{specifier: specifier, value: rawModule}
	return Module{
		call: func(ctx context.Context, export string, args ...any) (any, error) {
			raw, err := state.export(export)
			if err != nil {
				return nil, err
			}
			if raw.Type() != js.TypeFunction {
				return nil, wrapError("Module.Call", export, CodeNotFunction, errors.New("export is not a function"))
			}
			jsArgs := make([]interface{}, 0, len(args))
			for _, arg := range args {
				value, err := goValueToJS("Module.Call", export, arg)
				if err != nil {
					return nil, err
				}
				jsArgs = append(jsArgs, value)
			}
			result, err := awaitValue(ctx, "Module.Call", export, raw.Invoke(jsArgs...))
			if err != nil {
				return nil, err
			}
			return jsValueToGo("Module.Call", export, result)
		},
		callDefault: func(ctx context.Context, args ...any) (any, error) {
			return state.callDefault(ctx, args...)
		},
		value: func(ctx context.Context, export string) (any, error) {
			raw, err := state.export(export)
			if err != nil {
				return nil, err
			}
			resolved, err := awaitValue(ctx, "Module.Value", export, raw)
			if err != nil {
				return nil, err
			}
			return jsValueToGo("Module.Value", export, resolved)
		},
		dispose: state.dispose,
	}, nil
}

func OpenCrossTabChannel(options CrossTabChannelOptions) (CrossTabChannel, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return CrossTabChannel{}, wrapError("OpenCrossTabChannel", options.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	source := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
	if ctor := js.Global().Get("BroadcastChannel"); ctor.Type() == js.TypeFunction {
		return newBroadcastCrossTabChannel(name, source, ctor.New(name)), nil
	}
	return newStorageCrossTabChannel(name, source, resolveCrossTabStorageKey(name, options.StorageKey))
}

func OpenSecondaryWindowChannel(options WindowChannelOptions) (WindowChannel, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", options.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	rawWindow, err := globalProperty("Window", "window")
	if err != nil {
		return WindowChannel{}, err
	}
	openFn := rawWindow.Get("open")
	if openFn.Type() != js.TypeFunction {
		return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", name)
	}
	rawURL := strings.TrimSpace(options.URL)
	if rawURL == "" {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", name, CodeInvalid, errors.New("window URL is empty"))
	}
	raw := openFn.Invoke(rawURL, name, strings.TrimSpace(options.Features))
	if raw.IsUndefined() || raw.IsNull() {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", name, CodeUnavailable, errors.New("window.open returned no handle"))
	}
	return newWindowChannel(name, resolveWindowTargetOrigin(strings.TrimSpace(options.TargetOrigin)), raw, true), nil
}

func WindowOpenerChannel(options WindowChannelOptions) (WindowChannel, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return WindowChannel{}, wrapError("WindowOpenerChannel", options.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	rawWindow, err := globalProperty("Window", "window")
	if err != nil {
		return WindowChannel{}, err
	}
	opener := rawWindow.Get("opener")
	if opener.IsUndefined() || opener.IsNull() {
		return WindowChannel{}, unavailable("WindowOpenerChannel", name)
	}
	return newWindowChannel(name, resolveWindowTargetOrigin(strings.TrimSpace(options.TargetOrigin)), opener, false), nil
}

const defaultWorkerReadyTimeout = 5 * time.Second

type browserWorkerState struct {
	mu            sync.RWMutex
	options       WorkerOptions
	raw           js.Value
	active        bool
	nextRequestID int
}

func NewWorker(ctx context.Context, options WorkerOptions) (Worker, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(options.URL) == "" {
		return Worker{}, wrapError("NewWorker", options.URL, CodeInvalid, errors.New("worker URL is empty"))
	}
	state := &browserWorkerState{options: options}
	if err := state.start(ctx); err != nil {
		return Worker{}, err
	}
	return Worker{
		post: state.post,
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			return state.subscribe(handler)
		},
		request:   state.request,
		terminate: state.terminate,
		restart:   state.restart,
	}, nil
}

func (s *browserWorkerState) start(ctx context.Context) error {
	raw, err := createBrowserWorker(s.options)
	if err != nil {
		return err
	}
	if s.options.Ready {
		waitCtx := ctx
		if _, ok := waitCtx.Deadline(); !ok {
			timeout := s.options.ReadyTimeout
			if timeout <= 0 {
				timeout = defaultWorkerReadyTimeout
			}
			var cancel context.CancelFunc
			waitCtx, cancel = context.WithTimeout(waitCtx, timeout)
			defer cancel()
		}
		if err := waitWorkerReady(waitCtx, raw, s.options.URL); err != nil {
			raw.Call("terminate")
			return err
		}
	}
	s.mu.Lock()
	s.raw = raw
	s.active = true
	s.mu.Unlock()
	return nil
}

func (s *browserWorkerState) current(op string, target string) (js.Value, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.active || s.raw.IsUndefined() || s.raw.IsNull() {
		return js.Undefined(), wrapError(op, target, CodeDisposed, errors.New("worker is not active"))
	}
	return s.raw, nil
}

func (s *browserWorkerState) post(message any) error {
	raw, err := s.current("Worker.Post", s.options.URL)
	if err != nil {
		return err
	}
	value, err := goValueToJS("Worker.Post", s.options.URL, message)
	if err != nil {
		return err
	}
	raw.Call("postMessage", value)
	return nil
}

func (s *browserWorkerState) subscribe(handler func(WorkerMessage, error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("Worker.Subscribe", s.options.URL, CodeInvalid, errors.New("handler is nil"))
	}
	raw, err := s.current("Worker.Subscribe", s.options.URL)
	if err != nil {
		return Subscription{}, err
	}
	messageFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message, err := workerMessageFromEvent("Worker.Subscribe", s.options.URL, args)
		handler(message, err)
		return nil
	})
	errorFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler(WorkerMessage{}, wrapError("Worker.Subscribe", s.options.URL, CodeRemote, errors.New(workerRemoteErrorSummary(args))))
		return nil
	})
	raw.Call("addEventListener", "message", messageFn)
	raw.Call("addEventListener", "error", errorFn)
	return Subscription{cancel: func() {
		raw.Call("removeEventListener", "message", messageFn)
		raw.Call("removeEventListener", "error", errorFn)
		messageFn.Release()
		errorFn.Release()
	}}, nil
}

func (s *browserWorkerState) request(ctx context.Context, name string, payload any, onProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(name) == "" {
		return WorkerMessage{}, wrapError("Worker.Request", name, CodeInvalid, errors.New("request name is empty"))
	}
	raw, err := s.current("Worker.Request", name)
	if err != nil {
		return WorkerMessage{}, err
	}

	s.mu.Lock()
	s.nextRequestID++
	requestID := fmt.Sprintf("worker-%d", s.nextRequestID)
	s.mu.Unlock()

	resultCh := make(chan WorkerMessage, 1)
	errCh := make(chan error, 1)
	messageFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message, decodeErr := workerMessageFromEvent("Worker.Request", name, args)
		if decodeErr != nil {
			if onProgress != nil {
				onProgress(WorkerMessage{}, decodeErr)
			}
			return nil
		}
		if strings.TrimSpace(message.ID) != requestID {
			return nil
		}
		switch strings.TrimSpace(message.Phase) {
		case "progress":
			if onProgress != nil {
				onProgress(message, nil)
			}
		case "error":
			errCh <- wrapError("Worker.Request", name, CodeRemote, errors.New(workerRemoteEnvelopeError(message)))
		case "result", "message", "":
			resultCh <- message
		}
		return nil
	})
	errorFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errCh <- wrapError("Worker.Request", name, CodeRemote, errors.New(workerRemoteErrorSummary(args)))
		return nil
	})
	raw.Call("addEventListener", "message", messageFn)
	raw.Call("addEventListener", "error", errorFn)
	defer func() {
		raw.Call("removeEventListener", "message", messageFn)
		raw.Call("removeEventListener", "error", errorFn)
		messageFn.Release()
		errorFn.Release()
	}()

	if err := s.post(WorkerMessage{
		ID:      requestID,
		Phase:   "request",
		Name:    name,
		Payload: payload,
	}); err != nil {
		return WorkerMessage{}, err
	}

	select {
	case message := <-resultCh:
		return message, nil
	case err := <-errCh:
		return WorkerMessage{}, err
	case <-ctx.Done():
		return WorkerMessage{}, workerContextError("Worker.Request", name, ctx.Err())
	}
}

func (s *browserWorkerState) terminate() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active || s.raw.IsUndefined() || s.raw.IsNull() {
		return wrapError("Worker.Terminate", s.options.URL, CodeDisposed, errors.New("worker is not active"))
	}
	s.raw.Call("terminate")
	s.raw = js.Undefined()
	s.active = false
	return nil
}

func (s *browserWorkerState) restart(ctx context.Context) error {
	s.mu.Lock()
	raw := s.raw
	active := s.active
	s.raw = js.Undefined()
	s.active = false
	s.mu.Unlock()
	if active && !raw.IsUndefined() && !raw.IsNull() {
		raw.Call("terminate")
	}
	return s.start(ctx)
}

func createBrowserWorker(options WorkerOptions) (js.Value, error) {
	ctor, err := globalProperty("Worker", "Worker")
	if err != nil {
		return js.Undefined(), err
	}
	workerType := strings.TrimSpace(options.Type)
	if workerType != "" && workerType != "classic" && workerType != "module" {
		return js.Undefined(), wrapError("NewWorker", options.URL, CodeInvalid, errors.New("worker type must be classic or module"))
	}
	if workerType == "" && strings.HasSuffix(strings.ToLower(strings.TrimSpace(options.URL)), ".mjs") {
		workerType = "module"
	}
	if strings.TrimSpace(options.Name) == "" && workerType == "" {
		return ctor.New(options.URL), nil
	}
	init := js.Global().Get("Object").New()
	if strings.TrimSpace(options.Name) != "" {
		init.Set("name", options.Name)
	}
	if workerType != "" {
		init.Set("type", workerType)
	}
	return ctor.New(options.URL, init), nil
}

func waitWorkerReady(ctx context.Context, raw js.Value, target string) error {
	readyCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	messageFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message, err := workerMessageFromEvent("NewWorker", target, args)
		if err != nil {
			errCh <- err
			return nil
		}
		if strings.TrimSpace(message.Phase) == "ready" {
			readyCh <- struct{}{}
		}
		return nil
	})
	errorFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errCh <- wrapError("NewWorker", target, CodeRemote, errors.New(workerRemoteErrorSummary(args)))
		return nil
	})
	raw.Call("addEventListener", "message", messageFn)
	raw.Call("addEventListener", "error", errorFn)
	defer func() {
		raw.Call("removeEventListener", "message", messageFn)
		raw.Call("removeEventListener", "error", errorFn)
		messageFn.Release()
		errorFn.Release()
	}()
	select {
	case <-readyCh:
		return nil
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return workerContextError("NewWorker", target, ctx.Err())
	}
}

func workerMessageFromEvent(op string, target string, args []js.Value) (WorkerMessage, error) {
	if len(args) == 0 {
		return WorkerMessage{}, wrapError(op, target, CodeDecode, errors.New("worker event payload is missing"))
	}
	payload := args[0]
	if payload.IsUndefined() || payload.IsNull() {
		return WorkerMessage{}, wrapError(op, target, CodeDecode, errors.New("worker event payload is missing"))
	}
	data := payload.Get("data")
	if data.IsUndefined() || data.IsNull() {
		return WorkerMessage{}, wrapError(op, target, CodeDecode, errors.New("worker message is missing data"))
	}
	value, err := jsValueToGo(op, target, data)
	if err != nil {
		return WorkerMessage{}, err
	}
	return workerMessageFromGo(value), nil
}

func workerMessageFromGo(value any) WorkerMessage {
	message := WorkerMessage{
		Phase:   "message",
		Payload: value,
	}
	data, ok := value.(map[string]any)
	if !ok {
		return message
	}
	if id := workerStringField(data, "id"); id != "" {
		message.ID = id
	}
	if phase := workerStringField(data, "phase"); phase != "" {
		message.Phase = phase
	}
	if name := workerStringField(data, "name"); name != "" {
		message.Name = name
	} else if name := workerStringField(data, "type"); name != "" {
		message.Name = name
	}
	if payload, ok := data["payload"]; ok {
		message.Payload = payload
	}
	if remoteErr := workerStringField(data, "error"); remoteErr != "" {
		message.Error = remoteErr
	}
	return message
}

func workerStringField(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func workerRemoteEnvelopeError(message WorkerMessage) string {
	if strings.TrimSpace(message.Error) != "" {
		return message.Error
	}
	if summary := strings.TrimSpace(fmt.Sprint(message.Payload)); summary != "" && summary != "<nil>" {
		return summary
	}
	return "worker reported an error"
}

func workerRemoteErrorSummary(args []js.Value) string {
	if len(args) == 0 {
		return "worker reported an error"
	}
	if message := args[0].Get("message"); !message.IsUndefined() && !message.IsNull() {
		return strings.TrimSpace(message.String())
	}
	return strings.TrimSpace(jsValueSummary(args[0]))
}

func workerContextError(op string, target string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return wrapError(op, target, CodeTimeout, err)
	}
	return wrapError(op, target, CodeCancelled, err)
}

func newBroadcastCrossTabChannel(name string, source string, raw js.Value) CrossTabChannel {
	var (
		mu       sync.Mutex
		sequence int64
		active   = true
	)
	nextEnvelope := func(payload any) CrossTabEnvelope {
		mu.Lock()
		defer mu.Unlock()
		sequence++
		return CrossTabEnvelope{
			Name:     name,
			Payload:  payload,
			Source:   source,
			Sequence: sequence,
			SentAt:   time.Now().UTC(),
		}
	}
	current := func(op string) (js.Value, error) {
		mu.Lock()
		defer mu.Unlock()
		if !active || raw.IsUndefined() || raw.IsNull() {
			return js.Undefined(), wrapError(op, name, CodeDisposed, errors.New("cross-tab channel is closed"))
		}
		return raw, nil
	}
	return CrossTabChannel{
		name:      func() string { return name },
		transport: func() string { return "broadcast-channel" },
		publish: func(payload any) error {
			target, err := current("CrossTabChannel.Publish")
			if err != nil {
				return err
			}
			value, err := goValueToJS("CrossTabChannel.Publish", name, nextEnvelope(payload))
			if err != nil {
				return err
			}
			target.Call("postMessage", value)
			return nil
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("CrossTabChannel.Subscribe", name, CodeInvalid, errors.New("handler is nil"))
			}
			target, err := current("CrossTabChannel.Subscribe")
			if err != nil {
				return Subscription{}, err
			}
			listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				if len(args) == 0 {
					handler(CrossTabEnvelope{Name: name}, wrapError("CrossTabChannel.Subscribe", name, CodeDecode, errors.New("broadcast message event is missing")))
					return nil
				}
				data := args[0].Get("data")
				value, decodeErr := jsValueToGo("CrossTabChannel.Subscribe", name, data)
				if decodeErr != nil {
					handler(CrossTabEnvelope{Name: name}, decodeErr)
					return nil
				}
				handler(crossTabEnvelopeFromGo(value, name), nil)
				return nil
			})
			target.Call("addEventListener", "message", listener)
			return Subscription{cancel: func() {
				target.Call("removeEventListener", "message", listener)
				listener.Release()
			}}, nil
		},
		close: func() error {
			target, err := current("CrossTabChannel.Close")
			if err != nil {
				return err
			}
			target.Call("close")
			mu.Lock()
			active = false
			raw = js.Undefined()
			mu.Unlock()
			return nil
		},
	}
}

func newStorageCrossTabChannel(name string, source string, storageKey string) (CrossTabChannel, error) {
	storage, err := LocalStorage()
	if err != nil {
		return CrossTabChannel{}, err
	}
	window, err := globalProperty("EventTarget", "window")
	if err != nil {
		return CrossTabChannel{}, err
	}
	var (
		mu       sync.Mutex
		sequence int64
		active   = true
	)
	nextEnvelope := func(payload any) CrossTabEnvelope {
		mu.Lock()
		defer mu.Unlock()
		sequence++
		return CrossTabEnvelope{
			Name:     name,
			Payload:  payload,
			Source:   source,
			Sequence: sequence,
			SentAt:   time.Now().UTC(),
		}
	}
	ensureActive := func(op string) error {
		mu.Lock()
		defer mu.Unlock()
		if !active {
			return wrapError(op, name, CodeDisposed, errors.New("cross-tab channel is closed"))
		}
		return nil
	}
	return CrossTabChannel{
		name:      func() string { return name },
		transport: func() string { return "storage-event" },
		publish: func(payload any) error {
			if err := ensureActive("CrossTabChannel.Publish"); err != nil {
				return err
			}
			data, err := json.Marshal(nextEnvelope(payload))
			if err != nil {
				return wrapError("CrossTabChannel.Publish", name, CodeEncode, err)
			}
			if err := storage.SetItem(storageKey, string(data)); err != nil {
				return err
			}
			return storage.RemoveItem(storageKey)
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("CrossTabChannel.Subscribe", name, CodeInvalid, errors.New("handler is nil"))
			}
			if err := ensureActive("CrossTabChannel.Subscribe"); err != nil {
				return Subscription{}, err
			}
			listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				if len(args) == 0 {
					return nil
				}
				event := args[0]
				if event.IsUndefined() || event.IsNull() {
					return nil
				}
				if event.Get("key").String() != storageKey {
					return nil
				}
				newValue := event.Get("newValue")
				if newValue.IsUndefined() || newValue.IsNull() || strings.TrimSpace(newValue.String()) == "" {
					return nil
				}
				var message CrossTabEnvelope
				if err := json.Unmarshal([]byte(newValue.String()), &message); err != nil {
					handler(CrossTabEnvelope{Name: name}, wrapError("CrossTabChannel.Subscribe", name, CodeDecode, err))
					return nil
				}
				if strings.TrimSpace(message.Name) == "" {
					message.Name = name
				}
				handler(message, nil)
				return nil
			})
			window.Call("addEventListener", "storage", listener)
			return Subscription{cancel: func() {
				window.Call("removeEventListener", "storage", listener)
				listener.Release()
			}}, nil
		},
		close: func() error {
			if err := ensureActive("CrossTabChannel.Close"); err != nil {
				return err
			}
			mu.Lock()
			active = false
			mu.Unlock()
			return nil
		},
	}, nil
}

func newWindowChannel(name string, targetOrigin string, peer js.Value, allowClose bool) WindowChannel {
	source := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
	rawWindow := js.Global().Get("window")
	return WindowChannel{
		name:         func() string { return name },
		targetOrigin: func() string { return targetOrigin },
		publish: func(payload any) error {
			if peer.IsUndefined() || peer.IsNull() {
				return wrapError("WindowChannel.Publish", name, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if closed := peer.Get("closed"); !closed.IsUndefined() && !closed.IsNull() && closed.Bool() {
				return wrapError("WindowChannel.Publish", name, CodeDisposed, errors.New("window channel peer is closed"))
			}
			value, err := goValueToJS("WindowChannel.Publish", name, WindowEnvelope{
				Name:    name,
				Payload: payload,
				Source:  source,
				SentAt:  time.Now().UTC(),
			})
			if err != nil {
				return err
			}
			peer.Call("postMessage", value, targetOrigin)
			return nil
		},
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("WindowChannel.Subscribe", name, CodeInvalid, errors.New("handler is nil"))
			}
			listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				if len(args) == 0 {
					handler(WindowEnvelope{Name: name}, wrapError("WindowChannel.Subscribe", name, CodeDecode, errors.New("message event is missing")))
					return nil
				}
				event := args[0]
				if event.IsUndefined() || event.IsNull() {
					return nil
				}
				eventSource := event.Get("source")
				if !eventSource.IsUndefined() && !eventSource.IsNull() && !eventSource.Equal(peer) {
					return nil
				}
				if targetOrigin != "*" {
					origin := strings.TrimSpace(event.Get("origin").String())
					if origin != "" && origin != targetOrigin {
						return nil
					}
				}
				value, decodeErr := jsValueToGo("WindowChannel.Subscribe", name, event.Get("data"))
				if decodeErr != nil {
					handler(WindowEnvelope{Name: name}, decodeErr)
					return nil
				}
				handler(windowEnvelopeFromGo(value, name), nil)
				return nil
			})
			rawWindow.Call("addEventListener", "message", listener)
			return Subscription{cancel: func() {
				rawWindow.Call("removeEventListener", "message", listener)
				listener.Release()
			}}, nil
		},
		focus: func() error {
			if peer.IsUndefined() || peer.IsNull() {
				return wrapError("WindowChannel.Focus", name, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if fn := peer.Get("focus"); fn.Type() != js.TypeFunction {
				return unavailable("WindowChannel.Focus", name)
			}
			peer.Call("focus")
			return nil
		},
		close: func() error {
			if !allowClose {
				return unavailable("WindowChannel.Close", name)
			}
			if peer.IsUndefined() || peer.IsNull() {
				return wrapError("WindowChannel.Close", name, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if fn := peer.Get("close"); fn.Type() != js.TypeFunction {
				return unavailable("WindowChannel.Close", name)
			}
			peer.Call("close")
			return nil
		},
		closed: func() bool {
			if peer.IsUndefined() || peer.IsNull() {
				return true
			}
			closed := peer.Get("closed")
			if closed.IsUndefined() || closed.IsNull() {
				return false
			}
			return closed.Bool()
		},
	}
}

func resolveCrossTabStorageKey(name string, override string) string {
	trimmed := strings.TrimSpace(override)
	if trimmed != "" {
		return trimmed
	}
	return "__gwc_cross_tab__:" + name
}

func resolveWindowTargetOrigin(rawTargetOrigin string) string {
	trimmed := strings.TrimSpace(rawTargetOrigin)
	if trimmed != "" {
		return trimmed
	}
	window := js.Global().Get("window")
	if window.IsUndefined() || window.IsNull() {
		return "*"
	}
	location := window.Get("location")
	if location.IsUndefined() || location.IsNull() {
		return "*"
	}
	origin := strings.TrimSpace(location.Get("origin").String())
	if origin == "" {
		return "*"
	}
	return origin
}

func crossTabEnvelopeFromGo(value any, fallbackName string) CrossTabEnvelope {
	message := CrossTabEnvelope{
		Name:    fallbackName,
		Payload: value,
	}
	data, ok := value.(map[string]any)
	if !ok {
		return message
	}
	if name := workerStringField(data, "name"); name != "" {
		message.Name = name
	}
	if payload, ok := data["payload"]; ok {
		message.Payload = payload
	}
	if source := workerStringField(data, "source"); source != "" {
		message.Source = source
	}
	if sequence, ok := crossTabInt64Field(data["sequence"]); ok {
		message.Sequence = sequence
	}
	if sentAt, ok := crossTabTimeField(data["sentAt"]); ok {
		message.SentAt = sentAt
	}
	return message
}

func windowEnvelopeFromGo(value any, fallbackName string) WindowEnvelope {
	message := WindowEnvelope{
		Name:    fallbackName,
		Payload: value,
	}
	data, ok := value.(map[string]any)
	if !ok {
		return message
	}
	if name := workerStringField(data, "name"); name != "" {
		message.Name = name
	}
	if payload, ok := data["payload"]; ok {
		message.Payload = payload
	}
	if source := workerStringField(data, "source"); source != "" {
		message.Source = source
	}
	if sentAt, ok := crossTabTimeField(data["sentAt"]); ok {
		message.SentAt = sentAt
	}
	return message
}

func crossTabInt64Field(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case float32:
		return int64(typed), true
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func crossTabTimeField(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

type moduleState struct {
	mu        sync.RWMutex
	specifier string
	value     js.Value
	disposed  bool
}

func (m *moduleState) export(name string) (js.Value, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.disposed {
		return js.Undefined(), wrapError("Module", m.specifier, CodeDisposed, errors.New("module handle is disposed"))
	}
	raw := m.value.Get(name)
	if raw.IsUndefined() || raw.IsNull() {
		return js.Undefined(), wrapError("Module.Value", name, CodeMissingExport, errors.New("export not found"))
	}
	return raw, nil
}

func (m *moduleState) callDefault(ctx context.Context, args ...any) (any, error) {
	raw, err := m.export("default")
	if err != nil {
		return nil, err
	}
	if raw.Type() != js.TypeFunction {
		return nil, wrapError("Module.CallDefault", "default", CodeNotFunction, errors.New("default export is not a function"))
	}
	jsArgs := make([]interface{}, 0, len(args))
	for _, arg := range args {
		value, err := goValueToJS("Module.CallDefault", "default", arg)
		if err != nil {
			return nil, err
		}
		jsArgs = append(jsArgs, value)
	}
	result, err := awaitValue(ctx, "Module.CallDefault", "default", raw.Invoke(jsArgs...))
	if err != nil {
		return nil, err
	}
	return jsValueToGo("Module.CallDefault", "default", result)
}

func (m *moduleState) dispose() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disposed = true
	m.value = js.Undefined()
	return nil
}

func newEventTarget(name string, raw js.Value) EventTarget {
	return EventTarget{
		dispatch: func(eventName string, detail any) error {
			customEventCtor := js.Global().Get("CustomEvent")
			if customEventCtor.IsUndefined() || customEventCtor.IsNull() {
				return unavailable("EventTarget.Dispatch", name)
			}
			detailValue, err := goValueToJS("EventTarget.Dispatch", eventName, detail)
			if err != nil {
				return err
			}
			init := js.Global().Get("Object").New()
			init.Set("detail", detailValue)
			raw.Call("dispatchEvent", customEventCtor.New(eventName, init))
			return nil
		},
		listen: func(eventName string, handler func(BrowserEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("EventTarget.Listen", eventName, CodeInvalid, errors.New("handler is nil"))
			}
			if add := raw.Get("addEventListener"); add.Type() != js.TypeFunction {
				return Subscription{}, unavailable("EventTarget.Listen", name)
			}
			listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				if len(args) == 0 {
					handler(BrowserEvent{Type: eventName})
					return nil
				}
				event := args[0]
				detail, _ := jsValueToGo("EventTarget.Listen", eventName, event.Get("detail"))
				handler(BrowserEvent{
					Type:          event.Get("type").String(),
					Detail:        detail,
					Target:        elementFromJSValue(event.Get("target")),
					CurrentTarget: elementFromJSValue(event.Get("currentTarget")),
				})
				return nil
			})
			raw.Call("addEventListener", eventName, listener)
			return Subscription{cancel: func() {
				raw.Call("removeEventListener", eventName, listener)
				listener.Release()
			}}, nil
		},
	}
}

func newElement(name string, raw js.Value) Element {
	return Element{
		raw:       raw,
		tagName:   func() string { return raw.Get("tagName").String() },
		id:        func() string { return raw.Get("id").String() },
		className: func() string { return raw.Get("className").String() },
		focus: func() error {
			if fn := raw.Get("focus"); fn.Type() != js.TypeFunction {
				return unavailable("Element.Focus", name)
			}
			raw.Call("focus")
			return nil
		},
		blur: func() error {
			if fn := raw.Get("blur"); fn.Type() != js.TypeFunction {
				return unavailable("Element.Blur", name)
			}
			raw.Call("blur")
			return nil
		},
		click: func() error {
			if fn := raw.Get("click"); fn.Type() != js.TypeFunction {
				return unavailable("Element.Click", name)
			}
			raw.Call("click")
			return nil
		},
		scrollIntoView: func(options ScrollIntoViewOptions) error {
			if fn := raw.Get("scrollIntoView"); fn.Type() != js.TypeFunction {
				return unavailable("Element.ScrollIntoView", name)
			}
			init := js.Global().Get("Object").New()
			hasOptions := false
			if strings.TrimSpace(options.Behavior) != "" {
				init.Set("behavior", options.Behavior)
				hasOptions = true
			}
			if strings.TrimSpace(options.Block) != "" {
				init.Set("block", options.Block)
				hasOptions = true
			}
			if strings.TrimSpace(options.Inline) != "" {
				init.Set("inline", options.Inline)
				hasOptions = true
			}
			if !hasOptions {
				raw.Call("scrollIntoView")
				return nil
			}
			raw.Call("scrollIntoView", init)
			return nil
		},
		boundingClientRect: func() (Rect, error) {
			if fn := raw.Get("getBoundingClientRect"); fn.Type() != js.TypeFunction {
				return Rect{}, unavailable("Element.BoundingClientRect", name)
			}
			return rectFromJSValue(raw.Call("getBoundingClientRect")), nil
		},
		events: func() (EventTarget, error) {
			return newEventTarget(name, raw), nil
		},
		observeResize: func(handler func(ResizeEntry)) (Subscription, error) {
			return observeResize(name, raw, handler)
		},
		observeIntersection: func(options IntersectionObserverOptions, handler func(IntersectionEntry)) (Subscription, error) {
			return observeIntersection(name, raw, options, handler)
		},
	}
}

func elementFromJSValue(value js.Value) Element {
	if value.IsUndefined() || value.IsNull() {
		return Element{}
	}
	return newElement("element", value)
}

func rectFromJSValue(value js.Value) Rect {
	if value.IsUndefined() || value.IsNull() {
		return Rect{}
	}
	return Rect{
		X:      value.Get("x").Float(),
		Y:      value.Get("y").Float(),
		Width:  value.Get("width").Float(),
		Height: value.Get("height").Float(),
		Top:    value.Get("top").Float(),
		Right:  value.Get("right").Float(),
		Bottom: value.Get("bottom").Float(),
		Left:   value.Get("left").Float(),
	}
}

func observeResize(name string, raw js.Value, handler func(ResizeEntry)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("Element.ObserveResize", name, CodeInvalid, errors.New("handler is nil"))
	}
	ctor := js.Global().Get("ResizeObserver")
	if ctor.IsUndefined() || ctor.IsNull() {
		return Subscription{}, unavailable("Element.ObserveResize", name)
	}
	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		entries := args[0]
		length := entries.Get("length").Int()
		for index := 0; index < length; index++ {
			entry := entries.Index(index)
			handler(ResizeEntry{
				Target:      elementFromJSValue(entry.Get("target")),
				ContentRect: rectFromJSValue(entry.Get("contentRect")),
			})
		}
		return nil
	})
	observer := ctor.New(callback)
	observer.Call("observe", raw)
	return Subscription{cancel: func() {
		observer.Call("disconnect")
		callback.Release()
	}}, nil
}

func observeIntersection(name string, raw js.Value, options IntersectionObserverOptions, handler func(IntersectionEntry)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("Element.ObserveIntersection", name, CodeInvalid, errors.New("handler is nil"))
	}
	ctor := js.Global().Get("IntersectionObserver")
	if ctor.IsUndefined() || ctor.IsNull() {
		return Subscription{}, unavailable("Element.ObserveIntersection", name)
	}
	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		entries := args[0]
		length := entries.Get("length").Int()
		for index := 0; index < length; index++ {
			entry := entries.Index(index)
			var rootBounds *Rect
			if bounds := entry.Get("rootBounds"); !bounds.IsUndefined() && !bounds.IsNull() {
				rect := rectFromJSValue(bounds)
				rootBounds = &rect
			}
			handler(IntersectionEntry{
				Target:             elementFromJSValue(entry.Get("target")),
				IsIntersecting:     entry.Get("isIntersecting").Bool(),
				IntersectionRatio:  entry.Get("intersectionRatio").Float(),
				BoundingClientRect: rectFromJSValue(entry.Get("boundingClientRect")),
				IntersectionRect:   rectFromJSValue(entry.Get("intersectionRect")),
				RootBounds:         rootBounds,
			})
		}
		return nil
	})
	if len(options.Thresholds) == 0 && options.RootMargin == "" && options.Root.raw == nil {
		observer := ctor.New(callback)
		observer.Call("observe", raw)
		return Subscription{cancel: func() {
			observer.Call("disconnect")
			callback.Release()
		}}, nil
	}
	init := js.Global().Get("Object").New()
	if options.RootMargin != "" {
		init.Set("rootMargin", options.RootMargin)
	}
	if len(options.Thresholds) > 0 {
		thresholds := js.Global().Get("Array").New()
		for _, threshold := range options.Thresholds {
			thresholds.Call("push", threshold)
		}
		init.Set("threshold", thresholds)
	}
	if root, ok := options.Root.raw.(js.Value); ok && !root.IsUndefined() && !root.IsNull() {
		init.Set("root", root)
	}
	observer := ctor.New(callback, init)
	observer.Call("observe", raw)
	return Subscription{cancel: func() {
		observer.Call("disconnect")
		callback.Release()
	}}, nil
}

func globalProperty(op string, name string) (js.Value, error) {
	value := js.Global().Get(name)
	if (value.IsUndefined() || value.IsNull()) && name != "window" {
		if window := js.Global().Get("window"); !window.IsUndefined() && !window.IsNull() {
			value = window.Get(name)
		}
	}
	if value.IsUndefined() || value.IsNull() {
		return js.Undefined(), unavailable(op, name)
	}
	return value, nil
}

func globalPath(op string, head string, tail string) (js.Value, error) {
	root, err := globalProperty(op, head)
	if err != nil {
		return js.Undefined(), err
	}
	value := root.Get(tail)
	if (value.IsUndefined() || value.IsNull()) && head != "window" {
		if window := js.Global().Get("window"); !window.IsUndefined() && !window.IsNull() {
			nextRoot := window.Get(head)
			if !nextRoot.IsUndefined() && !nextRoot.IsNull() {
				value = nextRoot.Get(tail)
			}
		}
	}
	if value.IsUndefined() || value.IsNull() {
		return js.Undefined(), unavailable(op, head+"."+tail)
	}
	return value, nil
}

func durationMS(value time.Duration) int {
	if value <= 0 {
		return 0
	}
	return int(value / time.Millisecond)
}

func awaitValue(ctx context.Context, op string, target string, value js.Value) (js.Value, error) {
	if !isPromise(value) {
		return value, nil
	}
	resolvedCh := make(chan js.Value, 1)
	rejectedCh := make(chan js.Value, 1)
	var (
		resolve js.Func
		reject  js.Func
		once    sync.Once
	)
	cleanup := func() {
		resolve.Release()
		reject.Release()
	}
	resolve = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		once.Do(func() {
			if len(args) > 0 {
				resolvedCh <- args[0]
			} else {
				resolvedCh <- js.Undefined()
			}
			cleanup()
		})
		return nil
	})
	reject = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		once.Do(func() {
			if len(args) > 0 {
				rejectedCh <- args[0]
			} else {
				rejectedCh <- js.ValueOf("promise rejected")
			}
			cleanup()
		})
		return nil
	})
	value.Call("then", resolve)
	value.Call("catch", reject)

	select {
	case resolved := <-resolvedCh:
		return resolved, nil
	case rejected := <-rejectedCh:
		return js.Undefined(), wrapError(op, target, CodePromiseRejected, errors.New(jsValueSummary(rejected)))
	case <-ctx.Done():
		return js.Undefined(), wrapError(op, target, CodePromiseRejected, ctx.Err())
	}
}

func isPromise(value js.Value) bool {
	if value.IsUndefined() || value.IsNull() {
		return false
	}
	if value.Type() != js.TypeObject && value.Type() != js.TypeFunction {
		return false
	}
	then := value.Get("then")
	return then.Type() == js.TypeFunction
}

func goValueToJS(op string, target string, value any) (interface{}, error) {
	switch typed := value.(type) {
	case nil:
		return js.Null(), nil
	case Value:
		if raw, ok := typed.rawValue(); ok {
			return raw, nil
		}
		return js.Undefined(), unavailable(op, target)
	case js.Value:
		return typed, nil
	case bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return js.ValueOf(typed), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, wrapError(op, target, CodeEncode, err)
		}
		return js.Global().Get("JSON").Call("parse", string(data)), nil
	}
}

func goValuesToJS(op string, target string, values ...any) ([]interface{}, error) {
	if len(values) == 0 {
		return nil, nil
	}
	converted := make([]interface{}, len(values))
	for index, value := range values {
		jsValue, err := goValueToJS(op, target, value)
		if err != nil {
			return nil, err
		}
		converted[index] = jsValue
	}
	return converted, nil
}

func jsValueToGo(op string, target string, value js.Value) (any, error) {
	if value.IsUndefined() || value.IsNull() {
		return nil, nil
	}
	switch value.Type() {
	case js.TypeBoolean:
		return value.Bool(), nil
	case js.TypeString:
		return value.String(), nil
	case js.TypeNumber:
		return value.Float(), nil
	case js.TypeObject:
		stringified := js.Global().Get("JSON").Call("stringify", value)
		if stringified.IsUndefined() || stringified.IsNull() {
			return nil, wrapError(op, target, CodeDecode, errors.New("value is not serializable"))
		}
		var decoded any
		if err := json.Unmarshal([]byte(stringified.String()), &decoded); err != nil {
			return nil, wrapError(op, target, CodeDecode, err)
		}
		return decoded, nil
	case js.TypeFunction:
		return nil, wrapError(op, target, CodeDecode, errors.New("function values are not serializable"))
	default:
		return value.String(), nil
	}
}

func jsValueSummary(value js.Value) string {
	if value.IsUndefined() {
		return "undefined"
	}
	if value.IsNull() {
		return "null"
	}
	switch value.Type() {
	case js.TypeString:
		return value.String()
	case js.TypeBoolean:
		if value.Bool() {
			return "true"
		}
		return "false"
	case js.TypeNumber:
		return fmt.Sprint(value.Float())
	default:
		stringified := js.Global().Get("JSON").Call("stringify", value)
		if stringified.IsUndefined() || stringified.IsNull() {
			return value.String()
		}
		return stringified.String()
	}
}
