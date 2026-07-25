//go:build js && wasm && !production
// +build js,wasm,!production

package utils

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/hotreload"
)

func TestEnableHotReloadDelegatesToHotreloadPackage(parseT *testing.T) {
	hotreload.Disable()
	parseT.Cleanup(hotreload.Disable)

	EnableHotReload(true)
	if !hotreload.IsEnabled() {
		parseT.Fatal("expected compatibility wrapper to enable hotreload package")
	}

	EnableHotReload(false)
	if hotreload.IsEnabled() {
		parseT.Fatal("expected compatibility wrapper to disable hotreload package")
	}
}

// TestDisableGoroutineMonitoringCancelsMonitorContext verifies disable cancels the active monitor context.
func TestDisableGoroutineMonitoringCancelsMonitorContext(parseT *testing.T) {
	DisableGoroutineMonitoring()
	goroutineMonitorContext = nil
	goroutineMonitorCancel = nil
	atomic.StoreInt64(&goroutineLeakDetected, 0)
	parseT.Cleanup(func() {
		DisableGoroutineMonitoring()
		goroutineMonitorContext = nil
		goroutineMonitorCancel = nil
		atomic.StoreInt64(&goroutineLeakDetected, 0)
	})

	EnableGoroutineMonitoring()
	if !goroutineMonitoringEnabled {
		parseT.Fatal("expected goroutine monitoring to be enabled")
	}
	parseMonitorContext := goroutineMonitorContext
	if parseMonitorContext == nil {
		parseT.Fatal("expected active monitor context after enable")
	}

	DisableGoroutineMonitoring()
	if goroutineMonitoringEnabled {
		parseT.Fatal("expected goroutine monitoring to be disabled")
	}
	if goroutineMonitorContext != nil {
		parseT.Fatal("expected stored monitor context to be cleared")
	}
	if goroutineMonitorCancel != nil {
		parseT.Fatal("expected stored monitor cancel func to be cleared")
	}

	select {
	case <-parseMonitorContext.Done():
	case <-time.After(250 * time.Millisecond):
		parseT.Fatal("expected monitor context to be canceled on disable")
	}
}

// TestEnableGoroutineMonitoringBuildsFreshMonitorContext verifies re-enable allocates one fresh monitor context.
func TestEnableGoroutineMonitoringBuildsFreshMonitorContext(parseT *testing.T) {
	DisableGoroutineMonitoring()
	goroutineMonitorContext = nil
	goroutineMonitorCancel = nil
	atomic.StoreInt64(&goroutineLeakDetected, 0)
	parseT.Cleanup(func() {
		DisableGoroutineMonitoring()
		goroutineMonitorContext = nil
		goroutineMonitorCancel = nil
		atomic.StoreInt64(&goroutineLeakDetected, 0)
	})

	EnableGoroutineMonitoring()
	parseFirstMonitorContext := goroutineMonitorContext
	if parseFirstMonitorContext == nil {
		parseT.Fatal("expected first monitor context after enable")
	}

	DisableGoroutineMonitoring()
	select {
	case <-parseFirstMonitorContext.Done():
	case <-time.After(250 * time.Millisecond):
		parseT.Fatal("expected first monitor context to be canceled on disable")
	}

	EnableGoroutineMonitoring()
	parseSecondMonitorContext := goroutineMonitorContext
	if parseSecondMonitorContext == nil {
		parseT.Fatal("expected second monitor context after re-enable")
	}
	if parseSecondMonitorContext == parseFirstMonitorContext {
		parseT.Fatal("expected re-enable to allocate a fresh monitor context")
	}
}
