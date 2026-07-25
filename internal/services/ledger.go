package services

import "fmt"

// Idempotency ledger — the second half of P3.3.
//
// Extracted from runtime2's PatchIdempotencyTracker, which solved the hard part
// correctly: across an async worker bridge, deliveries duplicate and reorder, so
// a receiver needs to distinguish "already applied this" from "this is older
// than what I have" from "two different payloads claim the same version". Those
// three cases want three different outcomes and only one of them is an error.
//
// That reasoning is not about DOM patches at all. P3.4's command replay and
// bulk resume need exactly it, and so does any projection stream. What kept it
// from being reusable is that it lived beside the patch types and spoke their
// vocabulary.
//
// Two deliberate differences from the source, both documented at their code:
// this guards epoch ordering itself, and it bounds its own memory. See
// ledger_runtime2_conformance_test.go for the differential test that pins where
// the two must agree exactly.

// Decision is what a ledger says to do with an arriving message.
type Decision uint8

const (
	// DecisionApply means the message is new and in order.
	DecisionApply Decision = iota
	// DecisionSkipDuplicate means this exact (version, identity) already applied.
	// Skipping is the correct outcome, not an error — duplicate delivery is
	// normal across a worker bridge.
	DecisionSkipDuplicate
	// DecisionSkipStale means the version is below what has already been applied
	// in this epoch. Its payload describes older state, so applying it would
	// regress the receiver.
	DecisionSkipStale
	// DecisionSkipStaleEpoch means the message belongs to an epoch the stream has
	// already moved past.
	DecisionSkipStaleEpoch
)

// ShouldApply reports whether a decision means "apply this message".
//
// Callers should branch on this rather than comparing against DecisionApply, so
// a future skip reason does not silently become an apply at existing call sites.
func (parseDecision Decision) ShouldApply() bool {
	return parseDecision == DecisionApply
}

func (parseDecision Decision) String() string {
	switch parseDecision {
	case DecisionApply:
		return "apply"
	case DecisionSkipDuplicate:
		return "skip-duplicate"
	case DecisionSkipStale:
		return "skip-stale"
	case DecisionSkipStaleEpoch:
		return "skip-stale-epoch"
	default:
		return fmt.Sprintf("unknown-decision(%d)", uint8(parseDecision))
	}
}

// DefaultLedgerRetention is how many recent versions per stream keep their
// identity recorded.
//
// The source tracker kept every version's identity for the life of an epoch,
// which is unbounded: a long-lived stream at a high message rate grows a map
// entry per message forever. Acceptable when an epoch is short and a region is
// one of a few dozen; not acceptable for domain streams that may run for a
// session. See the retention argument on recordVersionLocked for why bounding
// this cannot change any apply/skip outcome.
const DefaultLedgerRetention = 1024

// streamLedgerState is one stream's dedup state within one epoch.
type streamLedgerState struct {
	epoch             uint64
	identityByVersion map[uint64]string
	// recordOrder holds recorded versions oldest-first so the window can evict.
	recordOrder []uint64
	// maxVersion is the high-water mark: the monotonicity guard that makes a
	// never-seen lower version detectable as a reordered delivery.
	maxVersion uint64
	// retentionFloor is the highest version whose identity has been evicted.
	// Everything at or below it is known-old but no longer identity-comparable.
	retentionFloor uint64
}

// Ledger tracks duplicate, stale, and conflicting messages across named streams.
//
// Not safe for concurrent use. A ledger belongs to one receiving thread, which
// is the arrangement everywhere it is used: one worker, one message loop.
type Ledger struct {
	stateByStreamID map[string]streamLedgerState
	retention       int
}

// NewLedger creates a ledger with the default retention window.
func NewLedger() *Ledger {
	return NewLedgerWithRetention(DefaultLedgerRetention)
}

// NewLedgerWithRetention creates a ledger retaining the given number of recent
// versions per stream. A non-positive retention uses the default.
func NewLedgerWithRetention(parseRetention int) *Ledger {
	if parseRetention <= 0 {
		parseRetention = DefaultLedgerRetention
	}
	return &Ledger{
		stateByStreamID: make(map[string]streamLedgerState),
		retention:       parseRetention,
	}
}

// Check reports what to do with a message WITHOUT mutating ledger state.
//
// The split between Check and Commit is the fix for a real bug in the source
// (runtime2 #72): committing version state before the message body validated let
// a message that then failed validation record its version, so a later corrected
// message at that same version was rejected as a conflict and the stream wedged
// permanently. Commit only after the body is known good.
//
// An error means a genuine conflict — two different payloads claiming one
// version — which is a producer bug the caller should surface. Every ordinary
// duplicate or reorder returns a skip decision and a nil error.
func (parseLedger *Ledger) Check(parseStreamID string, parseEpoch uint64, parseVersion uint64, parseIdentity string) (Decision, error) {
	if parseValidationErr := parseLedger.validate(parseStreamID, parseIdentity); parseValidationErr != nil {
		return DecisionSkipStale, parseValidationErr
	}

	getState, hasState := parseLedger.stateByStreamID[parseStreamID]
	if !hasState {
		return DecisionApply, nil
	}

	// Epoch ordering is guarded HERE, unlike in the source.
	//
	// runtime2 treats any epoch mismatch as a reset, which is safe there only
	// because its parse layer rejects an unexpected epoch before the tracker is
	// ever reached. A generic ledger has no such layer above it, so an older
	// epoch arriving late would reset the stream and wipe the current epoch's
	// high-water mark — reintroducing at the epoch level exactly the reordering
	// the version guard exists to prevent.
	if parseEpoch < getState.epoch {
		return DecisionSkipStaleEpoch, nil
	}
	if parseEpoch > getState.epoch {
		return DecisionApply, nil
	}

	if getIdentity, hasIdentity := getState.identityByVersion[parseVersion]; hasIdentity {
		if getIdentity == parseIdentity {
			return DecisionSkipDuplicate, nil
		}
		return DecisionSkipStale, fmt.Errorf(
			"services: stream %q version %d already applied as identity %q, received %q",
			parseStreamID, parseVersion, getIdentity, parseIdentity)
	}

	// A never-seen version below the high-water mark is a reordered delivery. Its
	// payload describes older state, so applying it would regress the receiver.
	if parseVersion < getState.maxVersion {
		return DecisionSkipStale, nil
	}

	return DecisionApply, nil
}

