package durability_test

import (
	"errors"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/db/durability"
)

// v5 P3.6 — OPFS backend, single-writer.
//
// The three criteria, and what these tests can and cannot establish:
//
//	incremental durability by kill-and-reopen        -- the TIER PROPERTY is
//	                                                    asserted here; the actual
//	                                                    kill-and-reopen is a
//	                                                    browser test
//	a second tab reaches read-only WITHOUT THROWING   -- fully tested here
//	OPFS unavailable -> IndexedDB, diagnostic, no     -- fully tested here
//	  app-visible error
//
// The browser half — opening a sync access handle, catching the lock error a
// second tab gets — is injected and NOT verified by these tests. That boundary
// is stated rather than implied. What is proven is that the right state is
// entered and reported, which is where the bugs that matter live: a second tab
// that throws takes down a window the user did not touch, and a fallback that
// reports an error where it should report a diagnostic makes a working app look
// broken.

// fakeLocker models the single OPFS sync access handle per file.
type fakeLocker struct {
	heldBy       map[string]bool
	acquireCalls int
	failWith     error
}

func newFakeLocker() *fakeLocker {
	return &fakeLocker{heldBy: map[string]bool{}}
}

func (parseLocker *fakeLocker) Acquire(parseName string) (bool, error) {
	parseLocker.acquireCalls++
	if parseLocker.failWith != nil {
		return false, parseLocker.failWith
	}
	if parseLocker.heldBy[parseName] {
		return false, nil
	}
	parseLocker.heldBy[parseName] = true
	return true, nil
}

func (parseLocker *fakeLocker) Release(parseName string) error {
	delete(parseLocker.heldBy, parseName)
	return nil
}

const opfsAvailable = true

func opfsCapabilities() durability.Capabilities {
	return durability.Capabilities{HasOPFS: opfsAvailable, HasIndexedDB: true}
}

// ------------------------------------------ criterion: the fallback is quiet

// TestOPFSUnavailableFallsBackWithADiagnosticAndNoError is the third criterion,
// stated exactly.
func TestOPFSUnavailableFallsBackWithADiagnosticAndNoError(parseT *testing.T) {
	parseSelection, parseErr := durability.Select(
		durability.Request{Preferred: durability.TierOPFS, AllowFallback: true},
		durability.Capabilities{HasOPFS: false, HasIndexedDB: true})

	// "no app-visible error" is the load-bearing half. An error would force
	// every caller to decide whether to abort, over a condition where continuing
	// is correct.
	if parseErr != nil {
		parseT.Fatalf("the fallback must not surface as an error: %v", parseErr)
	}
	if parseSelection.Tier != durability.TierIndexedDB {
		parseT.Errorf("tier = %q, want indexeddb", parseSelection.Tier)
	}
	if !parseSelection.Degraded {
		parseT.Error("a fallback must report itself degraded")
	}
	if parseSelection.Diagnostic == "" {
		parseT.Error("the criterion requires a diagnostic; a silent downgrade is what T9 is about")
	}
	if parseSelection.Requested != durability.TierOPFS {
		parseT.Errorf("requested = %q, want the caller's original request preserved", parseSelection.Requested)
	}
}

// TestMemoryIsNeverASilentFallback: a caller who asked for durability and
// receives memory storage loses the user's data on reload, and would have been
// told only in a diagnostic nobody reads.
func TestMemoryIsNeverASilentFallback(parseT *testing.T) {
	_, parseErr := durability.Select(
		durability.Request{Preferred: durability.TierOPFS, AllowFallback: true},
		durability.Capabilities{HasOPFS: false, HasIndexedDB: false})

	if parseErr == nil {
		parseT.Fatal("with no durable tier available the call must fail rather than quietly using memory")
	}
	if !errors.Is(parseErr, durability.ErrNoTierAvailable) {
		parseT.Errorf("err = %v, want ErrNoTierAvailable", parseErr)
	}

	// Memory is available when it is what was asked for.
	parseSelection, parseMemoryErr := durability.Select(
		durability.Request{Preferred: durability.TierMemory}, durability.Capabilities{})
	if parseMemoryErr != nil || parseSelection.Tier != durability.TierMemory {
		parseT.Errorf("selection = %+v, err = %v; memory must be available when requested", parseSelection, parseMemoryErr)
	}
	if parseSelection.Durable() {
		parseT.Error("memory must not report itself durable")
	}
}

func TestFallbackCanBeRefused(parseT *testing.T) {
	_, parseErr := durability.Select(
		durability.Request{Preferred: durability.TierOPFS, AllowFallback: false},
		durability.Capabilities{HasOPFS: false, HasIndexedDB: true})
	if parseErr == nil {
		parseT.Error("a caller who would rather fail than write somewhere unexpected must be able to say so")
	}
}

