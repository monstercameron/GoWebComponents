// stats.js — statistics for responsiveness benchmarking.
//
// Frame-time and input-latency samples are heavily right-skewed and often
// bimodal (a clean-frame cluster plus a jank cluster). Mean and standard
// deviation are the wrong summary for both. Everything here is either
// order-statistic based or resampling based, and nothing assumes normality.
//
// Three jobs:
//   1. summarize    — percentile-first description of a sample
//   2. equivalence  — prove two samples are the SAME (M1), not merely
//                     "not proven different"
//   3. drift        — detect thermal/background drift that invalidates a run
//
// Pure functions, no DOM, no globals. Unit-testable in node.

// ---------------------------------------------------------------- percentiles

/**
 * Percentile by linear interpolation between closest ranks (the "R-7" /
 * Excel PERCENTILE.INC definition). Chosen over nearest-rank because frame
 * samples are small enough (hundreds) that rank quantization visibly moves
 * p99, and because it matches what Chrome DevTools reports.
 *
 * @param {number[]} sortedValues ascending, non-empty
 * @param {number} fraction 0..1
 */
export function percentileSorted(sortedValues, fraction) {
  const n = sortedValues.length;
  if (n === 0) return NaN;
  if (n === 1) return sortedValues[0];
  if (fraction <= 0) return sortedValues[0];
  if (fraction >= 1) return sortedValues[n - 1];

  const position = (n - 1) * fraction;
  const lowerIndex = Math.floor(position);
  const upperIndex = Math.ceil(position);
  if (lowerIndex === upperIndex) return sortedValues[lowerIndex];

  const weight = position - lowerIndex;
  return sortedValues[lowerIndex] * (1 - weight) + sortedValues[upperIndex] * weight;
}

export function percentile(values, fraction) {
  return percentileSorted([...values].sort((a, b) => a - b), fraction);
}

/**
 * Minimum sample size for a percentile estimate to mean anything.
 *
 * With interpolated percentiles, a tail estimate is a weighted blend of the
 * two ranks straddling it. At n=100 the p99 sits between ranks 98 and 99
 * weighted 99:1 — so ONE 400ms jank frame among 99 clean ones yields a p99 of
 * 19.8ms, not 400ms. The metric silently hides the exact event this harness
 * exists to catch.
 *
 * Rule: keep at least ~10 observations beyond the quantile, i.e.
 * n * (1 - q) >= 10. p95 needs 200 samples; p99 needs 1000.
 *
 * The harness defaults (6 repetitions x 4000ms windows ~= 1440 frames/arm at
 * 60Hz) are sized to satisfy p99. If you shorten the run, `tailReliable`
 * starts reporting false and the report must not quote p99.
 */
export function percentileReliable(n, fraction) {
  return n * (1 - fraction) >= 10;
}

/**
 * Percentile-first summary. `mean` is included only for reporting continuity
 * with the 201 harness — never gate on it.
 *
 * `tailReliable` says whether p95/p99 are meaningful at this sample size.
 * When false, gate on `max` and long-frame counts instead: those detect a
 * single bad frame, which is precisely what an under-sampled percentile cannot.
 */
export function summarize(values) {
  const n = values.length;
  if (n === 0) {
    return {
      n: 0, min: NaN, p50: NaN, p75: NaN, p95: NaN, p99: NaN, max: NaN, mean: NaN, mad: NaN,
      tailReliable: { p95: false, p99: false },
    };
  }
  const sorted = [...values].sort((a, b) => a - b);
  const p50 = percentileSorted(sorted, 0.5);
  // Median absolute deviation: a spread measure that a single 400ms jank
  // frame cannot inflate, unlike stddev.
  const deviations = sorted.map((v) => Math.abs(v - p50)).sort((a, b) => a - b);

  return {
    n,
    min: sorted[0],
    p50,
    p75: percentileSorted(sorted, 0.75),
    p95: percentileSorted(sorted, 0.95),
    p99: percentileSorted(sorted, 0.99),
    max: sorted[n - 1],
    mean: sorted.reduce((sum, v) => sum + v, 0) / n,
    mad: percentileSorted(deviations, 0.5),
    tailReliable: {
      p95: percentileReliable(n, 0.95),
      p99: percentileReliable(n, 0.99),
    },
  };
}

// ---------------------------------------------------------------- resampling

/**
 * Deterministic PRNG (mulberry32). Bootstrap results must be reproducible
 * across runs or a CI gate becomes flaky for reasons unrelated to the code
 * under test.
 */