// Commit records an applied message. Call it only after Check returned an apply
// decision AND the message body validated. Repeating it for the same
// (version, identity) is harmless.
func (parseLedger *Ledger) Commit(parseStreamID string, parseEpoch uint64, parseVersion uint64, parseIdentity string) error {
	if parseValidationErr := parseLedger.validate(parseStreamID, parseIdentity); parseValidationErr != nil {
		return parseValidationErr
	}

	getState, hasState := parseLedger.stateByStreamID[parseStreamID]

	// Committing an epoch the stream has moved past would roll the high-water
	// mark backwards and let every already-applied message replay. Check returns
	// a skip for this case, so reaching it means the caller ignored the decision
	// — a caller bug worth surfacing loudly rather than absorbing.
	if hasState && parseEpoch < getState.epoch {
		return fmt.Errorf(
			"services: stream %q cannot commit epoch %d after epoch %d",
			parseStreamID, parseEpoch, getState.epoch)
	}

	if !hasState || getState.epoch != parseEpoch {
		getState = streamLedgerState{
			epoch:             parseEpoch,
			identityByVersion: make(map[uint64]string),
		}
	}

	parseLedger.recordVersionLocked(&getState, parseVersion, parseIdentity)
	parseLedger.stateByStreamID[parseStreamID] = getState
	return nil
}

// Apply is the check-and-commit convenience for callers with no separate body
// validation step. Callers that validate a body must use Check and Commit
// instead; see the note on Check.
func (parseLedger *Ledger) Apply(parseStreamID string, parseEpoch uint64, parseVersion uint64, parseIdentity string) (Decision, error) {
	parseDecision, parseCheckErr := parseLedger.Check(parseStreamID, parseEpoch, parseVersion, parseIdentity)
	if parseCheckErr != nil || !parseDecision.ShouldApply() {
		return parseDecision, parseCheckErr
	}
	if parseCommitErr := parseLedger.Commit(parseStreamID, parseEpoch, parseVersion, parseIdentity); parseCommitErr != nil {
		return DecisionSkipStale, parseCommitErr
	}
	return DecisionApply, nil
}

// Forget drops a stream's state entirely.
//
// The source had no way to do this: a region's state lived as long as the
// tracker. Streams that genuinely end — a closed subscription, a completed bulk
// command — should release their state rather than accumulate for the session.
// Forgetting a stream that later resumes is safe but not free: it loses the
// high-water mark, so an in-flight stale delivery would be applied.
func (parseLedger *Ledger) Forget(parseStreamID string) {
	if parseLedger == nil {
		return
	}
	delete(parseLedger.stateByStreamID, parseStreamID)
}

// TrackedStreams reports how many streams hold state, for diagnostics and for
// tests that assert the ledger is not growing without bound.
func (parseLedger *Ledger) TrackedStreams() int {
	if parseLedger == nil {
		return 0
	}
	return len(parseLedger.stateByStreamID)
}

// TrackedVersions reports how many versions one stream currently retains, which
// is what the retention window bounds.
func (parseLedger *Ledger) TrackedVersions(parseStreamID string) int {
	if parseLedger == nil {
		return 0
	}
	return len(parseLedger.stateByStreamID[parseStreamID].identityByVersion)
}

func (parseLedger *Ledger) validate(parseStreamID string, parseIdentity string) error {
	if parseLedger == nil {
		return fmt.Errorf("services: ledger is nil")
	}
	if parseStreamID == "" {
		return fmt.Errorf("services: ledger stream id is required")
	}
	if parseIdentity == "" {
		return fmt.Errorf("services: ledger message identity is required")
	}
	return nil
}

// recordVersionLocked records one version's identity and evicts past the window.
//
// Why bounding retention cannot change an apply/skip outcome: eviction only ever
// removes versions BELOW the high-water mark, and the monotonicity guard already
// skips every never-seen version below that mark. So a duplicate old enough to
// have been evicted is classified skip-stale instead of skip-duplicate — a
// different reason for the identical outcome of not applying it.
//
// What is genuinely lost is conflict DETECTION for versions older than the
// window: two different payloads claiming one ancient version return a skip
// rather than an error. Both are already being dropped; only the diagnostic is
// weaker. Pinned by TestLedgerRetentionNeverChangesApplyOutcome.
func (parseLedger *Ledger) recordVersionLocked(parseState *streamLedgerState, parseVersion uint64, parseIdentity string) {
	if _, hasVersion := parseState.identityByVersion[parseVersion]; !hasVersion {
		parseState.recordOrder = append(parseState.recordOrder, parseVersion)
	}
	parseState.identityByVersion[parseVersion] = parseIdentity
	if parseVersion > parseState.maxVersion {
		parseState.maxVersion = parseVersion
	}

	for len(parseState.recordOrder) > parseLedger.retention {
		getOldest := parseState.recordOrder[0]
		parseState.recordOrder = parseState.recordOrder[1:]
		delete(parseState.identityByVersion, getOldest)
		if getOldest > parseState.retentionFloor {
			parseState.retentionFloor = getOldest
		}
	}
}
