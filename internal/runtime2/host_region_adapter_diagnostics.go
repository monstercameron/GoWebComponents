package runtime2

import (
	"fmt"
	"strings"
)

func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionDiagnosticEnvelope(parseEnvelope ControlEnvelope) (HostRegionDiagnosticResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionDiagnosticResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	applyControlDiagnosticRedaction(&parseEnvelope)
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return HostRegionDiagnosticResult{}, parseErr
	}
	return parseHostRegionAdapter.handleHostRegionDiagnosticEnvelopeValidated(parseEnvelope)
}

// handleHostRegionDiagnosticEnvelopeValidated records one already-validated and redacted diagnostic control envelope.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionDiagnosticEnvelopeValidated(parseEnvelope ControlEnvelope) (HostRegionDiagnosticResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionDiagnosticResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseEnvelope.Kind != ControlKindDiagnostic {
		return HostRegionDiagnosticResult{}, fmt.Errorf("runtime2: control kind %q is not diagnostic", parseEnvelope.Kind)
	}
	if parseEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostRegionDiagnosticResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot store diagnostic for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseEnvelope.RegionInstanceID,
		)
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionDiagnosticResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if hasDiagnosticIgnored, getIgnoreReason := parseHostRegionAdapter.shouldHostRegionIgnoreDiagnostic(parseEnvelope, getCoordinatorEntry); hasDiagnosticIgnored {
		if shouldHostRegionCountStaleDiagnosticIgnore(getIgnoreReason) {
			if _, parseCountErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorIncrementRegionIgnoredStaleDiagnosticCountTrusted(parseHostRegionAdapter.storeRegionInstanceID); parseCountErr != nil {
				return HostRegionDiagnosticResult{}, parseCountErr
			}
		}
		return HostRegionDiagnosticResult{
			HasIgnored:       true,
			GetIgnoreReason:  getIgnoreReason,
			GetDiagnostic:    parseEnvelope,
			GetDiagnosticLen: len(parseHostRegionAdapter.storeHostRegionDiagnosticRing),
		}, nil
	}
	parseHostRegionAdapter.storeHostRegionDiagnosticDowngrade(parseEnvelope)
	parseHostRegionAdapter.storeHostRegionDiagnosticEnvelope(parseEnvelope)
	return HostRegionDiagnosticResult{
		HasStored:        true,
		GetDiagnostic:    parseEnvelope,
		GetDiagnosticLen: len(parseHostRegionAdapter.storeHostRegionDiagnosticRing),
	}, nil
}

// storeHostRegionDiagnosticEnvelope appends one diagnostic envelope while keeping only the newest bounded ring entries.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionDiagnosticEnvelope(parseEnvelope ControlEnvelope) {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionDiagnosticRing = append(parseHostRegionAdapter.storeHostRegionDiagnosticRing, parseEnvelope)
	if len(parseHostRegionAdapter.storeHostRegionDiagnosticRing) <= getHostRegionDiagnosticRingLimit {
		return
	}
	parseDropCount := len(parseHostRegionAdapter.storeHostRegionDiagnosticRing) - getHostRegionDiagnosticRingLimit
	parseHostRegionAdapter.storeHostRegionDiagnosticRing = append(
		[]ControlEnvelope(nil),
		parseHostRegionAdapter.storeHostRegionDiagnosticRing[parseDropCount:]...,
	)
}

// shouldHostRegionCountStaleDiagnosticIgnore reports whether one diagnostic ignore reason should increment stale-diagnostic counters.
func shouldHostRegionCountStaleDiagnosticIgnore(parseIgnoreReason string) bool {
	return strings.HasPrefix(parseIgnoreReason, "stale-")
}

