// integration.test.mjs — the seam tests. Run: node integration.test.mjs
//
// gate.test.mjs hand-builds report objects, so it verifies the DECISION but not
// the PLUMBING. Every false-pass bug found in review lived in the plumbing:
// probe.go emitted a field, metrics.js dropped it, harness.js never saw it.
//
// These drive the ACTUAL shipped functions — the real `diffGoProbe`, the real
// `buildReport`, the real `gate` — starting from raw probe JSON and raw window
// data. Nothing here re-implements production logic.
//
// That distinction is load-bearing and was itself a review finding: an earlier
// cut of this file maintained its own copy of buildReport's aggregation, which
// meant reverting a production fix left these tests green. If you are tempted
// to copy logic in here to make a test easier, that is the bug.

import assert from 'node:assert/strict';
import { diffGoProbe } from './metrics.js';
import { buildReport, gate } from './harness.js';

let passed = 0;
function test(name, fn) {
  try {
    fn();
    passed++;
    console.log(`  ok  ${name}`);
  } catch (error) {
    console.error(`FAIL  ${name}\n      ${error.message}`);
    process.exitCode = 1;
  }
}

const BUDGETS = {
  M2_maxLongFrames: 0,
  M3_interactionP99Ms: 50,
  M3_interactionMaxMs: 120,
  M7_maxGCPauseMs: 3.0,
};

/**
 * Raw probe payloads in exactly the shape probe.go emits, fed through the REAL
 * diffGoProbe so a dropped field fails a test instead of shipping.
 */
function goProbePair({ maxGCPauseNs = 1_200_000, pauseSampleTruncated = false } = {}) {
  const before = {
    renderNs: 0, diffNs: 0, commitNs: 0, effectNs: 0, cleanupNs: 0,
    processedUnits: 0, commitCount: 0, workLoopPasses: 0,
    numGC: 0, pauseTotalNs: 0, windowMaxPauseNs: 0, pauseSampleTruncated: false,
    heapAllocBytes: 5e6, nextGCBytes: 1e7,
  };
  const after = {
    renderNs: 1e6, diffNs: 1e6, commitNs: 1e6, effectNs: 1e6, cleanupNs: 0,
    processedUnits: 100, commitCount: 10, workLoopPasses: 10,
    numGC: 3, pauseTotalNs: maxGCPauseNs, windowMaxPauseNs: maxGCPauseNs,
    pauseSampleTruncated,
    heapAllocBytes: 1e7, nextGCBytes: 2e7,
  };
  return diffGoProbe(before, after); // <- the real function
}

/** A raw window in exactly the shape startMeasurement().stop() returns. */
function rawWindow({
  label = 'loaded-0',
  frameCount = 300,
  frameMs = 16,
  frames = null,
  longFrames = [],
  interactionCount = 400,
  interactionMs = 20,
  longFrameSource = 'long-animation-frame',
  gc = {},
} = {}) {
  const frameSeries = frames ?? Array.from({ length: frameCount }, () => frameMs);
  return {
    raw: {
      label,
      durationMs: frameSeries.length * frameMs,
      frameIntervalMs: 1000 / 60,
      frames: frameSeries,
      droppedFrames: frameSeries.filter((d) => d > (1000 / 60) * 1.5).length,
      longFrames,
      longFrameSource,
      interactions: Array.from({ length: interactionCount }, (_, i) => ({
        name: 'keydown',
        startTime: i * 50,
        duration: interactionMs,
        inputDelay: 1,
        processingTime: 2,
        presentationDelay: interactionMs - 3,
        interactionId: i + 1,
      })),
      interactionsSupported: true,
      go: goProbePair(gc),
    },
    // Non-empty so the load gate is satisfied: these fixtures exist to exercise
    // drift, M2, and M7, and a window that reports no work would fail on the
    // load gate before reaching the metric under test.
    workloadStats: [{ name: 'import', completed: 8000, errText: '' }],
  };
}

/** Calls the REAL buildReport with the argument shape runComparison uses. */
function report(loadedWindows, idleWindows) {
  return buildReport({
    idleWindows,
    loadedWindows,
    frameIntervalMs: 1000 / 60,
    options: { repetitions: loadedWindows.length, windowMs: 4000, seed: 0x5eed },
    probe: { name: 'typing' },
    workloads: [{ name: 'import' }],
  });
}

// ------------------------------------------------------------------- healthy

test('a clean run passes end to end through the real pipeline', () => {
  const result = gate(report([rawWindow()], [rawWindow({ label: 'idle-0' })]), BUDGETS);
  assert.equal(result.passed, true, JSON.stringify(result.failures));
});

// ------------------------------------------------- diffGoProbe (real) seam

