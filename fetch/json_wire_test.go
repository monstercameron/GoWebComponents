package fetch

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFetchJSONWireShapePreservesZeroTimestampFields(parseT *testing.T) {
	parseCases := []struct {
		name     string
		value    any
		expected []string
	}{
		{
			name:     "cache bootstrap entry",
			value:    CacheBootstrapEntry{},
			expected: []string{`"updatedAt":"0001-01-01T00:00:00Z"`},
		},
		{
			name:  "persisted cache record",
			value: persistedCachedResource{},
			expected: []string{
				`"updatedAt":"0001-01-01T00:00:00Z"`,
				`"lastLoaded":"0001-01-01T00:00:00Z"`,
			},
		},
		{
			name:  "queued mutation",
			value: QueuedMutation{},
			expected: []string{
				`"createdAt":"0001-01-01T00:00:00Z"`,
				`"updatedAt":"0001-01-01T00:00:00Z"`,
				`"nextAttemptAt":"0001-01-01T00:00:00Z"`,
			},
		},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT2 *testing.T) {
			parseEncoded, parseErr := json.Marshal(parseCase.value)
			if parseErr != nil {
				parseT2.Fatalf("marshal %s: %v", parseCase.name, parseErr)
			}
			parseText := string(parseEncoded)
			for _, parseExpected := range parseCase.expected {
				if !strings.Contains(parseText, parseExpected) {
					parseT2.Fatalf("expected %s to preserve %s, got %s", parseCase.name, parseExpected, parseText)
				}
			}
		})
	}
}