export function makeRng(seed) {
  let state = seed >>> 0;
  return function next() {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function bootstrapStatistic(sample, rng, statistic) {
  const n = sample.length;
  const resampled = new Array(n);
  for (let i = 0; i < n; i++) resampled[i] = sample[(rng() * n) | 0];
  return statistic(resampled);
}

/**
 * Bootstrap confidence interval for `statisticA - statisticB`.
 *
 * Percentile bootstrap: no distributional assumption, works on p95 and other
 * order statistics where a t-test is invalid.
 *
 * @returns {{lower:number, upper:number, point:number}} in the sample's units
 */
export function bootstrapDifferenceCI(sampleA, sampleB, options = {}) {
  const {
    statistic = (values) => percentile(values, 0.95),
    iterations = 4000,
    confidence = 0.95,
    seed = 0x5eed,
  } = options;

  if (sampleA.length < 8 || sampleB.length < 8) {
    return { lower: NaN, upper: NaN, point: NaN, insufficient: true };
  }

  const rng = makeRng(seed);
  const differences = new Array(iterations);
  for (let i = 0; i < iterations; i++) {
    differences[i] = bootstrapStatistic(sampleA, rng, statistic) - bootstrapStatistic(sampleB, rng, statistic);
  }
  differences.sort((a, b) => a - b);

  const alpha = (1 - confidence) / 2;
  return {
    lower: percentileSorted(differences, alpha),
    upper: percentileSorted(differences, 1 - alpha),
    point: statistic(sampleA) - statistic(sampleB),
    insufficient: false,
  };
}

// --------------------------------------------------------------- equivalence

/**
 * Equivalence test for M1 ("frame time under load is statistically
 * indistinguishable from idle").
 *
 * This is the part benchmarks usually get wrong. A non-significant difference
 * test does NOT show equivalence — it shows you failed to detect one, which a
 * noisy harness guarantees. Equivalence requires the opposite framing: the
 * whole confidence interval of the difference must fall inside a margin you
 * declared in advance.
 *
 * @param {number[]} loadedSample  frame times while background load runs
 * @param {number[]} idleSample    frame times with no background load
 * @param {object} options
 * @param {number} options.marginMs absolute equivalence margin (e.g. 1.0ms)
 * @param {number} options.marginRatio relative margin as a fraction of the
 *        idle statistic (e.g. 0.10). The effective margin is the LARGER of the
 *        two, so a fast baseline does not make the gate impossibly tight.
 * @returns {{equivalent:boolean, ci:object, marginMs:number, reason:string}}
 */
export function equivalent(loadedSample, idleSample, options = {}) {
  const { marginMs = 1.0, marginRatio = 0.1, statistic = (v) => percentile(v, 0.95), seed = 0x5eed } = options;

  const ci = bootstrapDifferenceCI(loadedSample, idleSample, { statistic, seed });
  if (ci.insufficient) {
    return { equivalent: false, ci, marginMs: NaN, reason: 'insufficient-samples' };
  }

  const idleStatistic = statistic(idleSample);
  const effectiveMargin = Math.max(marginMs, idleStatistic * marginRatio);

  // One-sided in practice: loaded being FASTER than idle is not a failure,
  // so only the upper bound is gated. The lower bound is reported for audit
  // because a large negative usually means the idle run was contaminated.
  const withinMargin = ci.upper <= effectiveMargin;

  return {
    equivalent: withinMargin,
    ci,
    marginMs: effectiveMargin,
    reason: withinMargin ? 'ok' : 'upper-bound-exceeds-margin',
  };
}

// -------------------------------------------------------------------- drift

/**
 * Thermal / background drift detection.
 *
 * This machine is fanless and DEVNOTES_PERF_LOOP records ±25–30% per-scenario
 * swings. A load harness runs far longer than a scenario harness, so drift is
 * the default failure mode, not an edge case. Any run whose second half is
 * materially slower than its first half is not measuring the code.
 *
 * Mitigation is structural (interleave A/B — see harness.js); this is the
 * detector that proves the mitigation worked.
 *
 * @param {number[]} orderedValues samples in COLLECTION order (not sorted)
 * @param {number} toleranceRatio allowed median drift, default 12%
 */
export function detectDrift(orderedValues, toleranceRatio = 0.12) {
  const n = orderedValues.length;
  if (n < 16) return { drifted: false, reason: 'insufficient-samples', ratio: NaN };

  const midpoint = Math.floor(n / 2);
  const firstMedian = percentile(orderedValues.slice(0, midpoint), 0.5);
  const secondMedian = percentile(orderedValues.slice(midpoint), 0.5);
  if (!(firstMedian > 0)) return { drifted: false, reason: 'degenerate-baseline', ratio: NaN };

  const ratio = (secondMedian - firstMedian) / firstMedian;
  return {
    drifted: Math.abs(ratio) > toleranceRatio,
    ratio,
    firstMedian,
    secondMedian,
    reason: Math.abs(ratio) > toleranceRatio ? 'median-drift-exceeds-tolerance' : 'ok',
  };
}

// ---------------------------------------------------------------- bimodality

/**
 * Two-cluster split, carried over from the 201 harness's bimodality handling.
 *
 * Frame data legitimately splits into "clean frame" and "janked frame"
 * clusters. Reporting one central value across both hides exactly the
 * behavior v5 exists to fix, so the split is surfaced rather than smoothed.
 *
 * Largest-gap split on sorted values: simple, deterministic, and adequate for
 * the strongly separated clusters this harness produces.
 */
export function splitClusters(values, minSeparationRatio = 2.0) {
  const sorted = [...values].sort((a, b) => a - b);
  if (sorted.length < 8) return { bimodal: false, clusters: [sorted] };

  let widestGap = 0;
  let splitIndex = -1;
  // Ignore the extreme tails: a lone outlier is not a cluster.
  const lowerBound = Math.max(1, Math.floor(sorted.length * 0.1));
  const upperBound = Math.min(sorted.length - 1, Math.ceil(sorted.length * 0.9));
  for (let i = lowerBound; i < upperBound; i++) {
    const gap = sorted[i] - sorted[i - 1];
    if (gap > widestGap) {
      widestGap = gap;
      splitIndex = i;
    }
  }
  if (splitIndex < 0) return { bimodal: false, clusters: [sorted] };

  const lower = sorted.slice(0, splitIndex);
  const upper = sorted.slice(splitIndex);
  const lowerMedian = percentileSorted(lower, 0.5);
  const upperMedian = percentileSorted(upper, 0.5);

  const separated = lowerMedian > 0 && upperMedian / lowerMedian >= minSeparationRatio;
  return separated
    ? { bimodal: true, clusters: [lower, upper], lowerMedian, upperMedian, gap: widestGap }
    : { bimodal: false, clusters: [sorted] };
}
