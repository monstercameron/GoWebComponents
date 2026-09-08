package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	serverauth "github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/server/db"
)

func main() {
	if parseErr := runAtlasServer(defaultAtlasMainDeps()); parseErr != nil {
		log.Fatal(parseErr)
	}
}

type atlasMainDeps struct {
	loadConfig   func() (config, error)
	openDB       func(context.Context, string) (*sql.DB, error)
	migrate      func(context.Context, *sql.DB, string, string) error
	seed         func(context.Context, *sql.DB) error
	signalNotify func(chan<- os.Signal, ...os.Signal)
	listen       func(string) (net.Listener, error)
	serve        func(*http.Server, net.Listener) error
	output       io.Writer
}

func defaultAtlasMainDeps() atlasMainDeps {
	return atlasMainDeps{
		loadConfig:   loadConfig,
		openDB:       serverdb.Open,
		migrate:      serverdb.Migrate,
		seed:         serverdb.Seed,
		signalNotify: signal.Notify,
		listen:       func(parseAddr string) (net.Listener, error) { return net.Listen("tcp", parseAddr) },
		serve: func(parseServer *http.Server, parseListener net.Listener) error {
			return parseServer.Serve(parseListener)
		},
		output: os.Stdout,
	}
}

func runAtlasServer(parseDeps atlasMainDeps) error {
	parseCfg, parseErr := parseDeps.loadConfig()
	if parseErr != nil {
		return parseErr
	}

	if parseWarning := atlasWASMStartupWarning(parseCfg); parseWarning != "" {
		_, _ = fmt.Fprint(parseDeps.output, parseWarning)
	}

	parseCtx := context.Background()
	parseDatabase, parseErr := parseDeps.openDB(parseCtx, parseCfg.SQLitePath)
	if parseErr != nil {
		return parseErr
	}
	defer parseDatabase.Close()

	if parseErr2 := parseDeps.migrate(parseCtx, parseDatabase, parseCfg.MigrationsDir, parseCfg.FallbackSchema); parseErr2 != nil {
		return parseErr2
	}
	if parseErr3 := parseDeps.seed(parseCtx, parseDatabase); parseErr3 != nil {
		return parseErr3
	}

	parseApp := newAtlasServer(parseCfg, serverdb.NewStore(parseDatabase), serverauth.NewMockSessionManager())
	parseHttpServer := &http.Server{
		Addr:              parseCfg.Addr,
		Handler:           parseApp.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Bind before announcing readiness: ListenAndServe would have let a port
	// collision print a success line and then fail.
	parseListener, parseErr4 := parseDeps.listen(parseCfg.Addr)
	if parseErr4 != nil {
		return fmt.Errorf("listen on %s: %w", parseCfg.Addr, parseErr4)
	}

	parseShutdown := make(chan os.Signal, 1)
	parseDeps.signalNotify(parseShutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-parseShutdown
		parseCtx2, parseCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer parseCancel()
		_ = parseHttpServer.Shutdown(parseCtx2)
	}()

	_, _ = fmt.Fprintf(parseDeps.output, "Atlas server listening on http://%s\n", parseListener.Addr().String())
	if parseErr5 := parseDeps.serve(parseHttpServer, parseListener); parseErr5 != nil && parseErr5 != http.ErrServerClosed {
		return parseErr5
	}
	return nil
}
