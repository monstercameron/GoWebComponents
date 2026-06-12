package interop

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
func Decode(parseValue any, parseTarget any) error {
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
	raw any
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
	title         func() (string, error)
	setTitle      func(string) error
}

// Title returns document.title. Unavailable on native/SSR builds.
func (parseD Document) Title() (string, error) {
	if parseD.title == nil {
		return "", unavailable("Document.Title", "document.title")
	}
	return parseD.title()
}

// SetTitle sets document.title (e.g. to reflect the active route or
// conversation in the tab strip). Unavailable on native/SSR builds.
func (parseD Document) SetTitle(parseTitle string) error {
	if parseD.setTitle == nil {
		return unavailable("Document.SetTitle", "document.title")
	}
	return parseD.setTitle(parseTitle)
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
