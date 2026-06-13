package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"testing"
)

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
	parseCfg := config{Addr: "127.0.0.1:0"}
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
		listenAndServe: func(parseServer *http.Server) error {
			if parseServer == nil || parseServer.Addr != parseCfg.Addr || parseServer.Handler == nil {
				t.Fatalf("server = %#v, want configured addr and handler", parseServer)
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
	if parseOutput.String() != "Atlas server listening on http://127.0.0.1:0\n" {
		t.Fatalf("output = %q, want listening message", parseOutput.String())
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
			parseDeps.listenAndServe = func(*http.Server) error { return parseExpected }
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
