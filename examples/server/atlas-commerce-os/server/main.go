package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	serverauth "github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/server/db"
)

func main() {
	if parseErr := runAtlasServer(defaultAtlasMainDeps()); parseErr != nil {
		log.Fatal(parseErr)
	}
}

type atlasMainDeps struct {
	loadConfig     func() (config, error)
	openDB         func(context.Context, string) (*sql.DB, error)
	migrate        func(context.Context, *sql.DB, string, string) error
	seed           func(context.Context, *sql.DB) error
	signalNotify   func(chan<- os.Signal, ...os.Signal)
	listenAndServe func(*http.Server) error
	output         io.Writer
}

func defaultAtlasMainDeps() atlasMainDeps {
	return atlasMainDeps{
		loadConfig:     loadConfig,
		openDB:         serverdb.Open,
		migrate:        serverdb.Migrate,
		seed:           serverdb.Seed,
		signalNotify:   signal.Notify,
		listenAndServe: func(parseServer *http.Server) error { return parseServer.ListenAndServe() },
		output:         os.Stdout,
	}
}

func runAtlasServer(parseDeps atlasMainDeps) error {
	parseCfg, parseErr := parseDeps.loadConfig()
	if parseErr != nil {
		return parseErr
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

	parseShutdown := make(chan os.Signal, 1)
	parseDeps.signalNotify(parseShutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-parseShutdown
		parseCtx2, parseCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer parseCancel()
		_ = parseHttpServer.Shutdown(parseCtx2)
	}()

	_, _ = fmt.Fprintf(parseDeps.output, "Atlas server listening on http://%s\n", parseCfg.Addr)
	if parseErr4 := parseDeps.listenAndServe(parseHttpServer); parseErr4 != nil && parseErr4 != http.ErrServerClosed {
		return parseErr4
	}
	return nil
}
