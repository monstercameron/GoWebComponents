package pluginruntime

import (
	"errors"
	"strings"
	"testing"
)

type reviewHandle struct{ stop func() error }

func (parseH reviewHandle) StopPlugin() error {
	if parseH.stop == nil {
		return nil
	}
	return parseH.stop()
}

type reviewPlugin struct {
	id    string
	start func(Context) (Handle, error)
}

func (parseP reviewPlugin) Manifest() Manifest {
	return Manifest{ID: parseP.id, Version: "1.0.0", ActivationPolicy: ActivationPolicyBoot}
}

func (parseP reviewPlugin) StartPlugin(parseContext Context) (Handle, error) {
	return parseP.start(parseContext)
}

func reviewRegistration(parseID string, parseStart func(Context) (Handle, error)) PluginRegistration {
	return PluginRegistration{
		Factory: func() Plugin { return reviewPlugin{id: parseID, start: parseStart} },
	}
}

// TestFailedStartRunsCleanupsItRegistered pins that a plugin which registered cleanups and then
// failed (or panicked) in StartPlugin has them run: the kernel deletes the plugin's state on
// failure, so without this the cleanups (timers, listeners, handles) leak for the life of the
// process.
func TestFailedStartRunsCleanupsItRegistered(parseT *testing.T) {
	for _, parseName := range []string{"error", "panic"} {
		parseCleaned := 0
		_, parseErr := NewKernel(BootstrapOptions{Registrations: []PluginRegistration{
			reviewRegistration("leaky", func(parseContext Context) (Handle, error) {
				parseContext.RegisterCleanup(func() error { parseCleaned++; return nil })
				if parseName == "panic" {
					panic("start exploded")
				}
				return nil, errors.New("start failed")
			}),
		}})
		if parseErr == nil {
			parseT.Fatalf("%s: expected NewKernel to fail", parseName)
		}
		if parseCleaned != 1 {
			parseT.Errorf("%s: cleanup ran %d times, want 1", parseName, parseCleaned)
		}
	}
}

// TestNewKernelFailureStopsAlreadyStartedPlugins pins that when plugin N fails to start, plugins
// 1..N-1 (already running, with handles and cleanups) are stopped in reverse order rather than
// abandoned with the half-built kernel.
func TestNewKernelFailureStopsAlreadyStartedPlugins(parseT *testing.T) {
	var parseEvents []string
	parseGood := func(parseID string) PluginRegistration {
		return reviewRegistration(parseID, func(parseContext Context) (Handle, error) {
			parseContext.RegisterCleanup(func() error { parseEvents = append(parseEvents, "cleanup:"+parseID); return nil })
			return reviewHandle{stop: func() error { parseEvents = append(parseEvents, "stop:"+parseID); return nil }}, nil
		})
	}
	_, parseErr := NewKernel(BootstrapOptions{Registrations: []PluginRegistration{
		parseGood("one"),
		parseGood("two"),
		reviewRegistration("three", func(Context) (Handle, error) { return nil, errors.New("three failed") }),
	}})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "three failed") {
		parseT.Fatalf("NewKernel error = %v, want the start failure", parseErr)
	}
	if parseGot := strings.Join(parseEvents, ","); parseGot != "stop:two,cleanup:two,stop:one,cleanup:one" {
		parseT.Fatalf("teardown events = %q", parseGot)
	}
}
