package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"example.com/gwc-wails-counter/assets"
	"example.com/gwc-wails-counter/contracts"
	"example.com/gwc-wails-counter/internal/services"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	wailsadapter "github.com/monstercameron/GoWebComponents/v6/desktop/wails"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// buildFeatures is the immutable linker-set ceiling; runtime flags can only narrow it.
var buildFeatures = "all"

// main starts the desktop host or its bounded native-WebView smoke test.
func main() {
	parseSmoke := flag.Bool("smoke-test", false, "validate the actual WebView and exit with JSON evidence")
	parseFault := flag.String("smoke-fault", "", "smoke only: missing-binding, missing-wasm, or streaming")
	parseTwoWindows := flag.Bool("two-windows", false, "open two independent windows sharing durable state")
	parseFileDialogs := flag.Bool("file-dialogs", true, "Enable path-selection APIs in this lab; does not control report export")
	parseFileDrop := flag.Bool("file-drop", false, "Explicitly enable bounded caller-window file-drop events")
	parseFeatures := flag.String("features", "all", "Runtime feature allowlist (all, none, or comma-separated desktop feature names)")
	flag.Parse()
	if *parseFault != "" && (!*parseSmoke || (*parseFault != "missing-binding" && *parseFault != "missing-wasm" && *parseFault != "streaming")) {
		fmt.Fprintln(os.Stderr, "invalid smoke-fault (requires --smoke-test)")
		os.Exit(2)
	}
	os.Exit(runDesktop(*parseSmoke, *parseFault, *parseTwoWindows, *parseFileDialogs, *parseFileDrop, *parseFeatures))
}

