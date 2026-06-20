package kvstate

import "context"

// Record is one stored key/value with the metadata used for conflict resolution.
type Record struct {
	Key       string
	Value     []byte
	Version   int64 // monotonic per key
	UpdatedAt int64 // unix milliseconds
}

// PersistenceBackend is the storage contract behind a binding. The default is
// the shared SQLite engine; implement this to persist elsewhere (a REST API, a
// custom OPFS layout, an in-memory mock) while keeping the same hook/atom API.
//
// Cross-tab notification is handled separately (see watch.go), so a backend only
// needs to implement durable CRUD.
type PersistenceBackend interface {
	Load(parseCtx context.Context, parseKey string) (parseRecord Record, parseFound bool, parseErr error)
	Save(parseCtx context.Context, parseRecord Record) error
	Delete(parseCtx context.Context, parseKey string) error
	Keys(parseCtx context.Context) ([]string, error)
}
