//go:build js && wasm
// +build js,wasm

package ui

import (
	"context"

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
func UseWorkerTask[Request any, Progress any, Result any](options interop.WorkerOptions, name string) WorkerTask[Request, Progress, Result] {
	state := UseState(WorkerTaskState[Progress, Result]{})
	cancelRef := UseRef((context.CancelFunc)(nil))
	requestSeq := UseRef(0)
	workerRef := UseRef(interop.Worker{})
	workerReady := UseRef(false)

	start := func(payload Request) {
		if cancel := cancelRef.Get(); cancel != nil {
			cancel()
		}

		requestSeq.Set(requestSeq.Get() + 1)
		seq := requestSeq.Get()
		ctx, cancel := context.WithCancel(context.Background())
		cancelRef.Set(cancel)

		state.Update(func(prev WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
			var zeroResult Result
			var zeroProgress Progress
			prev.Value = zeroResult
			prev.Progress = zeroProgress
			prev.ProgressReady = false
			prev.Running = true
			prev.Ready = false
			prev.Cancelled = false
			prev.Started = true
			prev.Error = nil
			return prev
		})

		go func(requestPayload Request) {
			worker := workerRef.Get()
			if !workerReady.Get() {
				nextWorker, err := interop.NewWorker(ctx, options)
				if err != nil {
					if ctx.Err() != nil || requestSeq.Get() != seq {
						return
					}
					state.Update(func(prev WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
						prev.Running = false
						prev.Ready = false
						prev.Error = err
						return prev
					})
					return
				}
				worker = nextWorker
				workerRef.Set(worker)
				workerReady.Set(true)
			}

			value, err := interop.RequestWorkerDecoded[Request, Progress, Result](ctx, worker, name, requestPayload, func(progress interop.DecodedWorkerMessage[Progress], progressErr error) {
				if progressErr != nil || ctx.Err() != nil || requestSeq.Get() != seq {
					return
				}
				state.Update(func(prev WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
					prev.Progress = progress.Payload
					prev.ProgressReady = true
					prev.Running = true
					prev.Started = true
					return prev
				})
			})
			if ctx.Err() != nil || requestSeq.Get() != seq {
				return
			}
			if err != nil {
				if interop.IsCode(err, interop.CodeDisposed) {
					workerReady.Set(false)
					workerRef.Set(interop.Worker{})
				}
				state.Update(func(prev WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
					prev.Running = false
					prev.Ready = false
					prev.Error = err
					return prev
				})
				return
			}

			state.Set(WorkerTaskState[Progress, Result]{
				Value:         value,
				Progress:      state.Get().Progress,
				ProgressReady: state.Get().ProgressReady,
				Running:       false,
				Ready:         true,
				Cancelled:     false,
				Started:       true,
				Error:         nil,
			})
		}(payload)
	}

	cancel := func() {
		if activeCancel := cancelRef.Get(); activeCancel != nil {
			activeCancel()
			cancelRef.Set(nil)
		}

		state.Update(func(prev WorkerTaskState[Progress, Result]) WorkerTaskState[Progress, Result] {
			prev.Running = false
			prev.Cancelled = true
			prev.Started = true
			return prev
		})
	}

	UseEffect(func() func() {
		return func() {
			if activeCancel := cancelRef.Get(); activeCancel != nil {
				activeCancel()
				cancelRef.Set(nil)
			}
			if workerReady.Get() {
				_ = workerRef.Get().Terminate()
				workerReady.Set(false)
				workerRef.Set(interop.Worker{})
			}
		}
	}, true)

	return WorkerTask[Request, Progress, Result]{
		get:    func() WorkerTaskState[Progress, Result] { return state.Get() },
		start:  start,
		cancel: cancel,
	}
}

// Get returns the current worker task state.
func (t WorkerTask[Request, Progress, Result]) Get() WorkerTaskState[Progress, Result] {
	if t.get == nil {
		var zero WorkerTaskState[Progress, Result]
		return zero
	}
	return t.get()
}

// Start launches a worker-backed request with payload, cancelling any previous
// in-flight run.
func (t WorkerTask[Request, Progress, Result]) Start(payload Request) {
	if t.start != nil {
		t.start(payload)
	}
}

// Cancel cancels the current in-flight worker request, if any.
func (t WorkerTask[Request, Progress, Result]) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}