test('SEAM: the real diffGoProbe carries pauseSampleTruncated through to a gate failure', () => {
  // The original bug: probe.go computed this, diffGoProbe dropped it, and an
  // overflowed GC window passed M7 with a low reported max.
  const built = report(
    [rawWindow({ gc: { maxGCPauseNs: 300_000, pauseSampleTruncated: true } })],
    [rawWindow({ label: 'idle-0' })],
  );
  assert.equal(built.metrics.m7_sampleTruncated, true, 'flag must survive diffGoProbe -> buildReport');
  const result = gate(built, BUDGETS);
  assert.equal(result.passed, false, 'a truncated sample must never pass M7');
  assert.ok(result.failures.some((f) => f.metric === 'M7'));
});

test('SEAM: diffGoProbe returns null when either probe sample is missing', () => {
  assert.equal(diffGoProbe(null, { numGC: 1 }), null);
  assert.equal(diffGoProbe({ numGC: 1 }, null), null);
});

test('SEAM: an over-budget GC pause propagates from raw probe JSON to an M7 failure', () => {
  const built = report(
    [rawWindow({ gc: { maxGCPauseNs: 9_100_000 } })],
    [rawWindow({ label: 'idle-0' })],
  );
  assert.ok(Math.abs(built.metrics.m7_maxGCPauseMs - 9.1) < 1e-6);
  assert.ok(gate(built, BUDGETS).failures.some((f) => f.metric === 'M7'));
});

// -------------------------------------------------- buildReport (real) seam

test('SEAM: a blind observer in ANY window poisons the run, not just window 0', () => {
  // buildReport originally read loadedMetrics[0].longFrameSource, so an
  // observer that died after window 0 reported the stronger instrument.
  const built = report(
    [rawWindow({ label: 'loaded-0' }), rawWindow({ label: 'loaded-1', longFrameSource: 'none' })],
    [rawWindow({ label: 'idle-0' })],
  );
  assert.equal(built.metrics.m2_longFrames.source, 'none', 'weakest observer must win');
  assert.equal(gate(built, BUDGETS).passed, false);
});

test('SEAM: a degraded-but-working observer reports longtask, not LoAF', () => {
  const built = report(
    [rawWindow({ label: 'loaded-0' }), rawWindow({ label: 'loaded-1', longFrameSource: 'longtask' })],
    [rawWindow({ label: 'idle-0' })],
  );
  assert.equal(built.metrics.m2_longFrames.source, 'longtask');
  assert.equal(gate(built, BUDGETS).passed, true, 'longtask is weaker but still valid');
});

test('SEAM: long frames aggregate across windows into an M2 failure', () => {
  const built = report(
    [
      rawWindow({ label: 'loaded-0', longFrames: [{ startTime: 10, duration: 180, blockingDuration: 130, scripts: [] }] }),
      rawWindow({ label: 'loaded-1', longFrames: [{ startTime: 20, duration: 90, blockingDuration: 40, scripts: [] }] }),
    ],
    [rawWindow({ label: 'idle-0' })],
  );
  assert.equal(built.metrics.m2_longFrames.count, 2, 'counts must sum across windows');
  assert.equal(built.metrics.m2_longFrames.worstMs, 180, 'worst must be the max, not the last');
  assert.ok(gate(built, BUDGETS).failures.some((f) => f.metric === 'M2'));
});

test('SEAM: degraded frames under load fail M1 through the real equivalence test', () => {
  const janky = Array.from({ length: 300 }, (_, i) => (i % 4 === 0 ? 60 : 16));
  const built = report(
    [rawWindow({ frames: janky })],
    [rawWindow({ label: 'idle-0' })],
  );
  assert.equal(built.metrics.m1_frameTimeEquivalence.equivalent, false);
  assert.ok(gate(built, BUDGETS).failures.some((f) => f.metric === 'M1'));
});

test('SEAM: interaction latency reaches the M3 statistic step-down', () => {
  const built = report(
    [rawWindow({ interactionCount: 400, interactionMs: 90 })],
    [rawWindow({ label: 'idle-0' })],
  );
  assert.equal(built.metrics.m3_interactionLatency.tailReliable.p95, true);
  assert.equal(built.metrics.m3_interactionLatency.tailReliable.p99, false);
  const m3Failure = gate(built, BUDGETS).failures.find((f) => f.metric === 'M3');
  assert.ok(m3Failure, '90ms interactions must breach the 50ms budget');
  assert.match(m3Failure.reason, /p95/);
});

test('SEAM: buildReport marks a drifted run invalid and gate fails it', () => {
  // Second half 50% slower than the first — a warming machine.
  const drifting = [...Array(150).fill(16), ...Array(150).fill(24)];
  const built = report([rawWindow({ frames: drifting })], [rawWindow({ label: 'idle-0' })]);
  assert.equal(built.valid, false, 'buildReport must flag drift');
  assert.ok(gate(built, BUDGETS).failures.some((f) => f.metric === 'validity'));
});

console.log(`\n${passed} passed`);
