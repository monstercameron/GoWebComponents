package pwa

import (
	"context"
	"time"
)

type ManifestDiagnostics struct {
	Valid bool
	Error string
}

type OfflineQueueDiagnostics struct {
	TotalEntries    int
	QueuedEntries   int
	RetryingEntries int
	DeadEntries     int
	OldestCreatedAt time.Time
	LatestUpdatedAt time.Time
}

type OfflineQueueEntry struct {
	State     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StoragePressureDiagnostics struct {
	Available         bool
	Persistent        bool
	UsageBytes        int64
	QuotaBytes        int64
	IndexedDBBytes    int64
	CacheStorageBytes int64
	UsageRatio        float64
	Pressure          string
}

type DiagnosticsOptions struct {
	Manifest         *Manifest
	Installability   *InstallabilityManager
	ServiceWorker    *ServiceWorkerRegistration
	CacheStorage     *CacheStorageManager
	CacheStoragePlan *CacheStoragePlan
	OfflineQueue     func() ([]OfflineQueueEntry, error)
}

type DiagnosticsSnapshot struct {
	Manifest       ManifestDiagnostics
	Installability InstallabilityState
	ServiceWorker  ServiceWorkerSnapshot
	CacheStorage   CacheStorageSnapshot
	OfflineQueue   OfflineQueueDiagnostics
	Storage        StoragePressureDiagnostics
}

func summarizeOfflineQueue(parseOfflineEntries []OfflineQueueEntry) OfflineQueueDiagnostics {
	parseOfflineSummary := OfflineQueueDiagnostics{TotalEntries: len(parseOfflineEntries)}
	for _, parseOfflineEntry := range parseOfflineEntries {
		switch parseOfflineEntry.State {
		case "retrying":
			parseOfflineSummary.RetryingEntries++
		case "dead":
			parseOfflineSummary.DeadEntries++
		default:
			parseOfflineSummary.QueuedEntries++
		}
		if parseOfflineSummary.OldestCreatedAt.IsZero() || (!parseOfflineEntry.CreatedAt.IsZero() && parseOfflineEntry.CreatedAt.Before(parseOfflineSummary.OldestCreatedAt)) {
			parseOfflineSummary.OldestCreatedAt = parseOfflineEntry.CreatedAt
		}
		if parseOfflineEntry.UpdatedAt.After(parseOfflineSummary.LatestUpdatedAt) {
			parseOfflineSummary.LatestUpdatedAt = parseOfflineEntry.UpdatedAt
		}
	}
	return parseOfflineSummary
}

func pressureLabel(parsePressureRatio float64) string {
	switch {
	case parsePressureRatio >= 0.9:
		return "critical"
	case parsePressureRatio >= 0.75:
		return "elevated"
	case parsePressureRatio > 0:
		return "normal"
	default:
		return "unknown"
	}
}

func (parseStorage StoragePressureDiagnostics) normalized() StoragePressureDiagnostics {
	if parseStorage.QuotaBytes > 0 && parseStorage.UsageRatio == 0 {
		parseStorage.UsageRatio = float64(parseStorage.UsageBytes) / float64(parseStorage.QuotaBytes)
	}
	if parseStorage.Pressure == "" {
		parseStorage.Pressure = pressureLabel(parseStorage.UsageRatio)
	}
	return parseStorage
}

func _diagnosticsContext(_ context.Context) {}
