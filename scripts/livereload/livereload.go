package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	debounceTime = 1000 * time.Millisecond
	buildCommand = "go"
)

var (
	buildArgs = []string{"build", "-o", "static/bin/main.wasm"}
	buildEnv  = []string{"GOOS=js", "GOARCH=wasm"}
)

type LiveReloader struct {
	watcher       *fsnotify.Watcher
	mutex         sync.Mutex
	currentBuild  *exec.Cmd
	debounceTimer *time.Timer
	projectRoot   string
}

func NewLiveReloader(projectRoot string) (*LiveReloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &LiveReloader{
		watcher:     watcher,
		projectRoot: projectRoot,
	}, nil
}

func (lr *LiveReloader) Start() error {
	// Add the project root and subdirectories to the watcher
	err := lr.addWatchers(lr.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to add watchers: %w", err)
	}

	// Handle interrupt signal for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🔄 Live reload started. Watching for .go file changes...")
	fmt.Println("📂 Watching directory:", lr.projectRoot)
	fmt.Println("⏱️  Debounce time:", debounceTime)
	fmt.Println("🛑 Press Ctrl+C to stop")

	// Trigger initial build
	lr.triggerBuild()

	go func() {
		for {
			select {
			case event, ok := <-lr.watcher.Events:
				if !ok {
					return
				}
				lr.handleFileEvent(event)

			case err, ok := <-lr.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("❌ Watcher error: %v", err)

			case <-c:
				fmt.Println("\n🛑 Shutting down live reloader...")
				lr.cleanup()
				os.Exit(0)
			}
		}
	}()

	// Keep the main goroutine alive
	select {}
}

func (lr *LiveReloader) addWatchers(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories, vendor, and node_modules
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}

			// Add the directory to the watcher
			err := lr.watcher.Add(path)
			if err != nil {
				log.Printf("⚠️  Failed to watch directory %s: %v", path, err)
			} else {
				fmt.Printf("👀 Watching: %s\n", path)
			}
		}

		return nil
	})
}

func (lr *LiveReloader) handleFileEvent(event fsnotify.Event) {
	// Only handle .go files
	if !strings.HasSuffix(event.Name, ".go") {
		return
	}

	// Skip temporary files and test files for now (you can adjust this)
	if strings.Contains(event.Name, ".tmp") || strings.Contains(event.Name, "~") {
		return
	}

	// Only handle write and create events
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		fmt.Printf("📝 File changed: %s\n", event.Name)
		lr.debounceAndBuild()
	}
}

func (lr *LiveReloader) debounceAndBuild() {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	// Kill current build if running
	if lr.currentBuild != nil && lr.currentBuild.Process != nil {
		fmt.Printf("⏹️  Killing current build process (PID: %d)...\n", lr.currentBuild.Process.Pid)
		err := lr.currentBuild.Process.Kill()
		if err != nil {
			log.Printf("❌ Failed to kill current build: %v", err)
		} else {
			fmt.Println("✅ Previous build process killed successfully")
			// Note: We don't wait here to avoid blocking, the triggerBuild function will handle the wait
		}
		lr.currentBuild = nil
	}

	// Reset the debounce timer
	if lr.debounceTimer != nil {
		lr.debounceTimer.Stop()
		fmt.Println("⏱️  Previous debounce timer cancelled")
	}

	lr.debounceTimer = time.AfterFunc(debounceTime, func() {
		lr.triggerBuild()
	})

	fmt.Printf("⏲️  Debouncing... will build in %v\n", debounceTime)
}

func (lr *LiveReloader) triggerBuild() {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	fmt.Println("🔨 Starting WASM build...")
	startTime := time.Now()

	// Create the build command
	cmd := exec.Command(buildCommand, buildArgs...)

	cmd.Dir = lr.projectRoot
	cmd.Env = append(os.Environ(), buildEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	lr.currentBuild = cmd
	fmt.Printf("🏗️  Build process started (PID: will be available after start)\n")

	// Start the build process (non-blocking)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start build: %v\n", err)
		lr.currentBuild = nil
		return
	}

	fmt.Printf("🏗️  Build process running (PID: %d)\n", cmd.Process.Pid)

	// Wait for the build to complete
	err = cmd.Wait()

	duration := time.Since(startTime)

	// Check if the process was killed vs completed naturally
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.String() == "signal: killed" {
			fmt.Printf("⏹️  Build was cancelled after %v\n", duration)
		} else {
			fmt.Printf("❌ Build failed after %v: %v\n", duration, err)
		}
	} else {
		fmt.Printf("✅ Build completed successfully in %v\n", duration)
	}

	lr.currentBuild = nil
}

func (lr *LiveReloader) cleanup() {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	if lr.debounceTimer != nil {
		lr.debounceTimer.Stop()
	}

	if lr.currentBuild != nil && lr.currentBuild.Process != nil {
		fmt.Println("🛑 Killing running build process...")
		lr.currentBuild.Process.Kill()
		lr.currentBuild.Wait()
	}

	if lr.watcher != nil {
		lr.watcher.Close()
	}
}

func main() {
	// Get the project root (parent directory of scripts)
	scriptDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ Failed to get current directory: %v", err)
	}

	// If we're in the livereload directory, go up two levels to reach project root
	if filepath.Base(scriptDir) == "livereload" {
		scriptDir = filepath.Dir(filepath.Dir(scriptDir))
	} else if filepath.Base(scriptDir) == "scripts" {
		// If we're in the scripts directory, go up one level
		scriptDir = filepath.Dir(scriptDir)
	}

	reloader, err := NewLiveReloader(scriptDir)
	if err != nil {
		log.Fatalf("❌ Failed to create live reloader: %v", err)
	}

	defer reloader.cleanup()

	err = reloader.Start()
	if err != nil {
		log.Fatalf("❌ Failed to start live reloader: %v", err)
	}
}
