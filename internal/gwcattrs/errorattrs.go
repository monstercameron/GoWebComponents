package gwcattrs

import "strings"

// MaxComponentStackBytes caps gwc.component_stack so a runaway ancestry cannot
// bloat an exported span.
const MaxComponentStackBytes = 4096

// ParseErrorCode returns the diagnostic code at the start of parseMessage when it
// has the form "[GWC-...] text", which is how errcode errors render, and "" when it
// does not.
func ParseErrorCode(parseMessage string) string {
	parseMessage = strings.TrimSpace(parseMessage)
	if !strings.HasPrefix(parseMessage, "[GWC-") {
		return ""
	}
	parseEnd := strings.IndexByte(parseMessage, ']')
	if parseEnd < 0 {
		return ""
	}
	parseCode := parseMessage[1:parseEnd]
	for _, parseRune := range parseCode {
		if !(parseRune >= 'A' && parseRune <= 'Z' || parseRune >= '0' && parseRune <= '9' || parseRune == '-') {
			return ""
		}
	}
	return parseCode
}

// NormalizeErrorAttrs makes the error attributes of a failed event consistent
// across every exporter. It fills gwc.error.code from parseMessage when the event
// carries none, and trims and caps gwc.component_stack, removing it when empty.
// It returns a new map; parseAttrs is not modified and may be nil.
func NormalizeErrorAttrs(parseAttrs map[string]string, parseMessage string) map[string]string {
	parseOut := make(map[string]string, len(parseAttrs)+1)
	for parseKey, parseValue := range parseAttrs {
		parseOut[parseKey] = parseValue
	}
	if strings.TrimSpace(parseOut[KeyErrorCode]) == "" {
		delete(parseOut, KeyErrorCode)
		if parseCode := ParseErrorCode(parseMessage); parseCode != "" {
			parseOut[KeyErrorCode] = parseCode
		}
	}
	if parseStack, parseHas := parseOut[KeyComponentStack]; parseHas {
		parseStack = strings.TrimSpace(parseStack)
		if len(parseStack) > MaxComponentStackBytes {
			parseStack = strings.ToValidUTF8(parseStack[:MaxComponentStackBytes], "")
		}
		if parseStack == "" {
			delete(parseOut, KeyComponentStack)
		} else {
			parseOut[KeyComponentStack] = parseStack
		}
	}
	return parseOut
}