// runDesktop owns the host lifecycle and fails smoke runs without frontend evidence.
func runDesktop(isSmoke bool, parseFault string, isTwoWindows bool, isFileDialogsEnabled bool, isFileDropEnabled bool, parseFeatureValue string) (parseExitCode int) {
	parseFeaturePolicy, parseFeatureErr := desktop.ParseFeaturePolicy(parseFeatureValue)
	if parseFeatureErr != nil {
		fmt.Fprintln(os.Stderr, parseFeatureErr)
		return 2
	}
	parseBuildPolicy, parseBuildErr := desktop.ParseFeaturePolicy(buildFeatures)
	if parseBuildErr != nil {
		fmt.Fprintln(os.Stderr, parseBuildErr)
		return 2
	}
	parseFeaturePolicy = parseFeaturePolicy.Intersect(parseBuildPolicy.FeatureNames())
	parseFileDialogsEnabled := isFileDialogsEnabled && parseFeaturePolicy.Allows(desktop.FileDialogs)
	parseStoragePath := ""
	var parseErr error
	if isSmoke {
		// Each automated launch gets its own store and never changes user app data.
		parseTemp, parseErr := os.MkdirTemp("", "gwc-wails-smoke-")
		if parseErr != nil {
			fmt.Fprintln(os.Stderr, parseErr)
			return 1
		}
		defer os.RemoveAll(parseTemp)
		parseStoragePath = filepath.Join(parseTemp, "storage.json")
	}
	var parseStorage *services.StorageService
	if parseFeaturePolicy.Allows(desktop.PersistentStorage) {
		parseStorage, parseErr = services.NewStorageService(parseStoragePath, func(parseCommit desktop.StorageCommit) {
			if parseApp := application.Get(); parseApp != nil {
				parseApp.Event.Emit("storage.changed", parseCommit)
			}
		})
		if parseErr != nil {
			fmt.Fprintln(os.Stderr, parseErr)
			return 1
		}
		defer func() {
			if parseCloseErr := parseStorage.ServiceShutdown(); parseCloseErr != nil {
				fmt.Fprintln(os.Stderr, "storage shutdown:", parseCloseErr)
				parseExitCode = 1
			}
		}()
	}
	parseReports := make(chan contracts.SmokeReport, 1)
	parseFinished := make(chan contracts.SmokeReport, 1)
	parseAPI, parseErr := services.NewAPIService()
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, "API fixtures:", parseErr)
		return 1
	}
	parseAPI.Policy = parseFeaturePolicy
	parseNativeBackend := wailsadapter.NewNativeBackend()
	if runtime.GOOS == "windows" {
		parseNativeBackend, parseErr = wailsadapter.NewNativeBackendWithWindowTemplates(map[string]wailsadapter.WindowTemplate{
			"counter": {Route: "/#/counter", Title: "Counter", Width: 720, Height: 560},
		})
		if parseErr != nil {
			fmt.Fprintln(os.Stderr, "native window templates:", parseErr)
			return 1
		}
	}
	parseNative := desktop.NewNativeHost(parseNativeBackend, parseFeaturePolicy)
	parseAPI.Native = parseNative
	defer func() {
		if parseErr := parseAPI.ServiceShutdown(); parseErr != nil {
			fmt.Fprintln(os.Stderr, "API fixture cleanup:", parseErr)
			parseExitCode = 1
		}
	}()
	// The lab explicitly opts in; library hosts default to denied.
	parseFiles := desktop.NewFileDialogHost(wailsadapter.NewFileDialogs(), parseFileDialogsEnabled)
	parseAPI.Files = parseFiles
	parseCounter := &services.CounterService{Files: parseFiles, Native: parseNative, Policy: parseFeaturePolicy}
	parseServices := []application.Service{application.NewService(parseCounter), application.NewService(parseAPI), application.NewService(parseFiles), application.NewService(parseNative)}
	if parseFeaturePolicy.Allows(desktop.PersistentStorage) {
		parseServices = append(parseServices, application.NewService(parseStorage))
	}
	parseURL := "/#/tester"
	if isSmoke {
		parseServices = append(parseServices, application.NewService(services.NewSmokeService(parseReports)))
		parseURL = "/?smoke=1&fault=" + parseFault
	}
	parseApp := application.New(application.Options{
		Name: "GWC Windows API Lab", Description: "GoWebComponents native Windows API tester",
		Services: parseServices,
		Assets:   application.AssetOptions{Handler: application.BundledAssetFileServer(assets.Files), Middleware: guardDesktopAssets},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	parseWindowEventsEnabled := parseFeaturePolicy.Allows(desktop.WindowEvents)
	parseWindowFileDropEnabled := parseWindowEventsEnabled && isFileDropEnabled
	parseCounter.WindowEventsInstalled = parseWindowEventsEnabled
	parseCounter.WindowFileDropEnabled = parseWindowFileDropEnabled
	var parseWindowCleanupMutex sync.Mutex
	parseWindowCleanups := []func(){}
	if parseWindowEventsEnabled {
		parseApp.Window.OnCreate(func(parseWindow application.Window) {
			parseCleanup, parseAttachErr := wailsadapter.AttachWindowEvents(parseWindow, wailsadapter.WindowEventOptions{EnableFileDrop: parseWindowFileDropEnabled})
			if parseAttachErr != nil {
				parseCounter.WindowEventsInstalled = false
				parseCounter.WindowFileDropEnabled = false
				return
			}
			parseWindowCleanupMutex.Lock()
			parseWindowCleanups = append(parseWindowCleanups, parseCleanup)
			parseWindowCleanupMutex.Unlock()
		})
		defer func() {
			parseWindowCleanupMutex.Lock()
			defer parseWindowCleanupMutex.Unlock()
			for parseIndex := len(parseWindowCleanups) - 1; parseIndex >= 0; parseIndex-- {
				parseWindowCleanups[parseIndex]()
			}
		}()
	}
	var parseMenu *application.Menu
	parseBindings := services.APIKeyBindings(parseAPI)
	if parseFeaturePolicy.Allows(desktop.NativeMenus) {
		parseMenu = services.InstallAPIMenus(parseApp, parseAPI)
	}
	parseApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "counter", Title: "Windows API Lab", Width: 1024, Height: 820,
		URL: parseURL, Hidden: isSmoke, EnableFileDrop: parseWindowFileDropEnabled,
		Windows: application.WindowsWindow{Menu: parseMenu}, KeyBindings: parseBindings,
	})
	if isTwoWindows || (isSmoke && (parseFault == "" || parseFault == "streaming")) {
		parseObserverURL := "/#/tester"
		if isSmoke {
			parseObserverURL = "/?smoke=1&observer=1&fault=" + parseFault
		}
		parseApp.Window.NewWithOptions(application.WebviewWindowOptions{
			Name: "observer", Title: "Windows API Lab — second window", Width: 1024, Height: 820,
			URL: parseObserverURL, Hidden: isSmoke, EnableFileDrop: parseWindowFileDropEnabled,
			Windows: application.WindowsWindow{Menu: parseMenu}, KeyBindings: parseBindings,
		})
	}
	if isSmoke {
		go func() {
			parseTimeout := time.NewTimer(45 * time.Second)
			defer parseTimeout.Stop()
			var parseReport contracts.SmokeReport
			select {
			case parseReport = <-parseReports:
			case <-parseTimeout.C:
				parseReport.Error = "native WebView smoke timed out before a complete report"
			case <-parseApp.Context().Done():
				return
			}
			parseFinished <- parseReport
			parseApp.Quit()
		}()
	}
	if parseErr := parseApp.Run(); parseErr != nil {
		fmt.Fprintln(os.Stderr, parseErr)
		return 1
	}
	if !isSmoke {
		return 0
	}
	var parseReport contracts.SmokeReport
	select {
	case parseReport = <-parseFinished:
	default:
		parseReport.Error = "host closed before native WebView verification completed"
	}
	if parseErr := json.NewEncoder(os.Stdout).Encode(parseReport); parseErr != nil {
		fmt.Fprintln(os.Stderr, parseErr)
		return 1
	}
	if !parseReport.OK {
		return 1
	}
	return 0
}
