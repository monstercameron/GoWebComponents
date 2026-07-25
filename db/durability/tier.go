// Package durability selects and reports a client-side storage tier
// (plan item P3.6).
//
// P3.6's three criteria are all about states and transitions rather than about
// file syscalls:
//
//   - incremental durability, verified by kill-and-reopen
//   - a second tab reaches a READ-ONLY state without throwing, and the app is
//     told
//   - when OPFS is unavailable the IndexedDB tier is used, with a diagnostic and
//     no app-visible error
//
// So the selection and the lock model live here, in a package with no engine and
// no syscall/js, and are tested natively. The browser half — opening an OPFS
// sync access handle, catching the lock error a second tab gets — is injected
// through Capabilities and Locker, and is NOT verified by these tests. That
// boundary is stated rather than implied: what is proven here is that the right
// state is entered and reported, not that the browser behaves as described.
//
// The whole point of separating them is that the state machine is where the
// bugs that matter live. A second tab that throws instead of degrading takes
// down a user's other window; a fallback that reports an error where it should
// report a diagnostic makes a working app look broken.
package durability

import (
	"errors"
	"fmt"
)

// Tier names a storage backend.
type Tier string

const (
	// TierOPFS is the Origin Private File System, accessed through sync access
	// handles in a worker. Incremental: a write reaches disk without rewriting
	// the database image.
	TierOPFS Tier = "opfs"
	// TierIndexedDB snapshots the whole database image on flush. Universally
	// available, and the cost is that durability is all-or-nothing per flush.
	TierIndexedDB Tier = "indexeddb"
	// TierMemory does not survive a reload. The correct tier when the caller
	// asked for it, and never a silent fallback — a user who expects their data
	// to persist must not be given memory storage quietly.
	TierMemory Tier = "memory"
)

// Access describes what the holder may do with the database.
type Access string

const (
	// AccessReadWrite is exclusive ownership.
	AccessReadWrite Access = "read-write"
	// AccessReadOnly means another context holds the writer lock.
	//
	// A state, not an error. OPFS allows one sync access handle per file, so a
	// second tab CANNOT write — and throwing there would break a window the user
	// did not touch, over a condition that is completely normal.
	AccessReadOnly Access = "read-only"
	// AccessNone means the database could not be opened at all.
	AccessNone Access = "none"
)

// Capabilities reports what the environment supports.
//
// Injected rather than probed here, so the state machine is testable without a
// browser and so a caller can force a tier in a test.
type Capabilities struct {
	// HasOPFS reports that the Origin Private File System is reachable AND that
	// sync access handles are usable, which requires a worker context.
	HasOPFS bool
	// HasIndexedDB reports that IndexedDB is reachable.
	HasIndexedDB bool
}

// Request is what the application asked for.
type Request struct {
	// Preferred is the tier the caller wants.
	Preferred Tier
	// AllowFallback permits a lower tier when the preferred one is unavailable.
	//
	// True is the right default for durability, and false is the right choice
	// for a caller who would rather fail loudly than write somewhere the user
	// did not expect.
	AllowFallback bool
}

// Selection is the outcome of choosing a tier.
type Selection struct {
	// Tier is what will actually be used.
	Tier Tier
	// Requested is what the caller asked for.
	Requested Tier
	// Degraded reports that the tier is lower than requested.
	Degraded bool
	// Diagnostic explains a degradation in one sentence, empty otherwise.
	//
	// P3.6 requires a diagnostic and NO app-visible error for the OPFS →
	// IndexedDB fallback, so the two are separate fields rather than one error:
	// an error would force every caller to decide whether to abort, over a
	// condition where continuing is correct.
	Diagnostic string
}

// ErrNoTierAvailable is returned when nothing can be used.
var ErrNoTierAvailable = errors.New("durability: no storage tier is available")

// Select chooses a tier.
//
// The fallback order is OPFS → IndexedDB and stops there. Memory is never a
// silent fallback: a caller who asked for durability and receives memory storage
// would lose the user's data on reload, and would have been told about it only
// in a diagnostic nobody reads. Falling back to memory therefore requires
// asking for memory.
func Select(parseRequest Request, parseCapabilities Capabilities) (Selection, error) {
	switch parseRequest.Preferred {
	case TierMemory:
		return Selection{Tier: TierMemory, Requested: TierMemory}, nil

	case TierIndexedDB:
		if parseCapabilities.HasIndexedDB {
			return Selection{Tier: TierIndexedDB, Requested: TierIndexedDB}, nil
		}
		return Selection{}, fmt.Errorf("%w: indexeddb was requested and is unavailable", ErrNoTierAvailable)

	case TierOPFS:
		if parseCapabilities.HasOPFS {
			return Selection{Tier: TierOPFS, Requested: TierOPFS}, nil
		}
		if !parseRequest.AllowFallback {
			return Selection{}, fmt.Errorf("%w: opfs was requested, is unavailable, and fallback was not allowed", ErrNoTierAvailable)
		}
		if parseCapabilities.HasIndexedDB {
			return Selection{
				Tier:      TierIndexedDB,
				Requested: TierOPFS,
				Degraded:  true,
				Diagnostic: "opfs is unavailable in this context, so the indexeddb tier is in use; " +
					"durability is per-flush rather than incremental",
			}, nil
		}
		return Selection{}, fmt.Errorf("%w: neither opfs nor indexeddb is available", ErrNoTierAvailable)

	default:
		return Selection{}, fmt.Errorf("durability: unknown tier %q", parseRequest.Preferred)
	}
}

