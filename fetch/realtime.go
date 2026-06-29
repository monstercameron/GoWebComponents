package fetch

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	gwcruntime "github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

const (
	defaultRealtimeMaxMessages    = 32
	defaultRealtimeMaxErrors      = 8
	defaultRealtimeMaxReconnects  = 3
	defaultRealtimeInitialBackoff = 250 * time.Millisecond
	defaultRealtimeMaxBackoff     = 5 * time.Second
	defaultRealtimeBackoffFactor  = 2
)

// RealtimeStatus describes the current lifecycle phase for a realtime hook.
type RealtimeStatus string

const (
	// RealtimeIdle means the hook has not opened a browser transport yet.
	RealtimeIdle RealtimeStatus = "idle"
	// RealtimeConnecting means a browser transport is being opened.
	RealtimeConnecting RealtimeStatus = "connecting"
	// RealtimeOpen means the browser transport is currently open.
	RealtimeOpen RealtimeStatus = "open"
	// RealtimeReconnecting means the hook is waiting for a bounded reconnect attempt.
	RealtimeReconnecting RealtimeStatus = "reconnecting"
	// RealtimeClosed means the transport is closed and no reconnect is pending.
	RealtimeClosed RealtimeStatus = "closed"
	// RealtimeUnsupported means the browser API is unavailable on this target.
	RealtimeUnsupported RealtimeStatus = "unsupported"
)

// RealtimeMessage stores one received WebSocket or EventSource message.
type RealtimeMessage struct {
	Data        string
	Type        string
	LastEventID string
	ReceivedAt  time.Time
}

// RealtimeError stores one recent realtime transport error.
type RealtimeError struct {
	Message string
	At      time.Time
}

// RealtimeState is the bounded state snapshot returned by realtime hooks.
type RealtimeState struct {
	Status            RealtimeStatus
	Supported         bool
	Connecting        bool
	Open              bool
	Closed            bool
	Messages          []RealtimeMessage
	LastMessage       RealtimeMessage
	Errors            []RealtimeError
	Error             error
	ConnectAttempts   int
	ReconnectAttempts int
	Reconnects        int
	NextReconnectAt   time.Time
	LastOpenAt        time.Time
	LastCloseAt       time.Time
	LastHeartbeatAt   time.Time
	HeartbeatMisses   int
}

// WebSocketOptions configures UseWebSocket.
type WebSocketOptions struct {
	Protocols         []string
	Manual            bool
	DisableReconnect  bool
	MaxReconnects     int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffFactor     float64
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	HeartbeatMessage  string
	MaxMessages       int
	MaxErrors         int
	Now               func() time.Time
}

// EventSourceOptions configures UseEventSource.
type EventSourceOptions struct {
	WithCredentials   bool
	Manual            bool
	DisableReconnect  bool
	MaxReconnects     int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffFactor     float64
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	MaxMessages       int
	MaxErrors         int
	Now               func() time.Time
}

// WebSocket exposes a UseWebSocket connection handle.
type WebSocket struct {
	get   func() RealtimeState
	open  func()
	close func()
	send  func(string) error
}

// EventSource exposes a UseEventSource connection handle.
type EventSource struct {
	get   func() RealtimeState
	open  func()
	close func()
}

type realtimeUnsupportedError struct {
	api string
}

// Error returns the unsupported transport message.
func (parseE realtimeUnsupportedError) Error() string {
	return fmt.Sprintf("%s API unavailable in this environment", parseE.api)
}

type realtimeTransport interface {
	send(string) error
	close() error
}

type realtimeTransportCallbacks struct {
	handleOpen    func()
	handleMessage func(RealtimeMessage)
	handleError   func(error)
	handleClose   func()
}

type realtimeResolvedOptions struct {
	protocols           []string
	withCredentials     bool
	manual              bool
	reconnect           bool
	maxReconnects       int
	initialBackoff      time.Duration
	maxBackoff          time.Duration
	backoffFactor       float64
	heartbeatInterval   time.Duration
	heartbeatTimeout    time.Duration
	heartbeatMessage    string
	shouldSendHeartbeat bool
	maxMessages         int
	maxErrors           int
	now                 func() time.Time
}

