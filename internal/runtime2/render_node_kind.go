package runtime2

import "fmt"

// RenderNodeKind identifies the node category encoded in render IR.
type RenderNodeKind uint8

const (
	renderNodeKindInvalid RenderNodeKind = iota

	RenderNodeKindText
	RenderNodeKindHostElement
	RenderNodeKindFragment
)

// ParseRenderNodeKind decodes and validates a raw render-node kind value.
func ParseRenderNodeKind(parseRaw uint8) (RenderNodeKind, error) {
	parseKind := RenderNodeKind(parseRaw)
	switch parseKind {
	case RenderNodeKindText, RenderNodeKindHostElement, RenderNodeKindFragment:
		return parseKind, nil
	case renderNodeKindInvalid:
		return renderNodeKindInvalid, fmt.Errorf("runtime2: render node kind %d is invalid", parseRaw)
	default:
		return renderNodeKindInvalid, fmt.Errorf("runtime2: render node kind %d is unsupported", parseRaw)
	}
}
