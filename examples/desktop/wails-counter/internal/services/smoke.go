package services

import (
	"errors"
	"fmt"
	"sync"

	"example.com/gwc-wails-counter/contracts"
)

// SmokeService accepts evidence only when explicitly registered by the smoke host.
type SmokeService struct {
	parseMutex    sync.Mutex
	parseReported bool
	parseResults  chan<- contracts.SmokeReport
	parseObserver contracts.ObserverState
	parseInitial  *contracts.ObserverState
}

// ReportObserver records only smoke-window rendered state, not persistence itself.
func (parseService *SmokeService) ReportObserver(parseState contracts.ObserverState) {
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	parseService.parseObserver = parseState
	if parseService.parseInitial == nil && parseState.Ready && parseState.Local == 0 {
		parseCopy := parseState
		parseService.parseInitial = &parseCopy
	}
}

// GetObserver returns the second window's latest rendered evidence.
func (parseService *SmokeService) GetObserver() contracts.ObserverState {
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	if parseService.parseInitial != nil {
		parseValue := *parseService.parseInitial
		parseService.parseInitial = nil
		return parseValue
	}
	return parseService.parseObserver
}

// NewSmokeService creates a reporter for an explicit smoke-test invocation.
func NewSmokeService(parseResults chan<- contracts.SmokeReport) *SmokeService {
	return &SmokeService{parseResults: parseResults}
}

// GetMode identifies a host started with the smoke-test flag.
func (parseService *SmokeService) GetMode() bool { return true }

// Reject exercises the generated binding's native error path.
func (parseService *SmokeService) Reject() error {
	return errors.New("smoke expected rejection")
}

// ReportResult accepts one report, rejecting incomplete success claims.
func (parseService *SmokeService) ReportResult(parseReport contracts.SmokeReport) error {
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	if parseService.parseReported {
		return errors.New("smoke result already reported")
	}
	if parseReport.OK {
		parseChecks := make(map[string]bool, len(parseReport.Checks))
		for _, parseCheck := range parseReport.Checks {
			parseChecks[parseCheck] = true
		}
		for _, parseRequired := range []string{"dom", "local-counter", "native-call", "native-error", "native-event", "routing", "route-remount", "wasm-mime", "csp", "backend-cancellation", "unmount-cancellation", "adapter-cleanup", "native-handshake", "origin-guard", "api-tester-ui", "api-session-report", "desktop-file-contract", "native-sdk-contract"} {
			if !parseChecks[parseRequired] {
				return fmt.Errorf("smoke report missing check %q", parseRequired)
			}
		}
		if parseChecks["persistent-storage-disabled"] {
			if parseChecks["durable-native-state"] || parseChecks["two-window-invalidation"] {
				return errors.New("disabled persistence report contains durable-state evidence")
			}
		} else {
			for _, parseRequired := range []string{"durable-native-state", "two-window-invalidation", "window-local-state"} {
				if !parseChecks[parseRequired] {
					return fmt.Errorf("smoke report missing check %q", parseRequired)
				}
			}
		}
		if parseChecks["native-window-disabled"] {
			if parseChecks["api-window-info"] {
				return errors.New("disabled window report contains native window evidence")
			}
		} else if !parseChecks["api-window-info"] {
			return errors.New("smoke report missing check \"api-window-info\"")
		}
		if parseReport.Error != "" {
			return errors.New("successful smoke report contains an error")
		}
	} else if parseReport.Error == "" {
		return errors.New("failed smoke report needs an error")
	}
	if parseService.parseResults == nil {
		return errors.New("smoke result receiver is unavailable")
	}
	select {
	case parseService.parseResults <- parseReport:
		parseService.parseReported = true
		return nil
	default:
		return errors.New("smoke result receiver is not ready")
	}
}