type realtimeConnectionController struct {
	api              string
	url              string
	options          realtimeResolvedOptions
	state            ui.State[RealtimeState]
	openTransport    func(string, realtimeResolvedOptions, realtimeTransportCallbacks) (realtimeTransport, error)
	transport        realtimeTransport
	active           bool
	manualClosed     bool
	connectionID     int
	connectionClosed bool
	reconnectTimer   *time.Timer
	heartbeatTimer   *time.Timer
}

// UseWebSocket opens a bounded browser WebSocket connection for a component.
func UseWebSocket(parseURL string, parseOptions ...WebSocketOptions) WebSocket {
	parseResolved := resolveWebSocketOptions(parseOptions)
	parseState := ui.UseState(buildWebSocketInitialState())
	parseOpenTick := ui.UseState(0)
	parseControllerRef := ui.UseRef((*realtimeConnectionController)(nil))
	parseTick := parseOpenTick.Get()

	ui.UseEffect(func() func() {
		parseController := &realtimeConnectionController{
			api:           "WebSocket",
			url:           parseURL,
			options:       parseResolved,
			state:         parseState,
			openTransport: openWebSocketTransport,
		}
		parseControllerRef.Set(parseController)
		if !parseResolved.manual || parseTick > 0 {
			parseController.start()
		}
		return func() {
			parseController.stop(false)
			if parseControllerRef.Get() == parseController {
				parseControllerRef.Set(nil)
			}
		}
	}, buildRealtimeDeps(parseURL, parseTick, parseResolved)...)

	return WebSocket{
		get: func() RealtimeState {
			return cloneRealtimeState(parseState.Get())
		},
		open: func() {
			if parseController := parseControllerRef.Get(); parseController != nil {
				parseController.start()
				return
			}
			parseOpenTick.Update(func(parsePrev int) int { return parsePrev + 1 })
		},
		close: func() {
			if parseController := parseControllerRef.Get(); parseController != nil {
				parseController.stop(true)
			}
		},
		send: func(parseMessage string) error {
			parseController := parseControllerRef.Get()
			if parseController == nil {
				return errors.New("WebSocket is not open")
			}
			return parseController.send(parseMessage)
		},
	}
}

// Get returns the current WebSocket state snapshot.
func (parseW WebSocket) Get() RealtimeState {
	if parseW.get == nil {
		return RealtimeState{}
	}
	return parseW.get()
}

// Open starts or restarts the WebSocket connection.
func (parseW WebSocket) Open() {
	if parseW.open != nil {
		parseW.open()
	}
}

// Close closes the WebSocket connection and suppresses reconnects.
func (parseW WebSocket) Close() {
	if parseW.close != nil {
		parseW.close()
	}
}

// Send sends one text message through the active WebSocket connection.
func (parseW WebSocket) Send(parseMessage string) error {
	if parseW.send == nil {
		return errors.New("WebSocket is not open")
	}
	return parseW.send(parseMessage)
}

// UseEventSource opens a bounded browser EventSource connection for a component.
func UseEventSource(parseURL string, parseOptions ...EventSourceOptions) EventSource {
	parseResolved := resolveEventSourceOptions(parseOptions)
	parseState := ui.UseState(buildEventSourceInitialState())
	parseOpenTick := ui.UseState(0)
	parseControllerRef := ui.UseRef((*realtimeConnectionController)(nil))
	parseTick := parseOpenTick.Get()

	ui.UseEffect(func() func() {
		parseController := &realtimeConnectionController{
			api:           "EventSource",
			url:           parseURL,
			options:       parseResolved,
			state:         parseState,
			openTransport: openEventSourceTransport,
		}
		parseControllerRef.Set(parseController)
		if !parseResolved.manual || parseTick > 0 {
			parseController.start()
		}
		return func() {
			parseController.stop(false)
			if parseControllerRef.Get() == parseController {
				parseControllerRef.Set(nil)
			}
		}
	}, buildRealtimeDeps(parseURL, parseTick, parseResolved)...)

	return EventSource{
		get: func() RealtimeState {
			return cloneRealtimeState(parseState.Get())
		},
		open: func() {
			if parseController := parseControllerRef.Get(); parseController != nil {
				parseController.start()
				return
			}
			parseOpenTick.Update(func(parsePrev int) int { return parsePrev + 1 })
		},
		close: func() {
			if parseController := parseControllerRef.Get(); parseController != nil {
				parseController.stop(true)
			}
		},
	}
}

