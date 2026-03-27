package interop

import (
	"context"
	"errors"
	"sync"
)

// WorkerRequester defines the request-capable worker surface shared by single
// workers and worker pools.
type WorkerRequester interface {
	Request(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error)
}

// WorkerPoolOptions configures a fixed-size pool of request-capable workers.
type WorkerPoolOptions struct {
	Size       int
	QueueLimit int
	OpenWorker func(context.Context) (Worker, error)
}

// WorkerPool routes request-style jobs across a fixed worker set with bounded
// queueing and explicit shutdown controls.
type WorkerPool struct {
	request       func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error)
	close         func() error
	drain         func(context.Context) error
	getSize       func() int
	getQueueLimit func() int
}

type workerPoolState struct {
	storeMu           sync.Mutex
	storeAccepting    bool
	storeClosed       bool
	storeCloseErr     error
	storeClose        chan struct{}
	storeCloseOnce    sync.Once
	storeAdmissions   chan struct{}
	storeWorkers      chan int
	storeWorkerList   []Worker
	storeOpenWorker   func(context.Context) (Worker, error)
	storeRepairCtx    context.Context
	storeRepairCancel context.CancelFunc
	storeSize         int
	storeQueueLimit   int
	storeRequestWG    sync.WaitGroup
}

// OpenWorkerPool creates a fixed-size worker pool with bounded queueing on top
// of the existing worker request surface.
func OpenWorkerPool(parseCtx context.Context, parseOptions WorkerPoolOptions) (WorkerPool, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if parseOptions.Size <= 0 {
		return WorkerPool{}, wrapError("OpenWorkerPool", "", CodeInvalid, errors.New("worker pool size must be positive"))
	}
	if parseOptions.QueueLimit < 0 {
		return WorkerPool{}, wrapError("OpenWorkerPool", "", CodeInvalid, errors.New("worker pool queue limit must be non-negative"))
	}
	if parseOptions.OpenWorker == nil {
		return WorkerPool{}, wrapError("OpenWorkerPool", "", CodeInvalid, errors.New("worker pool open worker callback is nil"))
	}

	parseRepairCtx, parseRepairCancel := context.WithCancel(context.Background())
	parseWorkers := make([]Worker, 0, parseOptions.Size)
	for parseIndex := 0; parseIndex < parseOptions.Size; parseIndex++ {
		parseWorker, parseErr := parseOptions.OpenWorker(parseCtx)
		if parseErr != nil {
			parseRepairCancel()
			_ = closeWorkerPoolWorkers(parseWorkers)
			return WorkerPool{}, parseErr
		}
		parseWorkers = append(parseWorkers, parseWorker)
	}

	parseState := &workerPoolState{
		storeAccepting:    true,
		storeClose:        make(chan struct{}),
		storeAdmissions:   make(chan struct{}, parseOptions.Size+parseOptions.QueueLimit),
		storeWorkers:      make(chan int, parseOptions.Size),
		storeWorkerList:   append([]Worker(nil), parseWorkers...),
		storeOpenWorker:   parseOptions.OpenWorker,
		storeRepairCtx:    parseRepairCtx,
		storeRepairCancel: parseRepairCancel,
		storeSize:         parseOptions.Size,
		storeQueueLimit:   parseOptions.QueueLimit,
	}
	for parseIndex := range parseWorkers {
		parseState.storeWorkers <- parseIndex
	}

	return WorkerPool{
		request:       parseState.request,
		close:         parseState.close,
		drain:         parseState.drain,
		getSize:       func() int { return parseState.storeSize },
		getQueueLimit: func() int { return parseState.storeQueueLimit },
	}, nil
}

// Request submits one job to the pool and waits for its final worker result.
func (parseP WorkerPool) Request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	if parseP.request == nil {
		return WorkerMessage{}, unavailable("WorkerPool.Request", "")
	}
	return parseP.request(parseCtx, parseName, parsePayload, parseOnProgress)
}

// Close stops accepting work, terminates the worker set, and causes active or
// queued requests to fail promptly.
func (parseP WorkerPool) Close() error {
	if parseP.close == nil {
		return unavailable("WorkerPool.Close", "")
	}
	return parseP.close()
}

// Drain stops accepting new requests, waits for accepted work to finish, and
// then terminates the worker set.
func (parseP WorkerPool) Drain(parseCtx context.Context) error {
	if parseP.drain == nil {
		return unavailable("WorkerPool.Drain", "")
	}
	return parseP.drain(parseCtx)
}

// GetSize returns the configured worker count for the pool.
func (parseP WorkerPool) GetSize() int {
	if parseP.getSize == nil {
		return 0
	}
	return parseP.getSize()
}

// GetQueueLimit returns the configured number of queued requests allowed in
// addition to the running worker count.
func (parseP WorkerPool) GetQueueLimit() int {
	if parseP.getQueueLimit == nil {
		return 0
	}
	return parseP.getQueueLimit()
}