// storeHostRegionDiagnosticDowngrade records one diagnostic downgrade into snapshot or patch accounting based on diagnostic event kind.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionDiagnosticDowngrade(parseEnvelope ControlEnvelope) {
	if parseHostRegionAdapter == nil || parseEnvelope.DiagnosticDowngrade == nil {
		return
	}
	parseDiagnosticKind, parseKindErr := ParseDiagnosticEventKind(parseEnvelope.DiagnosticType)
	if parseKindErr != nil {
		return
	}
	parseDowngrade := *parseEnvelope.DiagnosticDowngrade
	switch parseDiagnosticKind {
	case DiagnosticEventKindMount, DiagnosticEventKindUpdate:
		parseHostRegionAdapter.storeHostRegionSnapshotDowngrade = parseDowngrade
		parseHostRegionAdapter.hasHostRegionSnapshotDowngrade = true
		if parseEnvelope.TransportTier != "" {
			parseHostRegionAdapter.storeHostRegionSnapshotTier = parseEnvelope.TransportTier
		}
	case DiagnosticEventKindPatchReady:
		parseHostRegionAdapter.storeHostRegionPatchDowngrade = parseDowngrade
		parseHostRegionAdapter.hasHostRegionPatchDowngrade = true
		if parseEnvelope.TransportTier != "" {
			parseHostRegionAdapter.storeHostRegionPatchTier = parseEnvelope.TransportTier
			parseHostRegionAdapter.storeHostRegionTransportTier = parseEnvelope.TransportTier
		}
	}
}

// shouldHostRegionIgnoreDiagnostic reports whether one diagnostic envelope is stale for the region's latest fallback or repair state.
func (parseHostRegionAdapter *HostRegionAdapter) shouldHostRegionIgnoreDiagnostic(parseEnvelope ControlEnvelope, parseCoordinatorEntry CoordinatorEntry) (bool, string) {
	if parseHostRegionAdapter == nil {
		return false, ""
	}
	getRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
	getFallbackState, hasFallbackState := parseHostRegionAdapter.storeRecoveryCoordinator.GetRegionFallbackState(getRegionID)
	if parseEnvelope.Epoch > 0 {
		if parseEnvelope.Epoch < parseCoordinatorEntry.Epoch {
			return true, "stale-epoch"
		}
		if parseHostRegionAdapter.isHostRegionRepairPending &&
			parseHostRegionAdapter.storeHostRegionRepairRemountEpoch > 0 &&
			parseEnvelope.Epoch < parseHostRegionAdapter.storeHostRegionRepairRemountEpoch {
			return true, "stale-repair-epoch"
		}
		if hasFallbackState && getFallbackState.GetEpoch > 0 && parseEnvelope.Epoch < getFallbackState.GetEpoch {
			return true, "stale-fallback-epoch"
		}
	}
	getVersionFloor := parseHostRegionAdapter.getHostRegionDiagnosticVersionFloor(parseCoordinatorEntry)
	if parseEnvelope.InputVersion > 0 {
		if parseEnvelope.InputVersion < getVersionFloor {
			return true, "stale-version"
		}
		if hasFallbackState && getFallbackState.GetInputVersion > 0 && parseEnvelope.InputVersion < getFallbackState.GetInputVersion {
			return true, "stale-fallback-version"
		}
	}
	if parseHostRegionAdapter.storeRecoveryCoordinator.HandleWorkerDiagnostic(
		getRegionID,
		parseEnvelope.Epoch,
		parseEnvelope.InputVersion,
	).HasIgnored {
		return true, "fallback-active"
	}
	return false, ""
}

// getHostRegionDiagnosticVersionFloor reports the oldest diagnostic input version still considered current.
func (parseHostRegionAdapter *HostRegionAdapter) getHostRegionDiagnosticVersionFloor(parseCoordinatorEntry CoordinatorEntry) uint64 {
	if parseHostRegionAdapter == nil {
		return 0
	}
	return parseHostRegionMaxVersion(
		parseCoordinatorEntry.LastCommittedVersion,
		parseCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	)
}

// SetHostRegionPatchTransportTier stores one latest patch transport tier selected by worker patch-ready signaling.
func (parseHostRegionAdapter *HostRegionAdapter) SetHostRegionPatchTransportTier(parseTransportTier TransportTier) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	getTransportTier, parseTierErr := ParseTransportTier(string(parseTransportTier))
	if parseTierErr != nil {
		return parseTierErr
	}
	parseHostRegionAdapter.storeHostRegionPatchTier = getTransportTier
	parseHostRegionAdapter.storeHostRegionTransportTier = getTransportTier
	return nil
}

// clearHostRegionSourceSnapshotCache clears cached source snapshot state used by host update snapshot capture.