// Get returns the current EventSource state snapshot.
func (parseE EventSource) Get() RealtimeState {
	if parseE.get == nil {
		return RealtimeState{}
	}
	return parseE.get()
}

// Open starts or restarts the EventSource connection.
func (parseE EventSource) Open() {
	if parseE.open != nil {
		parseE.open()
	}
}

// Close closes the EventSource connection and suppresses reconnects.
func (parseE EventSource) Close() {
	if parseE.close != nil {
		parseE.close()
	}
}

// start opens the controller from an idle or manually closed state.
func (parseC *realtimeConnectionController) start() {
	if parseC == nil {
		return
	}
	parseC.stopTimers()
	parseC.active = true
	parseC.manualClosed = false
	parseC.connectionClosed = false
	parseC.options.now = getRealtimeNow(parseC.options.now)
	parseC.open()
}

// stop closes the active transport and records a terminal closed state.
func (parseC *realtimeConnectionController) stop(isManual bool) {
	if parseC == nil {
		return
	}
	parseC.active = false
	parseC.manualClosed = isManual
	parseC.connectionID++
	parseC.connectionClosed = true
	parseC.stopTimers()
	if parseC.transport != nil {
		_ = parseC.transport.close()
		parseC.transport = nil
	}
	parseNow := parseC.options.now()
	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.Status = RealtimeClosed
		parsePrev.Connecting = false
		parsePrev.Open = false
		parsePrev.Closed = true
		parsePrev.NextReconnectAt = time.Time{}
		parsePrev.LastCloseAt = parseNow
		return parsePrev
	})
}

// open creates one browser transport and wires callbacks for the active connection.
func (parseC *realtimeConnectionController) open() {
	if parseC == nil || !parseC.active {
		return
	}
	parseURL := strings.TrimSpace(parseC.url)
	if parseURL == "" {
		parseErr := fmt.Errorf("%s URL is empty", parseC.api)
		parseC.recordError(parseErr)
		parseC.finishClosed(parseErr)
		return
	}

	parseC.connectionID++
	parseID := parseC.connectionID
	parseC.connectionClosed = false
	if parseC.transport != nil {
		_ = parseC.transport.close()
		parseC.transport = nil
	}
	parseC.stopHeartbeat()

	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.Status = RealtimeConnecting
		parsePrev.Supported = true
		parsePrev.Connecting = true
		parsePrev.Open = false
		parsePrev.Closed = false
		parsePrev.NextReconnectAt = time.Time{}
		parsePrev.ConnectAttempts++
		return parsePrev
	})

	parseTransport, parseErr := parseC.openTransport(parseURL, parseC.options, realtimeTransportCallbacks{
		handleOpen: func() {
			parseC.handleOpen(parseID)
		},
		handleMessage: func(parseMessage RealtimeMessage) {
			parseC.handleMessage(parseID, parseMessage)
		},
		handleError: func(parseErr2 error) {
			parseC.handleError(parseID, parseErr2)
		},
		handleClose: func() {
			parseC.handleClose(parseID)
		},
	})
	if parseErr != nil {
		parseC.transport = nil
		if isRealtimeUnsupportedError(parseErr) {
			parseC.active = false
			updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
				parsePrev.Status = RealtimeUnsupported
				parsePrev.Supported = false
				parsePrev.Connecting = false
				parsePrev.Open = false
				parsePrev.Closed = true
				parsePrev.Error = parseErr
				parsePrev.Errors = appendRealtimeErrors(parsePrev.Errors, RealtimeError{Message: parseErr.Error(), At: parseC.options.now()}, parseC.options.maxErrors)
				return parsePrev
			})
			return
		}
		parseC.recordError(parseErr)
		parseC.scheduleReconnect(parseErr)
		return
	}
	parseC.transport = parseTransport
}

// send sends one text message through the active transport.
func (parseC *realtimeConnectionController) send(parseMessage string) error {
	if parseC == nil || !parseC.active || parseC.transport == nil {
		return errors.New("WebSocket is not open")
	}
	parseErr := parseC.transport.send(parseMessage)
	if parseErr != nil {
		parseC.recordError(parseErr)
	}
	return parseErr
}

