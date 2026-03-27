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

func (parseE *Error) Error() string {
	if parseE == nil {
		return ""
	}
	parseBase := "interop failure"
	if parseE.Op != "" {
		parseBase = parseE.Op
	}
	if parseE.Target != "" {
		parseBase += " " + parseE.Target
	}
	if parseE.Code != "" {
		parseBase += " [" + string(parseE.Code) + "]"
	}
	parseMessage := parseBase
	if parseE.Err != nil {
		parseMessage += ": " + parseE.Err.Error()
	}
	if parseDocs, parseRemediation := interopActionableGuidance(parseE.Code); parseDocs != "" {
		if parseRemediation != "" {
			parseMessage += ". " + parseRemediation
		}
		parseMessage += ". See " + parseDocs
	}
	return parseMessage
}

func (parseE *Error) Unwrap() error {
	if parseE == nil {
		return nil
	}
	return parseE.Err
}

func wrapError(parseOp, parseTarget string, parseCode ErrorCode, parseErr error) error {
	if parseErr == nil {
		return nil
	}
	return &Error{Op: parseOp, Target: parseTarget, Code: parseCode, Err: parseErr}
}

func unavailable(parseOp, parseTarget string) error {
	return &Error{Op: parseOp, Target: parseTarget, Code: CodeUnavailable, Err: errors.New("browser interop is unavailable in this build")}
}

// Decode projects JSON-shaped interop payloads into a typed target.
func Decode(parseValue any, parseTarget interface{}) error {
	if parseTarget == nil {
		return wrapError("Decode", "", CodeInvalid, errors.New("target is nil"))
	}
	parseData, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return wrapError("Decode", "", CodeEncode, parseErr)
	}
	if parseErr2 := json.Unmarshal(parseData, parseTarget); parseErr2 != nil {
		return wrapError("Decode", "", CodeDecode, parseErr2)
	}
	return nil
}

// IsCode reports whether err is an interop error with the provided code.
func IsCode(parseErr error, parseCode ErrorCode) bool {
	var parseInteropErr *Error
	if !errors.As(parseErr, &parseInteropErr) {
		return false
	}
	return parseInteropErr.Code == parseCode
}

// AsError unwraps an interop error into the structured Error form.
func AsError(parseErr error) (*Error, bool) {
	var parseInteropErr *Error
	if !errors.As(parseErr, &parseInteropErr) {
		return nil, false
	}
	return parseInteropErr, true
}

// CodeOf returns the interop error code for err when available.
func CodeOf(parseErr error) (ErrorCode, bool) {
	parseInteropErr, parseOk := AsError(parseErr)
	if !parseOk {
		return "", false
	}
	return parseInteropErr.Code, true
}