// TestIncrementalIsATierProperty is the part of criterion one that a native
// test can hold: which tier claims incremental durability.
//
// The kill-and-reopen verification itself is a browser test — nothing here
// touches a file.
func TestIncrementalIsATierProperty(parseT *testing.T) {
	parseOPFS, _ := durability.Select(durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities())
	if !parseOPFS.Incremental() {
		parseT.Error("opfs is the tier that makes writes durable without rewriting the image")
	}

	parseIndexedDB, _ := durability.Select(
		durability.Request{Preferred: durability.TierIndexedDB},
		durability.Capabilities{HasIndexedDB: true})
	if parseIndexedDB.Incremental() {
		parseT.Error("indexeddb snapshots the whole image per flush and must not claim incremental durability")
	}
	if !parseIndexedDB.Durable() {
		parseT.Error("indexeddb survives a reload")
	}
}

// ------------------------------------- criterion: the second tab degrades

// TestSecondTabIsReadOnlyAndNotAnError is the second criterion.
func TestSecondTabIsReadOnlyAndNotAnError(parseT *testing.T) {
	parseLocker := newFakeLocker()

	parseFirst, parseErr := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)
	if parseErr != nil {
		parseT.Fatalf("first tab: %v", parseErr)
	}
	if !parseFirst.CanWrite() {
		parseT.Fatal("the first tab must get the writer lock")
	}

	// The second tab. Opening must NOT fail: throwing here breaks a window the
	// user did not touch, over a completely normal condition.
	parseSecond, parseSecondErr := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)
	if parseSecondErr != nil {
		parseT.Fatalf("a second tab must not error, it must degrade: %v", parseSecondErr)
	}
	if parseSecond.Access != durability.AccessReadOnly {
		parseT.Errorf("access = %q, want read-only", parseSecond.Access)
	}
	if parseSecond.CanWrite() {
		parseT.Error("a second tab must not be able to write")
	}

	// "surfaces it to the app": the reason is what a UI shows.
	if parseSecond.ReadOnlyReason == "" {
		parseT.Error("a read-only handle with no reason is surfaceable only as a mystery")
	}

	// And a write attempt is refused with a sentinel a UI can branch on, not a
	// generic failure.
	parseWriteErr := parseSecond.CheckWrite()
	if !errors.Is(parseWriteErr, durability.ErrReadOnly) {
		parseT.Errorf("err = %v, want ErrReadOnly", parseWriteErr)
	}
	if parseFirst.CheckWrite() != nil {
		parseT.Error("the first tab must still be writable")
	}
}

// TestSecondTabPromotesWhenTheFirstCloses: without this a user's remaining
// window stays read-only until they reload, which reads as the app being broken.
func TestSecondTabPromotesWhenTheFirstCloses(parseT *testing.T) {
	parseLocker := newFakeLocker()

	parseFirst, _ := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)
	parseSecond, _ := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)

	if hasPromoted, parseErr := parseSecond.Promote(); parseErr != nil || hasPromoted {
		parseT.Errorf("promoted=%v err=%v; the first tab still holds the lock", hasPromoted, parseErr)
	}

	if parseErr := parseFirst.Close(); parseErr != nil {
		parseT.Fatalf("Close: %v", parseErr)
	}

	hasPromoted, parseErr := parseSecond.Promote()
	if parseErr != nil {
		parseT.Fatalf("Promote: %v", parseErr)
	}
	if !hasPromoted {
		parseT.Fatal("the remaining tab must be able to take the lock once it is free")
	}
	if !parseSecond.CanWrite() || parseSecond.ReadOnlyReason != "" {
		parseT.Errorf("after promotion: canWrite=%v reason=%q", parseSecond.CanWrite(), parseSecond.ReadOnlyReason)
	}
}

// TestAReadOnlyHandleDoesNotReleaseSomeoneElsesLock is the bug this would have.
//
// A read-only handle is read-only precisely BECAUSE another context holds the
// lock. Releasing on close by access mode rather than by what it actually holds
// would take the lock away from the tab that owns it.
func TestAReadOnlyHandleDoesNotReleaseSomeoneElsesLock(parseT *testing.T) {
	parseLocker := newFakeLocker()

	parseFirst, _ := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)
	parseSecond, _ := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)

	if parseErr := parseSecond.Close(); parseErr != nil {
		parseT.Fatalf("closing a read-only handle: %v", parseErr)
	}

	if !parseLocker.heldBy["app"] {
		parseT.Error("closing the read-only tab released the writer lock held by the first tab")
	}
	if parseFirst.CheckWrite() != nil {
		parseT.Error("the first tab lost its writability when another tab closed")
	}
}