// request schedules one request onto the next available pooled worker.
func (parseS *workerPoolState) request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}

	parseS.storeMu.Lock()
	if !parseS.storeAccepting {
		parseCloseErr := parseS.storeCloseErr
		parseS.storeMu.Unlock()
		if parseCloseErr != nil {
			return WorkerMessage{}, parseCloseErr
		}
		return WorkerMessage{}, wrapError("WorkerPool.Request", parseName, CodeDisposed, errors.New("worker pool is not accepting requests"))
	}
	parseClose := parseS.storeClose
	parseAdmissions := parseS.storeAdmissions
	parseWorkers := parseS.storeWorkers
	parseS.storeRequestWG.Add(1)
	parseS.storeMu.Unlock()

	parseHasAdmission := false
	defer func() {
		if parseHasAdmission {
			<-parseAdmissions
		}
		parseS.storeRequestWG.Done()
	}()

	select {
	case parseAdmissions <- struct{}{}:
		parseHasAdmission = true
	case <-parseClose:
		return WorkerMessage{}, wrapError("WorkerPool.Request", parseName, CodeDisposed, errors.New("worker pool is closed"))
	case <-parseCtx.Done():
		return WorkerMessage{}, workerPoolContextError("WorkerPool.Request", parseName, parseCtx.Err())
	}

	var parseWorkerIndex int
	select {
	case parseWorkerIndex = <-parseWorkers:
	case <-parseClose:
		return WorkerMessage{}, wrapError("WorkerPool.Request", parseName, CodeDisposed, errors.New("worker pool is closed"))
	case <-parseCtx.Done():
		return WorkerMessage{}, workerPoolContextError("WorkerPool.Request", parseName, parseCtx.Err())
	}

	parseWorker, hasWorker := parseS.getWorkerAtIndex(parseWorkerIndex)
	if !hasWorker {
		return WorkerMessage{}, wrapError("WorkerPool.Request", parseName, CodeDisposed, errors.New("worker pool worker is unavailable"))
	}
	parseShouldReturnWorker := true
	defer func() {
		if parseShouldReturnWorker {
			parseS.storeWorker(parseWorkerIndex)
		}
	}()

	parseMessage, parseErr := parseWorker.Request(parseCtx, parseName, parsePayload, parseOnProgress)
	if parseErr != nil && IsCode(parseErr, CodeDisposed) {
		parseShouldReturnWorker = false
		go func(parseIndex int) {
			_ = parseS.replaceWorker(parseIndex)
		}(parseWorkerIndex)
	}
	return parseMessage, parseErr
}

// close shuts the pool down immediately and terminates all workers.
func (parseS *workerPoolState) close() error {
	parseWorkers, parseErr := parseS.stop(false)
	if parseErr != nil {
		return parseErr
	}
	return closeWorkerPoolWorkers(parseWorkers)
}

// drain stops new admissions, waits for accepted jobs to finish, and then
// terminates all workers.
func (parseS *workerPoolState) drain(parseCtx context.Context) error {
	if parseCtx == nil {
		parseCtx = context.Background()
	}

	parseWorkers, parseErr := parseS.stop(true)
	if parseErr != nil {
		return parseErr
	}
	if parseErr := waitWorkerPoolRequests(parseCtx, &parseS.storeRequestWG); parseErr != nil {
		return parseErr
	}
	parseS.storeMu.Lock()
	parseS.storeClosed = true
	parseS.closeSignal()
	parseS.storeMu.Unlock()
	return closeWorkerPoolWorkers(parseWorkers)
}

// stop flips the pool into non-accepting mode and optionally marks it closed
// immediately for forced shutdown.
func (parseS *workerPoolState) stop(parseDrain bool) ([]Worker, error) {
	parseS.storeMu.Lock()
	defer parseS.storeMu.Unlock()
	if parseS.storeClosed {
		parseOp := "WorkerPool.Close"
		if parseDrain {
			parseOp = "WorkerPool.Drain"
		}
		if parseS.storeCloseErr != nil {
			return nil, parseS.storeCloseErr
		}
		return nil, wrapError(parseOp, "", CodeDisposed, errors.New("worker pool is closed"))
	}
	parseS.storeAccepting = false
	parseWorkers := append([]Worker(nil), parseS.storeWorkerList...)
	parseRepairCancel := parseS.storeRepairCancel
	parseS.storeRepairCancel = nil
	if !parseDrain {
		parseS.storeClosed = true
	}
	if parseRepairCancel != nil {
		parseRepairCancel()
	}
	if !parseDrain {
		parseS.closeSignal()
	}
	if parseDrain {
		return parseWorkers, nil
	}
	return parseWorkers, nil
}

// storeWorker returns one pooled slot to the available worker queue when the
// pool still owns it.
func (parseS *workerPoolState) storeWorker(parseIndex int) {
	parseS.storeMu.Lock()
	parseClosed := parseS.storeClosed
	parseWorkers := parseS.storeWorkers
	parseS.storeMu.Unlock()
	if parseClosed {
		return
	}
	parseWorkers <- parseIndex
}

