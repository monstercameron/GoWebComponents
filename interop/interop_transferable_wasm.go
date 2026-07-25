//go:build js && wasm

package interop

import (
	"fmt"
	"syscall/js"
)

// Binary transferables (v5 P3.1).
//
// postMessage clones its payload by default. A transfer list moves ownership
// instead: the buffer is detached from the sender and handed to the receiver
// without copying its contents, which is O(1) regardless of size.
//
// Before this, the only transfer list the interop layer built was for
// MessagePorts (messagePortTransferListJS), so every binary payload — including
// runtime2's patch and snapshot streams, which exist precisely to move bytes
// efficiently — was structured-cloned. The transports were binary; the delivery
// was not.
//
// The Go-side cost is unchanged and worth stating plainly: Go's wasm target
// cannot expose its linear memory to JS directly, so bytes still cross via
// js.CopyBytesToJS on the way out and js.CopyBytesToGo on the way in. What
// transferring removes is the browser's own clone of the buffer in between —
// the middle copy, not the crossings.

// Transferable is a binary payload whose ownership moves to the receiver.
//
// After a successful send the sender's buffer is DETACHED: reading it throws.
// That is the trade being made, and it is why this is an explicit type rather
// than an automatic optimization — a caller has to opt into losing the buffer.
type Transferable struct {
	raw js.Value
}

// NewTransferable copies bytes into a fresh ArrayBuffer that can be transferred.
//
// The copy here is unavoidable (Go heap to JS heap). The saving comes later:
// postMessage moves the result instead of cloning it, so a 2MB payload crosses
// the worker boundary without the browser duplicating it.
func NewTransferable(parseData []byte) (Transferable, error) {
	parseArray := js.Global().Get("Uint8Array").New(len(parseData))
	if len(parseData) > 0 {
		js.CopyBytesToJS(parseArray, parseData)
	}
	return Transferable{raw: parseArray.Get("buffer")}, nil
}

// Bytes copies a received transferable back into Go memory.
func (parseT Transferable) Bytes() []byte {
	if parseT.raw.IsUndefined() || parseT.raw.IsNull() {
		return nil
	}
	parseView := js.Global().Get("Uint8Array").New(parseT.raw)
	parseOut := make([]byte, parseView.Length())
	if len(parseOut) > 0 {
		js.CopyBytesToGo(parseOut, parseView)
	}
	return parseOut
}

// Len reports the byte length, or zero once the buffer has been transferred
// away — a detached buffer reports zero rather than throwing, so a caller can
// check without a panic.
func (parseT Transferable) Len() int {
	if parseT.raw.IsUndefined() || parseT.raw.IsNull() {
		return 0
	}
	return parseT.raw.Get("byteLength").Int()
}

// IsDetached reports whether ownership has already moved. Sending a detached
// transferable is an error, not a silent no-op.
func (parseT Transferable) IsDetached() bool {
	return parseT.Len() == 0 && !parseT.raw.IsUndefined() && !parseT.raw.IsNull()
}

// buildTransferList assembles a postMessage transfer list from ports and
// buffers together, since one message may legitimately carry both.
func buildTransferList(parseOp string, parseTarget string, parsePorts []MessagePort, parseBuffers []Transferable) (js.Value, error) {
	parseTransfers := js.Global().Get("Array").New(len(parsePorts) + len(parseBuffers))
	parseIndex := 0

	for parsePortIndex, parsePort := range parsePorts {
		parseRawPort, parseOk := parsePort.raw.(js.Value)
		if !parseOk || parseRawPort.IsUndefined() || parseRawPort.IsNull() {
			return js.Undefined(), wrapError(parseOp, parseTarget, CodeInvalid,
				fmt.Errorf("message port %d is unavailable", parsePortIndex))
		}
		parseTransfers.SetIndex(parseIndex, parseRawPort)
		parseIndex++
	}

	for parseBufferIndex, parseBuffer := range parseBuffers {
		if parseBuffer.raw.IsUndefined() || parseBuffer.raw.IsNull() {
			return js.Undefined(), wrapError(parseOp, parseTarget, CodeInvalid,
				fmt.Errorf("transferable %d is unavailable", parseBufferIndex))
		}
		if parseBuffer.IsDetached() {
			return js.Undefined(), wrapError(parseOp, parseTarget, CodeInvalid,
				fmt.Errorf("transferable %d was already transferred; ownership cannot move twice", parseBufferIndex))
		}
		parseTransfers.SetIndex(parseIndex, parseBuffer.raw)
		parseIndex++
	}

	return parseTransfers, nil
}

// PostTransferable sends a payload to a worker, moving the listed buffers
// instead of cloning them.
//
// The buffers are detached once this returns; reading them afterwards throws.
func (parseW Worker) PostTransferable(parsePayload any, parseBuffers ...Transferable) error {
	if parseW.postTransferable == nil {
		return unavailable("PostTransferable", "worker")
	}
	return parseW.postTransferable(parsePayload, parseBuffers)
}

// PostTransferable sends a payload over a message port, moving the listed
// buffers instead of cloning them.
func (parseP MessagePort) PostTransferable(parsePayload any, parseBuffers ...Transferable) error {
	parseRaw, parseOk := parseP.raw.(js.Value)
	if !parseOk || parseRaw.IsUndefined() || parseRaw.IsNull() {
		return unavailable("PostTransferable", "message port")
	}
	return postTransferableMessageJS("PostTransferable", "message port", parseRaw, parsePayload, parseBuffers)
}

// postTransferableMessageJS is the shared send path.
func postTransferableMessageJS(parseOp string, parseTarget string, parseRaw js.Value, parsePayload any, parseBuffers []Transferable) (parseErr error) {
	defer recoverInteropException(parseOp, parseTarget, &parseErr)

	parseConverted, parseErr := goValueToJSStructured(parseOp, parseTarget, parsePayload)
	if parseErr != nil {
		return parseErr
	}
	if len(parseBuffers) == 0 {
		parseRaw.Call("postMessage", parseConverted)
		return nil
	}
	parseTransfers, parseErr := buildTransferList(parseOp, parseTarget, nil, parseBuffers)
	if parseErr != nil {
		return parseErr
	}
	parseRaw.Call("postMessage", parseConverted, parseTransfers)
	return nil
}
