# Multi-Client Coordination

This page defines the current first slice and near-term direction for browser coordination between two or more sovereign `js/wasm` clients.

Use it when each client owns its own runtime, DOM root, and release lifecycle, but the clients still need typed browser-local coordination.

## At A Glance

- the current multi-client surface is experimental public API
- the shipped transport layer already lives in `interop` through cross-tab and window channels
- the current first-class value is a shared client-message contract over those transports, not a new distributed runtime
- devtools can already surface app-owned multi-client inspection state when the app provides it

## Quick Transport Chooser

- use `interop.OpenCrossTabChannel(...)` when separate tabs need presence, invalidation, discovery, or light event fanout
- use `interop.OpenSecondaryWindowChannel(...)` and `interop.WindowOpenerChannel(...)` when an opener and popup need targeted request or reply traffic
- keep worker messaging separate; workers are background compute peers, not sovereign UI clients
- keep JSON as the default control plane and only opt into binary where the resolved transport and topic both justify it

## Example Shape

This is the intended shape for the current shipped slice: open a browser-local channel, announce presence, subscribe with typed decoding, and keep app-owned registry or diagnostics state outside the transport itself.

```go
package shell

import (
    "github.com/atdiar/particleui/devtools"
    "github.com/atdiar/particleui/interop"
)

func attachPeerChannel() (func(), error) {
    channel, err := interop.OpenCrossTabChannel("atlas-peers")
    if err != nil {
        return nil, err
    }

    self := interop.ClientIdentity{
        ID:      "storefront-1",
        App:     "atlas",
        Surface: "storefront-tab",
        Role:    "storefront",
    }

    subscription, err := interop.SubscribeClientMessages(channel, func(message interop.ClientMessage, decodeErr error) {
        if decodeErr != nil {
            return
        }

        devtools.SetMultiClientInspection(devtools.MultiClient{
            Enabled:           true,
            LocalPeerID:       self.ID,
            ResolvedTransport: channel.Transport(),
        })
        _ = message
    })
    if err != nil {
        _ = channel.Close()
        return nil, err
    }

    if err := interop.PublishClientHello(channel, self); err != nil {
        _ = subscription.Cancel()
        _ = channel.Close()
        return nil, err
    }

    return func() {
        _ = interop.PublishClientGoodbye(channel, self)
        _ = subscription.Cancel()
        _ = channel.Close()
    }, nil
}
```

## Scope

This proposal is intentionally built on the existing shipped surfaces:

- tabs use `interop.OpenCrossTabChannel(...)`
- popup or opener pairs use `interop.OpenSecondaryWindowChannel(...)` and `interop.WindowOpenerChannel(...)`
- worker transport is out of scope because the target here is peer clients, not client-owned background compute

The missing first-class abstraction is not a new transport. It is one small shared message contract and helper layer over the current transports.

The current implemented slice now lives in `interop` as client-message helpers over `CrossTabChannel` and `WindowChannel`, with JSON control-plane support everywhere and binary payload support on the transports that can carry it natively.

## Current Shipped Slice

Today the documented first slice is:

- client identity and capability negotiation through `hello`
- teardown through `goodbye`
- app-owned topic traffic through `event`, `intent`, `query`, `result`, and `invalidate`
- transport-aware binary payload support only where the resolved transport can carry it safely
- application-provided inspection state surfaced through `devtools.SetMultiClientInspection(...)`

What the framework does not claim today:

- durable replay or offline delivery
- shared-memory semantics between runtimes
- worker unification under the same contract
- automatic authority resolution for competing clients

## Proposed Public API

```go
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

type ClientMessage struct {
    ID       string            `json:"id,omitempty"`
    Kind     ClientMessageKind `json:"kind"`
    Topic    string            `json:"topic"`
    Source   ClientIdentity    `json:"source"`
    Target   string            `json:"target,omitempty"`
    Capabilities *ClientCapabilities `json:"capabilities,omitempty"`
    Encoding ClientPayloadEncoding `json:"encoding,omitempty"`
    ContentType string         `json:"contentType,omitempty"`
    Payload  any               `json:"payload,omitempty"`
    Revision string            `json:"revision,omitempty"`
    Error    string            `json:"error,omitempty"`
    SentAt   time.Time         `json:"sentAt,omitempty"`
}

type ClientPayloadEncoding string

const (
    ClientPayloadJSON   ClientPayloadEncoding = "json"
    ClientPayloadBinary ClientPayloadEncoding = "binary"
)

type ClientBinaryPayload struct {
    ContentType string `json:"contentType,omitempty"`
    Bytes       []byte `json:"-"`
}
```

