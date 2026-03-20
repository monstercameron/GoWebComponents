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
	Headers map[string]interface{}
	Body    interface{}
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

func (r atlasCachedResource[T]) Get() atlasCachedResourceState[T] {
	if r.get == nil {
		var zero atlasCachedResourceState[T]
		return zero
	}
	return r.get()
}

func (r atlasCachedResource[T]) Reload() {
	if r.reload != nil {
		r.reload()
	}
}

func (r atlasCachedResource[T]) Set(value T) {
	if r.set != nil {
		r.set(value)
	}
}

func (r atlasCachedResource[T]) Update(fn func(T) T) {
	if r.update != nil {
		r.update(fn)
	}
}

func (r atlasResource[T]) Get() atlasResourceState[T] {
	if r.get == nil {
		var zero atlasResourceState[T]
		return zero
	}
	return r.get()
}

func (r atlasResource[T]) Reload() {
	if r.reload != nil {
		r.reload()
	}
}

func (t atlasWorkerTask[Request, Progress, Result]) Get() atlasWorkerTaskState[Progress, Result] {
	if t.get == nil {
		var zero atlasWorkerTaskState[Progress, Result]
		return zero
	}
	return t.get()
}

func (t atlasWorkerTask[Request, Progress, Result]) Start(payload Request) {
	if t.start != nil {
		t.start(payload)
	}
}

func (t atlasWorkerTask[Request, Progress, Result]) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (c atlasChannelValue[T]) Get() T {
	if c.get == nil {
		var zero T
		return zero
	}
	return c.get()
}

func (c atlasChannelValue[T]) Ok() bool {
	if c.ok == nil {
		return false
	}
	return c.ok()
}

func (c atlasChannelValue[T]) Closed() bool {
	if c.closed == nil {
		return false
	}
	return c.closed()
}
