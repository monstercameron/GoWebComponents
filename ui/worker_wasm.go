//go:build js && wasm
// +build js,wasm

package ui

import (
	"context"
	"sync"

	"github.com/monstercameron/GoWebComponents/interop"
)

// WorkerTaskState describes the lifecycle of a worker-backed task together with
// the latest decoded progress payload.
type WorkerTaskState[Progress any, Result any] struct {
	Value         Result
	Progress      Progress
	ProgressReady bool
	Running       bool
	Ready         bool
	Cancelled     bool
	Started       bool
	Error         error
}

// WorkerTask exposes control over a typed worker-backed request flow.
type WorkerTask[Request any, Progress any, Result any] struct {
	get    func() WorkerTaskState[Progress, Result]
	start  func(Request)
	cancel func()
}

// UseWorkerTask creates a worker-backed task handle that reuses one browser
// worker instance across runs and cleans it up when the owning component
// unmounts.
func UseWorkerTask[Request any, Progress any, Result any](parseOptions interop.WorkerOptions, parseName string) WorkerTask[Request, Progress, Result] {
	parseState := UseState(WorkerTaskState[Progress, Result]{})
	parseCancelRef := UseRef((context.CancelFunc)(nil))
	parseRequestSeq := UseRef(0)
	parseWorkerStateRef := UseRef(&struct {
		mu         sync.Mutex
		worker     interop.Worker
		openCtx    context.Context
		openCancel context.CancelFunc
		isReady    bool
		isOpening  bool
		isClosed   bool
		waitCh     chan struct{}
	}{})

	parseStart := func(parsePayload Request) {
		if parseCancel := parseCancelRef.Get(); parseCancel != nil {
			parseCancel()
		}

		parseRequestSeq.Set(parseRequestSeq.Get() + 1)
		parseSeq := parseRequestSeq.Get()
		parseCtx, parseCancel2 := context.WithCancel(context.Background())
		parseCancelRef.Set(parseCancel2)

		parseState.Update(func(parsePrev WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
			var parseZeroResult Result
			var parseZeroProgress Progress
			parsePrev.Value = parseZeroResult
			parsePrev.Progress = parseZeroProgress
			parsePrev.ProgressReady = false
			parsePrev.Running = true
			parsePrev.Ready = false
			parsePrev.Cancelled = false
			parsePrev.Started = true
			parsePrev.Error = nil
			return parsePrev
		})

		go func(parseRequestPayload Request) {
			parseWorkerState := parseWorkerStateRef.Get()
			var parseWorker interop.Worker
			for {
				parseWorkerState.mu.Lock()
				if parseWorkerState.isClosed {
					parseWorkerState.mu.Unlock()
					return
				}
				if parseWorkerState.isReady {
					parseWorker = parseWorkerState.worker
					parseWorkerState.mu.Unlock()
					break
				}
				if parseWorkerState.isOpening {
					parseWaitCh := parseWorkerState.waitCh
					parseWorkerState.mu.Unlock()
					select {
					case <-parseWaitCh:
						if parseCtx.Err() != nil {
							return
						}
						continue
					case <-parseCtx.Done():
						return
					}
				}
				parseWaitCh := make(chan struct{})
				if parseWorkerState.openCtx == nil {
					parseWorkerState.openCtx, parseWorkerState.openCancel = context.WithCancel(context.Background())
				}
				parseOpenCtx := parseWorkerState.openCtx
				parseWorkerState.isOpening = true
				parseWorkerState.waitCh = parseWaitCh
				parseWorkerState.mu.Unlock()

				parseNextWorker, parseErr := interop.OpenWorker(parseOpenCtx, parseOptions)

				parseWorkerState.mu.Lock()
				parseWorkerClosed := parseWorkerState.isClosed
				parseWorkerState.isOpening = false
				if parseWorkerState.waitCh == parseWaitCh {
					parseWorkerState.waitCh = nil
				}
				if parseErr == nil && !parseWorkerClosed {
					parseWorkerState.worker = parseNextWorker
					parseWorkerState.isReady = true
					parseWorker = parseNextWorker
				}
				parseWorkerState.mu.Unlock()
				close(parseWaitCh)

				if parseErr != nil {
					if parseCtx.Err() != nil || parseRequestSeq.Get() != parseSeq {
						return
					}
					parseCancelRef.Set(nil)
					parseState.Update(func(parsePrev2 WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
						parsePrev2.Running = false
						parsePrev2.Ready = false
						parsePrev2.Error = parseErr
						return parsePrev2
					})
					return
				}
				if parseWorkerClosed {
					_ = parseNextWorker.Terminate()
					return
				}
				break
			}

			parseValue, parseErr2 := interop.RequestWorkerDecoded[Request, Progress, Result](parseCtx, parseWorker, parseName, parseRequestPayload, func(parseProgress interop.DecodedWorkerMessage[Progress], parseProgressErr error) {
				if parseProgressErr != nil || parseCtx.Err() != nil || parseRequestSeq.Get() != parseSeq {
					return
				}
				parseState.Update(func(parsePrev3 WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
					parsePrev3.Progress = parseProgress.Payload
					parsePrev3.ProgressReady = true
					parsePrev3.Running = true
					parsePrev3.Started = true
					return parsePrev3
				})
			})
			if parseCtx.Err() != nil || parseRequestSeq.Get() != parseSeq {
				return
			}
			if parseErr2 != nil {
				parseCancelRef.Set(nil)
				if interop.IsCode(parseErr2, interop.CodeDisposed) {
					parseWorkerState.mu.Lock()
					parseWorkerState.worker = interop.Worker{}
					parseWorkerState.isReady = false
					parseWorkerState.mu.Unlock()
				}
				parseState.Update(func(parsePrev4 WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
					parsePrev4.Running = false
					parsePrev4.Ready = false
					parsePrev4.Error = parseErr2
					return parsePrev4
				})
				return
			}

			parseCancelRef.Set(nil)
			parseState.Set(WorkerTaskState[Progress, Result]{
				Value:         parseValue,
				Progress:      parseState.Get().Progress,
				ProgressReady: parseState.Get().ProgressReady,
				Running:       false,
				Ready:         true,
				Cancelled:     false,
				Started:       true,
				Error:         nil,
			})
		}(parsePayload)
	}

	parseCancel3 := func() {
		if parseActiveCancel := parseCancelRef.Get(); parseActiveCancel != nil {
			parseActiveCancel()
			parseCancelRef.Set(nil)
		} else {
			return
		}

		parseState.Update(func(parsePrev5 WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
			parsePrev5.Running = false
			parsePrev5.Cancelled = true
			parsePrev5.Started = true
			return parsePrev5
		})
	}

	UseEffect(func() func() {
		return func() {
			if parseActiveCancel2 := parseCancelRef.Get(); parseActiveCancel2 != nil {
				parseActiveCancel2()
				parseCancelRef.Set(nil)
			}
			parseWorkerState := parseWorkerStateRef.Get()
			parseWorkerState.mu.Lock()
			parseWorker := parseWorkerState.worker
			parseOpenCancel := parseWorkerState.openCancel
			parseWorkerReady := parseWorkerState.isReady
			parseWorkerState.worker = interop.Worker{}
			parseWorkerState.openCtx = nil
			parseWorkerState.openCancel = nil
			parseWorkerState.isReady = false
			parseWorkerState.isClosed = true
			parseWorkerState.mu.Unlock()
			if parseOpenCancel != nil {
				parseOpenCancel()
			}
			if parseWorkerReady {
				_ = parseWorker.Terminate()
			}
		}
	}, true)

	return WorkerTask[Request, Progress, Result]{
		get:    func() WorkerTaskState[Progress, Result] { return parseState.Get() },
		start:  parseStart,
		cancel: parseCancel3,
	}
}

// Get returns the current worker task state.
func (parseT WorkerTask[Request, Progress, Result]) Get() WorkerTaskState[Progress, Result] {
	if parseT.get == nil {
		var parseZero WorkerTaskState[Progress, Result]
		return parseZero
	}
	return parseT.get()
}

// Start launches a worker-backed request with payload, cancelling any previous
// in-flight run.
func (parseT WorkerTask[Request, Progress, Result]) Start(parsePayload Request) {
	if parseT.start != nil {
		parseT.start(parsePayload)
	}
}

// Cancel cancels the current in-flight worker request, if any.
func (parseT WorkerTask[Request, Progress, Result]) Cancel() {
	if parseT.cancel != nil {
		parseT.cancel()
	}
}
