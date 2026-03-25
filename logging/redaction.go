package logging

import "strings"

const redactedInteractionValue = "[redacted]"

func redactInteractionValue(parseInputKind string, parseInputValue string) string {
	if strings.EqualFold(strings.TrimSpace(parseInputKind), "password") {
		return redactedInteractionValue
	}
	return parseInputValue
}