// handleOpen records an opened transport and starts heartbeat tracking.
func (parseC *realtimeConnectionController) handleOpen(parseID int) {
	if !parseC.isCurrentConnection(parseID) {
		return
	}
	parseC.connectionClosed = false
	parseC.options.now = getRealtimeNow(parseC.options.now)
	parseC.options.maxErrors = getPositiveOrDefault(parseC.options.maxErrors, defaultRealtimeMaxErrors)
	parseC.options.maxMessages = getPositiveOrDefault(parseC.options.maxMessages, defaultRealtimeMaxMessages)
	parseNow := parseC.options.now()
	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.Status = RealtimeOpen
		parsePrev.Supported = true
		parsePrev.Connecting = false
		parsePrev.Open = true
		parsePrev.Closed = false
		parsePrev.ReconnectAttempts = 0
		parsePrev.NextReconnectAt = time.Time{}
		parsePrev.LastOpenAt = parseNow
		parsePrev.Error = nil
		return parsePrev
	})
	parseC.startHeartbeat()
}

// handleMessage appends one bounded message to state.
func (parseC *realtimeConnectionController) handleMessage(parseID int, parseMessage RealtimeMessage) {
	if !parseC.isCurrentConnection(parseID) {
		return
	}
	if parseMessage.ReceivedAt.IsZero() {
		parseMessage.ReceivedAt = parseC.options.now()
	}
	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.Messages = appendRealtimeMessages(parsePrev.Messages, parseMessage, parseC.options.maxMessages)
		parsePrev.LastMessage = parseMessage
		return parsePrev
	})
}

// handleError appends one bounded transport error to state.
func (parseC *realtimeConnectionController) handleError(parseID int, parseErr error) {
	if !parseC.isCurrentConnection(parseID) || parseErr == nil {
		return
	}
	parseC.recordError(parseErr)
}

// handleClose records a closed transport and schedules a bounded reconnect.
func (parseC *realtimeConnectionController) handleClose(parseID int) {
	if !parseC.isCurrentConnection(parseID) || parseC.connectionClosed {
		return
	}
	parseC.connectionClosed = true
	parseTransport := parseC.transport
	parseC.transport = nil
	parseC.stopHeartbeat()
	if parseTransport != nil {
		time.AfterFunc(0, func() {
			defer gwcruntime.RecoverContainedPanic("fetch", parseC.api+" close cleanup")
			_ = parseTransport.close()
		})
	}
	if parseC.manualClosed || !parseC.active {
		parseC.finishClosed(nil)
		return
	}
	parseC.scheduleReconnect(nil)
}

// isCurrentConnection reports whether a callback belongs to the active transport.
func (parseC *realtimeConnectionController) isCurrentConnection(parseID int) bool {
	return parseC != nil && parseC.active && parseID == parseC.connectionID
}

// scheduleReconnect waits for the next bounded reconnect attempt or closes permanently.
func (parseC *realtimeConnectionController) scheduleReconnect(parseErr error) {
	if parseC == nil || !parseC.active || parseC.manualClosed || !parseC.options.reconnect {
		parseC.finishClosed(parseErr)
		return
	}
	if parseC.options.maxReconnects <= 0 {
		parseC.options.maxReconnects = defaultRealtimeMaxReconnects
	}
	if parseC.options.maxReconnects > 0 && parseC.currentReconnectAttempts() >= parseC.options.maxReconnects {
		parseC.finishClosed(parseErr)
		return
	}

	parseAttempt := parseC.currentReconnectAttempts() + 1
	parseDelay := calculateRealtimeBackoff(parseAttempt, parseC.options.initialBackoff, parseC.options.maxBackoff, parseC.options.backoffFactor)
	parseNext := parseC.options.now().Add(parseDelay)
	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.Status = RealtimeReconnecting
		parsePrev.Connecting = false
		parsePrev.Open = false
		parsePrev.Closed = false
		parsePrev.ReconnectAttempts = parseAttempt
		parsePrev.Reconnects++
		parsePrev.NextReconnectAt = parseNext
		if parseErr != nil {
			parsePrev.Error = parseErr
		}
		return parsePrev
	})
	parseC.stopReconnectTimer()
	parseC.reconnectTimer = time.AfterFunc(parseDelay, func() {
		defer gwcruntime.RecoverContainedPanic("fetch", parseC.api+" reconnect")
		if parseC == nil || !parseC.active || parseC.manualClosed {
			return
		}
		parseC.open()
	})
}