Current helper surface:

```go
func DecodeClientMessage(value any) (ClientMessage, error)
func PublishClientMessage(channel interop.CrossTabChannel, message ClientMessage) error
func PublishClientWindowMessage(channel interop.WindowChannel, message ClientMessage) error

func PublishClientHello(channel interop.CrossTabChannel, self ClientIdentity) error
func PublishClientHelloWithCapabilities(channel interop.CrossTabChannel, self ClientIdentity, capabilities ClientCapabilities) error
func PublishClientHelloWindow(channel interop.WindowChannel, self ClientIdentity) error
func PublishClientHelloWindowWithCapabilities(channel interop.WindowChannel, self ClientIdentity, capabilities ClientCapabilities) error
func PublishClientGoodbye(channel interop.CrossTabChannel, self ClientIdentity) error
func PublishClientGoodbyeWindow(channel interop.WindowChannel, self ClientIdentity) error

func PublishClientEvent(channel interop.CrossTabChannel, topic string, self ClientIdentity, payload any) error
func PublishClientIntent(channel interop.WindowChannel, topic string, self ClientIdentity, target string, payload any) error
func PublishClientInvalidation(channel interop.CrossTabChannel, topic string, self ClientIdentity, revision string) error
func PublishClientQuery(channel interop.CrossTabChannel, topic string, self ClientIdentity) error
func PublishClientResult(channel interop.CrossTabChannel, topic string, self ClientIdentity, target string, payload any) error
func PublishClientBinaryWindow(channel interop.WindowChannel, topic string, self ClientIdentity, target string, payload ClientBinaryPayload) error
func PublishClientBinaryCrossTab(channel interop.CrossTabChannel, topic string, self ClientIdentity, payload ClientBinaryPayload) error

func ClientProtocolCompatible(local, peer ClientCapabilities) bool
func ClientSupportsEncoding(capabilities ClientCapabilities, encoding ClientPayloadEncoding) bool
func ClientSupportsTopic(capabilities ClientCapabilities, topic string) bool
func ClientCanExchange(local, peer ClientCapabilities, topic string, encoding ClientPayloadEncoding) bool

func SubscribeClientMessages(channel interop.CrossTabChannel, handler func(ClientMessage, error)) (interop.Subscription, error)
func SubscribeClientWindowMessages(channel interop.WindowChannel, handler func(ClientMessage, error)) (interop.Subscription, error)
```

## Payload Format And Duplex

The proposed model should be message-oriented and logically full duplex.

- every client may send and receive at any time
- request and reply are correlated by message `ID`
- the transport is not a byte stream and does not promise socket-style ordering across peers

Payload format should be hybrid:

- JSON is the current implemented control-plane encoding
- binary is the current experimental optional data-plane encoding for payload-heavy topics on supported transports

Recommended rule:

- `hello`, `goodbye`, `intent`, `invalidate`, `query`, `result`, and lifecycle traffic stay JSON
- binary is allowed only for topic payloads that materially benefit from it

Binary framing rules for the current experimental slice:

- `Encoding` must be `binary` when the payload is raw bytes
- `ContentType` should identify the application-owned payload format such as `application/cbor`, `application/octet-stream`, or another explicit domain type
- chunking is not part of the current first-class contract; oversized binary payloads should be rejected rather than silently fragmented
- integrity hashes and schema ids are application-owned fields until a stronger first-party binary envelope is justified

Transport limits matter:

- `WindowChannel` can support binary through browser `postMessage` structured clone
- `CrossTabChannel` can support binary only when the resolved transport is `BroadcastChannel`
- `CrossTabChannel` storage fallback should reject binary publish rather than silently base64-wrapping it

The implemented helper layer now does exactly that: binary publish succeeds on `WindowChannel` and `BroadcastChannel`, and binary publish is rejected on storage-event fallback with a structured interop error.

That means binary support is capability-based, not universal.

## Versioning And Capability Negotiation

