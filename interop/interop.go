package interop

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type ErrorCode string

const (
	CodeUnavailable     ErrorCode = "unavailable"
	CodePromiseRejected ErrorCode = "promise_rejected"
	CodeDisposed        ErrorCode = "disposed"
	CodeEncode          ErrorCode = "encode"
	CodeDecode          ErrorCode = "decode"
	CodeMissingExport   ErrorCode = "missing_export"
	CodeNotFunction     ErrorCode = "not_function"
	CodeInvalid         ErrorCode = "invalid"
	CodeBlocked         ErrorCode = "blocked"
	CodeQuotaExceeded   ErrorCode = "quota_exceeded"
	CodeUnauthorized    ErrorCode = "unauthorized"
	CodeCancelled       ErrorCode = "cancelled"
	CodeTimeout         ErrorCode = "timeout"
	CodeRemote          ErrorCode = "remote_error"
)

// Error reports a structured interop failure.
type Error struct {
	Op     string
	Target string
	Code   ErrorCode
	Err    error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	base := "interop failure"
	if e.Op != "" {
		base = e.Op
	}
	if e.Target != "" {
		base += " " + e.Target
	}
	if e.Code != "" {
		base += " [" + string(e.Code) + "]"
	}
	message := base
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	if docs, remediation := interopActionableGuidance(e.Code); docs != "" {
		if remediation != "" {
			message += ". " + remediation
		}
		message += ". See " + docs
	}
	return message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapError(op, target string, code ErrorCode, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Target: target, Code: code, Err: err}
}

func unavailable(op, target string) error {
	return &Error{Op: op, Target: target, Code: CodeUnavailable, Err: errors.New("browser interop is unavailable in this build")}
}

// Decode projects JSON-shaped interop payloads into a typed target.
func Decode(value any, target interface{}) error {
	if target == nil {
		return wrapError("Decode", "", CodeInvalid, errors.New("target is nil"))
	}
	data, err := json.Marshal(value)
	if err != nil {
		return wrapError("Decode", "", CodeEncode, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return wrapError("Decode", "", CodeDecode, err)
	}
	return nil
}

// IsCode reports whether err is an interop error with the provided code.
func IsCode(err error, code ErrorCode) bool {
	var interopErr *Error
	if !errors.As(err, &interopErr) {
		return false
	}
	return interopErr.Code == code
}

// AsError unwraps an interop error into the structured Error form.
func AsError(err error) (*Error, bool) {
	var interopErr *Error
	if !errors.As(err, &interopErr) {
		return nil, false
	}
	return interopErr, true
}

// CodeOf returns the interop error code for err when available.
func CodeOf(err error) (ErrorCode, bool) {
	interopErr, ok := AsError(err)
	if !ok {
		return "", false
	}
	return interopErr.Code, true
}

func interopActionableGuidance(code ErrorCode) (string, string) {
	switch code {
	case CodeInvalid:
		return "ACTIONABLE_ERRORS.md#gwc-interop-invalid", "Validate required names, URLs, and callbacks before creating the interop binding"
	case CodeBlocked:
		return "ACTIONABLE_ERRORS.md#gwc-interop-persistent-store-blocked", "Close older tabs, workers, or windows that still hold the same IndexedDB database open, then retry the upgrade"
	case CodeQuotaExceeded:
		return "ACTIONABLE_ERRORS.md#gwc-interop-persistent-store-quota", "Purge stale durable data or reduce the payload size before retrying the write"
	case CodeNotFunction:
		return "ACTIONABLE_ERRORS.md#gwc-interop-not-function", "Verify the target export or property exists and is callable before invoking it"
	case CodeUnauthorized:
		return "ACTIONABLE_ERRORS.md#gwc-interop-multi-client-unauthorized", "Verify topic ownership, peer role, and origin policy before accepting or publishing privileged multi-client traffic"
	default:
		return "", ""
	}
}

type Subscription struct {
	cancel func()
}

func (s Subscription) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

// WindowEnv exposes shared values attached directly to the browser window object.
// It is intended for simple cross-surface configuration such as mount selectors,
// bootstrap flags, and other host-provided runtime settings.
type WindowEnv struct {
	lookup func(string) (Value, bool)
}

// Lookup returns the raw shared window value when present.
func (e WindowEnv) Lookup(name string) (Value, bool) {
	if e.lookup == nil {
		return Value{}, false
	}
	return e.lookup(name)
}

// LookupString resolves a shared window value as a normalized string.
// Empty strings and JavaScript stringified nullish sentinel values are treated as missing.
func (e WindowEnv) LookupString(name string) (string, bool) {
	value, ok := e.Lookup(name)
	if !ok {
		return "", false
	}
	resolved := strings.TrimSpace(value.String())
	if resolved == "" || resolved == "<undefined>" || resolved == "<null>" {
		return "", false
	}
	return resolved, true
}

// String returns a normalized shared window string or the provided fallback.
func (e WindowEnv) String(name string, fallback string) string {
	if resolved, ok := e.LookupString(name); ok {
		return resolved
	}
	return fallback
}

// Value wraps a browser JavaScript value behind a typed interop surface.
// Platform-specific methods are attached in build-tagged files.
type Value struct {
	raw interface{}
}

type Storage struct {
	getItem    func(string) (string, bool, error)
	getMany    func([]string) (map[string]string, error)
	setItem    func(string, string) error
	removeItem func(string) error
	clear      func() error
	length     func() (int, error)
	key        func(int) (string, bool, error)
}

func (s Storage) GetItem(key string) (string, bool, error) {
	if s.getItem == nil {
		return "", false, unavailable("Storage.GetItem", "")
	}
	return s.getItem(key)
}

func (s Storage) SetItem(key string, value string) error {
	if s.setItem == nil {
		return unavailable("Storage.SetItem", "")
	}
	return s.setItem(key, value)
}

func (s Storage) GetMany(keys ...string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	if s.getMany != nil {
		return s.getMany(keys)
	}
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		value, ok, err := s.GetItem(key)
		if err != nil {
			return nil, err
		}
		if ok {
			values[key] = value
		}
	}
	return values, nil
}

func (s Storage) RemoveItem(key string) error {
	if s.removeItem == nil {
		return unavailable("Storage.RemoveItem", "")
	}
	return s.removeItem(key)
}

func (s Storage) Clear() error {
	if s.clear == nil {
		return unavailable("Storage.Clear", "")
	}
	return s.clear()
}

func (s Storage) Len() (int, error) {
	if s.length == nil {
		return 0, unavailable("Storage.Len", "")
	}
	return s.length()
}

func (s Storage) Key(index int) (string, bool, error) {
	if s.key == nil {
		return "", false, unavailable("Storage.Key", "")
	}
	return s.key(index)
}

type PersistentStoreOptions struct {
	Name               string
	DatabaseName       string
	Version            int
	DeleteOnCorruption bool
	OnBlocked          func(PersistentStoreBlockedEvent)
	FallbackResolver   func() (Storage, error)
	FallbackBackend    string
}

type PersistentStoreBlockedEvent struct {
	DatabaseName     string
	StoreName        string
	RequestedVersion int
}

