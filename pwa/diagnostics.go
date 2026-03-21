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

func summarizeOfflineQueue(entries []OfflineQueueEntry) OfflineQueueDiagnostics {
	summary := OfflineQueueDiagnostics{TotalEntries: len(entries)}
	for _, entry := range entries {
		switch entry.State {
		case "retrying":
			summary.RetryingEntries++
		case "dead":
			summary.DeadEntries++
		default:
			summary.QueuedEntries++
		}
		if summary.OldestCreatedAt.IsZero() || (!entry.CreatedAt.IsZero() && entry.CreatedAt.Before(summary.OldestCreatedAt)) {
			summary.OldestCreatedAt = entry.CreatedAt
		}
		if entry.UpdatedAt.After(summary.LatestUpdatedAt) {
			summary.LatestUpdatedAt = entry.UpdatedAt
		}
	}
	return summary
}

func pressureLabel(ratio float64) string {
	switch {
	case ratio >= 0.9:
		return "critical"
	case ratio >= 0.75:
		return "elevated"
	case ratio > 0:
		return "normal"
	default:
		return "unknown"
	}
}

func (s StoragePressureDiagnostics) normalized() StoragePressureDiagnostics {
	if s.QuotaBytes > 0 && s.UsageRatio == 0 {
		s.UsageRatio = float64(s.UsageBytes) / float64(s.QuotaBytes)
	}
	if s.Pressure == "" {
		s.Pressure = pressureLabel(s.UsageRatio)
	}
	return s
}

func _diagnosticsContext(_ context.Context) {}
