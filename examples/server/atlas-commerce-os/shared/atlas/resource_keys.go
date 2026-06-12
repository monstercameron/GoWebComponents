package atlas

type atlasCachedResourceState[T any] struct {
	Value   T
	Loading bool
	Error   error
	Ready   bool
	Stale   bool
}

type atlasCachedResource[T any] struct {
	get    func() atlasCachedResourceState[T]
	reload func()
	set    func(T)
	update func(func(T) T)
}

type atlasResourceState[T any] struct {
	Value   T
	Loading bool
	Error   error
	Ready   bool
}

type atlasResource[T any] struct {
	get    func() atlasResourceState[T]
	reload func()
}

type atlasFetchOptions struct {
	Method  string
	Headers map[string]any
	Body    any
}

type atlasImperativeFetchResult struct {
	Data    string
	Status  int
	Headers map[string]string
	Error   string
}

type atlasWorkerTaskState[Progress any, Result any] struct {
	Value         Result
	Progress      Progress
	ProgressReady bool
	Running       bool
	Ready         bool
	Cancelled     bool
	Started       bool
	Error         error
}

type atlasWorkerTask[Request any, Progress any, Result any] struct {
	get    func() atlasWorkerTaskState[Progress, Result]
	start  func(Request)
	cancel func()
}

type atlasChannelValue[T any] struct {
	get    func() T
	ok     func() bool
	closed func() bool
}

func (parseR atlasCachedResource[T]) Get() atlasCachedResourceState[T] {
	if parseR.get == nil {
		var parseZero atlasCachedResourceState[T]
		return parseZero
	}
	return parseR.get()
}

func (parseR atlasCachedResource[T]) Reload() {
	if parseR.reload != nil {
		parseR.reload()
	}
}

func (parseR atlasCachedResource[T]) Set(parseValue T) {
	if parseR.set != nil {
		parseR.set(parseValue)
	}
}

func (parseR atlasCachedResource[T]) Update(parseFn func(T) T) {
	if parseR.update != nil {
		parseR.update(parseFn)
	}
}

func (parseR atlasResource[T]) Get() atlasResourceState[T] {
	if parseR.get == nil {
		var parseZero atlasResourceState[T]
		return parseZero
	}
	return parseR.get()
}

func (parseR atlasResource[T]) Reload() {
	if parseR.reload != nil {
		parseR.reload()
	}
}

func (parseT atlasWorkerTask[Request, Progress, Result]) Get() atlasWorkerTaskState[Progress, Result] {
	if parseT.get == nil {
		var parseZero atlasWorkerTaskState[Progress, Result]
		return parseZero
	}
	return parseT.get()
}

func (parseT atlasWorkerTask[Request, Progress, Result]) Start(parsePayload Request) {
	if parseT.start != nil {
		parseT.start(parsePayload)
	}
}

func (parseT atlasWorkerTask[Request, Progress, Result]) Cancel() {
	if parseT.cancel != nil {
		parseT.cancel()
	}
}

func (parseC atlasChannelValue[T]) Get() T {
	if parseC.get == nil {
		var parseZero T
		return parseZero
	}
	return parseC.get()
}

func (parseC atlasChannelValue[T]) Ok() bool {
	if parseC.ok == nil {
		return false
	}
	return parseC.ok()
}

func (parseC atlasChannelValue[T]) Closed() bool {
	if parseC.closed == nil {
		return false
	}
	return parseC.closed()
}
