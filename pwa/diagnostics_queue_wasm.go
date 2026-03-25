//go:build js && wasm
// +build js,wasm

package pwa

import "github.com/monstercameron/GoWebComponents/fetch"

// BuildMutationQueueDiagnosticsSource wraps a MutationQueue as a DiagnosticsSource for offline queue entries.
func BuildMutationQueueDiagnosticsSource(parseMutationQueue *fetch.MutationQueue) func() ([]OfflineQueueEntry, error) {
	if parseMutationQueue == nil {
		return nil
	}
	return func() ([]OfflineQueueEntry, error) {
		parseQueueEntries, parseQueueErr := parseMutationQueue.List()
		if parseQueueErr != nil {
			return nil, parseQueueErr
		}
		parseQueueResult := make([]OfflineQueueEntry, 0, len(parseQueueEntries))
		for _, parseQueueEntry := range parseQueueEntries {
			parseQueueResult = append(parseQueueResult, OfflineQueueEntry{
				State:     string(parseQueueEntry.State),
				CreatedAt: parseQueueEntry.CreatedAt,
				UpdatedAt: parseQueueEntry.UpdatedAt,
			})
		}
		return parseQueueResult, nil
	}
}