// currentReconnectAttempts reads the current consecutive reconnect count.
func (parseC *realtimeConnectionController) currentReconnectAttempts() int {
	parseState := parseC.state.Get()
	return parseState.ReconnectAttempts
}

// finishClosed records a non-reconnecting closed state.
func (parseC *realtimeConnectionController) finishClosed(parseErr error) {
	if parseC == nil {
		return
	}
	parseC.active = false
	parseC.stopTimers()
	parseNow := parseC.options.now()
	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.Status = RealtimeClosed
		parsePrev.Connecting = false
		parsePrev.Open = false
		parsePrev.Closed = true
		parsePrev.NextReconnectAt = time.Time{}
		parsePrev.LastCloseAt = parseNow
		if parseErr != nil {
			parsePrev.Error = parseErr
		}
		return parsePrev
	})
}

// recordError appends one bounded error to state.
func (parseC *realtimeConnectionController) recordError(parseErr error) {
	if parseC == nil || parseErr == nil {
		return
	}
	parseNow := parseC.options.now()
	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.Error = parseErr
		parsePrev.Errors = appendRealtimeErrors(parsePrev.Errors, RealtimeError{Message: parseErr.Error(), At: parseNow}, parseC.options.maxErrors)
		return parsePrev
	})
}

// startHeartbeat starts optional heartbeat send and timeout tracking.
func (parseC *realtimeConnectionController) startHeartbeat() {
	if parseC == nil || parseC.options.heartbeatInterval <= 0 {
		return
	}
	parseC.stopHeartbeat()
	parseC.heartbeatTimer = time.AfterFunc(parseC.options.heartbeatInterval, func() {
		defer gwcruntime.RecoverContainedPanic("fetch", parseC.api+" heartbeat")
		parseC.handleHeartbeat()
	})
}

// handleHeartbeat records heartbeat state and reconnects timed-out connections.
func (parseC *realtimeConnectionController) handleHeartbeat() {
	if parseC == nil || !parseC.active || parseC.transport == nil {
		return
	}
	parseNow := parseC.options.now()
	if parseC.options.shouldSendHeartbeat {
		parseMessage := parseC.options.heartbeatMessage
		if parseMessage == "" {
			parseMessage = "ping"
		}
		if parseErr := parseC.transport.send(parseMessage); parseErr != nil {
			parseC.recordError(parseErr)
		}
	}

	var isTimedOut bool
	if parseC.options.heartbeatTimeout > 0 {
		parseState := parseC.state.Get()
		parseLastActivity := parseState.LastMessage.ReceivedAt
		if parseLastActivity.IsZero() {
			parseLastActivity = parseState.LastOpenAt
		}
		isTimedOut = !parseLastActivity.IsZero() && parseNow.Sub(parseLastActivity) > parseC.options.heartbeatTimeout
	}

	updateRealtimeState(parseC.state, func(parsePrev RealtimeState) RealtimeState {
		parsePrev.LastHeartbeatAt = parseNow
		if isTimedOut {
			parsePrev.HeartbeatMisses++
		}
		return parsePrev
	})

	if isTimedOut {
		parseErr := fmt.Errorf("%s heartbeat timed out after %s", parseC.api, parseC.options.heartbeatTimeout)
		parseC.recordError(parseErr)
		if parseC.transport != nil {
			_ = parseC.transport.close()
		}
		parseC.handleClose(parseC.connectionID)
		return
	}
	parseC.startHeartbeat()
}

// stopTimers stops all pending reconnect and heartbeat timers.
func (parseC *realtimeConnectionController) stopTimers() {
	parseC.stopReconnectTimer()
	parseC.stopHeartbeat()
}

// stopReconnectTimer stops a pending reconnect timer.
func (parseC *realtimeConnectionController) stopReconnectTimer() {
	if parseC != nil && parseC.reconnectTimer != nil {
		parseC.reconnectTimer.Stop()
		parseC.reconnectTimer = nil
	}
}

