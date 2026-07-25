package domain

import (
	"errors"
	"fmt"
)

// BulkCommand describes work over an indexed range of items.
//
// The shape is deliberately narrow — an item count and a step function — because
// the resume guarantee depends on items being addressable by a stable index. A
// bulk command over a stream whose order can change between runs cannot be
// resumed correctly by any bookkeeping, and this makes that constraint explicit
// rather than implied.
type BulkCommand struct {
	// ID is the replay and resume key. It must be the same across the original
	// run and every resumption of it.
	ID CommandID
	// Total is the number of items.
	Total int
	// YieldEvery is how many items run between turns given back to the host
	// event loop.
	//
	// Zero never yields, which is correct for a runtime with a thread to itself
	// and wrong for the case bulk commands are actually for. In a single-threaded
	// worker message loop, a run that never yields cannot be cancelled by a
	// message: the cancel is queued behind the run it is meant to stop. The
	// per-item flag check below is necessary but not sufficient, because with no
	// yield nothing can ever set that flag.
	//
	// Setting it requires Runtime.SetYield; ExecuteBulk refuses a command that
	// asks to yield with no yielder installed, rather than running to completion
	// while looking cancellable.
	//
	// Separate from CheckpointEvery on purpose. Checkpoints default to every
	// item, and a round trip through the event loop per item would cost far more
	// than the work.
	YieldEvery int
	// CheckpointEvery is how many items complete between durable progress writes.
	// Zero or one checkpoints every item, which is the only interval that gives
	// exactly-once against a non-transactional store — see CheckpointStore.
	// Larger intervals trade a bounded window of re-executed items for fewer
	// writes, and the window is reported rather than hidden.
	CheckpointEvery int
}

// BulkResult describes one run of a bulk command.
//
// "One run" is the important qualifier: after a crash and a resume there are two
// results, and Processed is per-run. Completed and NextIndex are the fields that
// describe the command as a whole.
type BulkResult struct {
	Total int
	// ResumedFrom is the index this run started at. Non-zero means it resumed.
	ResumedFrom int
	// Processed counts step invocations in this run.
	Processed int
	// MaxReplayWindow is the largest number of items this run MAY have
	// re-executed.
	//
	// Deliberately a bound rather than a count. A run that ends gracefully — a
	// step returning an error — records exactly how far it got, so nothing is
	// re-executed and this is zero. A run killed outright records nothing after
	// its last checkpoint, and no later run can discover how far it actually got:
	// that information died with it. The bound is the checkpoint interval, and
	// reporting the bound is honest where reporting a count would be invented.
	//
	// A caller whose steps are not idempotent needs this to be zero, which means
	// either a transactional CheckpointStore or accepting the window.
	MaxReplayWindow int
	// NextIndex is the first item not known to be complete after this run.
	NextIndex int
	// Completed reports whether every item is done.
	Completed bool
	// Outcome describes the run in the same vocabulary as an atomic command.
	Outcome Outcome
}