// Incremental reports whether the tier makes writes durable without rewriting
// the whole database image.
//
// The property that distinguishes the tiers in practice: a 50,000-row import
// against IndexedDB rewrites the image on every flush, which is what makes
// incremental durability worth the complexity of OPFS.
func (parseSelection Selection) Incremental() bool {
	return parseSelection.Tier == TierOPFS
}

// Durable reports whether data survives a reload.
func (parseSelection Selection) Durable() bool {
	return parseSelection.Tier == TierOPFS || parseSelection.Tier == TierIndexedDB
}

// ------------------------------------------------------------ single writer

// Locker acquires the single-writer lock for a database.
//
// Injected because the real implementation is an OPFS sync access handle, which
// exists only in a browser worker. The contract is narrow on purpose: acquiring
// either succeeds, or fails because someone else holds it, and those are
// different outcomes rather than one error to be string-matched.
type Locker interface {
	// Acquire attempts to take the writer lock.
	//
	// Returns (true, nil) on success, (false, nil) when another context holds
	// it, and (false, err) when something actually went wrong.
	Acquire(parseName string) (bool, error)
	// Release drops the lock.
	Release(parseName string) error
}

// Handle is an opened database's tier and access state.
type Handle struct {
	Name      string
	Selection Selection
	Access    Access
	// ReadOnlyReason explains a read-only handle, for the app to surface.
	//
	// P3.6 requires the second tab to reach read-only "without throwing and
	// surface it to the app". A read-only handle with no reason is surfaceable
	// only as a mystery.
	ReadOnlyReason string

	locker Locker
	held   bool
}

// Open selects a tier and attempts to take the writer lock.
//
// A second tab does NOT get an error. It gets a read-only handle with a reason,
// because that is the state the user's second window is actually in, and
// throwing would break a window they did not touch over an entirely normal
// condition.
func Open(parseName string, parseRequest Request, parseCapabilities Capabilities, parseLocker Locker) (*Handle, error) {
	if parseName == "" {
		return nil, errors.New("durability: a database name is required")
	}

	parseSelection, parseErr := Select(parseRequest, parseCapabilities)
	if parseErr != nil {
		return nil, parseErr
	}

	parseHandle := &Handle{
		Name:      parseName,
		Selection: parseSelection,
		Access:    AccessReadWrite,
		locker:    parseLocker,
	}

	// Memory is per-context by definition, so there is nothing to contend for.
	if parseSelection.Tier == TierMemory || parseLocker == nil {
		return parseHandle, nil
	}

	hasLock, parseLockErr := parseLocker.Acquire(parseName)
	if parseLockErr != nil {
		// A lock that failed for a reason other than contention is a real
		// failure: the caller asked to open a database and no usable handle
		// exists.
		parseHandle.Access = AccessNone
		return parseHandle, fmt.Errorf("durability: acquiring the writer lock for %q: %w", parseName, parseLockErr)
	}
	if !hasLock {
		parseHandle.Access = AccessReadOnly
		parseHandle.ReadOnlyReason = "another tab or worker holds the writer lock for this database; " +
			"reads are available and writes are not"
		return parseHandle, nil
	}

	parseHandle.held = true
	return parseHandle, nil
}

// CanWrite reports whether writes are permitted.
func (parseHandle *Handle) CanWrite() bool {
	if parseHandle == nil {
		return false
	}
	return parseHandle.Access == AccessReadWrite
}

// ErrReadOnly is returned when a write is attempted on a read-only handle.
//
// A distinct sentinel so an app can present "this window is read-only because
// another tab has the database" rather than a generic failure — which is the
// difference between an explanation and a bug report.
var ErrReadOnly = errors.New("durability: this database handle is read-only")

// CheckWrite reports whether a write may proceed.
func (parseHandle *Handle) CheckWrite() error {
	if parseHandle == nil {
		return errors.New("durability: handle is nil")
	}
	switch parseHandle.Access {
	case AccessReadWrite:
		return nil
	case AccessReadOnly:
		return fmt.Errorf("%w: %s", ErrReadOnly, parseHandle.ReadOnlyReason)
	default:
		return fmt.Errorf("durability: database %q is not open", parseHandle.Name)
	}
}

// Close releases the writer lock if this handle holds it.
//
// Releasing a lock this handle never held would take it from whoever does, so
// the check is on `held` rather than on the access mode — a handle can be
// read-only precisely because someone else is holding what it would release.
func (parseHandle *Handle) Close() error {
	if parseHandle == nil {
		return nil
	}
	if !parseHandle.held || parseHandle.locker == nil {
		parseHandle.Access = AccessNone
		return nil
	}
	parseHandle.held = false
	parseHandle.Access = AccessNone
	return parseHandle.locker.Release(parseHandle.Name)
}

// Promote retries the writer lock on a read-only handle.
//
// For the tab that was second and is now alone, after the first one closed.
// Without it a user's remaining window stays read-only until they reload it,
// which reads as the app being broken.
func (parseHandle *Handle) Promote() (bool, error) {
	if parseHandle == nil {
		return false, errors.New("durability: handle is nil")
	}
	if parseHandle.Access == AccessReadWrite {
		return true, nil
	}
	if parseHandle.Access != AccessReadOnly || parseHandle.locker == nil {
		return false, fmt.Errorf("durability: database %q cannot be promoted from %q", parseHandle.Name, parseHandle.Access)
	}

	hasLock, parseErr := parseHandle.locker.Acquire(parseHandle.Name)
	if parseErr != nil {
		return false, fmt.Errorf("durability: promoting %q: %w", parseHandle.Name, parseErr)
	}
	if !hasLock {
		return false, nil
	}

	parseHandle.held = true
	parseHandle.Access = AccessReadWrite
	parseHandle.ReadOnlyReason = ""
	return true, nil
}