// stopHeartbeat stops a pending heartbeat timer.
func (parseC *realtimeConnectionController) stopHeartbeat() {
	if parseC != nil && parseC.heartbeatTimer != nil {
		parseC.heartbeatTimer.Stop()
		parseC.heartbeatTimer = nil
	}
}

// resolveWebSocketOptions normalizes caller WebSocket options.
func resolveWebSocketOptions(parseOptions []WebSocketOptions) realtimeResolvedOptions {
	var parseOption WebSocketOptions
	if len(parseOptions) > 0 {
		parseOption = parseOptions[0]
	}
	parseResolved := realtimeResolvedOptions{
		protocols:           append([]string(nil), parseOption.Protocols...),
		manual:              parseOption.Manual,
		reconnect:           !parseOption.DisableReconnect,
		maxReconnects:       getPositiveOrDefault(parseOption.MaxReconnects, defaultRealtimeMaxReconnects),
		initialBackoff:      getDurationOrDefault(parseOption.InitialBackoff, defaultRealtimeInitialBackoff),
		maxBackoff:          getDurationOrDefault(parseOption.MaxBackoff, defaultRealtimeMaxBackoff),
		backoffFactor:       getFloatOrDefault(parseOption.BackoffFactor, defaultRealtimeBackoffFactor),
		heartbeatInterval:   parseOption.HeartbeatInterval,
		heartbeatTimeout:    parseOption.HeartbeatTimeout,
		heartbeatMessage:    parseOption.HeartbeatMessage,
		shouldSendHeartbeat: parseOption.HeartbeatInterval > 0,
		maxMessages:         getPositiveOrDefault(parseOption.MaxMessages, defaultRealtimeMaxMessages),
		maxErrors:           getPositiveOrDefault(parseOption.MaxErrors, defaultRealtimeMaxErrors),
		now:                 getRealtimeNow(parseOption.Now),
	}
	if parseResolved.maxBackoff < parseResolved.initialBackoff {
		parseResolved.maxBackoff = parseResolved.initialBackoff
	}
	return parseResolved
}

// resolveEventSourceOptions normalizes caller EventSource options.
func resolveEventSourceOptions(parseOptions []EventSourceOptions) realtimeResolvedOptions {
	var parseOption EventSourceOptions
	if len(parseOptions) > 0 {
		parseOption = parseOptions[0]
	}
	parseTimeout := parseOption.HeartbeatTimeout
	if parseTimeout <= 0 && parseOption.HeartbeatInterval > 0 {
		parseTimeout = parseOption.HeartbeatInterval * 2
	}
	parseResolved := realtimeResolvedOptions{
		withCredentials:   parseOption.WithCredentials,
		manual:            parseOption.Manual,
		reconnect:         !parseOption.DisableReconnect,
		maxReconnects:     getPositiveOrDefault(parseOption.MaxReconnects, defaultRealtimeMaxReconnects),
		initialBackoff:    getDurationOrDefault(parseOption.InitialBackoff, defaultRealtimeInitialBackoff),
		maxBackoff:        getDurationOrDefault(parseOption.MaxBackoff, defaultRealtimeMaxBackoff),
		backoffFactor:     getFloatOrDefault(parseOption.BackoffFactor, defaultRealtimeBackoffFactor),
		heartbeatInterval: parseOption.HeartbeatInterval,
		heartbeatTimeout:  parseTimeout,
		maxMessages:       getPositiveOrDefault(parseOption.MaxMessages, defaultRealtimeMaxMessages),
		maxErrors:         getPositiveOrDefault(parseOption.MaxErrors, defaultRealtimeMaxErrors),
		now:               getRealtimeNow(parseOption.Now),
	}
	if parseResolved.maxBackoff < parseResolved.initialBackoff {
		parseResolved.maxBackoff = parseResolved.initialBackoff
	}
	return parseResolved
}

// buildRealtimeDeps returns stable effect dependencies for realtime hooks.
func buildRealtimeDeps(parseURL string, parseOpenTick int, parseOptions realtimeResolvedOptions) []any {
	return []any{
		parseURL,
		parseOpenTick,
		strings.Join(parseOptions.protocols, "\x00"),
		parseOptions.withCredentials,
		parseOptions.manual,
		parseOptions.reconnect,
		parseOptions.maxReconnects,
		parseOptions.initialBackoff,
		parseOptions.maxBackoff,
		parseOptions.backoffFactor,
		parseOptions.heartbeatInterval,
		parseOptions.heartbeatTimeout,
		parseOptions.heartbeatMessage,
		parseOptions.shouldSendHeartbeat,
		parseOptions.maxMessages,
		parseOptions.maxErrors,
	}
}