The protocol should be additive-first and versioned explicitly.

Recommended rules:

- every client advertises `ProtocolVersion` and capability flags in its `hello` payload
- unknown optional fields must be ignored
- unknown message kinds must be rejected locally with structured diagnostics and no state mutation
- additive fields are allowed in minor protocol revisions when older clients can safely ignore them
- field removals, meaning changes, or required-field additions require a major protocol revision

Practical negotiation model:

- `hello` publishes `ClientIdentity` plus `ClientCapabilities`
- the current helper layer now publishes default transport-derived capabilities on `PublishClientHello(...)` and `PublishClientHelloWindow(...)`, and also exposes explicit `...WithCapabilities(...)` helpers plus compatibility predicates for higher-level registries
- receivers keep the peer in the registry even if some capabilities are unsupported
- topic- or encoding-specific work should proceed only when the peer and the local client both advertise support
- version mismatch should degrade to presence-only rather than pretending higher-level coordination is safe

## Delivery Guarantees

The intended delivery model is deliberately weak and explicit.

Current contract:

- browser-local delivery is best-effort
- the protocol should be treated as at-most-once from the sender point of view
- applications must tolerate duplicate `hello`, `goodbye`, `query`, or `result` messages
- per-sender ordering may be observed on one transport, but global ordering across peers is not guaranteed
- there is no built-in replay, durable queue, or offline delivery contract for multi-client traffic

Applications should treat `intent`, `invalidate`, and `query` flows as idempotent whenever possible.

## Authority Model

Multi-client coordination only stays sane when authority is explicit per topic.

Recommended rules:

- auth and session truth stay server-owned unless one clearly documented browser client is acting only as a mirror of server state
- route ownership stays local to the current surface, with other clients receiving route-focus hints rather than trying to co-own navigation
- cache invalidation may be peer-published, but authoritative data still comes from the cache owner or server
- diagnostics topics may be peer-published, but should never mutate business state
- when two clients can edit the same logical entity, one of them or the server must still be authoritative for acceptance of the final revision

Topic design should answer "who decides?" before defining message payloads.

## Request, Timeout, And Cancellation Semantics

`query` and `result` are request-reply messages, not a streaming RPC framework.

Recommended rules:

- `ID` is the request correlation key
- the requester owns timeout policy
- late `result` messages may be ignored after timeout or local cancellation
- duplicate `result` messages should be ignored after the first accepted terminal reply
- cancellation is local in the current first-class contract; no dedicated wire-level `cancel` message is required yet

Recommended defaults:

- short peer-discovery queries: 1 to 2 seconds
- targeted popup or opener queries: 3 to 5 seconds
- payload-heavy or operator-tool queries: application-owned timeout based on topic cost

## Lease, Heartbeat, And Expiry

Presence should use leases, not permanent connected state.

Recommended defaults:

- `hello` on first subscribe
- re-announce on reconnect and visibility resume
- optional heartbeat only for long-lived presence-sensitive surfaces
- default lease timeout should be 3 times the heartbeat interval when heartbeat is enabled
- background-tab throttling should be expected, so expiry handling must tolerate missed beats without panicking or spamming logs

For a first public default, a conservative rule is:

- no mandatory heartbeat
- lease refresh on first boot, resume, reconnect, and explicit discovery query handling
- expiry treated as degraded presence, not as a correctness failure

## Discovery And Late Join

Use a handshake, not a browser-wide scan.

The browser does not provide a first-class way to enumerate arbitrary sovereign tabs running the same app, so the proposed model should be:

- subscribe first
- publish `hello` immediately after the channel is ready
- maintain an in-memory peer registry keyed by `ClientIdentity.ID`
- remove peers on `goodbye` or after a lease timeout

For late joiners, the minimal discovery flow is:

```go
func PublishClientQuery(channel interop.CrossTabChannel, topic string, self ClientIdentity) error
func PublishClientResult(channel interop.CrossTabChannel, topic string, self ClientIdentity, target string, payload any) error
```

Recommended behavior:

- a newly started tab publishes `hello`
- if it needs a current peer list, it also publishes `query` on topic `clients`
- live peers answer with `result` targeted back to that client
- long-lived clients re-announce `hello` on reconnect, focus, or a small heartbeat interval when presence matters