// getWorkerAtIndex returns the worker handle currently stored in one pooled
// slot.
func (parseS *workerPoolState) getWorkerAtIndex(parseIndex int) (Worker, bool) {
	parseS.storeMu.Lock()
	defer parseS.storeMu.Unlock()
	if parseIndex < 0 || parseIndex >= len(parseS.storeWorkerList) {
		return Worker{}, false
	}
	return parseS.storeWorkerList[parseIndex], true
}

// replaceWorker reopens capacity after one pooled worker becomes disposed. If
// replacement fails, the pool is closed so future requests fail fast instead of
// silently running below the configured concurrency.
func (parseS *workerPoolState) replaceWorker(parseIndex int) error {
	parseS.storeMu.Lock()
	if parseS.storeClosed || !parseS.storeAccepting || parseIndex < 0 || parseIndex >= len(parseS.storeWorkerList) {
		parseS.storeMu.Unlock()
		return nil
	}
	parseOpenWorker := parseS.storeOpenWorker
	parseRepairCtx := parseS.storeRepairCtx
	parseS.storeMu.Unlock()

	parseWorker, parseErr := parseOpenWorker(parseRepairCtx)
	if parseErr != nil {
		parseS.storeMu.Lock()
		parseClosed := parseS.storeClosed
		parseAccepting := parseS.storeAccepting
		parseS.storeMu.Unlock()
		if parseClosed || !parseAccepting {
			return nil
		}
		parseRepairErr := wrapError("WorkerPool.ReplaceWorker", "", CodeRemote, parseErr)
		parseS.closeWithRepairError(parseRepairErr)
		return parseRepairErr
	}

	parseS.storeMu.Lock()
	parseClosed := parseS.storeClosed
	parseAccepting := parseS.storeAccepting
	if !parseClosed && parseAccepting {
		parseS.storeWorkerList[parseIndex] = parseWorker
	}
	parseWorkers := parseS.storeWorkers
	parseS.storeMu.Unlock()
	if parseClosed || !parseAccepting {
		_ = parseWorker.Terminate()
		return nil
	}
	parseWorkers <- parseIndex
	return nil
}

// closeWithRepairError flips the pool into a closed state after replacement
// failure so later requests fail clearly instead of hanging on reduced
// capacity.
func (parseS *workerPoolState) closeWithRepairError(parseErr error) {
	if parseErr == nil {
		return
	}
	parseS.storeMu.Lock()
	if parseS.storeClosed || !parseS.storeAccepting {
		parseS.storeMu.Unlock()
		return
	}
	parseWorkers := append([]Worker(nil), parseS.storeWorkerList...)
	parseS.storeAccepting = false
	parseS.storeClosed = true
	parseS.storeCloseErr = parseErr
	parseRepairCancel := parseS.storeRepairCancel
	parseS.storeRepairCancel = nil
	parseS.storeMu.Unlock()
	if parseRepairCancel != nil {
		parseRepairCancel()
	}
	parseS.closeSignal()
	_ = closeWorkerPoolWorkers(parseWorkers)
}

// cancelRepairContext cancels any in-flight worker replacement work tied to
// the pool.
func (parseS *workerPoolState) cancelRepairContext() {
	parseS.storeMu.Lock()
	parseRepairCancel := parseS.storeRepairCancel
	parseS.storeRepairCancel = nil
	parseS.storeMu.Unlock()
	if parseRepairCancel != nil {
		parseRepairCancel()
	}
}

// closeSignal closes the shared shutdown signal at most once.
func (parseS *workerPoolState) closeSignal() {
	parseS.storeCloseOnce.Do(func() {
		close(parseS.storeClose)
	})
}

// waitWorkerPoolRequests waits for all accepted pool requests to settle or the
// provided context to expire.
func waitWorkerPoolRequests(parseCtx context.Context, parseWG *sync.WaitGroup) error {
	parseDone := make(chan struct{})
	go func() {
		parseWG.Wait()
		close(parseDone)
	}()
	select {
	case <-parseDone:
		return nil
	case <-parseCtx.Done():
		return workerPoolContextError("WorkerPool.Drain", "", parseCtx.Err())
	}
}

// closeWorkerPoolWorkers terminates all workers in the provided slice and
// joins any non-disposed shutdown errors.
func closeWorkerPoolWorkers(parseWorkers []Worker) error {
	var parseErrors []error
	for _, parseWorker := range parseWorkers {
		if parseErr := parseWorker.Terminate(); parseErr != nil && !IsCode(parseErr, CodeDisposed) {
			parseErrors = append(parseErrors, parseErr)
		}
	}
	return errors.Join(parseErrors...)
}

// workerPoolContextError maps request or drain context failures into the
// shared interop timeout or cancellation error codes.
func workerPoolContextError(parseOp string, parseTarget string, parseErr error) error {
	if errors.Is(parseErr, context.DeadlineExceeded) {
		return wrapError(parseOp, parseTarget, CodeTimeout, parseErr)
	}
	return wrapError(parseOp, parseTarget, CodeCancelled, parseErr)
}
