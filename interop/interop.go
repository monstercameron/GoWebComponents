package interop

import (
	"context"
	"encoding/json"
	"errors"
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
	if e.Err != nil {
		return base + ": " + e.Err.Error()
	}
	return base
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

type Subscription struct {
	cancel func()
}

func (s Subscription) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

type Storage struct {
	getItem    func(string) (string, bool, error)
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

type EventTarget struct {
	dispatch  func(string, any) error
	subscribe func(string, func(CustomEvent)) (Subscription, error)
}

func (t EventTarget) Dispatch(name string, detail any) error {
	if t.dispatch == nil {
		return unavailable("EventTarget.Dispatch", "")
	}
	return t.dispatch(name, detail)
}

func (t EventTarget) Subscribe(name string, handler func(CustomEvent)) (Subscription, error) {
	if t.subscribe == nil {
		return Subscription{}, unavailable("EventTarget.Subscribe", "")
	}
	return t.subscribe(name, handler)
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
