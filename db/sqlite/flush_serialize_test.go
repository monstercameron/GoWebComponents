package sqlite

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestFlushSerializesWithOpenTransaction pins that Flush acquires the single
// pooled connection and therefore cannot snapshot the image while a transaction
// is in flight (the torn-write window). It installs a fake flush and holds the
// one connection with an open Tx: Flush must block until the Tx releases it.
func TestFlushSerializesWithOpenTransaction(parseT *testing.T) {
	parseDB, parseErr := Open(context.Background(), Options{Name: "flushserialize", Persistence: Memory})
	if parseErr != nil {
		parseT.Fatalf("open: %v", parseErr)
	}
	defer parseDB.Close()

	var parseFlushed int32
	parseDB.flush = func(context.Context) error {
		atomic.AddInt32(&parseFlushed, 1)
		return nil
	}

	// Hold the single connection with an open transaction.
	parseTx, parseTxErr := parseDB.sdb.BeginTx(context.Background(), nil)
	if parseTxErr != nil {
		parseT.Fatalf("begin tx: %v", parseTxErr)
	}

	parseFlushDone := make(chan error, 1)
	go func() { parseFlushDone <- parseDB.Flush(context.Background()) }()

	// Flush must NOT complete while the tx holds the only connection.
	select {
	case <-parseFlushDone:
		parseT.Fatal("Flush completed while a transaction held the connection (torn-write window)")
	case <-time.After(100 * time.Millisecond):
	}
	if atomic.LoadInt32(&parseFlushed) != 0 {
		parseT.Fatal("flush ran while a transaction was open")
	}

	// Releasing the connection lets Flush proceed.
	if parseCommitErr := parseTx.Commit(); parseCommitErr != nil {
		parseT.Fatalf("commit: %v", parseCommitErr)
	}
	select {
	case parseFlushErr := <-parseFlushDone:
		if parseFlushErr != nil {
			parseT.Fatalf("flush: %v", parseFlushErr)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatal("Flush did not complete after the transaction released the connection")
	}
	if atomic.LoadInt32(&parseFlushed) != 1 {
		parseT.Fatal("flush did not run after the transaction committed")
	}
}