For popup or opener pairs, discovery is simpler:

- there is no scan because the window handle is already known
- the child sends `hello` after boot
- the opener treats `hello` as readiness for targeted `intent`, `query`, or `event` messages

## Lifecycle Events

The architecture should model connection lifecycle explicitly, but only for states the browser surfaces can actually support.

Recommended client-side lifecycle events:

- `hello-sent`: this client announced itself on a channel
- `peer-seen`: a valid `hello` was received from another client
- `peer-ready`: a peer answered a targeted `query` or accepted a targeted window handshake
- `peer-disconnected`: a peer sent `goodbye`
- `peer-expired`: a peer lease timed out without refresh
- `channel-reconnected`: the local client re-subscribed and re-announced after transport loss or page resume

These should stay local runtime events derived from `ClientMessage`, not a second wire protocol.

What should not be modeled as a first-class wire event:

- `scan-found`
- `scan-started`
- `scan-complete`

Those do not fit the proposed architecture because cross-tab transport is announcement-based, not enumerable. A late joiner can ask "who is alive" with `query(topic="clients")`, but that is still a handshake round, not a true scan.

Minimal local registry shape:

```go
type ClientPeerState string

const (
    ClientPeerUnknown      ClientPeerState = "unknown"
    ClientPeerSeen         ClientPeerState = "seen"
    ClientPeerReady        ClientPeerState = "ready"
    ClientPeerDisconnected ClientPeerState = "disconnected"
    ClientPeerExpired      ClientPeerState = "expired"
)
```

Recommended transitions:

- `unknown -> seen` on `hello`
- `seen -> ready` on successful targeted reply or explicit window readiness
- `seen|ready -> disconnected` on `goodbye`
- `seen|ready -> expired` when lease renewal is missed
- `disconnected|expired -> seen` on a fresh `hello`

## Payload Limits And Backpressure

The first-party contract should reject oversized traffic instead of hiding it.

Recommended defaults:

- cap JSON control-plane payloads conservatively
- cap binary payloads separately and more tightly than whatever the browser might technically allow
- reject oversized publish attempts with structured errors
- avoid implicit chunking in the first public slice
- let applications debounce noisy topics instead of expecting the framework to queue arbitrarily

Backpressure remains application-owned until there is clear demand for a first-party scheduler or queue.

## Topic Namespaces And Schema Ownership

Topic sprawl becomes an enterprise problem quickly, so names and ownership need rules.

Recommended topic families:

- `clients`: presence and discovery
- `intent:*`: targeted commands or collaboration requests
- `invalidate:*`: cache or view invalidation
- `event:*`: informational peer events
- `diagnostics:*`: non-business coordination diagnostics

Recommended ownership rules:

- one team or subsystem owns each topic schema
- additive fields are preferred over field meaning changes
- required fields should stay small and stable
- schemas should live beside the topic owner, not only in ad hoc example code

## Error Codes And Failure Handling

The multi-client layer should use structured errors instead of string matching.

Recommended stable failure cases:

- unsupported transport
- unsupported binary encoding
- version mismatch
- unauthorized topic publish
- peer unavailable
- query timeout
- payload too large
- decode failure
- lease expired
- orphaned window

These should integrate with the existing structured `interop` error model instead of inventing a parallel string-only failure path.

## Trust, Authorization, And Origin Policy

Same origin reduces risk, but it does not erase trust decisions.

Recommended security rules:

- same-origin coordination is the default supported path
- `TargetOrigin` must stay explicit and validated for window messaging
- roles in `ClientIdentity` are descriptive unless the application explicitly enforces them
- privileged topics such as session or operator intents should still apply application-owned authorization checks before effecting change
- clients must not trust peer-provided claims more than same-origin application code and server policy justify

Current helper-layer enforcement:

- window subscriptions now surface same-peer origin mismatches as structured unauthorized interop errors instead of silently accepting or acting on the message
- privileged topic families such as `session:*`, `operator:*`, `intent:session*`, and `intent:operator*` are now rejected by the helper layer for non-privileged roles, while operator, admin, and system roles remain allowed to publish them
- stale popup or opener handles continue to fail closed with structured disposed errors before privileged traffic is sent

Recommended origin rules:

- popup and opener coordination should default to same-origin only
- navigation or origin changes should invalidate the existing trusted peer relationship
- orphaned windows should degrade into read-only or disconnected state instead of continuing privileged behavior optimistically

## Orphan And Degraded Modes

Degraded operation should be explicit.

Recommended rules:

- expired peers should remain visible in diagnostics until cleaned up locally
- orphaned popup clients should stop sending privileged intents automatically
- disconnected peers may be re-discovered only through a fresh handshake
- degraded presence is not itself a fatal runtime error

## Observability, Logging, And Diagnostics

The multi-client layer should emit stable instrumentation rather than relying on app-specific console logs.

Recommended observability events:

- peer seen
- peer expired
- query sent
- result received
- timeout
- binary rejected
- transport resolved
- channel reconnected

Recommended logging rules:

- log topic names, peer ids, correlation ids, payload sizes, and transport choice
- do not log raw binary payload bodies by default
- treat peer identity as operational metadata, not user identity

Recommended diagnostics rules:

- version mismatch, unsupported encoding, unauthorized topic, timeout, and decode failure should point back to this document
- devtools should surface the local peer registry, resolved transport, recent message summaries, and lease status rather than raw payload dumps

## Reference Topology

One concrete enterprise-shaped flow for this repo is a storefront tab, an operator tab, and a popup inspector.

Suggested ownership split:

- storefront tab: owns customer-visible route state, cart intent publication, and cache invalidation requests for shopper-facing data
- operator tab: owns privileged operational topics such as inventory overrides, fulfillment intent review, and escalation workflow state
- popup inspector: owns local diagnostics, recent message inspection, and targeted query or result exchanges with its opener only

Suggested transport split:

- storefront tab <-> operator tab: cross-tab channel for presence, discovery, invalidation, and non-privileged event fanout
- operator tab <-> popup inspector: window channel for targeted query, result, and diagnostics flows where the opener already knows the child window handle
- popup inspector !-> storefront tab directly: route through the operator tab when authority or trust boundaries matter

Suggested lifecycle:

- each long-lived surface publishes `hello` on boot and refreshes presence on reconnect or resume
- a late-joining operator tab publishes `query` on topic `clients` and accepts targeted `result` replies to rebuild its peer registry
- the popup inspector treats opener `hello` as readiness and downgrades to disconnected diagnostics mode if the opener becomes orphaned or its lease expires

Suggested authority rules:

- shopper-facing tabs may publish `intent` and `invalidate`, but the operator tab or server still decides whether privileged operational changes are accepted
- popup inspectors may observe diagnostics topics and issue targeted `query` messages, but they must not publish business-state mutation topics directly
- binary payloads stay on diagnostics or asset-preview topics and only when the resolved transport advertises binary capability

This topology exercises the main rules in this document without pretending every client is equally trusted or equally authoritative.

## Optional RPC Stream Transport

Typed RPC streams may become an optional companion transport for multi-client coordination, but they should not replace the current native topic mesh by default.

Recommended decision:

- keep presence, discovery, handshake, local invalidation, and lightweight peer event fanout on the existing multi-client JSON control plane
- allow an RPC stream layer only for flows that are already server-authoritative, schema-heavy, or long-lived enough that protobuf contracts and stream lifecycle management materially improve correctness
- preserve the authority, logging, and diagnostics rules from this document even when the payload path moves onto an RPC stream

When the current native topic contract is the right fit:

- tab or popup presence
- late-join discovery through `hello`, `goodbye`, `query`, and `result`
- cache or view invalidation
- low-volume coordination between sovereign browser clients on the same origin
- browser-local collaboration hints where missed or duplicated messages are tolerable and the server still owns final acceptance

When an optional typed RPC stream is justified instead:

- shared-session flows where the server must serialize authority and attach authenticated session context to every update
- live dashboards or operator consoles that need a long-lived server stream with typed resumable snapshots or high-cardinality event payloads
- collaboration flows whose value depends on protobuf-defined schema evolution, stricter per-stream ordering, or explicit server acks
- payload-heavy streams that would otherwise abuse topic fanout with large JSON envelopes or ad hoc binary encoding

Required guardrails if an RPC companion transport is added later:

