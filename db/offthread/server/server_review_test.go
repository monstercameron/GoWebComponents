package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/db/offthread"
)

// TestAutocommitStatementIsRefusedWhileTransactionOpen pins that a statement
// naming no transaction cannot hang the single database thread: the pool holds
// one connection and the open transaction owns it, so an autocommit statement
// would wait forever on a server that answers one request at a time.
func TestAutocommitStatementIsRefusedWhileTransactionOpen(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseCtx := context.Background()

	parseBegin := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpBegin})
	if parseBegin.Err != "" {
		parseT.Fatalf("begin: %s", parseBegin.Err)
	}

	parseDone := make(chan offthread.Response, 1)
	go func() {
		parseCallCtx, parseCancel := context.WithTimeout(parseCtx, 3*time.Second)
		defer parseCancel()
		parseDone <- parseServer.Handle(parseCallCtx, offthread.Request{Op: offthread.OpQuery, SQL: "SELECT 1"})
	}()
	select {
	case parseResponse := <-parseDone:
		if parseResponse.Err == "" {
			parseT.Fatalf("autocommit query during an open transaction succeeded; want a refusal")
		}
	case <-time.After(1 * time.Second):
		parseT.Fatal("autocommit query blocked on the connection held by the open transaction instead of being refused")
	}

	if parseRollback := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpRollback, TxID: parseBegin.TxID}); parseRollback.Err != "" {
		parseT.Fatalf("rollback: %s", parseRollback.Err)
	}
}

// TestTransactionSurvivesCancelledBeginContext pins that cancelling the context
// of the Begin request does not roll the transaction back behind the server's back.
func TestTransactionSurvivesCancelledBeginContext(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseCtx := context.Background()
	if parseCreate := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpExec, SQL: "CREATE TABLE t (id INTEGER)"}); parseCreate.Err != "" {
		parseT.Fatalf("create: %s", parseCreate.Err)
	}

	parseBeginCtx, parseCancel := context.WithCancel(parseCtx)
	parseBegin := parseServer.Handle(parseBeginCtx, offthread.Request{Op: offthread.OpBegin})
	if parseBegin.Err != "" {
		parseT.Fatalf("begin: %s", parseBegin.Err)
	}
	// The per-request context of a worker ends once its response is posted.
	parseCancel()

	parseExec := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpExec, SQL: "INSERT INTO t VALUES (1)", TxID: parseBegin.TxID})
	if parseExec.Err != "" {
		parseT.Fatalf("exec in transaction after begin context ended: %s", parseExec.Err)
	}
	if parseCommit := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpCommit, TxID: parseBegin.TxID}); parseCommit.Err != "" {
		parseT.Fatalf("commit: %s", parseCommit.Err)
	}
}
