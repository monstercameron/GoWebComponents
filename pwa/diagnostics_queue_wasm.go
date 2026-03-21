//go:build js && wasm
// +build js,wasm

package pwa

import "github.com/monstercameron/GoWebComponents/fetch"

func MutationQueueDiagnosticsSource(queue *fetch.MutationQueue) func() ([]OfflineQueueEntry, error) {
	if queue == nil {
		return nil
	}
	return func() ([]OfflineQueueEntry, error) {
		entries, err := queue.List()
		if err != nil {
			return nil, err
		}
		result := make([]OfflineQueueEntry, 0, len(entries))
		for _, entry := range entries {
			result = append(result, OfflineQueueEntry{
				State:     string(entry.State),
				CreatedAt: entry.CreatedAt,
				UpdatedAt: entry.UpdatedAt,
			})
		}
		return result, nil
	}
}
