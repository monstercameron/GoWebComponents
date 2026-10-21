//go:build js && wasm

package interop

import (
	"context"
	"errors"
	"syscall/js"
	"testing"
	"time"
)

// TestDecryptCanceledContextIsNotReportedAsTampering verifies a canceled
// context surfaces as a context error, not as an authentication-tag mismatch.
func TestDecryptCanceledContextIsNotReportedAsTampering(parseT *testing.T) {
	parseSubtle := js.Global().Get("crypto").Get("subtle")
	parseOriginal := parseSubtle.Get("decrypt")
	parseExecutor := js.FuncOf(func(js.Value, []js.Value) any { return nil })
	defer parseExecutor.Release()
	parsePending := js.FuncOf(func(js.Value, []js.Value) any {
		return js.Global().Get("Promise").New(parseExecutor)
	})
	defer parsePending.Release()
	parseSubtle.Set("decrypt", parsePending)
	defer parseSubtle.Set("decrypt", parseOriginal)

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	_, parseErr := Decrypt(parseCtx, CryptoKey{raw: js.Global().Get("Object").New()}, []byte{1, 2, 3}, make([]byte, 12))
	if parseErr == nil {
		parseT.Fatal("expected an error from a canceled decrypt")
	}
	if IsCode(parseErr, CodeDecode) {
		parseT.Fatalf("cancellation misreported as tampering (CodeDecode): %v", parseErr)
	}
	if !errors.Is(parseErr, context.Canceled) {
		parseT.Fatalf("expected context.Canceled in chain, got %v", parseErr)
	}
}

// installFakeIndexedDB installs an indexedDB whose write transactions abort at
// commit with QuotaExceededError after every request has already succeeded.
func installFakeIndexedDB(parseShouldFailCommit bool) func() {
	parseFlag := "false"
	if parseShouldFailCommit {
		parseFlag = "true"
	}
	parseOriginal := js.Global().Get("indexedDB")
	parseInstall := js.Global().Get("Function").New(`
		const failCommit = ` + parseFlag + `;
		const rows = new Map();
		const later = (fn) => setTimeout(fn, 0);
		const makeRequest = (compute) => {
			const request = {};
			later(() => { request.result = compute(); if (request.onsuccess) request.onsuccess({}); });
			return request;
		};
		const db = {
			objectStoreNames: { contains: () => true },
			close() {},
			transaction() {
				const listeners = {};
				const tx = {
					addEventListener(type, fn) { listeners[type] = fn; },
					removeEventListener(type) { delete listeners[type]; },
					objectStore() {
						return {
							put(entry) {
								const request = makeRequest(() => { rows.set(entry.key, entry.value); return entry.key; });
								setTimeout(() => {
									if (failCommit) {
										tx.error = { name: "QuotaExceededError", message: "quota exceeded at commit" };
										if (listeners.abort) listeners.abort({});
									} else if (listeners.complete) {
										listeners.complete({});
									}
								}, 5);
								return request;
							},
						};
					},
				};
				return tx;
			},
		};
		globalThis.indexedDB = {
			open() {
				const request = {};
				later(() => { request.result = db; if (request.onsuccess) request.onsuccess({}); });
				return request;
			},
		};
	`)
	parseInstall.Invoke()
	return func() { js.Global().Set("indexedDB", parseOriginal) }
}

// TestPersistentStoreSetItemReportsCommitTimeQuotaFailure verifies a write whose
// transaction aborts at commit is not reported as a success.
func TestPersistentStoreSetItemReportsCommitTimeQuotaFailure(parseT *testing.T) {
	parseRestore := installFakeIndexedDB(true)
	defer parseRestore()
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer parseCancel()
	parseStore, parseErr := OpenPersistentStore(parseCtx, PersistentStoreOptions{Name: "rows", DatabaseName: "e-review-quota"})
	if parseErr != nil {
		parseT.Fatalf("open: %v", parseErr)
	}
	parseSetErr := parseStore.SetItem(parseCtx, "k", "v")
	if !IsCode(parseSetErr, CodeQuotaExceeded) {
		parseT.Fatalf("expected CodeQuotaExceeded from the aborted commit, got %v", parseSetErr)
	}
}

// TestPersistentStoreSetItemWaitsForCommit verifies the happy path still
// succeeds once the transaction completes.
func TestPersistentStoreSetItemWaitsForCommit(parseT *testing.T) {
	parseRestore := installFakeIndexedDB(false)
	defer parseRestore()
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer parseCancel()
	parseStore, parseErr := OpenPersistentStore(parseCtx, PersistentStoreOptions{Name: "rows", DatabaseName: "e-review-ok"})
	if parseErr != nil {
		parseT.Fatalf("open: %v", parseErr)
	}
	if parseSetErr := parseStore.SetItem(parseCtx, "k", "v"); parseSetErr != nil {
		parseT.Fatalf("SetItem: %v", parseSetErr)
	}
}

// TestWorkerTerminateFailsInFlightRequest verifies a request that is waiting on
// a worker returns (disposed) when that worker is terminated, instead of waiting
// for a context that may never expire.
func TestWorkerTerminateFailsInFlightRequest(parseT *testing.T) {
	parseOriginal := js.Global().Get("Worker")
	js.Global().Get("Function").New(`
		globalThis.Worker = class {
			constructor() { this.listeners = []; }
			addEventListener(type, fn) { this.listeners.push([type, fn]); }
			removeEventListener() {}
			postMessage() {}
			terminate() {}
		};
	`).Invoke()
	defer js.Global().Set("Worker", parseOriginal)

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "fake.js"})
	if parseErr != nil {
		parseT.Fatalf("OpenWorker: %v", parseErr)
	}
	parseDone := make(chan error, 1)
	go func() {
		_, parseRequestErr := parseWorker.Request(context.Background(), "job", nil, nil)
		parseDone <- parseRequestErr
	}()
	time.Sleep(50 * time.Millisecond)
	if parseTermErr := parseWorker.Terminate(); parseTermErr != nil {
		parseT.Fatalf("Terminate: %v", parseTermErr)
	}
	select {
	case parseRequestErr := <-parseDone:
		if !IsCode(parseRequestErr, CodeDisposed) {
			parseT.Fatalf("expected CodeDisposed, got %v", parseRequestErr)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatal("in-flight request still blocked after the worker was terminated")
	}
}
