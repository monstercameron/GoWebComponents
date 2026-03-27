package runtime2

import "fmt"

// PatchOpCode identifies one patch operation type in a patch stream.
type PatchOpCode uint8

const (
	patchOpCodeInvalid PatchOpCode = iota

	PatchOpCodeInsertNode
	PatchOpCodeRemoveNode
	PatchOpCodeSetText
	PatchOpCodeSetAttr
	PatchOpCodeRemoveAttr
	PatchOpCodeSetStyle
	PatchOpCodeRemoveStyle
	PatchOpCodeMoveKeyedChild
	PatchOpCodeReplaceSubtree
)

// ParsePatchOpCode decodes and validates one raw patch op code.
func ParsePatchOpCode(parseRaw uint8) (PatchOpCode, error) {
	parseCode := PatchOpCode(parseRaw)
	switch parseCode {
	case PatchOpCodeInsertNode,
		PatchOpCodeRemoveNode,
		PatchOpCodeSetText,
		PatchOpCodeSetAttr,
		PatchOpCodeRemoveAttr,
		PatchOpCodeSetStyle,
		PatchOpCodeRemoveStyle,
		PatchOpCodeMoveKeyedChild,
		PatchOpCodeReplaceSubtree:
		return parseCode, nil
	case patchOpCodeInvalid:
		return patchOpCodeInvalid, fmt.Errorf("runtime2: patch op code %d is invalid", parseRaw)
	default:
		return patchOpCodeInvalid, fmt.Errorf("runtime2: patch op code %d is unsupported", parseRaw)
	}
}