// calculateRealtimeBackoff returns one bounded exponential backoff duration.
func calculateRealtimeBackoff(parseAttempt int, parseInitial time.Duration, parseMax time.Duration, parseFactor float64) time.Duration {
	parseInitial = getDurationOrDefault(parseInitial, defaultRealtimeInitialBackoff)
	parseMax = getDurationOrDefault(parseMax, defaultRealtimeMaxBackoff)
	parseFactor = getFloatOrDefault(parseFactor, defaultRealtimeBackoffFactor)
	if parseAttempt <= 1 {
		if parseInitial > parseMax {
			return parseMax
		}
		return parseInitial
	}
	parseScaled := float64(parseInitial) * math.Pow(parseFactor, float64(parseAttempt-1))
	if parseScaled <= 0 || parseScaled > float64(parseMax) {
		return parseMax
	}
	return time.Duration(parseScaled)
}

// updateRealtimeState applies a cloned bounded state update.
func updateRealtimeState(parseState ui.State[RealtimeState], parseUpdate func(RealtimeState) RealtimeState) {
	if parseUpdate == nil {
		return
	}
	parseState.Update(func(parsePrev RealtimeState) RealtimeState {
		return parseUpdate(cloneRealtimeState(parsePrev))
	})
}

// appendRealtimeMessages appends a message and keeps only the newest max entries.
func appendRealtimeMessages(parseMessages []RealtimeMessage, parseMessage RealtimeMessage, parseMax int) []RealtimeMessage {
	parseMax = getPositiveOrDefault(parseMax, defaultRealtimeMaxMessages)
	parseMessages = append(append([]RealtimeMessage(nil), parseMessages...), parseMessage)
	if len(parseMessages) > parseMax {
		parseMessages = parseMessages[len(parseMessages)-parseMax:]
	}
	return parseMessages
}

// appendRealtimeErrors appends an error and keeps only the newest max entries.
func appendRealtimeErrors(parseErrors []RealtimeError, parseError RealtimeError, parseMax int) []RealtimeError {
	parseMax = getPositiveOrDefault(parseMax, defaultRealtimeMaxErrors)
	parseErrors = append(append([]RealtimeError(nil), parseErrors...), parseError)
	if len(parseErrors) > parseMax {
		parseErrors = parseErrors[len(parseErrors)-parseMax:]
	}
	return parseErrors
}

// cloneRealtimeState clones slice-backed state fields before exposing a snapshot.
func cloneRealtimeState(parseState RealtimeState) RealtimeState {
	if parseState.Messages != nil {
		parseState.Messages = append([]RealtimeMessage(nil), parseState.Messages...)
	}
	if parseState.Errors != nil {
		parseState.Errors = append([]RealtimeError(nil), parseState.Errors...)
	}
	return parseState
}

// getRealtimeNow returns a stable clock function.
func getRealtimeNow(parseNow func() time.Time) func() time.Time {
	if parseNow != nil {
		return parseNow
	}
	return time.Now
}

// getDurationOrDefault returns fallback when value is not positive.
func getDurationOrDefault(parseValue time.Duration, parseFallback time.Duration) time.Duration {
	if parseValue > 0 {
		return parseValue
	}
	return parseFallback
}

// getFloatOrDefault returns fallback when value is not positive.
func getFloatOrDefault(parseValue float64, parseFallback float64) float64 {
	if parseValue > 0 {
		return parseValue
	}
	return parseFallback
}

// getPositiveOrDefault returns fallback when value is not positive.
func getPositiveOrDefault(parseValue int, parseFallback int) int {
	if parseValue > 0 {
		return parseValue
	}
	return parseFallback
}

// isRealtimeUnsupportedError reports whether err is an unsupported transport error.
func isRealtimeUnsupportedError(parseErr error) bool {
	var parseUnsupported realtimeUnsupportedError
	return errors.As(parseErr, &parseUnsupported)
}