// ExecuteBulk runs a bulk command, resuming from any recorded progress.
//
// The three P3.4 criteria all land here:
//
//   - Replay: a command already completed does no work and reports OutcomeReplayed.
//   - Resume: a command that died at item N restarts from its last checkpoint and
//     finishes the remaining items exactly once.
//   - Cancellation: a cancelled command neither replays nor resumes, because
//     Cancel clears its progress and its id stays refused.
//
// A step that fails or panics stops the run, preserves progress, and returns the
// error. Progress is preserved deliberately: the failure is usually about one
// item, and discarding 30,000 completed items to re-do them is the wrong
// response to a bad row.
func (parseRuntime *Runtime) ExecuteBulk(parseCommand BulkCommand, parseStep func(parseIndex int) error) (BulkResult, error) {
	if parseRuntime == nil {
		return BulkResult{Outcome: OutcomeFailed}, errors.New("domain: runtime is nil")
	}
	if parseCommand.ID == "" {
		return BulkResult{Outcome: OutcomeFailed}, errors.New("domain: bulk command id is required")
	}
	if parseCommand.Total < 0 {
		return BulkResult{Outcome: OutcomeFailed}, fmt.Errorf("domain: bulk command %q has a negative total", parseCommand.ID)
	}
	if parseStep == nil {
		return BulkResult{Outcome: OutcomeFailed}, errors.New("domain: bulk command step is required")
	}
	if parseCommand.YieldEvery < 0 {
		return BulkResult{Outcome: OutcomeFailed},
			fmt.Errorf("domain: bulk command %q has a negative yield interval", parseCommand.ID)
	}
	if parseCommand.YieldEvery > 0 && parseRuntime.yield == nil {
		// Refused rather than ignored. Running anyway would produce a command
		// that looks cancellable — it checks the flag on every item — while
		// being structurally incapable of receiving a cancel.
		return BulkResult{Outcome: OutcomeFailed},
			fmt.Errorf("domain: bulk command %q asks to yield every %d items but no yielder is installed; call Runtime.SetYield",
				parseCommand.ID, parseCommand.YieldEvery)
	}

	parseResult := BulkResult{Total: parseCommand.Total}

	if parseRuntime.cancelled[parseCommand.ID] {
		parseResult.Outcome = OutcomeCancelled
		return parseResult, nil
	}

	// A completed bulk command is recorded in the same ledger as an atomic one,
	// so re-running it is a replay rather than a second pass over every item.
	parseDecision, parseCheckErr := parseRuntime.ledger.Check(
		string(parseCommand.ID), commandEpoch, commandVersion, string(parseCommand.ID))
	if parseCheckErr != nil {
		return BulkResult{Total: parseCommand.Total, Outcome: OutcomeFailed},
			fmt.Errorf("domain: bulk command %q: %w", parseCommand.ID, parseCheckErr)
	}
	if !parseDecision.ShouldApply() {
		parseResult.Outcome = OutcomeReplayed
		parseResult.NextIndex = parseCommand.Total
		parseResult.Completed = true
		return parseResult, nil
	}

	parseStartIndex, hasExactResume, parseResumeErr := parseRuntime.resumeIndex(parseCommand)
	if parseResumeErr != nil {
		parseResult.Outcome = OutcomeFailed
		return parseResult, parseResumeErr
	}
	parseResult.ResumedFrom = parseStartIndex
	parseResult.NextIndex = parseStartIndex

	parseInterval := max(parseCommand.CheckpointEvery, 1)
	if parseStartIndex > 0 && !hasExactResume {
		// Resuming from an interval checkpoint means the previous run's fate past
		// that point is unknown. See MaxReplayWindow for why this is a bound and
		// not a count. An exact checkpoint leaves it at zero: the previous run
		// ended gracefully and recorded precisely what it completed.
		parseResult.MaxReplayWindow = parseInterval
	}

	for parseIndex := parseStartIndex; parseIndex < parseCommand.Total; parseIndex++ {
		// Yield BEFORE the cancellation check, never after: the yield is what
		// lets a cancel message be delivered at all, so a turn given back
		// without then re-reading the flag would waste the round trip and delay
		// the stop by a whole interval.
		if parseCommand.YieldEvery > 0 && parseIndex > parseStartIndex &&
			(parseIndex-parseStartIndex)%parseCommand.YieldEvery == 0 {
			parseRuntime.yield()
		}

		// Checked every item rather than once up front: a cancel arriving during a
		// 50,000-item run should stop it, not be noticed after it finishes.
		if parseRuntime.cancelled[parseCommand.ID] {
			parseResult.Outcome = OutcomeCancelled
			return parseResult, nil
		}

		if parseStepErr := runContainedStep(parseStep, parseIndex); parseStepErr != nil {
			parseResult.Outcome = OutcomeFailed
			// Progress up to the last checkpoint is already durable; write what
			// this run confirmed so a resume does not redo it.
			if parseSaveErr := parseRuntime.saveExactProgress(parseCommand, parseResult.NextIndex); parseSaveErr != nil {
				return parseResult, fmt.Errorf("domain: bulk command %q failed at item %d and its progress could not be saved: %w",
					parseCommand.ID, parseIndex, parseSaveErr)
			}
			return parseResult, fmt.Errorf("domain: bulk command %q failed at item %d: %w", parseCommand.ID, parseIndex, parseStepErr)
		}

		parseResult.Processed++
		// Confirmed in memory on every item, made durable only at the interval.
		// The gap between the two is precisely MaxReplayWindow.
		parseResult.NextIndex = parseIndex + 1

		if (parseIndex+1)%parseInterval == 0 {
			if parseSaveErr := parseRuntime.saveProgress(parseCommand, parseIndex+1); parseSaveErr != nil {
				parseResult.Outcome = OutcomeFailed
				return parseResult, fmt.Errorf("domain: bulk command %q could not checkpoint at item %d: %w",
					parseCommand.ID, parseIndex+1, parseSaveErr)
			}
		}
	}

	parseResult.NextIndex = parseCommand.Total
	parseResult.Completed = true
	parseResult.Outcome = OutcomeApplied

	// Record completion BEFORE clearing progress. The other order has a window
	// where a crash leaves neither a checkpoint nor a completion record, and the
	// whole command runs again.
	if parseCommitErr := parseRuntime.ledger.Commit(
		string(parseCommand.ID), commandEpoch, commandVersion, string(parseCommand.ID)); parseCommitErr != nil {
		parseResult.Outcome = OutcomeFailed
		return parseResult, fmt.Errorf("domain: bulk command %q: %w", parseCommand.ID, parseCommitErr)
	}
	if parseRuntime.checkpoints != nil {
		if parseClearErr := parseRuntime.checkpoints.Clear(parseCommand.ID); parseClearErr != nil {
			return parseResult, fmt.Errorf("domain: bulk command %q completed but its progress could not be cleared: %w",
				parseCommand.ID, parseClearErr)
		}
	}
	return parseResult, nil
}

