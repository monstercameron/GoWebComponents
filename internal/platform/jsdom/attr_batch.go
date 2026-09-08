//go:build js && wasm

package jsdom

import (
	"strconv"
	"syscall/js"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// wasmDOMMutation is deliberately typed. Building a []interface{} with four
// entries per mutation costs more in Go wasm than the JavaScript loop saves.
type wasmDOMMutation struct {
	node  *WASMDOMNode
	op    byte
	name  string
	value string
}

// Mutated nodes receive a JavaScript-side WeakRef ID on first use. Subsequent
// dense commits cross the Go/JS bridge once with a length-prefixed payload;
// sparse commits stay as direct DOM calls. Length prefixes make every string,
// including control characters, lossless.
const attrBatchHelperScript = `(() => {
	if (!window.__gwcAttrRegister || !window.__gwcAttrFlush) {
		const refs = [];
		const hasWeakRef = typeof WeakRef === "function";
		const finalizer = typeof FinalizationRegistry === "function"
			? new FinalizationRegistry((id) => { refs[id] = undefined; })
			: null;
		let nextID = 1;
		window.__gwcAttrRegister = (node) => {
			const id = nextID++;
			refs[id] = hasWeakRef ? new WeakRef(node) : node;
			if (finalizer) { finalizer.register(node, id); }
			return id;
		};
		window.__gwcAttrFlush = (payload) => {
			let i = 0;
			const readNumber = () => {
				let value = 0;
				while (i < payload.length) {
					const code = payload.charCodeAt(i++);
					if (code === 58) { break; }
					value = value * 10 + code - 48;
				}
				return value;
			};
			while (i < payload.length) {
				const op = payload.charCodeAt(i++);
				const id = readNumber();
				const nameLength = readNumber();
				const valueLength = readNumber();
				const name = payload.slice(i, i + nameLength);
				i += nameLength;
				const value = payload.slice(i, i + valueLength);
				i += valueLength;
				const entry = refs[id];
				const node = hasWeakRef ? entry?.deref() : entry;
				if (!node) { continue; }
				if (op === 97) {
					node.setAttribute(name, value);
				} else if (op === 114) {
					node.removeAttribute(name);
				} else if (op === 116) {
					if (node.nodeType === 3) {
						node.nodeValue = name;
					} else if (name !== "" && node.firstChild && node.firstChild.nodeType === 3 && !node.firstChild.nextSibling) {
						node.firstChild.nodeValue = name;
					} else {
						node.textContent = name;
					}
				}
			}
		};
	}
	if (!window.__gwcRemoveFlush) {
		window.__gwcRemoveFlush = (...pairs) => {
			for (let i = 0; i + 1 < pairs.length; i += 2) {
				const parent = pairs[i];
				const child = pairs[i + 1];
				if (child && child.parentNode === parent) { child.remove(); }
			}
		};
	}
})()`

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
	parseA.attrRegisterNode = js.Global().Get("__gwcAttrRegister")
	parseA.attrFlushBatch = js.Global().Get("__gwcAttrFlush")
	parseA.removeFlushBatch = js.Global().Get("__gwcRemoveFlush")
	if parseA.attrRegisterNode.Type() != js.TypeFunction || parseA.attrFlushBatch.Type() != js.TypeFunction || parseA.removeFlushBatch.Type() != js.TypeFunction {
		parseA.attrHelpersFailed = true
		return false
	}
	return true
}

func (parseA *WASMDOMAdapter) BeginAttrUpdateBatch() {
	if !parseA.ensureAttrBatchHelpers() {
		return
	}
	parseA.attrBatchActive = true
	clear(parseA.mutationBatchOps)
	parseA.mutationBatchOps = parseA.mutationBatchOps[:0]
	clear(parseA.removeBatchPairs)
	parseA.removeBatchPairs = parseA.removeBatchPairs[:0]
}

const denseMutationBatchThreshold = 12

