package logging

import "strings"

const redactedInteractionValue = "[redacted]"

func redactInteractionValue(inputType string, value string) string {
	if strings.EqualFold(strings.TrimSpace(inputType), "password") {
		return redactedInteractionValue
	}
	return value
}
