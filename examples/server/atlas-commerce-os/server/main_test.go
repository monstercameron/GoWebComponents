package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// atlasStubAddr and atlasStubListener stand in for a bound TCP listener so the
// readiness line can be asserted against the listener's address rather than the
// requested address.
type atlasStubAddr string

func (parseAddr atlasStubAddr) Network() string { return "tcp" }

func (parseAddr atlasStubAddr) String() string { return string(parseAddr) }

type atlasStubListener struct {
	addr atlasStubAddr
}

func (parseListener *atlasStubListener) Accept() (net.Conn, error) {
	return nil, errors.New("stub listener does not accept")
}

func (parseListener *atlasStubListener) Close() error { return nil }

func (parseListener *atlasStubListener) Addr() net.Addr { return parseListener.addr }

func openAtlasMainTestDB(t *testing.T) *sql.DB {
	t.Helper()
	parseDB, parseErr := sql.Open("sqlite", "file:atlas-main-test?mode=memory&cache=shared")
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	t.Cleanup(func() {
		_ = parseDB.Close()
	})
	return parseDB
}

func atlasMainTestDeps(t *testing.T) (atlasMainDeps, *bytes.Buffer) {
	t.Helper()
	parseDB := openAtlasMainTestDB(t)
	parseOutput := &bytes.Buffer{}
	parseWasmPath := filepath.Join(t.TempDir(), "atlas-commerce-os.wasm")
	if parseWriteErr := os.WriteFile(parseWasmPath, []byte("stub"), 0o600); parseWriteErr != nil {
		t.Fatalf("write stub wasm: %v", parseWriteErr)
	}
	parseCfg := config{Addr: "127.0.0.1:0", AtlasWASM: parseWasmPath}
	parseDeps := atlasMainDeps{
		loadConfig: func() (config, error) { return parseCfg, nil },
		openDB: func(parseCtx context.Context, parsePath string) (*sql.DB, error) {
			if parsePath != "" {
				t.Fatalf("sqlite path = %q, want empty test path", parsePath)
			}
			return parseDB, nil
		},
		migrate: func(parseCtx context.Context, parseDB *sql.DB, parseMigrationsDir string, parseFallbackSchema string) error {
			if parseDB == nil {
				t.Fatal("migrate received nil db")
			}
			return nil
		},
		seed: func(parseCtx context.Context, parseDB *sql.DB) error {
			if parseDB == nil {
				t.Fatal("seed received nil db")
			}
			return nil
		},
		signalNotify: func(parseChan chan<- os.Signal, parseSignals ...os.Signal) {
			if len(parseSignals) != 2 {
				t.Fatalf("signal count = %d, want SIGINT/SIGTERM", len(parseSignals))
			}
		},
		listen: func(parseAddr string) (net.Listener, error) {
			if parseAddr != parseCfg.Addr {
				t.Fatalf("listen addr = %q, want %q", parseAddr, parseCfg.Addr)
			}
			return &atlasStubListener{addr: "127.0.0.1:8096"}, nil
		},
		serve: func(parseServer *http.Server, parseListener net.Listener) error {
			if parseServer == nil || parseServer.Addr != parseCfg.Addr || parseServer.Handler == nil {
				t.Fatalf("server = %#v, want configured addr and handler", parseServer)
			}
			if parseListener == nil {
				t.Fatal("serve received nil listener")
			}
			return http.ErrServerClosed
		},
		output: parseOutput,
	}
	return parseDeps, parseOutput
}

func TestRunAtlasServerBuildsServerAndAcceptsClosedServer(t *testing.T) {
	parseDeps, parseOutput := atlasMainTestDeps(t)

	if parseErr := runAtlasServer(parseDeps); parseErr != nil {
		t.Fatalf("runAtlasServer() error = %v, want nil", parseErr)
	}
	if parseOutput.String() != "Atlas server listening on http://127.0.0.1:8096\n" {
		t.Fatalf("output = %q, want listening message with the bound listener address", parseOutput.String())
	}
}

func TestRunAtlasServerWarnsWhenClientBundleMissing(t *testing.T) {
	parseDeps, parseOutput := atlasMainTestDeps(t)
	parseBaseLoadConfig := parseDeps.loadConfig
	parseDeps.loadConfig = func() (config, error) {
		parseCfg, parseErr := parseBaseLoadConfig()
		parseCfg.AtlasWASM = filepath.Join(t.TempDir(), "missing-atlas-commerce-os.wasm")
		return parseCfg, parseErr
	}

	if parseErr := runAtlasServer(parseDeps); parseErr != nil {
		t.Fatalf("runAtlasServer() error = %v, want nil", parseErr)
	}
	parseText := parseOutput.String()
	if !strings.Contains(parseText, "missing-atlas-commerce-os.wasm") {
		t.Fatalf("output = %q, want the expected absolute wasm path", parseText)
	}
	if !strings.Contains(parseText, atlasWASMAssetURL) || !strings.Contains(parseText, atlasWASMBuildCommand) {
		t.Fatalf("output = %q, want served URL and build command in the warning", parseText)
	}
	if !strings.Contains(parseText, "Atlas server listening on http://127.0.0.1:8096\n") {
		t.Fatalf("output = %q, want the readiness line after the warning", parseText)
	}
}

func TestRunAtlasServerFailsBeforeAnnouncingWhenBindFails(t *testing.T) {
	parseDeps, parseOutput := atlasMainTestDeps(t)
	parseBindErr := errors.New("address already in use")
	parseDeps.listen = func(string) (net.Listener, error) { return nil, parseBindErr }
	parseDeps.serve = func(*http.Server, net.Listener) error {
		t.Fatal("serve must not run when bind fails")
		return nil
	}

	parseErr := runAtlasServer(parseDeps)
	if !errors.Is(parseErr, parseBindErr) {
		t.Fatalf("runAtlasServer() error = %v, want %v", parseErr, parseBindErr)
	}
	if strings.Contains(parseOutput.String(), "listening") {
		t.Fatalf("output = %q, want no readiness line when bind fails", parseOutput.String())
	}
}

func TestRunAtlasServerStartupErrors(t *testing.T) {
	parseExpected := errors.New("boom")
	parseCases := []struct {
		name   string
		mutate func(*atlasMainDeps)
	}{
		{"load config", func(parseDeps *atlasMainDeps) {
			parseDeps.loadConfig = func() (config, error) { return config{}, parseExpected }
		}},
		{"open db", func(parseDeps *atlasMainDeps) {
			parseDeps.openDB = func(context.Context, string) (*sql.DB, error) { return nil, parseExpected }
		}},
		{"migrate", func(parseDeps *atlasMainDeps) {
			parseDeps.migrate = func(context.Context, *sql.DB, string, string) error { return parseExpected }
		}},
		{"seed", func(parseDeps *atlasMainDeps) {
			parseDeps.seed = func(context.Context, *sql.DB) error { return parseExpected }
		}},
		{"listen", func(parseDeps *atlasMainDeps) {
			parseDeps.listen = func(string) (net.Listener, error) { return nil, parseExpected }
		}},
		{"serve", func(parseDeps *atlasMainDeps) {
			parseDeps.serve = func(*http.Server, net.Listener) error { return parseExpected }
		}},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			parseDeps, _ := atlasMainTestDeps(t)
			parseCase.mutate(&parseDeps)
			if parseErr := runAtlasServer(parseDeps); !errors.Is(parseErr, parseExpected) {
				t.Fatalf("runAtlasServer() error = %v, want %v", parseErr, parseExpected)
			}
		})
	}
}
