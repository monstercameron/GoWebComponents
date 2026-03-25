package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
)

func main() {
	parseCfg, parseErr := loadConfig()
	if parseErr != nil {
		log.Fatal(parseErr)
	}

	parseCtx := context.Background()
	parseDatabase, parseErr := serverdb.Open(parseCtx, parseCfg.SQLitePath)
	if parseErr != nil {
		log.Fatal(parseErr)
	}
	defer parseDatabase.Close()

	if parseErr2 := serverdb.Migrate(parseCtx, parseDatabase, parseCfg.MigrationsDir, parseCfg.FallbackSchema); parseErr2 != nil {
		log.Fatal(parseErr2)
	}
	if parseErr3 := serverdb.Seed(parseCtx, parseDatabase); parseErr3 != nil {
		log.Fatal(parseErr3)
	}

	parseApp := newAtlasServer(parseCfg, serverdb.NewStore(parseDatabase), serverauth.NewMockSessionManager())
	parseHttpServer := &http.Server{
		Addr:              parseCfg.Addr,
		Handler:           parseApp.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	parseShutdown := make(chan os.Signal, 1)
	signal.Notify(parseShutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-parseShutdown
		parseCtx2, parseCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer parseCancel()
		_ = parseHttpServer.Shutdown(parseCtx2)
	}()

	fmt.Printf("Atlas server listening on http://%s\n", parseCfg.Addr)
	if parseErr4 := parseHttpServer.ListenAndServe(); parseErr4 != nil && parseErr4 != http.ErrServerClosed {
		log.Fatal(parseErr4)
	}
}
