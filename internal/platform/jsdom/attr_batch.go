//go:build js && wasm

package jsdom

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// Cross-node attribute update batching.
//
// Every setAttribute/removeAttribute is one syscall/js bridge call, so a
// commit touching one attribute on each of 200 rows pays 200 hops — measured
// as ~78% of the primitive-attribute-update scenario's time. During a commit
// the adapter instead encodes attribute writes into one control-separated
// string and applies them with a single bridge call into a JS-side loop.
//
// Node identity crosses the bridge as an integer: nodes register lazily in a
// JS-side WeakRef table (id cached on the wasm node wrapper AND on the DOM
// node itself, so re-wrapped nodes reuse their id). WeakRefs keep removed
// subtrees collectable. Registration costs one bridge call per node once;
// steady-state commits pay one call total.
//
// The helpers are installed via eval, which a strict CSP may block — the
// capability then reports unavailable and every write takes the direct path.

const (
	attrBatchRecordSep = "\x1e"
	attrBatchFieldSep  = "\x1f"
)

const attrBatchHelperScript = `(() => {
	if (window.__gwcAttrNid) { return; }
	const reg = [null];
	const free = [];
	const fin = (typeof FinalizationRegistry === "function")
		? new FinalizationRegistry((id) => {
			const ref = reg[id];
			if (ref && !ref.deref()) { reg[id] = null; free.push(id); }
		})
		: null;
	window.__gwcAttrNid = (el) => {
		let id = el.__gwcAttrId;
		if (id === undefined) {
			id = free.length ? free.pop() : reg.length;
			reg[id] = new WeakRef(el);
			el.__gwcAttrId = id;
			if (fin) { fin.register(el, id); }
		}
		return id;
	};
	window.__gwcAttrFlush = (payload) => {
		const recs = payload.split("");
		for (let i = 0; i < recs.length; i++) {
			const r = recs[i];
			if (!r) { continue; }
			const parts = r.slice(1).split("");
			const ref = reg[+parts[0]];
			const el = ref && ref.deref();
			if (!el) { continue; }
			if (r[0] === "a") {
				el.setAttribute(parts[1], parts[2]);
			} else {
				el.removeAttribute(parts[1]);
			}
		}
	};
})()`

// ensureAttrBatchHelpers installs the JS-side registry and flush loop once.
func (parseA *WASMDOMAdapter) ensureAttrBatchHelpers() bool {
	if parseA.attrHelpersBound {
		return !parseA.attrHelpersFailed
	}
	parseA.attrHelpersBound = true
	defer func() {
		if recover() != nil {
			parseA.attrHelpersFailed = true
		}
	}()
	parseEval := js.Global().Get("eval")
	if parseEval.Type() != js.TypeFunction {
		parseA.attrHelpersFailed = true
		return false
	}
	parseEval.Invoke(attrBatchHelperScript)
	parseA.attrRegisterNode = js.Global().Get("__gwcAttrNid")
	parseA.attrFlushBatch = js.Global().Get("__gwcAttrFlush")
	if parseA.attrRegisterNode.Type() != js.TypeFunction || parseA.attrFlushBatch.Type() != js.TypeFunction {
		parseA.attrHelpersFailed = true
		return false
	}
	return true
}

// attrBatchNodeID returns the node's JS-side registry id, registering lazily.
func (parseA *WASMDOMAdapter) attrBatchNodeID(parseNode *WASMDOMNode) int {
	if parseNode.batchID > 0 {
		return parseNode.batchID
	}
	parseID := parseA.attrRegisterNode.Invoke(parseNode.value).Int()
	parseNode.batchID = parseID
	return parseID
}

// BeginAttrUpdateBatch starts buffering attribute writes for the current
// commit. Reads of attributes must not occur until EndAttrUpdateBatch.
func (parseA *WASMDOMAdapter) BeginAttrUpdateBatch() {
	if !parseA.ensureAttrBatchHelpers() {
		return
	}
	parseA.attrBatchActive = true
	parseA.attrBatchPayload.Reset()
}

// EndAttrUpdateBatch applies all buffered attribute writes in one bridge
// call. Idempotent; safe to call defensively.
func (parseA *WASMDOMAdapter) EndAttrUpdateBatch() {
	if !parseA.attrBatchActive {
		return
	}
	parseA.attrBatchActive = false
	if parseA.attrBatchPayload.Len() == 0 {
		return
	}
	parseA.attrFlushBatch.Invoke(parseA.attrBatchPayload.String())
	parseA.attrBatchPayload.Reset()
}

// queueAttrWrite buffers one attribute write, reporting false when the value
// cannot be encoded (contains the separators) so the caller writes directly.
func (parseA *WASMDOMAdapter) queueAttrWrite(parseNode *WASMDOMNode, parseOp byte, parseName, parseValue string) bool {
	if !parseA.attrBatchActive {
		return false
	}
	if strings.ContainsAny(parseName, attrBatchRecordSep+attrBatchFieldSep) ||
		strings.ContainsAny(parseValue, attrBatchRecordSep+attrBatchFieldSep) {
		return false
	}
	parseID := parseA.attrBatchNodeID(parseNode)
	parseBuilder := &parseA.attrBatchPayload
	parseBuilder.WriteByte(parseOp)
	parseBuilder.WriteString(itoaSmall(parseID))
	parseBuilder.WriteString(attrBatchFieldSep)
	parseBuilder.WriteString(parseName)
	if parseOp == 'a' {
		parseBuilder.WriteString(attrBatchFieldSep)
		parseBuilder.WriteString(parseValue)
	}
	parseBuilder.WriteString(attrBatchRecordSep)
	return true
}

// itoaSmall formats a non-negative int without strconv's interface costs on
// this very hot encode path.
func itoaSmall(parseValue int) string {
	if parseValue < 10 {
		return string([]byte{byte('0' + parseValue)})
	}
	var parseBuf [20]byte
	parseIndex := len(parseBuf)
	for parseValue > 0 {
		parseIndex--
		parseBuf[parseIndex] = byte('0' + parseValue%10)
		parseValue /= 10
	}
	return string(parseBuf[parseIndex:])
}

var _ runtime.DOMAdapter = (*WASMDOMAdapter)(nil)
