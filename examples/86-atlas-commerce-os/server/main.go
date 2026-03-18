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
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	database, err := serverdb.Open(ctx, cfg.SQLitePath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := serverdb.Migrate(ctx, database, cfg.MigrationsDir, cfg.FallbackSchema); err != nil {
		log.Fatal(err)
	}
	if err := serverdb.Seed(ctx, database); err != nil {
		log.Fatal(err)
	}

	app := newAtlasServer(cfg, serverdb.NewStore(database), serverauth.NewMockSessionManager())
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	fmt.Printf("Atlas server listening on http://%s\n", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
