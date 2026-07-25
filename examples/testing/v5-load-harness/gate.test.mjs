// Tests for gate() — the function that decides pass/fail. Run: node gate.test.mjs
//
// This layer had ZERO coverage in the first cut, and all three false-pass bugs
// found in adversarial review lived here rather than in stats.js. The math was
// tested; the decision was not. Every test below pins a way the gate could
// wrongly report success — a benchmark that lies is worse than no benchmark,
// because it converts "we don't know" into "we verified".

import assert from 'node:assert/strict';
import { gate } from './harness.js';
import { summarize } from './stats.js';

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

/** A report that passes everything, as the base for negative tests. */
function healthyReport(overrides = {}) {
  const interactions = Array.from({ length: 400 }, () => 20);
  return {
    valid: true,
    drift: {},
    metrics: {
      m1_frameTimeEquivalence: { equivalent: true, marginMs: 1.0, ci: { upper: 0.3 } },
      m2_longFrames: { count: 0, worstMs: 0, totalBlockingMs: 0, source: 'long-animation-frame' },
      m3_interactionLatency: summarize(interactions),
      m7_maxGCPauseMs: 1.2,
      m7_sampleTruncated: false,
    },
    ...overrides,
  };
}

function failureMetrics(result) {
  return result.failures.map((failure) => failure.metric);
}

test('a healthy report passes', () => {
  const result = gate(healthyReport(), BUDGETS);
  assert.equal(result.passed, true, JSON.stringify(result.failures));
});

// ---------------------------------------------------------------- validity

test('a drifted run fails even when every metric looks good', () => {
  const report = healthyReport({ valid: false, drift: { loaded: { drifted: true, ratio: 0.4 } } });
  const result = gate(report, BUDGETS);
  assert.equal(result.passed, false);
  assert.ok(failureMetrics(result).includes('validity'));
});

// ---------------------------------------------------------------------- M1

test('M1 fails when loaded frames are not equivalent to idle', () => {
  const report = healthyReport();
  report.metrics.m1_frameTimeEquivalence = { equivalent: false, marginMs: 1.0, ci: { upper: 7.4 } };
  assert.ok(failureMetrics(gate(report, BUDGETS)).includes('M1'));
});

// ---------------------------------------------------------------------- M2

test('M2 FAILS when no long-frame observer exists — a blind count of zero is not a pass', () => {
  // The original bug: source==='none' leaves longFrames permanently empty, so
  // `count > budget` is false and M2 passed trivially in any environment
  // lacking both entry types. "None observed" is not "none occurred".
  const report = healthyReport();
  report.metrics.m2_longFrames = { count: 0, worstMs: 0, totalBlockingMs: 0, source: 'none' };
  const result = gate(report, BUDGETS);
  assert.equal(result.passed, false, 'a blind observer must not produce a pass');
  assert.ok(failureMetrics(result).includes('M2'));
});

test('M2 passes on a real zero from a working observer', () => {
  const report = healthyReport();
  report.metrics.m2_longFrames = { count: 0, worstMs: 0, totalBlockingMs: 0, source: 'longtask' };
  assert.equal(gate(report, BUDGETS).passed, true);
});

test('M2 fails on any long frame', () => {
  const report = healthyReport();
  report.metrics.m2_longFrames = { count: 3, worstMs: 210, totalBlockingMs: 480, source: 'long-animation-frame' };
  assert.ok(failureMetrics(gate(report, BUDGETS)).includes('M2'));
});

// ---------------------------------------------------------------------- M3

test('M3 fails when the probe generated no interactions', () => {
  const report = healthyReport();
  report.metrics.m3_interactionLatency = { n: 0 };
  assert.ok(failureMetrics(gate(report, BUDGETS)).includes('M3'));
});

test('M3 steps down to p95 when p99 is not supportable, and says so', () => {
  // 400 interactions supports p95 (n>=200) but not p99 (n>=1000). The gate
  // must use p95 rather than reading an untrustworthy p99 — the original bug
  // gated on p99 unconditionally while summarize() was already reporting that
  // it was unreliable.
  // 25% of interactions at 80ms. Note 5% would NOT breach: at n=400 the p95
  // lands on the boundary rank and interpolates to ~23ms. Percentile gates are
  // less sensitive than they look, which is exactly why max is gated too.
  const report = healthyReport();
  const slow = [...Array(300).fill(20), ...Array(100).fill(80)];
  report.metrics.m3_interactionLatency = summarize(slow);
  assert.equal(report.metrics.m3_interactionLatency.tailReliable.p99, false);
  assert.equal(report.metrics.m3_interactionLatency.tailReliable.p95, true);

  const result = gate(report, BUDGETS);
  const m3Failure = result.failures.find((failure) => failure.metric === 'M3');
  assert.ok(m3Failure, 'p95 of 80ms must breach the 50ms budget');
  assert.match(m3Failure.reason, /p95/, 'the gate must report which statistic it used');
});

test('M3 fails outright when the sample cannot support even p95', () => {
  const report = healthyReport();
  report.metrics.m3_interactionLatency = summarize(Array.from({ length: 40 }, () => 10));
  const result = gate(report, BUDGETS);
  const m3Failure = result.failures.find((failure) => failure.metric === 'M3');
  assert.ok(m3Failure, 'an unjudgeable sample must fail, not silently pass');
  assert.match(m3Failure.reason, /too few/);
});

test('M3 catches one catastrophic interaction that percentiles cannot see', () => {
  // 399 fast interactions and one 900ms stall. Every percentile stays healthy;
  // only the absolute ceiling detects it. This is why max is gated separately.
  const report = healthyReport();
  report.metrics.m3_interactionLatency = summarize([...Array(399).fill(20), 900]);
  assert.ok(report.metrics.m3_interactionLatency.p95 < 50, 'percentiles stay clean by construction');
  const result = gate(report, BUDGETS);
  assert.ok(failureMetrics(result).includes('M3'), 'the max ceiling must catch it');
});

// ---------------------------------------------------------------------- M7

test('M7 FAILS on a truncated GC sample even when the reported max looks fine', () => {
  // The original bug: probe.go computes pauseSampleTruncated, but metrics.js
  // dropped the field, so an overflowed window (>256 collections, with the
  // true worst pause evicted from the circular buffer) reported a clean low
  // max and passed. The reported value is a lower bound, not a maximum.
  const report = healthyReport();
  report.metrics.m7_maxGCPauseMs = 0.4;
  report.metrics.m7_sampleTruncated = true;
  const result = gate(report, BUDGETS);
  assert.equal(result.passed, false, 'a lower bound must never satisfy a maximum budget');
  assert.ok(failureMetrics(result).includes('M7'));
});

test('M7 fails on an over-budget pause', () => {
  const report = healthyReport();
  report.metrics.m7_maxGCPauseMs = 9.1;
  assert.ok(failureMetrics(gate(report, BUDGETS)).includes('M7'));
});

test('gate reports every independent failure, not just the first', () => {
  const report = healthyReport({ valid: false, drift: {} });
  report.metrics.m1_frameTimeEquivalence = { equivalent: false, marginMs: 1, ci: { upper: 9 } };
  report.metrics.m2_longFrames = { count: 5, worstMs: 300, totalBlockingMs: 900, source: 'longtask' };
  report.metrics.m7_maxGCPauseMs = 12;
  const metrics = failureMetrics(gate(report, BUDGETS));
  for (const expected of ['validity', 'M1', 'M2', 'M7']) {
    assert.ok(metrics.includes(expected), `expected ${expected} in ${metrics.join(',')}`);
  }
});

console.log(`\n${passed} passed`);
