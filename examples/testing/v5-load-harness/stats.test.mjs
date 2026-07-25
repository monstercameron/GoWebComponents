// Tests for stats.js. Run: node stats.test.mjs
//
// stats.js is where a wrong result would be least obvious — a percentile off
// by one rank or an inverted equivalence check silently changes every verdict
// the harness produces. These pin the behaviors the gate depends on.

import assert from 'node:assert/strict';
import {
  percentile,
  summarize,
  makeRng,
  bootstrapDifferenceCI,
  equivalent,
  detectDrift,
  splitClusters,
} from './stats.js';

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

const near = (actual, expected, tolerance = 1e-9) =>
  assert.ok(Math.abs(actual - expected) <= tolerance, `expected ~${expected}, got ${actual}`);

// ---------------------------------------------------------------- percentile

test('percentile interpolates between ranks (R-7)', () => {
  const values = [1, 2, 3, 4];
  near(percentile(values, 0), 1);
  near(percentile(values, 1), 4);
  near(percentile(values, 0.5), 2.5);
});

test('percentile handles single and empty samples', () => {
  near(percentile([7], 0.99), 7);
  assert.ok(Number.isNaN(percentile([], 0.5)));
});

test('percentile does not mutate its input', () => {
  const values = [5, 1, 3];
  percentile(values, 0.5);
  assert.deepEqual(values, [5, 1, 3]);
});

// ----------------------------------------------------------------- summarize

test('summarize reports order statistics, not just mean', () => {
  const frames = [...Array(99).fill(16), 400];
  const summary = summarize(frames);
  assert.ok(summary.mean < 21, 'mean hides the jank frame');
  near(summary.max, 400);
  near(summary.mad, 0, 1e-9); // MAD is immune to the single outlier
});

test('p99 CANNOT see a lone jank frame at n=100 — max and long-frame counts must', () => {
  // Pins the reason `tailReliable` exists. Interpolated p99 at n=100 blends
  // ranks 98 and 99 at 99:1, so one 400ms frame moves p99 by ~4ms. Anyone
  // "simplifying" the gate down to p99 alone would silently stop detecting
  // single catastrophic frames — the exact event v5 targets.
  const frames = [...Array(99).fill(16), 400];
  const summary = summarize(frames);
  assert.ok(summary.p99 < 25, `p99 is blind here by construction, got ${summary.p99}`);
  assert.equal(summary.tailReliable.p99, false, 'and the summary must say so');
  near(summary.max, 400, 1e-9);
});

test('tailReliable turns true once the sample supports the quantile', () => {
  const many = Array.from({ length: 1200 }, () => 16);
  const summary = summarize(many);
  assert.equal(summary.tailReliable.p95, true);
  assert.equal(summary.tailReliable.p99, true);

  // 6 reps x 4000ms windows at 60Hz ~= 1440 frames/arm, which is why the
  // harness defaults clear the p99 bar. Shorten the run and they stop doing so.
  const oneWindow = Array.from({ length: 240 }, () => 16);
  assert.equal(summarize(oneWindow).tailReliable.p95, true);
  assert.equal(summarize(oneWindow).tailReliable.p99, false, 'one window cannot support p99');
});

// ------------------------------------------------------------------ bootstrap

test('bootstrap is deterministic for a fixed seed', () => {
  const a = Array.from({ length: 200 }, (_, i) => 16 + (i % 5));
  const b = Array.from({ length: 200 }, (_, i) => 16 + (i % 5));
  const first = bootstrapDifferenceCI(a, b, { seed: 42, iterations: 500 });
  const second = bootstrapDifferenceCI(a, b, { seed: 42, iterations: 500 });
  assert.deepEqual(first, second, 'same seed must reproduce the CI exactly');
});

test('bootstrap refuses tiny samples instead of inventing a CI', () => {
  const result = bootstrapDifferenceCI([1, 2, 3], [1, 2, 3]);
  assert.equal(result.insufficient, true);
});

test('makeRng is deterministic and in range', () => {
  const rng = makeRng(1);
  const draws = Array.from({ length: 100 }, () => rng());
  assert.ok(draws.every((v) => v >= 0 && v < 1));
  const rngAgain = makeRng(1);
  assert.deepEqual(draws, Array.from({ length: 100 }, () => rngAgain()));
});

// ---------------------------------------------------------------- equivalence

test('equivalence passes when loaded matches idle', () => {
  const idle = Array.from({ length: 300 }, (_, i) => 16 + (i % 3) * 0.2);
  const loaded = Array.from({ length: 300 }, (_, i) => 16 + (i % 3) * 0.2);
  const result = equivalent(loaded, idle, { marginMs: 1.0, marginRatio: 0.1 });
  assert.equal(result.equivalent, true, result.reason);
});

test('equivalence FAILS when loaded is materially slower', () => {
  // This is the case v5 exists to prevent: background work degrading frames.
  const idle = Array.from({ length: 300 }, () => 16);
  const loaded = Array.from({ length: 300 }, (_, i) => (i % 4 === 0 ? 60 : 16));
  const result = equivalent(loaded, idle, { marginMs: 1.0, marginRatio: 0.1 });
  assert.equal(result.equivalent, false, 'a 60ms every 4th frame must not pass');
});

test('equivalence does not penalize loaded being faster', () => {
  const idle = Array.from({ length: 300 }, () => 20);
  const loaded = Array.from({ length: 300 }, () => 16);
  const result = equivalent(loaded, idle, { marginMs: 1.0, marginRatio: 0.1 });
  assert.equal(result.equivalent, true, 'faster-under-load is not a failure');
});

test('equivalence margin scales with a slow baseline', () => {
  // idle p95 = 100ms, so marginRatio 0.1 gives a 10ms margin, not 1ms.
  const idle = Array.from({ length: 300 }, () => 100);
  const loaded = Array.from({ length: 300 }, () => 105);
  const result = equivalent(loaded, idle, { marginMs: 1.0, marginRatio: 0.1 });
  near(result.marginMs, 10);
  assert.equal(result.equivalent, true);
});

// --------------------------------------------------------------------- drift

test('drift detects a warming machine', () => {
  const cool = Array.from({ length: 100 }, () => 16);
  const hot = Array.from({ length: 100 }, () => 24); // +50%
  const result = detectDrift([...cool, ...hot]);
  assert.equal(result.drifted, true);
  assert.ok(result.ratio > 0.12);
});

test('drift accepts a stable run', () => {
  const stable = Array.from({ length: 200 }, (_, i) => 16 + (i % 2) * 0.1);
  assert.equal(detectDrift(stable).drifted, false);
});

test('drift refuses to judge a tiny sample', () => {
  assert.equal(detectDrift([16, 17, 16]).reason, 'insufficient-samples');
});

// ------------------------------------------------------------------ clusters

test('splitClusters finds a clean/jank split', () => {
  const frames = [...Array(60).fill(16), ...Array(40).fill(90)];
  const result = splitClusters(frames);
  assert.equal(result.bimodal, true);
  near(result.lowerMedian, 16);
  near(result.upperMedian, 90);
});

test('splitClusters does not invent a split in unimodal data', () => {
  const frames = Array.from({ length: 100 }, (_, i) => 16 + (i % 4) * 0.25);
  assert.equal(splitClusters(frames).bimodal, false);
});

test('splitClusters ignores a lone outlier', () => {
  const frames = [...Array(99).fill(16), 400];
  assert.equal(splitClusters(frames).bimodal, false, 'one spike is not a cluster');
});

console.log(`\n${passed} passed`);
