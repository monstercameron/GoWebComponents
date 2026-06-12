package interop

import (
	"context"
	"errors"
	"time"
)

func (parseB SharedBuffer) GetSharedBufferRaw() any {
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
func (parseP MessagePort) GetMessagePortRaw() any {
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
