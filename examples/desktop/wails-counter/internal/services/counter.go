package services

import (
	"context"
	"errors"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v5/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// CounterService owns the native counter state and OS integrations.
type CounterService struct {
	Files      *desktop.FileDialogHost
	Native     *desktop.NativeHost
	Policy     desktop.FeaturePolicy
	parseMutex sync.Mutex
	parseValue int
	parseWork  contracts.WorkState
}

// GetCapabilities returns a native protocol handshake before the frontend starts.
// Build metadata reports the declared module version; doctor verifies local source pin.
func (parseService *CounterService) GetCapabilities() desktop.Capabilities {
	parseVersion := "unknown"
	if parseInfo, isAvailable := debug.ReadBuildInfo(); isAvailable {
		for _, parseDependency := range parseInfo.Deps {
			if parseDependency.Path == "github.com/wailsapp/wails/v3" {
				parseVersion = parseDependency.Version
				break
			}
		}
	}
	parseMethods := []string{"counter.increment", "counter.openFile", "counter.progress", "counter.reject", "counter.work"}
	parseFeatures := []desktop.Feature{}
	if parseService == nil || parseService.Policy.Allows(desktop.PersistentStorage) {
		parseMethods = append(parseMethods, "storage.load", "storage.save", "storage.delete", "storage.keys")
	}
	if parseService == nil || (parseService.Policy.Allows(desktop.ReportExport) && parseService.Files != nil && len(parseService.Files.GetMethods()) > 0) {
		parseMethods = append(parseMethods, "api.export")
	}
	parseMethods = append(parseMethods, "api.run", "api.report", "api.observe")
	if parseService != nil && parseService.Policy.Allows(desktop.FileDialogs) && parseService.Files != nil {
		parseMethods = append(parseMethods, parseService.Files.GetMethods()...)
		if len(parseService.Files.GetMethods()) > 0 {
			parseFeatures = append(parseFeatures, desktop.FileDialogs)
		}
	}
	if parseService != nil && parseService.Native != nil {
		parseMethods = append(parseMethods, parseService.Native.GetMethods()...)
		parseFeatures = append(parseFeatures, parseService.Native.GetFeatures()...)
	}
	if parseService == nil || parseService.Policy.Allows(desktop.NativeMenus) {
		parseFeatures = append(parseFeatures, desktop.NativeMenus)
	}
	if parseService == nil || parseService.Policy.Allows(desktop.PersistentStorage) {
		parseFeatures = append(parseFeatures, desktop.PersistentStorage)
	}
	if parseService == nil || (parseService.Policy.Allows(desktop.ReportExport) && parseService.Files != nil && len(parseService.Files.GetMethods()) > 0) {
		parseFeatures = append(parseFeatures, desktop.ReportExport)
	}
	return desktop.Capabilities{
		Protocol: desktop.ProtocolVersion, Platform: runtime.GOOS, HostVersion: parseVersion,
		Methods: parseMethods,
		Features: parseFeatures,
		Topics:  []string{"counter.progress", "storage.changed"},
	}
}

// GetWorkState snapshots host-side work so cancellation is verified independently of UI state.
func (parseService *CounterService) GetWorkState() contracts.WorkState {
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	return parseService.parseWork
}

// RunWork waits cooperatively for cancellation or completion after ten seconds.
func (parseService *CounterService) RunWork(parseContext context.Context) error {
	if parseContext == nil {
		return errors.New("work context unavailable")
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return parseErr
	}
	parseService.parseMutex.Lock()
	parseService.parseWork.Active++
	parseService.parseMutex.Unlock()
	defer func() {
		parseService.parseMutex.Lock()
		parseService.parseWork.Active--
		parseService.parseMutex.Unlock()
	}()
	parseTimer := time.NewTimer(10 * time.Second)
	defer parseTimer.Stop()
	select {
	case <-parseContext.Done():
		parseService.parseMutex.Lock()
		parseService.parseWork.Cancelled++
		parseService.parseMutex.Unlock()
		return parseContext.Err()
	case <-parseTimer.C:
		return nil
	}
}

// Reject demonstrates a native service failure without a successful mutation.
func (parseService *CounterService) Reject() error {
	return errors.New("counter expected rejection")
}

// Increment updates the counter and returns its portable state.
func (parseService *CounterService) Increment() contracts.CounterState {
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	parseService.parseValue++
	return contracts.CounterState{Value: parseService.parseValue}
}

// OpenFile displays the native picker, preserving cancellation as an empty result.
func (parseService *CounterService) OpenFile(parseContext context.Context) (string, error) {
	parseSelection, parseErr := selectFileDialog(parseContext, parseService.Files, "open-file", desktop.FileDialogOptions{Title: "Choose a file"})
	if parseErr != nil || parseSelection.Cancelled {
		return "", parseErr
	}
	return parseSelection.Paths[0], nil
}

// RunProgress emits a small native progress sequence to the current window.
func (parseService *CounterService) RunProgress(parseContext context.Context) error {
	parseWindow, parseErr := getCallingWindow(parseContext)
	if parseErr != nil {
		return parseErr
	}
	for parsePercent := 0; parsePercent <= 100; parsePercent += 25 {
		select {
		case <-parseContext.Done():
			return parseContext.Err()
		default:
		}
		parseWindow.EmitEvent("counter.progress", contracts.Progress{Percent: parsePercent, Stage: "native"})
		parseTimer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-parseTimer.C:
		case <-parseContext.Done():
			parseTimer.Stop()
			return parseContext.Err()
		}
	}
	return nil
}

// getCallingWindow resolves the RPC caller rather than whichever window has focus.
func getCallingWindow(parseContext context.Context) (application.Window, error) {
	if parseContext == nil {
		return nil, errors.New("counter call context is unavailable")
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return nil, parseErr
	}
	parseWindow, isWindow := parseContext.Value(application.WindowKey).(application.Window)
	if !isWindow || parseWindow == nil {
		return nil, errors.New("counter calling window is unavailable")
	}
	return parseWindow, nil
}
