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

func summarizeOfflineQueue(parseEntries []OfflineQueueEntry) OfflineQueueDiagnostics {
	parseSummary := OfflineQueueDiagnostics{TotalEntries: len(parseEntries)}
	for _, parseEntry := range parseEntries {
		switch parseEntry.State {
		case "retrying":
			parseSummary.RetryingEntries++
		case "dead":
			parseSummary.DeadEntries++
		default:
			parseSummary.QueuedEntries++
		}
		if parseSummary.OldestCreatedAt.IsZero() || (!parseEntry.CreatedAt.IsZero() && parseEntry.CreatedAt.Before(parseSummary.OldestCreatedAt)) {
			parseSummary.OldestCreatedAt = parseEntry.CreatedAt
		}
		if parseEntry.UpdatedAt.After(parseSummary.LatestUpdatedAt) {
			parseSummary.LatestUpdatedAt = parseEntry.UpdatedAt
		}
	}
	return parseSummary
}

func pressureLabel(parseRatio float64) string {
	switch {
	case parseRatio >= 0.9:
		return "critical"
	case parseRatio >= 0.75:
		return "elevated"
	case parseRatio > 0:
		return "normal"
	default:
		return "unknown"
	}
}

func (parseS StoragePressureDiagnostics) normalized() StoragePressureDiagnostics {
	if parseS.QuotaBytes > 0 && parseS.UsageRatio == 0 {
		parseS.UsageRatio = float64(parseS.UsageBytes) / float64(parseS.QuotaBytes)
	}
	if parseS.Pressure == "" {
		parseS.Pressure = pressureLabel(parseS.UsageRatio)
	}
	return parseS
}

func _diagnosticsContext(_ context.Context) {}