func TestDifferentDatabasesDoNotContend(parseT *testing.T) {
	parseLocker := newFakeLocker()

	parseOne, _ := durability.Open("alpha", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)
	parseTwo, parseErr := durability.Open("beta", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)
	if parseErr != nil {
		parseT.Fatalf("second database: %v", parseErr)
	}
	if !parseOne.CanWrite() || !parseTwo.CanWrite() {
		parseT.Error("two different databases must both be writable; the lock is per-file")
	}
}

// TestALockFailureIsAnErrorUnlikeContention: a lock that failed for a reason
// other than contention means no usable handle exists, which is different from
// a second tab and must not be presented as one.
func TestALockFailureIsAnErrorUnlikeContention(parseT *testing.T) {
	parseLocker := newFakeLocker()
	parseLocker.failWith = errors.New("quota exceeded")

	parseHandle, parseErr := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)
	if parseErr == nil {
		parseT.Fatal("a genuine lock failure must surface as an error")
	}
	if parseHandle.Access != durability.AccessNone {
		parseT.Errorf("access = %q, want none", parseHandle.Access)
	}
	if errors.Is(parseErr, durability.ErrReadOnly) {
		parseT.Error("a lock failure must not be presented as read-only contention")
	}
}

// -------------------------------------------------------------- mechanics

func TestMemoryNeedsNoLock(parseT *testing.T) {
	parseLocker := newFakeLocker()

	parseFirst, _ := durability.Open("app", durability.Request{Preferred: durability.TierMemory}, durability.Capabilities{}, parseLocker)
	parseSecond, parseErr := durability.Open("app", durability.Request{Preferred: durability.TierMemory}, durability.Capabilities{}, parseLocker)
	if parseErr != nil {
		parseT.Fatalf("second memory handle: %v", parseErr)
	}

	if !parseFirst.CanWrite() || !parseSecond.CanWrite() {
		parseT.Error("memory databases are per-context; there is nothing to contend for")
	}
	if parseLocker.acquireCalls != 0 {
		parseT.Errorf("the locker was called %d times for memory storage", parseLocker.acquireCalls)
	}
}

func TestClosedHandleRefusesWrites(parseT *testing.T) {
	parseLocker := newFakeLocker()
	parseHandle, _ := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)

	if parseErr := parseHandle.Close(); parseErr != nil {
		parseT.Fatalf("Close: %v", parseErr)
	}
	if parseHandle.CheckWrite() == nil {
		parseT.Error("a closed handle must refuse writes")
	}
	if parseHandle.CanWrite() {
		parseT.Error("a closed handle is not writable")
	}
	// And the lock is free for the next context.
	if parseLocker.heldBy["app"] {
		parseT.Error("closing the writer did not release the lock")
	}
}

func TestUnknownTierIsRejected(parseT *testing.T) {
	if _, parseErr := durability.Select(durability.Request{Preferred: durability.Tier("floppy")}, durability.Capabilities{}); parseErr == nil {
		parseT.Error("an unknown tier must be rejected rather than silently defaulting")
	}
}

func TestOpenRequiresAName(parseT *testing.T) {
	if _, parseErr := durability.Open("", durability.Request{Preferred: durability.TierMemory}, durability.Capabilities{}, nil); parseErr == nil {
		parseT.Error("a database without a name cannot be locked and must be rejected")
	}
}

func TestNilHandleIsSafe(parseT *testing.T) {
	var parseHandle *durability.Handle
	if parseHandle.CanWrite() {
		parseT.Error("a nil handle is not writable")
	}
	if parseHandle.CheckWrite() == nil {
		parseT.Error("a nil handle must error rather than panic")
	}
	if parseErr := parseHandle.Close(); parseErr != nil {
		parseT.Errorf("closing a nil handle must be a no-op: %v", parseErr)
	}
	if _, parseErr := parseHandle.Promote(); parseErr == nil {
		parseT.Error("promoting a nil handle must error rather than panic")
	}
}

func TestPromotingAWritableHandleIsANoOp(parseT *testing.T) {
	parseLocker := newFakeLocker()
	parseHandle, _ := durability.Open("app", durability.Request{Preferred: durability.TierOPFS}, opfsCapabilities(), parseLocker)

	parseCallsBefore := parseLocker.acquireCalls
	hasPromoted, parseErr := parseHandle.Promote()
	if parseErr != nil || !hasPromoted {
		parseT.Errorf("promoted=%v err=%v, want an immediate yes", hasPromoted, parseErr)
	}
	if parseLocker.acquireCalls != parseCallsBefore {
		parseT.Error("promoting an already-writable handle must not re-acquire the lock")
	}
}
