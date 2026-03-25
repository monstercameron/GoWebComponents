//go:build js && wasm
// +build js,wasm

package pwa

import "github.com/monstercameron/GoWebComponents/fetch"

// BuildMutationQueueDiagnosticsSource wraps a MutationQueue as a DiagnosticsSource for offline queue entries.
func BuildMutationQueueDiagnosticsSource(parseQueue *fetch.MutationQueue) func() ([]OfflineQueueEntry, error) {
	if parseQueue == nil {
		return nil
	}
	return func() ([]OfflineQueueEntry, error) {
		parseEntries, parseErr := parseQueue.List()
		if parseErr != nil {
			return nil, parseErr
		}
		parseResult := make([]OfflineQueueEntry, 0, len(parseEntries))
		for _, parseEntry := range parseEntries {
			parseResult = append(parseResult, OfflineQueueEntry{
				State:     string(parseEntry.State),
				CreatedAt: parseEntry.CreatedAt,
				UpdatedAt: parseEntry.UpdatedAt,
			})
		}
		return parseResult, nil
	}
}