type PersistentStore struct {
	backend    func() string
	getItem    func(context.Context, string) (string, bool, error)
	setItem    func(context.Context, string, string) error
	removeItem func(context.Context, string) error
	clear      func(context.Context) error
	keys       func(context.Context) ([]string, error)
	length     func(context.Context) (int, error)
	close      func() error
}

func (s PersistentStore) Backend() string {
	if s.backend == nil {
		return ""
	}
	return s.backend()
}

func (s PersistentStore) GetItem(ctx context.Context, key string) (string, bool, error) {
	if s.getItem == nil {
		return "", false, unavailable("PersistentStore.GetItem", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.getItem(ctx, key)
}

func (s PersistentStore) GetMany(ctx context.Context, keys ...string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		value, ok, err := s.GetItem(ctx, key)
		if err != nil {
			return nil, err
		}
		if ok {
			values[key] = value
		}
	}
	return values, nil
}

func (s PersistentStore) SetItem(ctx context.Context, key string, value string) error {
	if s.setItem == nil {
		return unavailable("PersistentStore.SetItem", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.setItem(ctx, key, value)
}

func (s PersistentStore) SetJSON(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return wrapError("PersistentStore.SetJSON", key, CodeEncode, err)
	}
	return s.SetItem(ctx, key, string(data))
}

func (s PersistentStore) DecodeJSON(ctx context.Context, key string, target any) (bool, error) {
	if target == nil {
		return false, wrapError("PersistentStore.DecodeJSON", key, CodeInvalid, errors.New("target is nil"))
	}
	value, ok, err := s.GetItem(ctx, key)
	if err != nil || !ok {
		return ok, err
	}
	if err := json.Unmarshal([]byte(value), target); err != nil {
		return false, wrapError("PersistentStore.DecodeJSON", key, CodeDecode, err)
	}
	return true, nil
}

func (s PersistentStore) RemoveItem(ctx context.Context, key string) error {
	if s.removeItem == nil {
		return unavailable("PersistentStore.RemoveItem", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.removeItem(ctx, key)
}

func (s PersistentStore) Clear(ctx context.Context) error {
	if s.clear == nil {
		return unavailable("PersistentStore.Clear", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.clear(ctx)
}

func (s PersistentStore) Keys(ctx context.Context) ([]string, error) {
	if s.keys == nil {
		return nil, unavailable("PersistentStore.Keys", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.keys(ctx)
}

func (s PersistentStore) Len(ctx context.Context) (int, error) {
	if s.length == nil {
		return 0, unavailable("PersistentStore.Len", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.length(ctx)
}

func (s PersistentStore) Close() error {
	if s.close == nil {
		return nil
	}
	return s.close()
}

func LoadPersistentJSON[T any](ctx context.Context, store PersistentStore, key string) (T, bool, error) {
	var value T
	ok, err := store.DecodeJSON(ctx, key, &value)
	if err != nil || !ok {
		return value, ok, err
	}
	return value, true, nil
}

type Location struct {
	href     func() string
	pathname func() string
	search   func() string
	hash     func() string
	origin   func() string
	assign   func(string) error
	replace  func(string) error
	reload   func() error
}

func (l Location) Href() string {
	if l.href == nil {
		return ""
	}
	return l.href()
}

func (l Location) Pathname() string {
	if l.pathname == nil {
		return ""
	}
	return l.pathname()
}

func (l Location) Search() string {
	if l.search == nil {
		return ""
	}
	return l.search()
}

func (l Location) Hash() string {
	if l.hash == nil {
		return ""
	}
	return l.hash()
}

func (l Location) Origin() string {
	if l.origin == nil {
		return ""
	}
	return l.origin()
}

func (l Location) Assign(rawURL string) error {
	if l.assign == nil {
		return unavailable("Location.Assign", "")
	}
	return l.assign(rawURL)
}

func (l Location) Replace(rawURL string) error {
	if l.replace == nil {
		return unavailable("Location.Replace", "")
	}
	return l.replace(rawURL)
}

func (l Location) Reload() error {
	if l.reload == nil {
		return unavailable("Location.Reload", "")
	}
	return l.reload()
}

type History struct {
	length       func() (int, error)
	state        func() (any, error)
	back         func() error
	forward      func() error
	goDelta      func(int) error
	pushState    func(any, string, string) error
	replaceState func(any, string, string) error
}

func (h History) Len() (int, error) {
	if h.length == nil {
		return 0, unavailable("History.Len", "")
	}
	return h.length()
}

func (h History) State() (any, error) {
	if h.state == nil {
		return nil, unavailable("History.State", "")
	}
	return h.state()
}

func (h History) Back() error {
	if h.back == nil {
		return unavailable("History.Back", "")
	}
	return h.back()
}

func (h History) Forward() error {
	if h.forward == nil {
		return unavailable("History.Forward", "")
	}
	return h.forward()
}

func (h History) Go(delta int) error {
	if h.goDelta == nil {
		return unavailable("History.Go", "")
	}
	return h.goDelta(delta)
}

func (h History) PushState(state any, title string, rawURL string) error {
	if h.pushState == nil {
		return unavailable("History.PushState", "")
	}
	return h.pushState(state, title, rawURL)
}

func (h History) ReplaceState(state any, title string, rawURL string) error {
	if h.replaceState == nil {
		return unavailable("History.ReplaceState", "")
	}
	return h.replaceState(state, title, rawURL)
}

type Clipboard struct {
	writeText func(context.Context, string) error
	readText  func(context.Context) (string, error)
}

func (c Clipboard) WriteText(ctx context.Context, text string) error {
	if c.writeText == nil {
		return unavailable("Clipboard.WriteText", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.writeText(ctx, text)
}

func (c Clipboard) ReadText(ctx context.Context) (string, error) {
	if c.readText == nil {
		return "", unavailable("Clipboard.ReadText", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.readText(ctx)
}

type Timer struct {
	cancel func() error
}

func (t Timer) Cancel() error {
	if t.cancel == nil {
		return unavailable("Timer.Cancel", "")
	}
	return t.cancel()
}

type CustomEvent struct {
	Type   string
	Detail any
}

type DecodedCustomEvent[T any] struct {
	Type   string
	Detail T
}

type BrowserEvent struct {
	Type          string
	Detail        any
	Target        Element
	CurrentTarget Element
}

type EventTarget struct {
	dispatch  func(string, any) error
	listen    func(string, func(BrowserEvent)) (Subscription, error)
	subscribe func(string, func(CustomEvent)) (Subscription, error)
}

func (t EventTarget) Dispatch(name string, detail any) error {
	if t.dispatch == nil {
		return unavailable("EventTarget.Dispatch", "")
	}
	return t.dispatch(name, detail)
}

func (t EventTarget) Listen(name string, handler func(BrowserEvent)) (Subscription, error) {
	if t.listen == nil {
		return Subscription{}, unavailable("EventTarget.Listen", "")
	}
	return t.listen(name, handler)
}

func (t EventTarget) Subscribe(name string, handler func(CustomEvent)) (Subscription, error) {
	if t.subscribe != nil {
		return t.subscribe(name, handler)
	}
	if t.listen == nil {
		return Subscription{}, unavailable("EventTarget.Subscribe", "")
	}
	return t.listen(name, func(event BrowserEvent) {
		handler(CustomEvent{
			Type:   event.Type,
			Detail: event.Detail,
		})
	})
}

// DecodeCustomEvent projects a custom-event detail payload into a typed value.
func DecodeCustomEvent[T any](event CustomEvent) (DecodedCustomEvent[T], error) {
	var detail T
	if err := Decode(event.Detail, &detail); err != nil {
		return DecodedCustomEvent[T]{Type: event.Type}, wrapError("DecodeCustomEvent", event.Type, CodeDecode, err)
	}
	return DecodedCustomEvent[T]{
		Type:   event.Type,
		Detail: detail,
	}, nil
}

// SubscribeDecoded decodes custom-event detail payloads before invoking the handler.
func SubscribeDecoded[T any](target EventTarget, name string, handler func(DecodedCustomEvent[T], error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("SubscribeDecoded", name, CodeInvalid, errors.New("handler is nil"))
	}
	return target.Subscribe(name, func(event CustomEvent) {
		decoded, err := DecodeCustomEvent[T](event)
		handler(decoded, err)
	})
}

type MediaQueryEvent struct {
	Matches bool
	Media   string
}

type MediaQueryList struct {
	matches   func() bool
	media     func() string
	subscribe func(func(MediaQueryEvent)) (Subscription, error)
}

func (m MediaQueryList) Matches() bool {
	if m.matches == nil {
		return false
	}
	return m.matches()
}

func (m MediaQueryList) Media() string {
	if m.media == nil {
		return ""
	}
	return m.media()
}

func (m MediaQueryList) Subscribe(handler func(MediaQueryEvent)) (Subscription, error) {
	if m.subscribe == nil {
		return Subscription{}, unavailable("MediaQueryList.Subscribe", "")
	}
	return m.subscribe(handler)
}

type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

type ScrollIntoViewOptions struct {
	Behavior string
	Block    string
	Inline   string
}

type ResizeEntry struct {
	Target      Element
	ContentRect Rect
}

type IntersectionObserverOptions struct {
	Root       Element
	RootMargin string
	Thresholds []float64
}

type IntersectionEntry struct {
	Target             Element
	IsIntersecting     bool
	IntersectionRatio  float64
	BoundingClientRect Rect
	IntersectionRect   Rect
	RootBounds         *Rect
}

type Element struct {
	raw                 any
	tagName             func() string
	id                  func() string
	className           func() string
	focus               func() error
	blur                func() error
	click               func() error
	setScrollTop        func(float64) error
	scrollIntoView      func(ScrollIntoViewOptions) error
	boundingClientRect  func() (Rect, error)
	events              func() (EventTarget, error)
	observeResize       func(func(ResizeEntry)) (Subscription, error)
	observeIntersection func(IntersectionObserverOptions, func(IntersectionEntry)) (Subscription, error)
	scrollMetrics       func() (float64, float64, float64, error)
}

func (e Element) TagName() string {
	if e.tagName == nil {
		return ""
	}
	return e.tagName()
}

func (e Element) ID() string {
	if e.id == nil {
		return ""
	}
	return e.id()
}

func (e Element) ClassName() string {
	if e.className == nil {
		return ""
	}
	return e.className()
}

func (e Element) Focus() error {
	if e.focus == nil {
		return unavailable("Element.Focus", "")
	}
	return e.focus()
}

func (e Element) Blur() error {
	if e.blur == nil {
		return unavailable("Element.Blur", "")
	}
	return e.blur()
}

func (e Element) Click() error {
	if e.click == nil {
		return unavailable("Element.Click", "")
	}
	return e.click()
}

func (e Element) SetScrollTop(scrollTop float64) error {
	if e.setScrollTop == nil {
		return unavailable("Element.SetScrollTop", "")
	}
	return e.setScrollTop(scrollTop)
}

func (e Element) ScrollIntoView(options ...ScrollIntoViewOptions) error {
	if e.scrollIntoView == nil {
		return unavailable("Element.ScrollIntoView", "")
	}
	var resolved ScrollIntoViewOptions
	if len(options) > 0 {
		resolved = options[0]
	}
	return e.scrollIntoView(resolved)
}

func (e Element) BoundingClientRect() (Rect, error) {
	if e.boundingClientRect == nil {
		return Rect{}, unavailable("Element.BoundingClientRect", "")
	}
	return e.boundingClientRect()
}

func (e Element) Events() (EventTarget, error) {
	if e.events == nil {
		return EventTarget{}, unavailable("Element.Events", "")
	}
	return e.events()
}

func (e Element) Listen(name string, handler func(BrowserEvent)) (Subscription, error) {
	target, err := e.Events()
	if err != nil {
		return Subscription{}, err
	}
	return target.Listen(name, handler)
}

func (e Element) Subscribe(name string, handler func(CustomEvent)) (Subscription, error) {
	target, err := e.Events()
	if err != nil {
		return Subscription{}, err
	}
	return target.Subscribe(name, handler)
}

func (e Element) Dispatch(name string, detail any) error {
	target, err := e.Events()
	if err != nil {
		return err
	}
	return target.Dispatch(name, detail)
}

func (e Element) ObserveResize(handler func(ResizeEntry)) (Subscription, error) {
	if e.observeResize == nil {
		return Subscription{}, unavailable("Element.ObserveResize", "")
	}
	return e.observeResize(handler)
}

func (e Element) ObserveIntersection(handler func(IntersectionEntry), options ...IntersectionObserverOptions) (Subscription, error) {
	if e.observeIntersection == nil {
		return Subscription{}, unavailable("Element.ObserveIntersection", "")
	}
	var resolved IntersectionObserverOptions
	if len(options) > 0 {
		resolved = options[0]
	}
	return e.observeIntersection(resolved, handler)
}

// ScrollMetrics returns the scrollTop, scrollHeight, and clientHeight of the
// element — the three values needed to determine scroll position within a
// scrollable container.
func (e Element) ScrollMetrics() (scrollTop, scrollHeight, clientHeight float64, err error) {
	if e.scrollMetrics == nil {
		return 0, 0, 0, unavailable("Element.ScrollMetrics", "")
	}
	return e.scrollMetrics()
}

type Document struct {
	elementByID   func(string) (Element, bool, error)
	elementsByID  func([]string) (map[string]Element, error)
	querySelector func(string) (Element, bool, error)
}

func (d Document) ElementByID(id string) (Element, bool, error) {
	if d.elementByID == nil {
		return Element{}, false, unavailable("Document.ElementByID", "")
	}
	return d.elementByID(id)
}

func (d Document) QuerySelector(selector string) (Element, bool, error) {
	if d.querySelector == nil {
		return Element{}, false, unavailable("Document.QuerySelector", "")
	}
	return d.querySelector(selector)
}

func (d Document) ElementsByID(ids ...string) (map[string]Element, error) {
	if len(ids) == 0 {
		return map[string]Element{}, nil
	}
	if d.elementsByID != nil {
		return d.elementsByID(ids)
	}
	values := make(map[string]Element, len(ids))
	for _, id := range ids {
		element, ok, err := d.ElementByID(id)
		if err != nil {
			return nil, err
		}
		if ok {
			values[id] = element
		}
	}
	return values, nil
}

type Module struct {
	call        func(context.Context, string, ...any) (any, error)
	callDefault func(context.Context, ...any) (any, error)
	value       func(context.Context, string) (any, error)
	dispose     func() error
}

func (m Module) Call(ctx context.Context, export string, args ...any) (any, error) {
	if m.call == nil {
		return nil, unavailable("Module.Call", export)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return m.call(ctx, export, args...)
}

func (m Module) CallDefault(ctx context.Context, args ...any) (any, error) {
	if m.callDefault == nil {
		return nil, unavailable("Module.CallDefault", "default")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return m.callDefault(ctx, args...)
}

func (m Module) Value(ctx context.Context, export string) (any, error) {
	if m.value == nil {
		return nil, unavailable("Module.Value", export)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return m.value(ctx, export)
}

func (m Module) Dispose() error {
	if m.dispose == nil {
		return unavailable("Module.Dispose", "")
	}
	return m.dispose()
}

type WorkerOptions struct {
	URL          string
	Name         string
	Type         string
	Ready        bool
	ReadyTimeout time.Duration
}

type GoWASMWorkerOptions struct {
	RuntimeURL   string
	WASMURL      string
	Name         string
	Ready        bool
	ReadyTimeout time.Duration
}

type CrossTabChannelOptions struct {
	Name       string
	StorageKey string
}

type WindowChannelOptions struct {
	URL          string
	Name         string
	Features     string
	TargetOrigin string
}

type CrossTabEnvelope struct {
	Name     string    `json:"name"`
	Payload  any       `json:"payload"`
	Source   string    `json:"source,omitempty"`
	Sequence int64     `json:"sequence,omitempty"`
	SentAt   time.Time `json:"sentAt,omitempty"`
}

type WindowEnvelope struct {
	Name    string    `json:"name"`
	Payload any       `json:"payload"`
	Source  string    `json:"source,omitempty"`
	SentAt  time.Time `json:"sentAt,omitempty"`
}

type ClientIdentity struct {
	ID      string `json:"id"`
	App     string `json:"app"`
	Surface string `json:"surface"`
	Role    string `json:"role,omitempty"`
	Version string `json:"version,omitempty"`
}

type ClientCapabilities struct {
	ProtocolVersion string   `json:"protocolVersion,omitempty"`
	Transports      []string `json:"transports,omitempty"`
	Encodings       []string `json:"encodings,omitempty"`
	Topics          []string `json:"topics,omitempty"`
	MaxJSONBytes    int      `json:"maxJsonBytes,omitempty"`
	MaxBinaryBytes  int      `json:"maxBinaryBytes,omitempty"`
}

type ClientMessageKind string

const (
	ClientHello      ClientMessageKind = "hello"
	ClientGoodbye    ClientMessageKind = "goodbye"
	ClientEvent      ClientMessageKind = "event"
	ClientIntent     ClientMessageKind = "intent"
	ClientQuery      ClientMessageKind = "query"
	ClientResult     ClientMessageKind = "result"
	ClientInvalidate ClientMessageKind = "invalidate"
	ClientError      ClientMessageKind = "error"
)

type ClientPayloadEncoding string

const (
	ClientPayloadJSON   ClientPayloadEncoding = "json"
	ClientPayloadBinary ClientPayloadEncoding = "binary"
)

const ClientPresenceTopic = "clients"

type ClientMessage struct {
	ID           string                `json:"id,omitempty"`
	Kind         ClientMessageKind     `json:"kind"`
	Topic        string                `json:"topic"`
	Source       ClientIdentity        `json:"source"`
	Capabilities *ClientCapabilities   `json:"capabilities,omitempty"`
	Target       string                `json:"target,omitempty"`
	Encoding     ClientPayloadEncoding `json:"encoding,omitempty"`
	ContentType  string                `json:"contentType,omitempty"`
	Payload      any                   `json:"payload,omitempty"`
	Revision     string                `json:"revision,omitempty"`
	Error        string                `json:"error,omitempty"`
	SentAt       time.Time             `json:"sentAt,omitempty"`
}

type ClientBinaryPayload struct {
	ContentType string `json:"contentType,omitempty"`
	Bytes       []byte `json:"-"`
}

type SurfaceSignalKind string

const (
	SurfaceSignalSession   SurfaceSignalKind = "session"
	SurfaceSignalRoute     SurfaceSignalKind = "route"
	SurfaceSignalSelection SurfaceSignalKind = "selection"
	SurfaceSignalIntent    SurfaceSignalKind = "intent"
)

type SurfaceIntentAction string

const (
	SurfaceIntentFocusWindow SurfaceIntentAction = "focus-window"
	SurfaceIntentFocusPanel  SurfaceIntentAction = "focus-panel"
	SurfaceIntentOpenPanel   SurfaceIntentAction = "open-panel"
	SurfaceIntentCloseWindow SurfaceIntentAction = "close-window"
)

type SurfaceSignal struct {
	Kind      SurfaceSignalKind       `json:"kind"`
	Session   *SurfaceSessionSignal   `json:"session,omitempty"`
	Route     *SurfaceRouteSignal     `json:"route,omitempty"`
	Selection *SurfaceSelectionSignal `json:"selection,omitempty"`
	Intent    *SurfaceIntentSignal    `json:"intent,omitempty"`
}

type SurfaceSessionSignal struct {
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	ReturnTo  string    `json:"returnTo,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

type SurfaceRouteSignal struct {
	Path    string `json:"path"`
	Query   string `json:"query,omitempty"`
	FocusID string `json:"focusId,omitempty"`
	Replace bool   `json:"replace,omitempty"`
}

type SurfaceSelectionSignal struct {
	Scope    string `json:"scope,omitempty"`
	ID       string `json:"id"`
	Revision string `json:"revision,omitempty"`
}

type SurfaceIntentSignal struct {
	Action SurfaceIntentAction `json:"action"`
	Target string              `json:"target,omitempty"`
	Params map[string]string   `json:"params,omitempty"`
}

type DecodedCrossTabEnvelope[T any] struct {
	Name     string
	Payload  T
	Source   string
	Sequence int64
	SentAt   time.Time
}

type DecodedWindowEnvelope[T any] struct {
	Name    string
	Payload T
	Source  string
	SentAt  time.Time
}

type WorkerMessage struct {
	ID      string `json:"id"`
	Phase   string `json:"phase"`
	Name    string `json:"name"`
	Payload any    `json:"payload"`
	Error   string `json:"error,omitempty"`
}

type DecodedWorkerMessage[T any] struct {
	ID      string
	Phase   string
	Name    string
	Payload T
	Error   string
}

type Worker struct {
	post      func(any) error
	subscribe func(func(WorkerMessage, error)) (Subscription, error)
	request   func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error)
	terminate func() error
	restart   func(context.Context) error
}

type WorkerScope struct {
	post      func(WorkerMessage) error
	subscribe func(func(WorkerMessage, error)) (Subscription, error)
}

type CrossTabChannel struct {
	name                func() string
	transport           func() string
	publish             func(any) error
	publishClientBinary func(ClientMessage) error
	subscribe           func(func(CrossTabEnvelope, error)) (Subscription, error)
	close               func() error
}

type WindowChannel struct {
	name                func() string
	targetOrigin        func() string
	publish             func(any) error
	publishClientBinary func(ClientMessage) error
	subscribe           func(func(WindowEnvelope, error)) (Subscription, error)
	focus               func() error
	close               func() error
	closed              func() bool
}

func (c CrossTabChannel) Name() string {
	if c.name == nil {
		return ""
	}
	return c.name()
}

func (c CrossTabChannel) Transport() string {
	if c.transport == nil {
		return ""
	}
	return c.transport()
}

func (c CrossTabChannel) Publish(payload any) error {
	if c.publish == nil {
		return unavailable("CrossTabChannel.Publish", "")
	}
	return c.publish(payload)
}

func (c CrossTabChannel) Subscribe(handler func(CrossTabEnvelope, error)) (Subscription, error) {
	if c.subscribe == nil {
		return Subscription{}, unavailable("CrossTabChannel.Subscribe", "")
	}
	return c.subscribe(handler)
}

func (c CrossTabChannel) Close() error {
	if c.close == nil {
		return unavailable("CrossTabChannel.Close", "")
	}
	return c.close()
}

func (c WindowChannel) Name() string {
	if c.name == nil {
		return ""
	}
	return c.name()
}

func (c WindowChannel) TargetOrigin() string {
	if c.targetOrigin == nil {
		return ""
	}
	return c.targetOrigin()
}

func (c WindowChannel) Publish(payload any) error {
	if c.publish == nil {
		return unavailable("WindowChannel.Publish", "")
	}
	return c.publish(payload)
}

func (c WindowChannel) Subscribe(handler func(WindowEnvelope, error)) (Subscription, error) {
	if c.subscribe == nil {
		return Subscription{}, unavailable("WindowChannel.Subscribe", "")
	}
	return c.subscribe(handler)
}

func (c WindowChannel) Focus() error {
	if c.focus == nil {
		return unavailable("WindowChannel.Focus", "")
	}
	return c.focus()
}

func (c WindowChannel) Close() error {
	if c.close == nil {
		return unavailable("WindowChannel.Close", "")
	}
	return c.close()
}

func (c WindowChannel) Closed() bool {
	if c.closed == nil {
		return false
	}
	return c.closed()
}

func (w Worker) Post(message any) error {
	if w.post == nil {
		return unavailable("Worker.Post", "")
	}
	return w.post(message)
}

func (w Worker) Subscribe(handler func(WorkerMessage, error)) (Subscription, error) {
	if w.subscribe == nil {
		return Subscription{}, unavailable("Worker.Subscribe", "")
	}
	return w.subscribe(handler)
}

func (w Worker) Request(ctx context.Context, name string, payload any, onProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	if w.request == nil {
		return WorkerMessage{}, unavailable("Worker.Request", name)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return w.request(ctx, name, payload, onProgress)
}

func (w Worker) Terminate() error {
	if w.terminate == nil {
		return unavailable("Worker.Terminate", "")
	}
	return w.terminate()
}

func (w Worker) Restart(ctx context.Context) error {
	if w.restart == nil {
		return unavailable("Worker.Restart", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return w.restart(ctx)
}

func (w WorkerScope) Post(message WorkerMessage) error {
	if w.post == nil {
		return unavailable("WorkerScope.Post", "")
	}
	return w.post(message)
}

func (w WorkerScope) Subscribe(handler func(WorkerMessage, error)) (Subscription, error) {
	if w.subscribe == nil {
		return Subscription{}, unavailable("WorkerScope.Subscribe", "")
	}
	return w.subscribe(handler)
}

func (w WorkerScope) Ready(name string) error {
	return w.Post(WorkerMessage{Phase: "ready", Name: name})
}

func (w WorkerScope) Message(name string, payload any) error {
	return w.Post(WorkerMessage{Phase: "message", Name: name, Payload: payload})
}

func (w WorkerScope) Progress(id string, name string, payload any) error {
	return w.Post(WorkerMessage{ID: id, Phase: "progress", Name: name, Payload: payload})
}

func (w WorkerScope) Result(id string, name string, payload any) error {
	return w.Post(WorkerMessage{ID: id, Phase: "result", Name: name, Payload: payload})
}

func (w WorkerScope) Error(id string, name string, errText string, payload any) error {
	return w.Post(WorkerMessage{ID: id, Phase: "error", Name: name, Error: errText, Payload: payload})
}

func DecodeWorkerMessage[T any](message WorkerMessage) (DecodedWorkerMessage[T], error) {
	var payload T
	if err := Decode(message.Payload, &payload); err != nil {
		return DecodedWorkerMessage[T]{
			ID:    message.ID,
			Phase: message.Phase,
			Name:  message.Name,
			Error: message.Error,
		}, wrapError("DecodeWorkerMessage", message.Name, CodeDecode, err)
	}
	return DecodedWorkerMessage[T]{
		ID:      message.ID,
		Phase:   message.Phase,
		Name:    message.Name,
		Payload: payload,
		Error:   message.Error,
	}, nil
}

func SubscribeDecodedWorker[T any](worker Worker, handler func(DecodedWorkerMessage[T], error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("SubscribeDecodedWorker", "", CodeInvalid, errors.New("handler is nil"))
	}
	return worker.Subscribe(func(message WorkerMessage, err error) {
		if err != nil {
			handler(DecodedWorkerMessage[T]{}, err)
			return
		}
		decoded, decodeErr := DecodeWorkerMessage[T](message)
		handler(decoded, decodeErr)
	})
}

func RequestWorkerDecoded[Req any, Progress any, Result any](ctx context.Context, worker Worker, name string, payload Req, onProgress func(DecodedWorkerMessage[Progress], error)) (Result, error) {
	var zero Result
	response, err := worker.Request(ctx, name, payload, func(message WorkerMessage, messageErr error) {
		if onProgress == nil {
			return
		}
		if messageErr != nil {
			onProgress(DecodedWorkerMessage[Progress]{}, messageErr)
			return
		}
		decoded, decodeErr := DecodeWorkerMessage[Progress](message)
		onProgress(decoded, decodeErr)
	})
	if err != nil {
		return zero, err
	}
	decoded, err := DecodeWorkerMessage[Result](response)
	if err != nil {
		return zero, err
	}
	return decoded.Payload, nil
}

func DecodeCrossTabEnvelope[T any](message CrossTabEnvelope) (DecodedCrossTabEnvelope[T], error) {
	var payload T
	if err := Decode(message.Payload, &payload); err != nil {
		return DecodedCrossTabEnvelope[T]{
			Name:     message.Name,
			Source:   message.Source,
			Sequence: message.Sequence,
			SentAt:   message.SentAt,
		}, wrapError("DecodeCrossTabEnvelope", message.Name, CodeDecode, err)
	}
	return DecodedCrossTabEnvelope[T]{
		Name:     message.Name,
		Payload:  payload,
		Source:   message.Source,
		Sequence: message.Sequence,
		SentAt:   message.SentAt,
	}, nil
}

func SubscribeDecodedCrossTab[T any](channel CrossTabChannel, handler func(DecodedCrossTabEnvelope[T], error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("SubscribeDecodedCrossTab", channel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return channel.Subscribe(func(message CrossTabEnvelope, err error) {
		if err != nil {
			handler(DecodedCrossTabEnvelope[T]{}, err)
			return
		}
		decoded, decodeErr := DecodeCrossTabEnvelope[T](message)
		handler(decoded, decodeErr)
	})
}

func DecodeWindowEnvelope[T any](message WindowEnvelope) (DecodedWindowEnvelope[T], error) {
	var payload T
	if err := Decode(message.Payload, &payload); err != nil {
		return DecodedWindowEnvelope[T]{
			Name:   message.Name,
			Source: message.Source,
			SentAt: message.SentAt,
		}, wrapError("DecodeWindowEnvelope", message.Name, CodeDecode, err)
	}
	return DecodedWindowEnvelope[T]{
		Name:    message.Name,
		Payload: payload,
		Source:  message.Source,
		SentAt:  message.SentAt,
	}, nil
}

func DecodeClientMessage(value any) (ClientMessage, error) {
	message, err := decodeClientMessageValue(value)
	if err != nil {
		return ClientMessage{}, err
	}
	if err := validateClientMessage("DecodeClientMessage", message.Topic, message); err != nil {
		return ClientMessage{}, err
	}
	return message, nil
}

func PublishClientMessage(channel CrossTabChannel, message ClientMessage) error {
	prepared, err := prepareClientMessage("PublishClientMessage", channel.Name(), message)
	if err != nil {
		return err
	}
	return channel.Publish(prepared)
}

func PublishClientWindowMessage(channel WindowChannel, message ClientMessage) error {
	prepared, err := prepareClientMessage("PublishClientWindowMessage", channel.Name(), message)
	if err != nil {
		return err
	}
	return channel.Publish(prepared)
}

func SubscribeClientMessages(channel CrossTabChannel, handler func(ClientMessage, error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("SubscribeClientMessages", channel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return channel.Subscribe(func(message CrossTabEnvelope, err error) {
		if err != nil {
			handler(ClientMessage{}, err)
			return
		}
		decoded, decodeErr := DecodeClientMessage(message.Payload)
		handler(decoded, decodeErr)
	})
}

func SubscribeClientWindowMessages(channel WindowChannel, handler func(ClientMessage, error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("SubscribeClientWindowMessages", channel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return channel.Subscribe(func(message WindowEnvelope, err error) {
		if err != nil {
			handler(ClientMessage{}, err)
			return
		}
		decoded, decodeErr := DecodeClientMessage(message.Payload)
		handler(decoded, decodeErr)
	})
}

func PublishClientHello(channel CrossTabChannel, self ClientIdentity) error {
	capabilities := defaultCrossTabClientCapabilities(channel)
	return PublishClientHelloWithCapabilities(channel, self, capabilities)
}

func PublishClientHelloWithCapabilities(channel CrossTabChannel, self ClientIdentity, capabilities ClientCapabilities) error {
	return PublishClientMessage(channel, ClientMessage{
		Kind:         ClientHello,
		Topic:        ClientPresenceTopic,
		Source:       self,
		Capabilities: &capabilities,
	})
}

func PublishClientHelloWindow(channel WindowChannel, self ClientIdentity) error {
	capabilities := defaultWindowClientCapabilities(channel)
	return PublishClientHelloWindowWithCapabilities(channel, self, capabilities)
}

func PublishClientHelloWindowWithCapabilities(channel WindowChannel, self ClientIdentity, capabilities ClientCapabilities) error {
	return PublishClientWindowMessage(channel, ClientMessage{
		Kind:         ClientHello,
		Topic:        ClientPresenceTopic,
		Source:       self,
		Capabilities: &capabilities,
	})
}

func PublishClientGoodbye(channel CrossTabChannel, self ClientIdentity) error {
	return PublishClientMessage(channel, ClientMessage{
		Kind:   ClientGoodbye,
		Topic:  ClientPresenceTopic,
		Source: self,
	})
}

func PublishClientGoodbyeWindow(channel WindowChannel, self ClientIdentity) error {
	return PublishClientWindowMessage(channel, ClientMessage{
		Kind:   ClientGoodbye,
		Topic:  ClientPresenceTopic,
		Source: self,
	})
}

func PublishClientEvent(channel CrossTabChannel, topic string, self ClientIdentity, payload any) error {
	return PublishClientMessage(channel, ClientMessage{
		Kind:    ClientEvent,
		Topic:   strings.TrimSpace(topic),
		Source:  self,
		Payload: payload,
	})
}

func PublishClientIntent(channel WindowChannel, topic string, self ClientIdentity, target string, payload any) error {
	trimmedTarget := strings.TrimSpace(target)
	if trimmedTarget == "" {
		return wrapError("PublishClientIntent", channel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	return PublishClientWindowMessage(channel, ClientMessage{
		Kind:    ClientIntent,
		Topic:   strings.TrimSpace(topic),
		Source:  self,
		Target:  trimmedTarget,
		Payload: payload,
	})
}

func PublishClientInvalidation(channel CrossTabChannel, topic string, self ClientIdentity, revision string) error {
	trimmedRevision := strings.TrimSpace(revision)
	if trimmedRevision == "" {
		return wrapError("PublishClientInvalidation", channel.Name(), CodeInvalid, errors.New("revision is empty"))
	}
	return PublishClientMessage(channel, ClientMessage{
		Kind:     ClientInvalidate,
		Topic:    strings.TrimSpace(topic),
		Source:   self,
		Revision: trimmedRevision,
	})
}

func PublishClientQuery(channel CrossTabChannel, topic string, self ClientIdentity) error {
	return PublishClientMessage(channel, ClientMessage{
		Kind:   ClientQuery,
		Topic:  strings.TrimSpace(topic),
		Source: self,
	})
}

func PublishClientResult(channel CrossTabChannel, topic string, self ClientIdentity, target string, payload any) error {
	trimmedTarget := strings.TrimSpace(target)
	if trimmedTarget == "" {
		return wrapError("PublishClientResult", channel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	return PublishClientMessage(channel, ClientMessage{
		Kind:    ClientResult,
		Topic:   strings.TrimSpace(topic),
		Source:  self,
		Target:  trimmedTarget,
		Payload: payload,
	})
}

func PublishClientBinaryWindow(channel WindowChannel, topic string, self ClientIdentity, target string, payload ClientBinaryPayload) error {
	trimmedTarget := strings.TrimSpace(target)
	if trimmedTarget == "" {
		return wrapError("PublishClientBinaryWindow", channel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	prepared, err := prepareClientBinaryMessage("PublishClientBinaryWindow", channel.Name(), ClientMessage{
		Kind:        ClientEvent,
		Topic:       strings.TrimSpace(topic),
		Source:      self,
		Target:      trimmedTarget,
		Encoding:    ClientPayloadBinary,
		ContentType: strings.TrimSpace(payload.ContentType),
		Payload:     append([]byte(nil), payload.Bytes...),
	})
	if err != nil {
		return err
	}
	if channel.publishClientBinary != nil {
		return channel.publishClientBinary(prepared)
	}
	return PublishClientWindowMessage(channel, prepared)
}

func PublishClientBinaryCrossTab(channel CrossTabChannel, topic string, self ClientIdentity, payload ClientBinaryPayload) error {
	prepared, err := prepareClientBinaryMessage("PublishClientBinaryCrossTab", channel.Name(), ClientMessage{
		Kind:        ClientEvent,
		Topic:       strings.TrimSpace(topic),
		Source:      self,
		Encoding:    ClientPayloadBinary,
		ContentType: strings.TrimSpace(payload.ContentType),
		Payload:     append([]byte(nil), payload.Bytes...),
	})
	if err != nil {
		return err
	}
	if channel.publishClientBinary != nil {
		return channel.publishClientBinary(prepared)
	}
	return PublishClientMessage(channel, prepared)
}

func prepareClientMessage(op string, target string, message ClientMessage) (ClientMessage, error) {
	if err := validateClientMessage(op, target, message); err != nil {
		return ClientMessage{}, err
	}
	if message.SentAt.IsZero() {
		message.SentAt = time.Now().UTC()
	}
	return message, nil
}

func prepareClientBinaryMessage(op string, target string, message ClientMessage) (ClientMessage, error) {
	prepared, err := prepareClientMessage(op, target, message)
	if err != nil {
		return ClientMessage{}, err
	}
	prepared.Encoding = ClientPayloadBinary
	bytes, ok := prepared.Payload.([]byte)
	if !ok {
		return ClientMessage{}, wrapError(op, target, CodeInvalid, errors.New("binary payload must be []byte"))
	}
	if len(bytes) == 0 {
		return ClientMessage{}, wrapError(op, target, CodeInvalid, errors.New("binary payload is empty"))
	}
	if strings.TrimSpace(prepared.ContentType) == "" {
		return ClientMessage{}, wrapError(op, target, CodeInvalid, errors.New("binary content type is empty"))
	}
	prepared.Payload = append([]byte(nil), bytes...)
	return prepared, nil
}

func validateClientMessage(op string, target string, message ClientMessage) error {
	if err := validateClientIdentity(op, target, message.Source); err != nil {
		return err
	}
	if strings.TrimSpace(message.Topic) == "" {
		return wrapError(op, target, CodeInvalid, errors.New("client topic is empty"))
	}
	switch message.Kind {
	case ClientHello, ClientGoodbye, ClientEvent, ClientIntent, ClientQuery, ClientResult, ClientInvalidate, ClientError:
	default:
		return wrapError(op, target, CodeInvalid, errors.New("client message kind is empty or unknown"))
	}
	switch message.Encoding {
	case "", ClientPayloadJSON, ClientPayloadBinary:
	default:
		return wrapError(op, target, CodeInvalid, errors.New("client payload encoding is empty or unknown"))
	}
	if message.Encoding == ClientPayloadBinary {
		if _, ok := message.Payload.([]byte); !ok {
			return wrapError(op, target, CodeInvalid, errors.New("binary client payload must be []byte"))
		}
	}
	if err := validateClientTopicAuthorization(op, target, message); err != nil {
		return err
	}
	return nil
}

func validateClientTopicAuthorization(op string, target string, message ClientMessage) error {
	topic := strings.ToLower(strings.TrimSpace(message.Topic))
	if !isPrivilegedClientTopic(topic) {
		return nil
	}
	role := normalizeClientRole(message.Source.Role)
	if clientRoleMayUsePrivilegedTopic(role) {
		return nil
	}
	if message.Kind == ClientIntent {
		return wrapError(op, target, CodeUnauthorized, errors.New("client role is not authorized to publish privileged intent topic"))
	}
	return wrapError(op, target, CodeUnauthorized, errors.New("client role is not authorized for privileged topic"))
}

func isPrivilegedClientTopic(topic string) bool {
	switch {
	case strings.HasPrefix(topic, "session:"), strings.HasPrefix(topic, "operator:"), strings.HasPrefix(topic, "intent:session"), strings.HasPrefix(topic, "intent:operator"):
		return true
	default:
		return false
	}
}

func normalizeClientRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func clientRoleMayUsePrivilegedTopic(role string) bool {
	switch role {
	case "operator", "admin", "system":
		return true
	default:
		return false
	}
}

func decodeClientMessageValue(value any) (ClientMessage, error) {
	switch typed := value.(type) {
	case ClientMessage:
		return typed, nil
	case map[string]any:
		return decodeClientMessageMap(typed)
	default:
		var message ClientMessage
		if err := Decode(value, &message); err != nil {
			return ClientMessage{}, err
		}
		return message, nil
	}
}

func decodeClientMessageMap(data map[string]any) (ClientMessage, error) {
	message := ClientMessage{
		ID:          stringField(data, "id"),
		Kind:        ClientMessageKind(stringField(data, "kind")),
		Topic:       stringField(data, "topic"),
		Target:      stringField(data, "target"),
		Revision:    stringField(data, "revision"),
		Error:       stringField(data, "error"),
		Encoding:    ClientPayloadEncoding(stringField(data, "encoding")),
		ContentType: stringField(data, "contentType"),
	}
	if source, ok := data["source"].(map[string]any); ok {
		message.Source = ClientIdentity{
			ID:      stringField(source, "id"),
			App:     stringField(source, "app"),
			Surface: stringField(source, "surface"),
			Role:    stringField(source, "role"),
			Version: stringField(source, "version"),
		}
	}
	if capabilities, ok := data["capabilities"].(map[string]any); ok {
		message.Capabilities = decodeClientCapabilitiesMap(capabilities)
	}
	if payload, ok := data["payload"]; ok {
		message.Payload = payload
	}
	if sentAt, ok := clientTimeField(data["sentAt"]); ok {
		message.SentAt = sentAt
	}
	return message, nil
}

func decodeClientCapabilitiesMap(data map[string]any) *ClientCapabilities {
	capabilities := &ClientCapabilities{
		ProtocolVersion: stringField(data, "protocolVersion"),
		Transports:      stringSliceField(data["transports"]),
		Encodings:       stringSliceField(data["encodings"]),
		Topics:          stringSliceField(data["topics"]),
		MaxJSONBytes:    intField(data["maxJsonBytes"]),
		MaxBinaryBytes:  intField(data["maxBinaryBytes"]),
	}
	return capabilities
}

func stringField(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func stringSliceField(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		values := make([]string, 0, len(typed))
		for _, entry := range typed {
			text, ok := entry.(string)
			if ok && strings.TrimSpace(text) != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func intField(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func clientTimeField(value any) (time.Time, bool) {
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

func defaultCrossTabClientCapabilities(channel CrossTabChannel) ClientCapabilities {
	encodings := []string{string(ClientPayloadJSON)}
	if channel.Transport() == "broadcast-channel" {
		encodings = append(encodings, string(ClientPayloadBinary))
	}
	transports := []string{}
	if transport := strings.TrimSpace(channel.Transport()); transport != "" {
		transports = append(transports, transport)
	}
	return ClientCapabilities{
		ProtocolVersion: "v1",
		Transports:      transports,
		Encodings:       encodings,
	}
}

func defaultWindowClientCapabilities(channel WindowChannel) ClientCapabilities {
	transports := []string{"window-message"}
	if target := strings.TrimSpace(channel.Name()); target != "" {
		_ = target
	}
	return ClientCapabilities{
		ProtocolVersion: "v1",
		Transports:      transports,
		Encodings:       []string{string(ClientPayloadJSON), string(ClientPayloadBinary)},
	}
}

func ClientProtocolCompatible(local ClientCapabilities, peer ClientCapabilities) bool {
	localVersion := normalizeProtocolVersion(local.ProtocolVersion)
	peerVersion := normalizeProtocolVersion(peer.ProtocolVersion)
	if localVersion == "" || peerVersion == "" {
		return false
	}
	return protocolMajor(localVersion) == protocolMajor(peerVersion)
}

func ClientSupportsEncoding(capabilities ClientCapabilities, encoding ClientPayloadEncoding) bool {
	trimmed := strings.TrimSpace(string(encoding))
	if trimmed == "" {
		trimmed = string(ClientPayloadJSON)
	}
	for _, candidate := range capabilities.Encodings {
		if strings.EqualFold(strings.TrimSpace(candidate), trimmed) {
			return true
		}
	}
	return false
}

func ClientSupportsTopic(capabilities ClientCapabilities, topic string) bool {
	trimmed := strings.TrimSpace(topic)
	if trimmed == "" {
		return false
	}
	if len(capabilities.Topics) == 0 {
		return true
	}
	for _, candidate := range capabilities.Topics {
		if strings.EqualFold(strings.TrimSpace(candidate), trimmed) {
			return true
		}
	}
	return false
}

func ClientCanExchange(local ClientCapabilities, peer ClientCapabilities, topic string, encoding ClientPayloadEncoding) bool {
	if !ClientProtocolCompatible(local, peer) {
		return false
	}
	if !ClientSupportsEncoding(local, encoding) || !ClientSupportsEncoding(peer, encoding) {
		return false
	}
	if !ClientSupportsTopic(local, topic) || !ClientSupportsTopic(peer, topic) {
		return false
	}
	return true
}

func normalizeProtocolVersion(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = strings.TrimPrefix(trimmed, "v")
	return trimmed
}

func protocolMajor(value string) string {
	trimmed := normalizeProtocolVersion(value)
	if trimmed == "" {
		return ""
	}
	if dot := strings.Index(trimmed, "."); dot >= 0 {
		return trimmed[:dot]
	}
	return trimmed
}

func validateClientIdentity(op string, target string, identity ClientIdentity) error {
	if strings.TrimSpace(identity.ID) == "" {
		return wrapError(op, target, CodeInvalid, errors.New("client identity id is empty"))
	}
	if strings.TrimSpace(identity.App) == "" {
		return wrapError(op, target, CodeInvalid, errors.New("client identity app is empty"))
	}
	if strings.TrimSpace(identity.Surface) == "" {
		return wrapError(op, target, CodeInvalid, errors.New("client identity surface is empty"))
	}
	return nil
}

func SubscribeDecodedWindow[T any](channel WindowChannel, handler func(DecodedWindowEnvelope[T], error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("SubscribeDecodedWindow", channel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return channel.Subscribe(func(message WindowEnvelope, err error) {
		if err != nil {
			handler(DecodedWindowEnvelope[T]{}, err)
			return
		}
		decoded, decodeErr := DecodeWindowEnvelope[T](message)
		handler(decoded, decodeErr)
	})
}

func DecodeSurfaceSignal(message WindowEnvelope) (DecodedWindowEnvelope[SurfaceSignal], error) {
	return DecodeWindowEnvelope[SurfaceSignal](message)
}

func SubscribeSurfaceSignals(channel WindowChannel, handler func(DecodedWindowEnvelope[SurfaceSignal], error)) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("SubscribeSurfaceSignals", channel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return SubscribeDecodedWindow[SurfaceSignal](channel, handler)
}

func PublishSurfaceSignal(channel WindowChannel, signal SurfaceSignal) error {
	switch signal.Kind {
	case SurfaceSignalSession:
		if signal.Session == nil {
			return wrapError("PublishSurfaceSignal", channel.Name(), CodeInvalid, errors.New("session signal is missing session payload"))
		}
	case SurfaceSignalRoute:
		if signal.Route == nil || strings.TrimSpace(signal.Route.Path) == "" {
			return wrapError("PublishSurfaceSignal", channel.Name(), CodeInvalid, errors.New("route signal is missing path"))
		}
	case SurfaceSignalSelection:
		if signal.Selection == nil || strings.TrimSpace(signal.Selection.ID) == "" {
			return wrapError("PublishSurfaceSignal", channel.Name(), CodeInvalid, errors.New("selection signal is missing id"))
		}
	case SurfaceSignalIntent:
		if signal.Intent == nil || strings.TrimSpace(string(signal.Intent.Action)) == "" {
			return wrapError("PublishSurfaceSignal", channel.Name(), CodeInvalid, errors.New("intent signal is missing action"))
		}
	default:
		return wrapError("PublishSurfaceSignal", channel.Name(), CodeInvalid, errors.New("surface signal kind is empty or unknown"))
	}
	return channel.Publish(signal)
}

func PublishLogout(channel WindowChannel, reason string) error {
	return PublishSurfaceSignal(channel, SurfaceSignal{
		Kind: SurfaceSignalSession,
		Session: &SurfaceSessionSignal{
			Status: "signed-out",
			Reason: strings.TrimSpace(reason),
		},
	})
}

func PublishSessionExpired(channel WindowChannel, reason string, returnTo string, expiresAt time.Time) error {
	return PublishSurfaceSignal(channel, SurfaceSignal{
		Kind: SurfaceSignalSession,
		Session: &SurfaceSessionSignal{
			Status:    "expired",
			Reason:    strings.TrimSpace(reason),
			ReturnTo:  strings.TrimSpace(returnTo),
			ExpiresAt: expiresAt,
		},
	})
}

func PublishRouteFocus(channel WindowChannel, path string, query string, focusID string) error {
	return PublishSurfaceSignal(channel, SurfaceSignal{
		Kind: SurfaceSignalRoute,
		Route: &SurfaceRouteSignal{
			Path:    strings.TrimSpace(path),
			Query:   strings.TrimSpace(query),
			FocusID: strings.TrimSpace(focusID),
		},
	})
}

func PublishSelection(channel WindowChannel, scope string, id string, revision string) error {
	return PublishSurfaceSignal(channel, SurfaceSignal{
		Kind: SurfaceSignalSelection,
		Selection: &SurfaceSelectionSignal{
			Scope:    strings.TrimSpace(scope),
			ID:       strings.TrimSpace(id),
			Revision: strings.TrimSpace(revision),
		},
	})
}

func PublishIntent(channel WindowChannel, action SurfaceIntentAction, target string, params map[string]string) error {
	return PublishSurfaceSignal(channel, SurfaceSignal{
		Kind: SurfaceSignalIntent,
		Intent: &SurfaceIntentSignal{
			Action: action,
			Target: strings.TrimSpace(target),
			Params: params,
		},
	})
}