func interopActionableGuidance(parseCode ErrorCode) (string, string) {
	switch parseCode {
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

func (parseS Subscription) Cancel() {
	if parseS.cancel != nil {
		parseS.cancel()
	}
}

// WindowEnv exposes shared values attached directly to the browser window object.
// It is intended for simple cross-surface configuration such as mount selectors,
// bootstrap flags, and other host-provided runtime settings.
type WindowEnv struct {
	lookup func(string) (Value, bool)
}

// Lookup returns the raw shared window value when present.
func (parseE WindowEnv) Lookup(parseName string) (Value, bool) {
	if parseE.lookup == nil {
		return Value{}, false
	}
	return parseE.lookup(parseName)
}

// LookupString resolves a shared window value as a normalized string.
// Empty strings and JavaScript stringified nullish sentinel values are treated as missing.
func (parseE WindowEnv) LookupString(parseName string) (string, bool) {
	parseValue, parseOk := parseE.Lookup(parseName)
	if !parseOk {
		return "", false
	}
	parseResolved := strings.TrimSpace(parseValue.String())
	if parseResolved == "" || parseResolved == "<undefined>" || parseResolved == "<null>" {
		return "", false
	}
	return parseResolved, true
}

// String returns a normalized shared window string or the provided fallback.
func (parseE WindowEnv) String(parseName string, parseFallback string) string {
	if parseResolved, parseOk := parseE.LookupString(parseName); parseOk {
		return parseResolved
	}
	return parseFallback
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

func (parseS Storage) GetItem(parseKey string) (string, bool, error) {
	if parseS.getItem == nil {
		return "", false, unavailable("Storage.GetItem", "")
	}
	return parseS.getItem(parseKey)
}

func (parseS Storage) SetItem(parseKey string, parseValue string) error {
	if parseS.setItem == nil {
		return unavailable("Storage.SetItem", "")
	}
	return parseS.setItem(parseKey, parseValue)
}

func (parseS Storage) GetMany(parseKeys ...string) (map[string]string, error) {
	if len(parseKeys) == 0 {
		return map[string]string{}, nil
	}
	if parseS.getMany != nil {
		return parseS.getMany(parseKeys)
	}
	parseValues := make(map[string]string, len(parseKeys))
	for _, parseKey := range parseKeys {
		parseValue, parseOk, parseErr := parseS.GetItem(parseKey)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseOk {
			parseValues[parseKey] = parseValue
		}
	}
	return parseValues, nil
}

func (parseS Storage) RemoveItem(parseKey string) error {
	if parseS.removeItem == nil {
		return unavailable("Storage.RemoveItem", "")
	}
	return parseS.removeItem(parseKey)
}

func (parseS Storage) Clear() error {
	if parseS.clear == nil {
		return unavailable("Storage.Clear", "")
	}
	return parseS.clear()
}

func (parseS Storage) Len() (int, error) {
	if parseS.length == nil {
		return 0, unavailable("Storage.Len", "")
	}
	return parseS.length()
}

func (parseS Storage) Key(parseIndex int) (string, bool, error) {
	if parseS.key == nil {
		return "", false, unavailable("Storage.Key", "")
	}
	return parseS.key(parseIndex)
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

func (parseS PersistentStore) Backend() string {
	if parseS.backend == nil {
		return ""
	}
	return parseS.backend()
}

func (parseS PersistentStore) GetItem(parseCtx context.Context, parseKey string) (string, bool, error) {
	if parseS.getItem == nil {
		return "", false, unavailable("PersistentStore.GetItem", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseS.getItem(parseCtx, parseKey)
}

func (parseS PersistentStore) GetMany(parseCtx context.Context, parseKeys ...string) (map[string]string, error) {
	if len(parseKeys) == 0 {
		return map[string]string{}, nil
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseValues := make(map[string]string, len(parseKeys))
	for _, parseKey := range parseKeys {
		parseValue, parseOk, parseErr := parseS.GetItem(parseCtx, parseKey)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseOk {
			parseValues[parseKey] = parseValue
		}
	}
	return parseValues, nil
}

func (parseS PersistentStore) SetItem(parseCtx context.Context, parseKey string, parseValue string) error {
	if parseS.setItem == nil {
		return unavailable("PersistentStore.SetItem", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseS.setItem(parseCtx, parseKey, parseValue)
}

func (parseS PersistentStore) SetJSON(parseCtx context.Context, parseKey string, parseValue any) error {
	parseData, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return wrapError("PersistentStore.SetJSON", parseKey, CodeEncode, parseErr)
	}
	return parseS.SetItem(parseCtx, parseKey, string(parseData))
}

func (parseS PersistentStore) DecodeJSON(parseCtx context.Context, parseKey string, parseTarget any) (bool, error) {
	if parseTarget == nil {
		return false, wrapError("PersistentStore.DecodeJSON", parseKey, CodeInvalid, errors.New("target is nil"))
	}
	parseValue, parseOk, parseErr := parseS.GetItem(parseCtx, parseKey)
	if parseErr != nil || !parseOk {
		return parseOk, parseErr
	}
	if parseErr2 := json.Unmarshal([]byte(parseValue), parseTarget); parseErr2 != nil {
		return false, wrapError("PersistentStore.DecodeJSON", parseKey, CodeDecode, parseErr2)
	}
	return true, nil
}

func (parseS PersistentStore) RemoveItem(parseCtx context.Context, parseKey string) error {
	if parseS.removeItem == nil {
		return unavailable("PersistentStore.RemoveItem", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseS.removeItem(parseCtx, parseKey)
}

func (parseS PersistentStore) Clear(parseCtx context.Context) error {
	if parseS.clear == nil {
		return unavailable("PersistentStore.Clear", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseS.clear(parseCtx)
}

func (parseS PersistentStore) Keys(parseCtx context.Context) ([]string, error) {
	if parseS.keys == nil {
		return nil, unavailable("PersistentStore.Keys", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseS.keys(parseCtx)
}

func (parseS PersistentStore) Len(parseCtx context.Context) (int, error) {
	if parseS.length == nil {
		return 0, unavailable("PersistentStore.Len", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseS.length(parseCtx)
}

func (parseS PersistentStore) Close() error {
	if parseS.close == nil {
		return nil
	}
	return parseS.close()
}

func LoadPersistentJSON[T any](parseCtx context.Context, store PersistentStore, parseKey string) (T, bool, error) {
	var parseValue T
	parseOk, parseErr := store.DecodeJSON(parseCtx, parseKey, &parseValue)
	if parseErr != nil || !parseOk {
		return parseValue, parseOk, parseErr
	}
	return parseValue, true, nil
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

func (parseL Location) Href() string {
	if parseL.href == nil {
		return ""
	}
	return parseL.href()
}

func (parseL Location) Pathname() string {
	if parseL.pathname == nil {
		return ""
	}
	return parseL.pathname()
}

func (parseL Location) Search() string {
	if parseL.search == nil {
		return ""
	}
	return parseL.search()
}

func (parseL Location) Hash() string {
	if parseL.hash == nil {
		return ""
	}
	return parseL.hash()
}

func (parseL Location) Origin() string {
	if parseL.origin == nil {
		return ""
	}
	return parseL.origin()
}

func (parseL Location) Assign(parseRawURL string) error {
	if parseL.assign == nil {
		return unavailable("Location.Assign", "")
	}
	return parseL.assign(parseRawURL)
}

func (parseL Location) Replace(parseRawURL string) error {
	if parseL.replace == nil {
		return unavailable("Location.Replace", "")
	}
	return parseL.replace(parseRawURL)
}

func (parseL Location) Reload() error {
	if parseL.reload == nil {
		return unavailable("Location.Reload", "")
	}
	return parseL.reload()
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

func (parseH History) Len() (int, error) {
	if parseH.length == nil {
		return 0, unavailable("History.Len", "")
	}
	return parseH.length()
}

func (parseH History) State() (any, error) {
	if parseH.state == nil {
		return nil, unavailable("History.State", "")
	}
	return parseH.state()
}

func (parseH History) Back() error {
	if parseH.back == nil {
		return unavailable("History.Back", "")
	}
	return parseH.back()
}

func (parseH History) Forward() error {
	if parseH.forward == nil {
		return unavailable("History.Forward", "")
	}
	return parseH.forward()
}

func (parseH History) Go(parseDelta int) error {
	if parseH.goDelta == nil {
		return unavailable("History.Go", "")
	}
	return parseH.goDelta(parseDelta)
}

func (parseH History) PushState(parseState any, parseTitle string, parseRawURL string) error {
	if parseH.pushState == nil {
		return unavailable("History.PushState", "")
	}
	return parseH.pushState(parseState, parseTitle, parseRawURL)
}

func (parseH History) ReplaceState(parseState any, parseTitle string, parseRawURL string) error {
	if parseH.replaceState == nil {
		return unavailable("History.ReplaceState", "")
	}
	return parseH.replaceState(parseState, parseTitle, parseRawURL)
}

type Clipboard struct {
	writeText func(context.Context, string) error
	readText  func(context.Context) (string, error)
}

func (parseC Clipboard) WriteText(parseCtx context.Context, parseText string) error {
	if parseC.writeText == nil {
		return unavailable("Clipboard.WriteText", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseC.writeText(parseCtx, parseText)
}

func (parseC Clipboard) ReadText(parseCtx context.Context) (string, error) {
	if parseC.readText == nil {
		return "", unavailable("Clipboard.ReadText", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseC.readText(parseCtx)
}

type Timer struct {
	cancel func() error
}

func (parseT Timer) Cancel() error {
	if parseT.cancel == nil {
		return unavailable("Timer.Cancel", "")
	}
	return parseT.cancel()
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

func (parseT EventTarget) Dispatch(parseName string, parseDetail any) error {
	if parseT.dispatch == nil {
		return unavailable("EventTarget.Dispatch", "")
	}
	return parseT.dispatch(parseName, parseDetail)
}

func (parseT EventTarget) Listen(parseName string, parseHandler func(BrowserEvent)) (Subscription, error) {
	if parseT.listen == nil {
		return Subscription{}, unavailable("EventTarget.Listen", "")
	}
	return parseT.listen(parseName, parseHandler)
}

func (parseT EventTarget) Subscribe(parseName string, parseHandler func(CustomEvent)) (Subscription, error) {
	if parseT.subscribe != nil {
		return parseT.subscribe(parseName, parseHandler)
	}
	if parseT.listen == nil {
		return Subscription{}, unavailable("EventTarget.Subscribe", "")
	}
	return parseT.listen(parseName, func(parseEvent BrowserEvent) {
		parseHandler(CustomEvent{
			Type:   parseEvent.Type,
			Detail: parseEvent.Detail,
		})
	})
}

// DecodeCustomEvent projects a custom-event detail payload into a typed value.
func DecodeCustomEvent[T any](parseEvent CustomEvent) (DecodedCustomEvent[T], error) {
	var parseDetail T
	if parseErr := Decode(parseEvent.Detail, &parseDetail); parseErr != nil {
		return DecodedCustomEvent[T]{Type: parseEvent.Type}, wrapError("DecodeCustomEvent", parseEvent.Type, CodeDecode, parseErr)
	}
	return DecodedCustomEvent[T]{
		Type:   parseEvent.Type,
		Detail: parseDetail,
	}, nil
}

// SubscribeDecoded decodes custom-event detail payloads before invoking the handler.
func SubscribeDecoded[T any](parseTarget EventTarget, parseName string, parseHandler func(DecodedCustomEvent[T], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeDecoded", parseName, CodeInvalid, errors.New("handler is nil"))
	}
	return parseTarget.Subscribe(parseName, func(parseEvent CustomEvent) {
		parseDecoded, parseErr := DecodeCustomEvent[T](parseEvent)
		parseHandler(parseDecoded, parseErr)
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

func (parseM MediaQueryList) Matches() bool {
	if parseM.matches == nil {
		return false
	}
	return parseM.matches()
}

func (parseM MediaQueryList) Media() string {
	if parseM.media == nil {
		return ""
	}
	return parseM.media()
}

func (parseM MediaQueryList) Subscribe(parseHandler func(MediaQueryEvent)) (Subscription, error) {
	if parseM.subscribe == nil {
		return Subscription{}, unavailable("MediaQueryList.Subscribe", "")
	}
	return parseM.subscribe(parseHandler)
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

func (parseE Element) TagName() string {
	_ = parseE.raw
	if parseE.tagName == nil {
		return ""
	}
	return parseE.tagName()
}

func (parseE Element) ID() string {
	if parseE.id == nil {
		return ""
	}
	return parseE.id()
}

func (parseE Element) ClassName() string {
	if parseE.className == nil {
		return ""
	}
	return parseE.className()
}

func (parseE Element) Focus() error {
	if parseE.focus == nil {
		return unavailable("Element.Focus", "")
	}
	return parseE.focus()
}

func (parseE Element) Blur() error {
	if parseE.blur == nil {
		return unavailable("Element.Blur", "")
	}
	return parseE.blur()
}

func (parseE Element) Click() error {
	if parseE.click == nil {
		return unavailable("Element.Click", "")
	}
	return parseE.click()
}

func (parseE Element) SetScrollTop(parseScrollTop float64) error {
	if parseE.setScrollTop == nil {
		return unavailable("Element.SetScrollTop", "")
	}
	return parseE.setScrollTop(parseScrollTop)
}

func (parseE Element) ScrollIntoView(parseOptions ...ScrollIntoViewOptions) error {
	if parseE.scrollIntoView == nil {
		return unavailable("Element.ScrollIntoView", "")
	}
	var parseResolved ScrollIntoViewOptions
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	return parseE.scrollIntoView(parseResolved)
}

func (parseE Element) BoundingClientRect() (Rect, error) {
	if parseE.boundingClientRect == nil {
		return Rect{}, unavailable("Element.BoundingClientRect", "")
	}
	return parseE.boundingClientRect()
}

func (parseE Element) Events() (EventTarget, error) {
	if parseE.events == nil {
		return EventTarget{}, unavailable("Element.Events", "")
	}
	return parseE.events()
}

func (parseE Element) Listen(parseName string, parseHandler func(BrowserEvent)) (Subscription, error) {
	parseTarget, parseErr := parseE.Events()
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	return parseTarget.Listen(parseName, parseHandler)
}

func (parseE Element) Subscribe(parseName string, parseHandler func(CustomEvent)) (Subscription, error) {
	parseTarget, parseErr := parseE.Events()
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	return parseTarget.Subscribe(parseName, parseHandler)
}

func (parseE Element) Dispatch(parseName string, parseDetail any) error {
	parseTarget, parseErr := parseE.Events()
	if parseErr != nil {
		return parseErr
	}
	return parseTarget.Dispatch(parseName, parseDetail)
}

func (parseE Element) ObserveResize(parseHandler func(ResizeEntry)) (Subscription, error) {
	if parseE.observeResize == nil {
		return Subscription{}, unavailable("Element.ObserveResize", "")
	}
	return parseE.observeResize(parseHandler)
}

func (parseE Element) ObserveIntersection(parseHandler func(IntersectionEntry), parseOptions ...IntersectionObserverOptions) (Subscription, error) {
	if parseE.observeIntersection == nil {
		return Subscription{}, unavailable("Element.ObserveIntersection", "")
	}
	var parseResolved IntersectionObserverOptions
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	return parseE.observeIntersection(parseResolved, parseHandler)
}

// ScrollMetrics returns the scrollTop, scrollHeight, and clientHeight of the
// element — the three values needed to determine scroll position within a
// scrollable container.
func (parseE Element) ScrollMetrics() (parseScrollTop, parseScrollHeight, parseClientHeight float64, parseErr error) {
	if parseE.scrollMetrics == nil {
		return 0, 0, 0, unavailable("Element.ScrollMetrics", "")
	}
	return parseE.scrollMetrics()
}

type Document struct {
	elementByID   func(string) (Element, bool, error)
	elementsByID  func([]string) (map[string]Element, error)
	querySelector func(string) (Element, bool, error)
}

func (parseD Document) ElementByID(parseId string) (Element, bool, error) {
	if parseD.elementByID == nil {
		return Element{}, false, unavailable("Document.ElementByID", "")
	}
	return parseD.elementByID(parseId)
}

func (parseD Document) QuerySelector(parseSelector string) (Element, bool, error) {
	if parseD.querySelector == nil {
		return Element{}, false, unavailable("Document.QuerySelector", "")
	}
	return parseD.querySelector(parseSelector)
}

func (parseD Document) ElementsByID(parseIds ...string) (map[string]Element, error) {
	if len(parseIds) == 0 {
		return map[string]Element{}, nil
	}
	if parseD.elementsByID != nil {
		return parseD.elementsByID(parseIds)
	}
	parseValues := make(map[string]Element, len(parseIds))
	for _, parseId := range parseIds {
		parseElement, parseOk, parseErr := parseD.ElementByID(parseId)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseOk {
			parseValues[parseId] = parseElement
		}
	}
	return parseValues, nil
}

type Module struct {
	call        func(context.Context, string, ...any) (any, error)
	callDefault func(context.Context, ...any) (any, error)
	value       func(context.Context, string) (any, error)
	dispose     func() error
}

func (parseM Module) Call(parseCtx context.Context, parseExport string, parseArgs ...any) (any, error) {
	if parseM.call == nil {
		return nil, unavailable("Module.Call", parseExport)
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseM.call(parseCtx, parseExport, parseArgs...)
}

func (parseM Module) CallDefault(parseCtx context.Context, parseArgs ...any) (any, error) {
	if parseM.callDefault == nil {
		return nil, unavailable("Module.CallDefault", "default")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseM.callDefault(parseCtx, parseArgs...)
}

func (parseM Module) Value(parseCtx context.Context, parseExport string) (any, error) {
	if parseM.value == nil {
		return nil, unavailable("Module.Value", parseExport)
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseM.value(parseCtx, parseExport)
}

func (parseM Module) Dispose() error {
	if parseM.dispose == nil {
		return unavailable("Module.Dispose", "")
	}
	return parseM.dispose()
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

// SharedMemorySupport reports whether the current browser context can use the
// shared-memory worker path.
type SharedMemorySupport struct {
	IsCrossOriginIsolated bool
	HasSharedArrayBuffer  bool
	HasAtomics            bool
	CanUseSharedMemory    bool
}

// SharedBuffer wraps a browser SharedArrayBuffer with byte access and int32
// atomic helpers for worker coordination.
type SharedBuffer struct {
	raw                  interface{}
	getByteLength        func() int
	readBytes            func(int, []byte) (int, error)
	writeBytes           func(int, []byte) (int, error)
	getInt32Length       func() int
	loadInt32            func(int) (int32, error)
	storeInt32           func(int, int32) error
	addInt32             func(int, int32) (int32, error)
	subInt32             func(int, int32) (int32, error)
	andInt32             func(int, int32) (int32, error)
	orInt32              func(int, int32) (int32, error)
	xorInt32             func(int, int32) (int32, error)
	exchangeInt32        func(int, int32) (int32, error)
	compareExchangeInt32 func(int, int32, int32) (int32, error)
	waitInt32            func(int, int32, time.Duration) (string, error)
	notifyInt32          func(int, int) (int, error)
}

// MessagePortMessage carries one payload received on a MessagePort together
// with any transferred ports attached to the same event.
type MessagePortMessage struct {
	Payload any
	Ports   []MessagePort
}

// DecodedMessagePortMessage carries a typed payload received on a MessagePort
// together with any transferred ports attached to the same event.
type DecodedMessagePortMessage[T any] struct {
	Payload T
	Ports   []MessagePort
}

type WorkerMessage struct {
	ID      string `json:"id"`
	Phase   string `json:"phase"`
	Name    string `json:"name"`
	Payload any    `json:"payload"`
	Error   string `json:"error,omitempty"`
	Ports   []MessagePort
}

type DecodedWorkerMessage[T any] struct {
	ID      string
	Phase   string
	Name    string
	Payload T
	Error   string
	Ports   []MessagePort
}

type Worker struct {
	post      func(any) error
	postPorts func(any, ...MessagePort) error
	subscribe func(func(WorkerMessage, error)) (Subscription, error)
	request   func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error)
	terminate func() error
	restart   func(context.Context) error
}

type WorkerScope struct {
	post      func(WorkerMessage) error
	postPorts func(WorkerMessage, ...MessagePort) error
	subscribe func(func(WorkerMessage, error)) (Subscription, error)
}

// MessagePort wraps a browser MessagePort with post, subscribe, and close
// helpers suitable for worker-owned subchannels.
type MessagePort struct {
	raw       interface{}
	post      func(any) error
	postPorts func(any, ...MessagePort) error
	subscribe func(func(MessagePortMessage, error)) (Subscription, error)
	close     func() error
}

// MessageChannel wraps the two linked endpoints produced by the browser
// MessageChannel constructor.
type MessageChannel struct {
	port1 MessagePort
	port2 MessagePort
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

// GetSharedBufferRaw returns the platform-specific shared-buffer handle.
func (parseB SharedBuffer) GetSharedBufferRaw() interface{} {
	return parseB.raw
}

// GetByteLength returns the SharedArrayBuffer length in bytes.
func (parseB SharedBuffer) GetByteLength() int {
	if parseB.getByteLength == nil {
		return 0
	}
	return parseB.getByteLength()
}

// ReadBytes copies shared-memory bytes starting at offset into dest.
func (parseB SharedBuffer) ReadBytes(parseOffset int, parseDest []byte) (int, error) {
	if parseB.readBytes == nil {
		return 0, unavailable("SharedBuffer.ReadBytes", "")
	}
	return parseB.readBytes(parseOffset, parseDest)
}

// WriteBytes copies source bytes into shared memory starting at offset.
func (parseB SharedBuffer) WriteBytes(parseOffset int, parseSource []byte) (int, error) {
	if parseB.writeBytes == nil {
		return 0, unavailable("SharedBuffer.WriteBytes", "")
	}
	return parseB.writeBytes(parseOffset, parseSource)
}

// GetInt32Length returns the number of addressable int32 slots in the shared
// buffer.
func (parseB SharedBuffer) GetInt32Length() int {
	if parseB.getInt32Length == nil {
		return 0
	}
	return parseB.getInt32Length()
}

// LoadInt32 atomically reads the int32 value at index.
func (parseB SharedBuffer) LoadInt32(parseIndex int) (int32, error) {
	if parseB.loadInt32 == nil {
		return 0, unavailable("SharedBuffer.LoadInt32", "")
	}
	return parseB.loadInt32(parseIndex)
}

// StoreInt32 atomically writes value to the int32 slot at index.
func (parseB SharedBuffer) StoreInt32(parseIndex int, parseValue int32) error {
	if parseB.storeInt32 == nil {
		return unavailable("SharedBuffer.StoreInt32", "")
	}
	return parseB.storeInt32(parseIndex, parseValue)
}

// AddInt32 atomically adds delta to the int32 slot at index and returns the
// previous value.
func (parseB SharedBuffer) AddInt32(parseIndex int, parseDelta int32) (int32, error) {
	if parseB.addInt32 == nil {
		return 0, unavailable("SharedBuffer.AddInt32", "")
	}
	return parseB.addInt32(parseIndex, parseDelta)
}

// SubInt32 atomically subtracts delta from the int32 slot at index and returns
// the previous value.
func (parseB SharedBuffer) SubInt32(parseIndex int, parseDelta int32) (int32, error) {
	if parseB.subInt32 == nil {
		return 0, unavailable("SharedBuffer.SubInt32", "")
	}
	return parseB.subInt32(parseIndex, parseDelta)
}

// AndInt32 atomically ANDs mask with the int32 slot at index and returns the
// previous value.
func (parseB SharedBuffer) AndInt32(parseIndex int, parseMask int32) (int32, error) {
	if parseB.andInt32 == nil {
		return 0, unavailable("SharedBuffer.AndInt32", "")
	}
	return parseB.andInt32(parseIndex, parseMask)
}

// OrInt32 atomically ORs mask with the int32 slot at index and returns the
// previous value.
func (parseB SharedBuffer) OrInt32(parseIndex int, parseMask int32) (int32, error) {
	if parseB.orInt32 == nil {
		return 0, unavailable("SharedBuffer.OrInt32", "")
	}
	return parseB.orInt32(parseIndex, parseMask)
}

// XorInt32 atomically XORs mask with the int32 slot at index and returns the
// previous value.
func (parseB SharedBuffer) XorInt32(parseIndex int, parseMask int32) (int32, error) {
	if parseB.xorInt32 == nil {
		return 0, unavailable("SharedBuffer.XorInt32", "")
	}
	return parseB.xorInt32(parseIndex, parseMask)
}

// ExchangeInt32 atomically swaps value into the int32 slot at index and
// returns the previous value.
func (parseB SharedBuffer) ExchangeInt32(parseIndex int, parseValue int32) (int32, error) {
	if parseB.exchangeInt32 == nil {
		return 0, unavailable("SharedBuffer.ExchangeInt32", "")
	}
	return parseB.exchangeInt32(parseIndex, parseValue)
}

// CompareExchangeInt32 atomically swaps newValue into the int32 slot at index
// when the current value equals oldValue, and returns the previous value.
func (parseB SharedBuffer) CompareExchangeInt32(parseIndex int, parseOldValue int32, parseNewValue int32) (int32, error) {
	if parseB.compareExchangeInt32 == nil {
		return 0, unavailable("SharedBuffer.CompareExchangeInt32", "")
	}
	return parseB.compareExchangeInt32(parseIndex, parseOldValue, parseNewValue)
}

// WaitInt32 blocks in a worker context until the int32 slot at index changes
// from expected or the optional timeout expires.
func (parseB SharedBuffer) WaitInt32(parseIndex int, parseExpected int32, parseTimeout time.Duration) (string, error) {
	if parseB.waitInt32 == nil {
		return "", unavailable("SharedBuffer.WaitInt32", "")
	}
	return parseB.waitInt32(parseIndex, parseExpected, parseTimeout)
}

// NotifyInt32 wakes blocked waiters for the int32 slot at index and returns the
// number of workers notified.
func (parseB SharedBuffer) NotifyInt32(parseIndex int, parseCount int) (int, error) {
	if parseB.notifyInt32 == nil {
		return 0, unavailable("SharedBuffer.NotifyInt32", "")
	}
	return parseB.notifyInt32(parseIndex, parseCount)
}

func (parseC CrossTabChannel) Name() string {
	if parseC.name == nil {
		return ""
	}
	return parseC.name()
}

func (parseC CrossTabChannel) Transport() string {
	if parseC.transport == nil {
		return ""
	}
	return parseC.transport()
}

func (parseC CrossTabChannel) Publish(parsePayload any) error {
	if parseC.publish == nil {
		return unavailable("CrossTabChannel.Publish", "")
	}
	return parseC.publish(parsePayload)
}

func (parseC CrossTabChannel) Subscribe(parseHandler func(CrossTabEnvelope, error)) (Subscription, error) {
	if parseC.subscribe == nil {
		return Subscription{}, unavailable("CrossTabChannel.Subscribe", "")
	}
	return parseC.subscribe(parseHandler)
}

func (parseC CrossTabChannel) Close() error {
	if parseC.close == nil {
		return unavailable("CrossTabChannel.Close", "")
	}
	return parseC.close()
}

func (parseC WindowChannel) Name() string {
	if parseC.name == nil {
		return ""
	}
	return parseC.name()
}

func (parseC WindowChannel) TargetOrigin() string {
	if parseC.targetOrigin == nil {
		return ""
	}
	return parseC.targetOrigin()
}

func (parseC WindowChannel) Publish(parsePayload any) error {
	if parseC.publish == nil {
		return unavailable("WindowChannel.Publish", "")
	}
	return parseC.publish(parsePayload)
}

func (parseC WindowChannel) Subscribe(parseHandler func(WindowEnvelope, error)) (Subscription, error) {
	if parseC.subscribe == nil {
		return Subscription{}, unavailable("WindowChannel.Subscribe", "")
	}
	return parseC.subscribe(parseHandler)
}

func (parseC WindowChannel) Focus() error {
	if parseC.focus == nil {
		return unavailable("WindowChannel.Focus", "")
	}
	return parseC.focus()
}

func (parseC WindowChannel) Close() error {
	if parseC.close == nil {
		return unavailable("WindowChannel.Close", "")
	}
	return parseC.close()
}

func (parseC WindowChannel) Closed() bool {
	if parseC.closed == nil {
		return false
	}
	return parseC.closed()
}

func (parseW Worker) Post(parseMessage any) error {
	if parseW.post == nil {
		return unavailable("Worker.Post", "")
	}
	return parseW.post(parseMessage)
}

// PostPorts sends a payload to the worker together with transferred
// MessagePorts.
func (parseW Worker) PostPorts(parseMessage any, parsePorts ...MessagePort) error {
	if parseW.postPorts == nil {
		return unavailable("Worker.PostPorts", "")
	}
	return parseW.postPorts(parseMessage, parsePorts...)
}

func (parseW Worker) Subscribe(parseHandler func(WorkerMessage, error)) (Subscription, error) {
	if parseW.subscribe == nil {
		return Subscription{}, unavailable("Worker.Subscribe", "")
	}
	return parseW.subscribe(parseHandler)
}

func (parseW Worker) Request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	if parseW.request == nil {
		return WorkerMessage{}, unavailable("Worker.Request", parseName)
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseW.request(parseCtx, parseName, parsePayload, parseOnProgress)
}

func (parseW Worker) Terminate() error {
	if parseW.terminate == nil {
		return unavailable("Worker.Terminate", "")
	}
	return parseW.terminate()
}

func (parseW Worker) Restart(parseCtx context.Context) error {
	if parseW.restart == nil {
		return unavailable("Worker.Restart", "")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseW.restart(parseCtx)
}

func (parseW WorkerScope) Post(parseMessage WorkerMessage) error {
	if parseW.post == nil {
		return unavailable("WorkerScope.Post", "")
	}
	return parseW.post(parseMessage)
}

// PostPorts sends a worker-scope message together with transferred
// MessagePorts.
func (parseW WorkerScope) PostPorts(parseMessage WorkerMessage, parsePorts ...MessagePort) error {
	if parseW.postPorts == nil {
		return unavailable("WorkerScope.PostPorts", "")
	}
	return parseW.postPorts(parseMessage, parsePorts...)
}

func (parseW WorkerScope) Subscribe(parseHandler func(WorkerMessage, error)) (Subscription, error) {
	if parseW.subscribe == nil {
		return Subscription{}, unavailable("WorkerScope.Subscribe", "")
	}
	return parseW.subscribe(parseHandler)
}

// Port1 returns the first endpoint of the message channel.
func (parseC MessageChannel) Port1() MessagePort {
	return parseC.port1
}

// Port2 returns the second endpoint of the message channel.
func (parseC MessageChannel) Port2() MessagePort {
	return parseC.port2
}

// GetMessagePortRaw returns the platform-specific message-port handle.
func (parseP MessagePort) GetMessagePortRaw() interface{} {
	return parseP.raw
}

// Post sends a payload over the message port.
func (parseP MessagePort) Post(parsePayload any) error {
	if parseP.post == nil {
		return unavailable("MessagePort.Post", "")
	}
	return parseP.post(parsePayload)
}

// PostPorts sends a payload over the message port together with transferred
// MessagePorts.
func (parseP MessagePort) PostPorts(parsePayload any, parsePorts ...MessagePort) error {
	if parseP.postPorts == nil {
		return unavailable("MessagePort.PostPorts", "")
	}
	return parseP.postPorts(parsePayload, parsePorts...)
}

// Subscribe receives payloads from the message port.
func (parseP MessagePort) Subscribe(parseHandler func(MessagePortMessage, error)) (Subscription, error) {
	if parseP.subscribe == nil {
		return Subscription{}, unavailable("MessagePort.Subscribe", "")
	}
	return parseP.subscribe(parseHandler)
}

// Close closes the message port.
func (parseP MessagePort) Close() error {
	if parseP.close == nil {
		return unavailable("MessagePort.Close", "")
	}
	return parseP.close()
}

func (parseW WorkerScope) Ready(parseName string) error {
	return parseW.Post(WorkerMessage{Phase: "ready", Name: parseName})
}

func (parseW WorkerScope) Message(parseName string, parsePayload any) error {
	return parseW.Post(WorkerMessage{Phase: "message", Name: parseName, Payload: parsePayload})
}

func (parseW WorkerScope) Progress(parseId string, parseName string, parsePayload any) error {
	return parseW.Post(WorkerMessage{ID: parseId, Phase: "progress", Name: parseName, Payload: parsePayload})
}

func (parseW WorkerScope) Result(parseId string, parseName string, parsePayload any) error {
	return parseW.Post(WorkerMessage{ID: parseId, Phase: "result", Name: parseName, Payload: parsePayload})
}

func (parseW WorkerScope) Error(parseId string, parseName string, parseErrText string, parsePayload any) error {
	return parseW.Post(WorkerMessage{ID: parseId, Phase: "error", Name: parseName, Error: parseErrText, Payload: parsePayload})
}

func DecodeWorkerMessage[T any](parseMessage WorkerMessage) (DecodedWorkerMessage[T], error) {
	var parsePayload T
	if parseErr := Decode(parseMessage.Payload, &parsePayload); parseErr != nil {
		return DecodedWorkerMessage[T]{
			ID:    parseMessage.ID,
			Phase: parseMessage.Phase,
			Name:  parseMessage.Name,
			Error: parseMessage.Error,
			Ports: parseMessage.Ports,
		}, wrapError("DecodeWorkerMessage", parseMessage.Name, CodeDecode, parseErr)
	}
	return DecodedWorkerMessage[T]{
		ID:      parseMessage.ID,
		Phase:   parseMessage.Phase,
		Name:    parseMessage.Name,
		Payload: parsePayload,
		Error:   parseMessage.Error,
		Ports:   parseMessage.Ports,
	}, nil
}

// DecodeMessagePortMessage projects a message-port payload into a typed value.
func DecodeMessagePortMessage[T any](parseMessage MessagePortMessage) (DecodedMessagePortMessage[T], error) {
	var parsePayload T
	if parseErr := Decode(parseMessage.Payload, &parsePayload); parseErr != nil {
		return DecodedMessagePortMessage[T]{
			Ports: parseMessage.Ports,
		}, wrapError("DecodeMessagePortMessage", "", CodeDecode, parseErr)
	}
	return DecodedMessagePortMessage[T]{
		Payload: parsePayload,
		Ports:   parseMessage.Ports,
	}, nil
}

func SubscribeDecodedWorker[T any](parseWorker Worker, parseHandler func(DecodedWorkerMessage[T], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeDecodedWorker", "", CodeInvalid, errors.New("handler is nil"))
	}
	return parseWorker.Subscribe(func(parseMessage WorkerMessage, parseErr error) {
		if parseErr != nil {
			parseHandler(DecodedWorkerMessage[T]{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeWorkerMessage[T](parseMessage)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

// RequestWorkerDecoded issues a typed request against any worker-compatible
// requester surface and decodes both progress and final result payloads.
func RequestWorkerDecoded[Req any, Progress any, Result any](parseCtx context.Context, parseWorker WorkerRequester, parseName string, parsePayload Req, parseOnProgress func(DecodedWorkerMessage[Progress], error)) (Result, error) {
	var parseZero Result
	parseResponse, parseErr := parseWorker.Request(parseCtx, parseName, parsePayload, func(parseMessage WorkerMessage, parseMessageErr error) {
		if parseOnProgress == nil {
			return
		}
		if parseMessageErr != nil {
			parseOnProgress(DecodedWorkerMessage[Progress]{}, parseMessageErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeWorkerMessage[Progress](parseMessage)
		parseOnProgress(parseDecoded, parseDecodeErr)
	})
	if parseErr != nil {
		return parseZero, parseErr
	}
	parseDecoded2, parseErr := DecodeWorkerMessage[Result](parseResponse)
	if parseErr != nil {
		return parseZero, parseErr
	}
	return parseDecoded2.Payload, nil
}

// SubscribeDecodedMessagePort receives typed payloads from a MessagePort.
func SubscribeDecodedMessagePort[T any](parsePort MessagePort, parseHandler func(DecodedMessagePortMessage[T], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeDecodedMessagePort", "", CodeInvalid, errors.New("handler is nil"))
	}
	return parsePort.Subscribe(func(parseMessage MessagePortMessage, parseErr error) {
		if parseErr != nil {
			parseHandler(DecodedMessagePortMessage[T]{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeMessagePortMessage[T](parseMessage)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

func DecodeCrossTabEnvelope[T any](parseMessage CrossTabEnvelope) (DecodedCrossTabEnvelope[T], error) {
	var parsePayload T
	if parseErr := Decode(parseMessage.Payload, &parsePayload); parseErr != nil {
		return DecodedCrossTabEnvelope[T]{
			Name:     parseMessage.Name,
			Source:   parseMessage.Source,
			Sequence: parseMessage.Sequence,
			SentAt:   parseMessage.SentAt,
		}, wrapError("DecodeCrossTabEnvelope", parseMessage.Name, CodeDecode, parseErr)
	}
	return DecodedCrossTabEnvelope[T]{
		Name:     parseMessage.Name,
		Payload:  parsePayload,
		Source:   parseMessage.Source,
		Sequence: parseMessage.Sequence,
		SentAt:   parseMessage.SentAt,
	}, nil
}

func SubscribeDecodedCrossTab[T any](parseChannel CrossTabChannel, parseHandler func(DecodedCrossTabEnvelope[T], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeDecodedCrossTab", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return parseChannel.Subscribe(func(parseMessage CrossTabEnvelope, parseErr error) {
		if parseErr != nil {
			parseHandler(DecodedCrossTabEnvelope[T]{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeCrossTabEnvelope[T](parseMessage)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

func DecodeWindowEnvelope[T any](parseMessage WindowEnvelope) (DecodedWindowEnvelope[T], error) {
	var parsePayload T
	if parseErr := Decode(parseMessage.Payload, &parsePayload); parseErr != nil {
		return DecodedWindowEnvelope[T]{
			Name:   parseMessage.Name,
			Source: parseMessage.Source,
			SentAt: parseMessage.SentAt,
		}, wrapError("DecodeWindowEnvelope", parseMessage.Name, CodeDecode, parseErr)
	}
	return DecodedWindowEnvelope[T]{
		Name:    parseMessage.Name,
		Payload: parsePayload,
		Source:  parseMessage.Source,
		SentAt:  parseMessage.SentAt,
	}, nil
}

// DecodeClientMessage decodes a ClientMessage from an interop payload value.
func DecodeClientMessage(parseValue any) (ClientMessage, error) {
	parseMessage, parseErr := decodeClientMessageValue(parseValue)
	if parseErr != nil {
		return ClientMessage{}, parseErr
	}
	if parseErr2 := validateClientMessage("DecodeClientMessage", parseMessage.Topic, parseMessage); parseErr2 != nil {
		return ClientMessage{}, parseErr2
	}
	return parseMessage, nil
}

// PublishClientMessage encodes and sends a ClientMessage over a CrossTabChannel.
func PublishClientMessage(parseChannel CrossTabChannel, parseMessage ClientMessage) error {
	parsePrepared, parseErr := prepareClientMessage("PublishClientMessage", parseChannel.Name(), parseMessage)
	if parseErr != nil {
		return parseErr
	}
	return parseChannel.Publish(parsePrepared)
}

// PublishClientWindowMessage encodes and sends a ClientMessage over a WindowChannel.
func PublishClientWindowMessage(parseChannel WindowChannel, parseMessage ClientMessage) error {
	parsePrepared, parseErr := prepareClientMessage("PublishClientWindowMessage", parseChannel.Name(), parseMessage)
	if parseErr != nil {
		return parseErr
	}
	return parseChannel.Publish(parsePrepared)
}

// SubscribeClientMessages decodes incoming CrossTabChannel envelopes as ClientMessages and invokes handler.
func SubscribeClientMessages(parseChannel CrossTabChannel, parseHandler func(ClientMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeClientMessages", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return parseChannel.Subscribe(func(parseMessage CrossTabEnvelope, parseErr error) {
		if parseErr != nil {
			parseHandler(ClientMessage{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeClientMessage(parseMessage.Payload)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

// SubscribeClientWindowMessages decodes incoming WindowChannel envelopes as ClientMessages and invokes handler.
func SubscribeClientWindowMessages(parseChannel WindowChannel, parseHandler func(ClientMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeClientWindowMessages", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return parseChannel.Subscribe(func(parseMessage WindowEnvelope, parseErr error) {
		if parseErr != nil {
			parseHandler(ClientMessage{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeClientMessage(parseMessage.Payload)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

// PublishClientHello sends a ClientHello presence message on a CrossTabChannel with default capabilities.
func PublishClientHello(parseChannel CrossTabChannel, parseSelf ClientIdentity) error {
	parseCapabilities := defaultCrossTabClientCapabilities(parseChannel)
	return PublishClientHelloWithCapabilities(parseChannel, parseSelf, parseCapabilities)
}

// PublishClientHelloWithCapabilities sends a ClientHello presence message on a CrossTabChannel with explicit capabilities.
func PublishClientHelloWithCapabilities(parseChannel CrossTabChannel, parseSelf ClientIdentity, parseCapabilities ClientCapabilities) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:         ClientHello,
		Topic:        ClientPresenceTopic,
		Source:       parseSelf,
		Capabilities: &parseCapabilities,
	})
}

// PublishClientHelloWindow sends a ClientHello presence message on a WindowChannel with default capabilities.
func PublishClientHelloWindow(parseChannel WindowChannel, parseSelf ClientIdentity) error {
	parseCapabilities := defaultWindowClientCapabilities(parseChannel)
	return PublishClientHelloWindowWithCapabilities(parseChannel, parseSelf, parseCapabilities)
}

// PublishClientHelloWindowWithCapabilities sends a ClientHello presence message on a WindowChannel with explicit capabilities.
func PublishClientHelloWindowWithCapabilities(parseChannel WindowChannel, parseSelf ClientIdentity, parseCapabilities ClientCapabilities) error {
	return PublishClientWindowMessage(parseChannel, ClientMessage{
		Kind:         ClientHello,
		Topic:        ClientPresenceTopic,
		Source:       parseSelf,
		Capabilities: &parseCapabilities,
	})
}

// PublishClientGoodbye sends a ClientGoodbye presence message on a CrossTabChannel.
func PublishClientGoodbye(parseChannel CrossTabChannel, parseSelf ClientIdentity) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:   ClientGoodbye,
		Topic:  ClientPresenceTopic,
		Source: parseSelf,
	})
}

// PublishClientGoodbyeWindow sends a ClientGoodbye presence message on a WindowChannel.
func PublishClientGoodbyeWindow(parseChannel WindowChannel, parseSelf ClientIdentity) error {
	return PublishClientWindowMessage(parseChannel, ClientMessage{
		Kind:   ClientGoodbye,
		Topic:  ClientPresenceTopic,
		Source: parseSelf,
	})
}

// PublishClientEvent sends a ClientEvent message on a CrossTabChannel to a named topic.
func PublishClientEvent(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parsePayload any) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:    ClientEvent,
		Topic:   strings.TrimSpace(parseTopic),
		Source:  parseSelf,
		Payload: parsePayload,
	})
}

// PublishClientIntent sends a directed ClientIntent message on a WindowChannel.
func PublishClientIntent(parseChannel WindowChannel, parseTopic string, parseSelf ClientIdentity, parseTarget string, parsePayload any) error {
	parseTrimmedTarget := strings.TrimSpace(parseTarget)
	if parseTrimmedTarget == "" {
		return wrapError("PublishClientIntent", parseChannel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	return PublishClientWindowMessage(parseChannel, ClientMessage{
		Kind:    ClientIntent,
		Topic:   strings.TrimSpace(parseTopic),
		Source:  parseSelf,
		Target:  parseTrimmedTarget,
		Payload: parsePayload,
	})
}

// PublishClientInvalidation sends a ClientInvalidate message on a CrossTabChannel for a given topic and revision.
func PublishClientInvalidation(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parseRevision string) error {
	parseTrimmedRevision := strings.TrimSpace(parseRevision)
	if parseTrimmedRevision == "" {
		return wrapError("PublishClientInvalidation", parseChannel.Name(), CodeInvalid, errors.New("revision is empty"))
	}
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:     ClientInvalidate,
		Topic:    strings.TrimSpace(parseTopic),
		Source:   parseSelf,
		Revision: parseTrimmedRevision,
	})
}

// PublishClientQuery sends a ClientQuery message on a CrossTabChannel for a given topic.
func PublishClientQuery(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:   ClientQuery,
		Topic:  strings.TrimSpace(parseTopic),
		Source: parseSelf,
	})
}

// PublishClientResult sends a directed ClientResult message on a CrossTabChannel.
func PublishClientResult(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parseTarget string, parsePayload any) error {
	parseTrimmedTarget := strings.TrimSpace(parseTarget)
	if parseTrimmedTarget == "" {
		return wrapError("PublishClientResult", parseChannel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:    ClientResult,
		Topic:   strings.TrimSpace(parseTopic),
		Source:  parseSelf,
		Target:  parseTrimmedTarget,
		Payload: parsePayload,
	})
}

// PublishClientBinaryWindow sends a binary ClientEvent on a WindowChannel to a specific target.
func PublishClientBinaryWindow(parseChannel WindowChannel, parseTopic string, parseSelf ClientIdentity, parseTarget string, parsePayload ClientBinaryPayload) error {
	parseTrimmedTarget := strings.TrimSpace(parseTarget)
	if parseTrimmedTarget == "" {
		return wrapError("PublishClientBinaryWindow", parseChannel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	parsePrepared, parseErr := prepareClientBinaryMessage("PublishClientBinaryWindow", parseChannel.Name(), ClientMessage{
		Kind:        ClientEvent,
		Topic:       strings.TrimSpace(parseTopic),
		Source:      parseSelf,
		Target:      parseTrimmedTarget,
		Encoding:    ClientPayloadBinary,
		ContentType: strings.TrimSpace(parsePayload.ContentType),
		Payload:     append([]byte(nil), parsePayload.Bytes...),
	})
	if parseErr != nil {
		return parseErr
	}
	if parseChannel.publishClientBinary != nil {
		return parseChannel.publishClientBinary(parsePrepared)
	}
	return PublishClientWindowMessage(parseChannel, parsePrepared)
}

// PublishClientBinaryCrossTab sends a binary ClientEvent on a CrossTabChannel.
func PublishClientBinaryCrossTab(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parsePayload ClientBinaryPayload) error {
	parsePrepared, parseErr := prepareClientBinaryMessage("PublishClientBinaryCrossTab", parseChannel.Name(), ClientMessage{
		Kind:        ClientEvent,
		Topic:       strings.TrimSpace(parseTopic),
		Source:      parseSelf,
		Encoding:    ClientPayloadBinary,
		ContentType: strings.TrimSpace(parsePayload.ContentType),
		Payload:     append([]byte(nil), parsePayload.Bytes...),
	})
	if parseErr != nil {
		return parseErr
	}
	if parseChannel.publishClientBinary != nil {
		return parseChannel.publishClientBinary(parsePrepared)
	}
	return PublishClientMessage(parseChannel, parsePrepared)
}

func prepareClientMessage(parseOp string, parseTarget string, parseMessage ClientMessage) (ClientMessage, error) {
	if parseErr := validateClientMessage(parseOp, parseTarget, parseMessage); parseErr != nil {
		return ClientMessage{}, parseErr
	}
	if parseMessage.SentAt.IsZero() {
		parseMessage.SentAt = time.Now().UTC()
	}
	return parseMessage, nil
}

func prepareClientBinaryMessage(parseOp string, parseTarget string, parseMessage ClientMessage) (ClientMessage, error) {
	parsePrepared, parseErr := prepareClientMessage(parseOp, parseTarget, parseMessage)
	if parseErr != nil {
		return ClientMessage{}, parseErr
	}
	parsePrepared.Encoding = ClientPayloadBinary
	parseBytes, parseOk := parsePrepared.Payload.([]byte)
	if !parseOk {
		return ClientMessage{}, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary payload must be []byte"))
	}
	if len(parseBytes) == 0 {
		return ClientMessage{}, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary payload is empty"))
	}
	if strings.TrimSpace(parsePrepared.ContentType) == "" {
		return ClientMessage{}, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary content type is empty"))
	}
	parsePrepared.Payload = append([]byte(nil), parseBytes...)
	return parsePrepared, nil
}

func validateClientMessage(parseOp string, parseTarget string, parseMessage ClientMessage) error {
	if parseErr := validateClientIdentity(parseOp, parseTarget, parseMessage.Source); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(parseMessage.Topic) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client topic is empty"))
	}
	switch parseMessage.Kind {
	case ClientHello, ClientGoodbye, ClientEvent, ClientIntent, ClientQuery, ClientResult, ClientInvalidate, ClientError:
	default:
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client message kind is empty or unknown"))
	}
	switch parseMessage.Encoding {
	case "", ClientPayloadJSON, ClientPayloadBinary:
	default:
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client payload encoding is empty or unknown"))
	}
	if parseMessage.Encoding == ClientPayloadBinary {
		if _, parseOk := parseMessage.Payload.([]byte); !parseOk {
			return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary client payload must be []byte"))
		}
	}
	if parseErr2 := validateClientTopicAuthorization(parseOp, parseTarget, parseMessage); parseErr2 != nil {
		return parseErr2
	}
	return nil
}

func validateClientTopicAuthorization(parseOp string, parseTarget string, parseMessage ClientMessage) error {
	parseTopic := strings.ToLower(strings.TrimSpace(parseMessage.Topic))
	if !isPrivilegedClientTopic(parseTopic) {
		return nil
	}
	parseRole := normalizeClientRole(parseMessage.Source.Role)
	if clientRoleMayUsePrivilegedTopic(parseRole) {
		return nil
	}
	if parseMessage.Kind == ClientIntent {
		return wrapError(parseOp, parseTarget, CodeUnauthorized, errors.New("client role is not authorized to publish privileged intent topic"))
	}
	return wrapError(parseOp, parseTarget, CodeUnauthorized, errors.New("client role is not authorized for privileged topic"))
}

func isPrivilegedClientTopic(parseTopic string) bool {
	switch {
	case strings.HasPrefix(parseTopic, "session:"), strings.HasPrefix(parseTopic, "operator:"), strings.HasPrefix(parseTopic, "intent:session"), strings.HasPrefix(parseTopic, "intent:operator"):
		return true
	default:
		return false
	}
}

func normalizeClientRole(parseRole string) string {
	return strings.ToLower(strings.TrimSpace(parseRole))
}

func clientRoleMayUsePrivilegedTopic(parseRole string) bool {
	switch parseRole {
	case "operator", "admin", "system":
		return true
	default:
		return false
	}
}

func decodeClientMessageValue(parseValue any) (ClientMessage, error) {
	switch parseTyped := parseValue.(type) {
	case ClientMessage:
		return parseTyped, nil
	case map[string]any:
		return decodeClientMessageMap(parseTyped)
	default:
		var parseMessage ClientMessage
		if parseErr := Decode(parseValue, &parseMessage); parseErr != nil {
			return ClientMessage{}, parseErr
		}
		return parseMessage, nil
	}
}

func decodeClientMessageMap(parseData map[string]any) (ClientMessage, error) {
	parseMessage := ClientMessage{
		ID:          stringField(parseData, "id"),
		Kind:        ClientMessageKind(stringField(parseData, "kind")),
		Topic:       stringField(parseData, "topic"),
		Target:      stringField(parseData, "target"),
		Revision:    stringField(parseData, "revision"),
		Error:       stringField(parseData, "error"),
		Encoding:    ClientPayloadEncoding(stringField(parseData, "encoding")),
		ContentType: stringField(parseData, "contentType"),
	}
	if parseSource, parseOk := parseData["source"].(map[string]any); parseOk {
		parseMessage.Source = ClientIdentity{
			ID:      stringField(parseSource, "id"),
			App:     stringField(parseSource, "app"),
			Surface: stringField(parseSource, "surface"),
			Role:    stringField(parseSource, "role"),
			Version: stringField(parseSource, "version"),
		}
	}
	if parseCapabilities, parseOk2 := parseData["capabilities"].(map[string]any); parseOk2 {
		parseMessage.Capabilities = decodeClientCapabilitiesMap(parseCapabilities)
	}
	if parsePayload, parseOk3 := parseData["payload"]; parseOk3 {
		parseMessage.Payload = parsePayload
	}
	if parseSentAt, parseOk4 := clientTimeField(parseData["sentAt"]); parseOk4 {
		parseMessage.SentAt = parseSentAt
	}
	return parseMessage, nil
}

func decodeClientCapabilitiesMap(parseData map[string]any) *ClientCapabilities {
	parseCapabilities := &ClientCapabilities{
		ProtocolVersion: stringField(parseData, "protocolVersion"),
		Transports:      stringSliceField(parseData["transports"]),
		Encodings:       stringSliceField(parseData["encodings"]),
		Topics:          stringSliceField(parseData["topics"]),
		MaxJSONBytes:    intField(parseData["maxJsonBytes"]),
		MaxBinaryBytes:  intField(parseData["maxBinaryBytes"]),
	}
	return parseCapabilities
}

func stringField(parseData map[string]any, parseKey string) string {
	parseValue, parseOk := parseData[parseKey]
	if !parseOk {
		return ""
	}
	switch parseTyped := parseValue.(type) {
	case string:
		return parseTyped
	default:
		return ""
	}
}

func stringSliceField(parseValue any) []string {
	switch parseTyped := parseValue.(type) {
	case []string:
		return append([]string(nil), parseTyped...)
	case []any:
		parseValues := make([]string, 0, len(parseTyped))
		for _, parseEntry := range parseTyped {
			parseText, parseOk := parseEntry.(string)
			if parseOk && strings.TrimSpace(parseText) != "" {
				parseValues = append(parseValues, parseText)
			}
		}
		return parseValues
	default:
		return nil
	}
}

func intField(parseValue any) int {
	switch parseTyped := parseValue.(type) {
	case int:
		return parseTyped
	case int64:
		return int(parseTyped)
	case float64:
		return int(parseTyped)
	default:
		return 0
	}
}

func clientTimeField(parseValue any) (time.Time, bool) {
	parseText, parseOk := parseValue.(string)
	if !parseOk || strings.TrimSpace(parseText) == "" {
		return time.Time{}, false
	}
	parseParsed, parseErr := time.Parse(time.RFC3339Nano, parseText)
	if parseErr != nil {
		return time.Time{}, false
	}
	return parseParsed, true
}

func defaultCrossTabClientCapabilities(parseChannel CrossTabChannel) ClientCapabilities {
	parseEncodings := []string{string(ClientPayloadJSON)}
	if parseChannel.Transport() == "broadcast-channel" {
		parseEncodings = append(parseEncodings, string(ClientPayloadBinary))
	}
	parseTransports := []string{}
	if parseTransport := strings.TrimSpace(parseChannel.Transport()); parseTransport != "" {
		parseTransports = append(parseTransports, parseTransport)
	}
	return ClientCapabilities{
		ProtocolVersion: "v1",
		Transports:      parseTransports,
		Encodings:       parseEncodings,
	}
}

func defaultWindowClientCapabilities(parseChannel WindowChannel) ClientCapabilities {
	parseTransports := []string{"window-message"}
	if parseTarget := strings.TrimSpace(parseChannel.Name()); parseTarget != "" {
		_ = parseTarget
	}
	return ClientCapabilities{
		ProtocolVersion: "v1",
		Transports:      parseTransports,
		Encodings:       []string{string(ClientPayloadJSON), string(ClientPayloadBinary)},
	}
}

// ClientProtocolCompatible reports whether two capability sets share a compatible protocol major version.
func ClientProtocolCompatible(parseLocal ClientCapabilities, parsePeer ClientCapabilities) bool {
	parseLocalVersion := normalizeProtocolVersion(parseLocal.ProtocolVersion)
	parsePeerVersion := normalizeProtocolVersion(parsePeer.ProtocolVersion)
	if parseLocalVersion == "" || parsePeerVersion == "" {
		return false
	}
	return protocolMajor(parseLocalVersion) == protocolMajor(parsePeerVersion)
}

// ClientSupportsEncoding reports whether the capabilities include the given payload encoding.
func ClientSupportsEncoding(parseCapabilities ClientCapabilities, parseEncoding ClientPayloadEncoding) bool {
	parseTrimmed := strings.TrimSpace(string(parseEncoding))
	if parseTrimmed == "" {
		parseTrimmed = string(ClientPayloadJSON)
	}
	for _, parseCandidate := range parseCapabilities.Encodings {
		if strings.EqualFold(strings.TrimSpace(parseCandidate), parseTrimmed) {
			return true
		}
	}
	return false
}

// ClientSupportsTopic reports whether the capabilities include the given topic (empty topic list means all).
func ClientSupportsTopic(parseCapabilities ClientCapabilities, parseTopic string) bool {
	parseTrimmed := strings.TrimSpace(parseTopic)
	if parseTrimmed == "" {
		return false
	}
	if len(parseCapabilities.Topics) == 0 {
		return true
	}
	for _, parseCandidate := range parseCapabilities.Topics {
		if strings.EqualFold(strings.TrimSpace(parseCandidate), parseTrimmed) {
			return true
		}
	}
	return false
}

// ClientCanExchange reports whether two clients can communicate on a topic with a shared encoding.
func ClientCanExchange(parseLocal ClientCapabilities, parsePeer ClientCapabilities, parseTopic string, parseEncoding ClientPayloadEncoding) bool {
	if !ClientProtocolCompatible(parseLocal, parsePeer) {
		return false
	}
	if !ClientSupportsEncoding(parseLocal, parseEncoding) || !ClientSupportsEncoding(parsePeer, parseEncoding) {
		return false
	}
	if !ClientSupportsTopic(parseLocal, parseTopic) || !ClientSupportsTopic(parsePeer, parseTopic) {
		return false
	}
	return true
}

func normalizeProtocolVersion(parseValue string) string {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseValue))
	parseTrimmed = strings.TrimPrefix(parseTrimmed, "v")
	return parseTrimmed
}

func protocolMajor(parseValue string) string {
	parseTrimmed := normalizeProtocolVersion(parseValue)
	if parseTrimmed == "" {
		return ""
	}
	if parseDot := strings.Index(parseTrimmed, "."); parseDot >= 0 {
		return parseTrimmed[:parseDot]
	}
	return parseTrimmed
}

func validateClientIdentity(parseOp string, parseTarget string, parseIdentity ClientIdentity) error {
	if strings.TrimSpace(parseIdentity.ID) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client identity id is empty"))
	}
	if strings.TrimSpace(parseIdentity.App) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client identity app is empty"))
	}
	if strings.TrimSpace(parseIdentity.Surface) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client identity surface is empty"))
	}
	return nil
}

func SubscribeDecodedWindow[T any](parseChannel WindowChannel, parseHandler func(DecodedWindowEnvelope[T], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeDecodedWindow", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return parseChannel.Subscribe(func(parseMessage WindowEnvelope, parseErr error) {
		if parseErr != nil {
			parseHandler(DecodedWindowEnvelope[T]{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeWindowEnvelope[T](parseMessage)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

// DecodeSurfaceSignal decodes a WindowEnvelope payload as a SurfaceSignal.
func DecodeSurfaceSignal(parseMessage WindowEnvelope) (DecodedWindowEnvelope[SurfaceSignal], error) {
	return DecodeWindowEnvelope[SurfaceSignal](parseMessage)
}

// SubscribeSurfaceSignals receives decoded SurfaceSignal messages on a WindowChannel.
func SubscribeSurfaceSignals(parseChannel WindowChannel, parseHandler func(DecodedWindowEnvelope[SurfaceSignal], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeSurfaceSignals", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return SubscribeDecodedWindow(parseChannel, parseHandler)
}

// PublishSurfaceSignal validates and sends a SurfaceSignal over a WindowChannel.
func PublishSurfaceSignal(parseChannel WindowChannel, parseSignal SurfaceSignal) error {
	switch parseSignal.Kind {
	case SurfaceSignalSession:
		if parseSignal.Session == nil {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("session signal is missing session payload"))
		}
	case SurfaceSignalRoute:
		if parseSignal.Route == nil || strings.TrimSpace(parseSignal.Route.Path) == "" {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("route signal is missing path"))
		}
	case SurfaceSignalSelection:
		if parseSignal.Selection == nil || strings.TrimSpace(parseSignal.Selection.ID) == "" {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("selection signal is missing id"))
		}
	case SurfaceSignalIntent:
		if parseSignal.Intent == nil || strings.TrimSpace(string(parseSignal.Intent.Action)) == "" {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("intent signal is missing action"))
		}
	default:
		return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("surface signal kind is empty or unknown"))
	}
	return parseChannel.Publish(parseSignal)
}

// PublishLogout sends a signed-out session signal on a WindowChannel.
func PublishLogout(parseChannel WindowChannel, parseReason string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalSession,
		Session: &SurfaceSessionSignal{
			Status: "signed-out",
			Reason: strings.TrimSpace(parseReason),
		},
	})
}

// PublishSessionExpired sends a session-expired signal on a WindowChannel.
func PublishSessionExpired(parseChannel WindowChannel, parseReason string, parseReturnTo string, parseExpiresAt time.Time) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalSession,
		Session: &SurfaceSessionSignal{
			Status:    "expired",
			Reason:    strings.TrimSpace(parseReason),
			ReturnTo:  strings.TrimSpace(parseReturnTo),
			ExpiresAt: parseExpiresAt,
		},
	})
}

// PublishRouteFocus sends a route-focus surface signal on a WindowChannel.
func PublishRouteFocus(parseChannel WindowChannel, parsePath string, parseQuery string, parseFocusID string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalRoute,
		Route: &SurfaceRouteSignal{
			Path:    strings.TrimSpace(parsePath),
			Query:   strings.TrimSpace(parseQuery),
			FocusID: strings.TrimSpace(parseFocusID),
		},
	})
}

// PublishSelection sends a selection surface signal on a WindowChannel.
func PublishSelection(parseChannel WindowChannel, parseScope string, parseId string, parseRevision string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalSelection,
		Selection: &SurfaceSelectionSignal{
			Scope:    strings.TrimSpace(parseScope),
			ID:       strings.TrimSpace(parseId),
			Revision: strings.TrimSpace(parseRevision),
		},
	})
}

// PublishIntent sends an intent surface signal on a WindowChannel.
func PublishIntent(parseChannel WindowChannel, parseAction SurfaceIntentAction, parseTarget string, parseParams map[string]string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalIntent,
		Intent: &SurfaceIntentSignal{
			Action: parseAction,
			Target: strings.TrimSpace(parseTarget),
			Params: parseParams,
		},
	})
}
