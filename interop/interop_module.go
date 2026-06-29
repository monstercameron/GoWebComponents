package interop

import (
	"context"
	"time"
)

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
	SentAt   time.Time `json:"sentAt"`
}

type WindowEnvelope struct {
	Name    string    `json:"name"`
	Payload any       `json:"payload"`
	Source  string    `json:"source,omitempty"`
	SentAt  time.Time `json:"sentAt"`
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
	SentAt       time.Time             `json:"sentAt"`
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
	ExpiresAt time.Time `json:"expiresAt"`
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
	raw                  any
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
	raw       any
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