func (parseA *WASMDOMAdapter) EndAttrUpdateBatch() {
	if !parseA.attrBatchActive {
		return
	}
	parseA.attrBatchActive = false
	for parsePairs := parseA.removeBatchPairs; len(parsePairs) > 0; {
		parseChunkSize := min(len(parsePairs), 256)
		parseA.removeFlushBatch.Invoke(parsePairs[:parseChunkSize]...)
		parsePairs = parsePairs[parseChunkSize:]
	}
	clear(parseA.removeBatchPairs)
	parseA.removeBatchPairs = parseA.removeBatchPairs[:0]

	if len(parseA.mutationBatchOps) < denseMutationBatchThreshold {
		for parseIndex := range parseA.mutationBatchOps {
			parseA.applyMutationDirect(&parseA.mutationBatchOps[parseIndex])
		}
	} else {
		parseA.attrBatchPayload = parseA.attrBatchPayload[:0]
		for parseIndex := range parseA.mutationBatchOps {
			parseMutation := &parseA.mutationBatchOps[parseIndex]
			if parseMutation.node.batchID == 0 {
				parseMutation.node.batchID = parseA.attrRegisterNode.Invoke(parseMutation.node.value).Int()
			}
			parseA.appendMutationRecord(parseMutation)
		}
		parseA.attrFlushBatch.Invoke(string(parseA.attrBatchPayload))
	}
	clear(parseA.mutationBatchOps)
	parseA.mutationBatchOps = parseA.mutationBatchOps[:0]
}

func (parseA *WASMDOMAdapter) applyMutationDirect(parseMutation *wasmDOMMutation) {
	if parseMutation == nil || parseMutation.node == nil {
		return
	}
	switch parseMutation.op {
	case 'a':
		parseMutation.node.value.Call("setAttribute", parseMutation.name, parseMutation.value)
	case 'r':
		parseMutation.node.value.Call("removeAttribute", parseMutation.name)
	case 't':
		parseA.SetTextContent(parseMutation.node, parseMutation.name)
	}
}

func (parseA *WASMDOMAdapter) appendMutationRecord(parseMutation *wasmDOMMutation) {
	parseA.attrBatchPayload = append(parseA.attrBatchPayload, parseMutation.op)
	parseA.attrBatchPayload = strconv.AppendInt(parseA.attrBatchPayload, int64(parseMutation.node.batchID), 10)
	parseA.attrBatchPayload = append(parseA.attrBatchPayload, ':')
	parseA.attrBatchPayload = strconv.AppendInt(parseA.attrBatchPayload, int64(jsUTF16Length(parseMutation.name)), 10)
	parseA.attrBatchPayload = append(parseA.attrBatchPayload, ':')
	parseA.attrBatchPayload = strconv.AppendInt(parseA.attrBatchPayload, int64(jsUTF16Length(parseMutation.value)), 10)
	parseA.attrBatchPayload = append(parseA.attrBatchPayload, ':')
	parseA.attrBatchPayload = append(parseA.attrBatchPayload, parseMutation.name...)
	parseA.attrBatchPayload = append(parseA.attrBatchPayload, parseMutation.value...)
}

func jsUTF16Length(parseValue string) int {
	parseLength := 0
	for len(parseValue) > 0 {
		parseRune, parseSize := utf8.DecodeRuneInString(parseValue)
		parseValue = parseValue[parseSize:]
		parseLength++
		if parseRune > 0xffff {
			parseLength++
		}
	}
	return parseLength
}

func (parseA *WASMDOMAdapter) queueAttrWrite(parseNode *WASMDOMNode, parseOp byte, parseName, parseValue string) bool {
	if !parseA.attrBatchActive {
		return false
	}
	parseA.mutationBatchOps = append(parseA.mutationBatchOps, wasmDOMMutation{node: parseNode, op: parseOp, name: parseName, value: parseValue})
	return true
}

func (parseA *WASMDOMAdapter) queueTextWrite(parseNode *WASMDOMNode, parseText string) bool {
	if !parseA.attrBatchActive {
		return false
	}
	parseA.mutationBatchOps = append(parseA.mutationBatchOps, wasmDOMMutation{node: parseNode, op: 't', name: parseText})
	return true
}

func (parseA *WASMDOMAdapter) queueChildRemoval(parseParent, parseChild *WASMDOMNode) bool {
	if !parseA.attrBatchActive {
		return false
	}
	parseA.removeBatchPairs = append(parseA.removeBatchPairs, parseParent.value, parseChild.value)
	return true
}

var _ runtime.DOMAdapter = (*WASMDOMAdapter)(nil)