- treat the RPC stream as a separate data plane layered beside the existing multi-client control plane, not as a silent transport swap under `interop`
- open the RPC connection from an app-owned shell or service boundary as described in `docs/RPC_TRANSPORT.md`
- fail closed on handshake, auth, or capability mismatch instead of degrading into implicit topic mirroring
- keep retry, replay, and resumability application-owned unless a future companion package documents them explicitly

## Interoperability With RPC-Backed Live Streams

Native multi-client topics and RPC-backed live streams may coexist, but one concern gets one authoritative wire contract at a time.

Primary ownership rules:

- `clients` presence, `hello`, `goodbye`, and discovery `query` or `result` traffic remain native multi-client topics
- browser-local invalidation remains a native multi-client concern even when the invalidated data is later refetched through RPC
- server-authoritative live data, typed collaboration state, and authenticated shared-session updates belong on RPC unary or streaming methods rather than on peer topics
- targeted `query` and `result` stay native only for browser-local peer service patterns such as opener or popup coordination; once the server is the authority, use an RPC method family instead of tunneling the same exchange through peer topics
- binary payloads stay native only for browser-local diagnostics or asset-preview topics on transports that already support them; use RPC streams when binary transfer needs server auth, stream framing, backpressure, or typed acks

Coexistence rules:

- presence topics may advertise that an RPC capability exists, but they must not carry authoritative copies of the stream payload
- an RPC update may trigger local `invalidate:*` publication, but the invalidation topic is a hint to refresh, not a second source of truth for the updated entity
- do not mirror one logical live feed onto both `event:*` topics and an RPC stream at the same time
- if a feature mixes the two planes, document the ownership split explicitly in app code and docs before shipping it

Recommended split by concern:

- presence and discovery: native multi-client topics
- cache invalidation: native multi-client topics
- targeted popup or opener requests: native `query` and `result`
- authenticated shared-session state: RPC
- live dashboards and high-volume operator feeds: RPC
- browser-local diagnostics fanout: native topics, with optional RPC only when the server must ingest or arbitrate the stream

Bridge policy:

- any topic-to-RPC or RPC-to-topic bridge must live in application code, not as an automatic helper hidden inside `interop`
- bridges should translate between the two planes at one boundary and preserve correlation ids, authority ownership, and structured diagnostics
- if ownership becomes ambiguous, prefer RPC for server truth and keep topics as advisory-only local signals

## Stability Tier

The current recommended stability tier is experimental public API.

That means:

- the JSON helper layer can evolve while the protocol proves itself in examples and application integration
- binary transport, capability negotiation, and richer diagnostics can still reshape in minor releases
- promotion to stable should wait until the example, conformance, and observability slices exist

## Non-Goals

This feature is not intended to be:

- a distributed database between browser runtimes
- shared-memory state between sovereign wasm clients
- a generic byte-stream transport
- offline replay or durable messaging by default
- same-document multi-client co-ownership without isolation boundaries
- a hidden replacement for server authority

## Rules

- use tabs for peer broadcast and invalidation
- use window channels for targeted opener or popup interaction
- keep the control plane JSON-shaped
- allow binary only on transports that can carry it natively
- prefer handshake and presence over active surface scanning
- treat lifecycle as presence plus lease state, not as socket-style connection state
- treat one client or the server as authoritative per topic
- prefer `intent` and `invalidate` over full state replication
- use `query` and `result` only when one client is deliberately acting as a local service for another

## Fit For This Repo

This fits the current project if it stays a thin typed layer over `CrossTabChannel` and `WindowChannel`.

This does not fit if it turns into:

- direct shared-state semantics between runtimes
- same-document co-ownership without isolation boundaries
- a generic distributed runtime hidden behind one magic client bus

## Recommended First Slice

If adopted, the first shipped slice should cover only:

- client `hello` presence
- client `goodbye` teardown
- topic events
- invalidation messages
- targeted window intents

That is enough for storefront plus console, popup inspector, or multi-tab operator flows without inventing a heavier browser mesh than the repo can support today.

## Review Checklist

- does the design choose cross-tab versus window transport deliberately instead of treating all peers as equivalent
- does each topic have an explicit authority owner before state mutation is allowed
- are binary payloads limited to transports and topics that can justify them
- do timeout, lease, and degraded-mode rules stay explicit instead of pretending delivery is durable
- does diagnostics or devtools state expose peer registry and transport facts without dumping raw payload bodies by default