// resumeIndex reads recorded progress, validates it, and reports whether the
// recorded position is exact.
func (parseRuntime *Runtime) resumeIndex(parseCommand BulkCommand) (int, bool, error) {
	if parseRuntime.checkpoints == nil {
		return 0, false, nil
	}
	parseCheckpoint, hasCheckpoint, parseLoadErr := parseRuntime.checkpoints.Load(parseCommand.ID)
	if parseLoadErr != nil {
		return 0, false, fmt.Errorf("domain: bulk command %q could not read its progress: %w", parseCommand.ID, parseLoadErr)
	}
	if !hasCheckpoint {
		return 0, false, nil
	}

	// A checkpoint written against a different item count describes a different
	// command wearing the same id. Resuming into it would skip or repeat an
	// arbitrary range, so refuse rather than guess.
	if parseCheckpoint.Total != parseCommand.Total {
		return 0, false, fmt.Errorf(
			"domain: bulk command %q has progress recorded against %d items but was asked to run %d",
			parseCommand.ID, parseCheckpoint.Total, parseCommand.Total)
	}
	if parseCheckpoint.NextIndex < 0 || parseCheckpoint.NextIndex > parseCommand.Total {
		return 0, false, fmt.Errorf("domain: bulk command %q has progress at item %d, outside 0..%d",
			parseCommand.ID, parseCheckpoint.NextIndex, parseCommand.Total)
	}
	return parseCheckpoint.NextIndex, parseCheckpoint.Exact, nil
}

// saveProgress records an interval checkpoint, which may lag a hard crash.
func (parseRuntime *Runtime) saveProgress(parseCommand BulkCommand, parseNextIndex int) error {
	return parseRuntime.save(parseCommand, parseNextIndex, false)
}

// saveExactProgress records precisely where a gracefully-failed run stopped, so
// its resume re-executes nothing.
func (parseRuntime *Runtime) saveExactProgress(parseCommand BulkCommand, parseNextIndex int) error {
	return parseRuntime.save(parseCommand, parseNextIndex, true)
}

func (parseRuntime *Runtime) save(parseCommand BulkCommand, parseNextIndex int, hasExact bool) error {
	if parseRuntime.checkpoints == nil {
		return nil
	}
	return parseRuntime.checkpoints.Save(Checkpoint{
		CommandID: parseCommand.ID,
		NextIndex: parseNextIndex,
		Total:     parseCommand.Total,
		Exact:     hasExact,
	})
}

// runContainedStep contains a panic in one item's step, for the same reason
// runContained does for an atomic effect: one bad row must not kill the worker,
// and progress up to that row must survive.
func runContainedStep(parseStep func(parseIndex int) error, parseIndex int) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("panic at item %d: %v", parseIndex, parseRecovered)
		}
	}()
	return parseStep(parseIndex)
}
