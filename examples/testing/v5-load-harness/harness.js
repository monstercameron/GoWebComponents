// harness.js — orchestration and gating.
//
// The measurement design, and why it is shaped this way:
//
// M1 asks whether frame time under load is indistinguishable from idle. That
// is a PAIRED comparison, and the naive way to run it — measure idle N times,
// then measure loaded N times — is invalid on this machine. DEVNOTES_PERF_LOOP
// records ±25–30% per-scenario swings on a fanless chassis, and a load run
// heats it. Sequential A-then-B assigns all of that thermal drift to B.
//
// So: interleave. IDLE, LOADED, IDLE, LOADED, ... Drift then lands on both
// arms roughly equally, and detectDrift() proves whether it did.
//
// Everything else follows from that: fixed seeds so bootstrap CIs reproduce,
// warmup discarded, raw samples retained so a run can be re-scored against
// changed budgets without re-running.

import * as stats from './stats.js';
import { detectFrameInterval, startMeasurement, deriveMetrics } from './metrics.js';

/**
 * A workload is anything the subject app can start and stop. Implementations
 * live in the Go subject (main.go) and are exposed on window.__gwcV5Workloads.
 *
 * @typedef {object} Workload
 * @property {string} name
 * @property {() => Promise<void>} start
 * @property {() => Promise<void>} stop
 * @property {() => object} stats  work completed, for the throughput column
 */

/**
 * The interaction probe. This is the ONLY thing measured. Background load is
 * never measured directly — it exists to perturb the probe, and the whole
 * thesis of v5 is that it must fail to.
 *
 * @typedef {object} Probe
 * @property {string} name
 * @property {(durationMs:number) => Promise<void>} run
 */

const DEFAULT_OPTIONS = {
  repetitions: 6,          // per arm; 12 windows total when interleaved
  windowMs: 4000,          // per window
  warmupMs: 3000,          // discarded
  cooldownMs: 1500,        // between windows, lets the machine settle
  seed: 0x5eed,
};

function sleep(durationMs) {
  return new Promise((resolve) => setTimeout(resolve, durationMs));
}

/**
 * Runs one arm: optionally start load, run the probe for windowMs while
 * measuring, then stop load.
 */
async function runWindow({ label, workloads, probe, windowMs, frameIntervalMs }) {
  for (const workload of workloads) await workload.start();

  const session = startMeasurement({ frameIntervalMs, label });
  await probe.run(windowMs);
  const raw = await session.stop();

  for (const workload of workloads) await workload.stop();

  return {
    raw,
    workloadStats: workloads.map((workload) => ({ name: workload.name, ...workload.stats() })),
  };
}

/**
 * @param {object} config
 * @param {Probe} config.probe            the measured interaction
 * @param {Workload[]} config.workloads   background load for the LOADED arm
 */
export async function runComparison(config) {
  const options = { ...DEFAULT_OPTIONS, ...config.options };
  const { probe, workloads } = config;

  // Detect the real refresh rate before anything else; every dropped-frame
  // threshold downstream derives from it.
  const frameIntervalMs = await detectFrameInterval();

  // Warmup: wasm instantiation, JIT tiering, first-paint allocation, and the
  // Go heap reaching steady state all land here and are discarded.
  {
    for (const workload of workloads) await workload.start();
    const warmupSession = startMeasurement({ frameIntervalMs, label: 'warmup' });
    await probe.run(options.warmupMs);
    await warmupSession.stop();
    for (const workload of workloads) await workload.stop();
    await sleep(options.cooldownMs);
  }

  const idleWindows = [];
  const loadedWindows = [];

  // Interleaved. See the header comment — this ordering is load-bearing.
  for (let repetition = 0; repetition < options.repetitions; repetition++) {
    idleWindows.push(
      await runWindow({
        label: `idle-${repetition}`,
        workloads: [],
        probe,
        windowMs: options.windowMs,
        frameIntervalMs,
      }),
    );
    await sleep(options.cooldownMs);

    loadedWindows.push(
      await runWindow({
        label: `loaded-${repetition}`,
        workloads,
        probe,
        windowMs: options.windowMs,
        frameIntervalMs,
      }),
    );
    await sleep(options.cooldownMs);
  }

  return buildReport({ idleWindows, loadedWindows, frameIntervalMs, options, probe, workloads });
}

function poolFrames(windows) {
  return windows.flatMap((window_) => window_.raw.frames);
}

function poolInteractions(windows) {
  return windows.flatMap((window_) => window_.raw.interactions.map((entry) => entry.duration));
}

/**
 * Aggregate how much background work the LOADED arm actually completed.
 *
 * This is the evidence for the whole comparison, and it was being collected per
 * window and then thrown away. M1 asks whether a loaded frame is equivalent to
 * an idle one — and a loaded arm that ran NO load answers "yes" perfectly. A
 * worker that failed to instantiate, a workload that errored on its first call,
 * a message dropped before onmessage existed: each produces a flawless M1 and a
 * meaningless one.
 *
 * Counts are cumulative per workload, and each loaded window restarts them, so
 * the totals sum across windows while the per-window figures stay comparable.
 */
function poolWorkloadThroughput(windows) {
  const byName = new Map();
  for (const window_ of windows) {
    for (const stat of window_.workloadStats ?? []) {
      const entry = byName.get(stat.name) ?? { name: stat.name, completed: 0, windows: 0, errors: [] };
      entry.completed += stat.completed ?? 0;
      entry.windows += 1;
      // The field is `err`, matching what the app's stats() returns and what
      // index.html forwards. Reading a different name here silently reported a
      // failing worker as a clean one, which is precisely the case this exists
      // to catch.
      if (stat.err) entry.errors.push(stat.err);
      byName.set(stat.name, entry);
    }
  }
  return [...byName.values()];
}

// Exported for integration.test.mjs. A test that maintains its own copy of this
// aggregation cannot catch a regression in the shipped one — reverting the
// weakest-observer-wins rule below would leave such a test green while
// runComparison() silently reintroduced the M2 false pass.
export function buildReport({ idleWindows, loadedWindows, frameIntervalMs, options, probe, workloads }) {
  const idleFrames = poolFrames(idleWindows);
  const loadedFrames = poolFrames(loadedWindows);

  // Drift is checked on the IDLE arm only, and that distinction is load-bearing.
  //
  // The idle arm is the control: nothing is running, so any trend in it is the
  // machine changing under us — thermal throttling, a background process — and
  // that invalidates the comparison.
  //
  // The loaded arm legitimately trends. A 50k-row import gets slower as the
  // table grows; frame time rising across a loaded run is the phenomenon being
  // measured, not an artifact. The first real run of this harness failed on
  // exactly that: idle p95 16.8ms against loaded p95 1835ms, flagged "drift"
  // when it was the load working as designed. Gating on the loaded arm would
  // make the harness reject precisely the runs it exists to capture.
  const idleDrift = stats.detectDrift(idleFrames);
  const loadedDrift = stats.detectDrift(loadedFrames); // reported, never gated

  const m1 = stats.equivalent(loadedFrames, idleFrames, {
    statistic: (values) => stats.percentile(values, 0.95),
    marginMs: 1.0,
    marginRatio: 0.1,
    seed: options.seed,
  });

  const loadedMetrics = loadedWindows.map((window_) => deriveMetrics(window_.raw, stats));
  const idleMetrics = idleWindows.map((window_) => deriveMetrics(window_.raw, stats));

  const loadedInteractions = poolInteractions(loadedWindows);
  const longFrameCount = loadedMetrics.reduce((sum, metric) => sum + metric.longFrameCount, 0);
  const maxGCPauseMs = loadedMetrics.reduce(
    (worst, metric) => Math.max(worst, metric.maxGCPauseMs ?? 0),
    0,
  );

  return {
    probe: probe.name,
    workloads: workloads.map((workload) => workload.name),
    frameIntervalMs,
    options,

    valid: !idleDrift.drifted,
    drift: { idle: idleDrift, loaded: loadedDrift, gatedOn: 'idle' },

    metrics: {
      // M1 — the acceptance test for v5.
      m1_frameTimeEquivalence: {
        equivalent: m1.equivalent,
        marginMs: m1.marginMs,
        ci: m1.ci,
        idle: stats.summarize(idleFrames),
        loaded: stats.summarize(loadedFrames),
      },
      // M2 — long frames during background work.
      m2_longFrames: {
        count: longFrameCount,
        worstMs: loadedMetrics.reduce((worst, m) => Math.max(worst, m.worstLongFrameMs), 0),
        totalBlockingMs: loadedMetrics.reduce((sum, m) => sum + m.totalBlockingMs, 0),
        // Weakest observer across ALL windows, not window 0's. Reading only the
        // first window would let a mid-session observer failure go unnoticed and
        // report the stronger instrument — the same shape of bug as trusting a
        // count of zero from a blind observer. 'none' anywhere poisons the run.
        source: loadedMetrics.some((m) => m.longFrameSource === 'none')
          ? 'none'
          : loadedMetrics.some((m) => m.longFrameSource === 'longtask')
            ? 'longtask'
            : (loadedMetrics[0]?.longFrameSource ?? 'none'),
      },
      // M3 — input-to-paint under load.
      m3_interactionLatency: loadedInteractions.length
        ? stats.summarize(loadedInteractions)
        : { n: 0, note: 'no interactions recorded — probe did not generate discrete input' },
      // M7 — worst GC pause on the render thread while loaded.
      m7_maxGCPauseMs: maxGCPauseMs,
      m7_sampleTruncated: loadedMetrics.some((metric) => metric.gcSampleTruncated),
      // How many collections the pause figure above was derived from.
      //
      // Without this, maxGCPauseMs === 0 is ambiguous: it means either "every
      // collection was fast" (a pass) or "no collection happened, so nothing was
      // measured" (not a result). The gate cannot tell those apart from a single
      // number, and the second one reads as a perfect score.
      m7_collectionsObserved: loadedMetrics.reduce(
        (total, metric) => total + (metric.go?.numGC ?? 0), 0),
      // Whether the Go probe answered at all. metrics.js readGoProbe() returns
      // null when window.__gwcV5Probe is missing or unparseable, so a non-null
      // `go` block IS the liveness proof.
      //
      // This separates the two ways to reach zero collections. A live probe that
      // saw none is reporting something true and good: the render thread allocated
      // little enough never to collect, which is precisely what the two-artifact
      // split exists to produce. A missing probe is reporting nothing.
      //
      // Deliberately NOT inferred from per-window commit or render deltas. That
      // was the first attempt and it was wrong for the same reason the frame-clock
      // proxy was: an idle render thread legitimately commits zero times in a
      // window, so a healthy run looked blind. Measured on this harness the render
      // thread collects ONCE at boot (numGC stays at 1 for the whole session), so
      // every per-window delta is zero by design.
      m7_probeLive: loadedMetrics.some((metric) => metric.go != null),
    },

    // Retained so a stored run can be re-scored without re-running.
    raw: { idle: idleMetrics, loaded: loadedMetrics },

    // What the loaded arm actually did.
    //
    // This was previously reported per-window and unaggregated, and nothing —
    // not the gate, not the Go driver — ever read it. That made it decoration
    // rather than evidence, on the one fact every other number depends on: M1
    // asks whether a loaded frame matches an idle one, and a loaded arm that
    // ran NO load answers yes perfectly. The gate now refuses such a run.
    workloadThroughput: poolWorkloadThroughput(loadedWindows),
    workloadThroughputByWindow: loadedWindows.map((window_) => window_.workloadStats),
  };
}

/**
 * Gate a report against budgets.json. Returns pass/fail per metric so CI can
 * block, and so a regression names the metric it broke rather than a score.
 *
 * A run that failed drift validation fails the gate regardless of its numbers:
 * an invalid measurement must never be reported as a pass.
 */
export function gate(report, budgets) {
  const failures = [];

  if (!report.valid) {
    failures.push({
      metric: 'validity',
      reason: 'thermal or background drift exceeded tolerance; measurement discarded',
      detail: report.drift,
    });
  }

  // Before any metric: did the loaded arm carry load at all?
  //
  // Checked FIRST and separately from validity, because the failure it catches
  // is the one that looks most like success. A broken worker makes every metric
  // below pass — M1 equivalent, zero long frames, no GC pauses — and the report
  // would read as the strongest possible result while measuring an idle page
  // twice.
  const throughput = report.workloadThroughput ?? [];
  const totalCompleted = throughput.reduce((sum, entry) => sum + (entry.completed ?? 0), 0);
  const workloadErrors = throughput.flatMap((entry) =>
    (entry.errors ?? []).map((text) => `${entry.name}: ${text}`),
  );
  if (throughput.length === 0 || totalCompleted <= 0) {
    failures.push({
      metric: 'load',
      reason:
        'the loaded arm completed no background work; M1 equivalence is vacuous because there was nothing to be equivalent to',
      detail: { throughput, workloadErrors },
    });
  } else if (workloadErrors.length > 0) {
    failures.push({
      metric: 'load',
      reason: `background workloads reported ${workloadErrors.length} error(s); the load was partial`,
      detail: { throughput, workloadErrors },
    });
  }

  const m1 = report.metrics.m1_frameTimeEquivalence;
  // M1 is only meaningful when the probe actually ran.
  //
  // Equivalence between two arms that both did nothing is trivially true, and
  // that is how M1 was recorded as met: the typing probe never typed, so idle and
  // loaded were both an idle page at vsync. Gating M1 on the interaction count
  // ties it to the same evidence M3 already demanded.
  const interactionSamples = report.metrics.m3_interactionLatency?.n ?? 0;
  if (!interactionSamples) {
    failures.push({
      metric: 'M1',
      reason:
        'no interactions were recorded, so the probe did not exercise the app; comparing two idle arms ' +
        'reports equivalence regardless of the runtime. M1 is UNMEASURED. Drive real (trusted) input — ' +
        'Event Timing ignores synthetic events, so it must come from the automation driver.',
    });
  } else if (!m1.equivalent) {
    failures.push({
      metric: 'M1',
      reason: `loaded p95 frame time is not equivalent to idle (CI upper ${m1.ci.upper?.toFixed(2)}ms > margin ${m1.marginMs?.toFixed(2)}ms)`,
    });
  }

  const m2 = report.metrics.m2_longFrames;
  // A count of zero means nothing if nothing was watching. When no long-frame
  // observer is available the sink stays permanently empty and a naive
  // `count > 0` check passes trivially — "no long frames observed" is not
  // "no long frames occurred". Blind instrument = failed run.
  //
  // The observer EXISTING is not the same as the observer FIRING, and the second
  // failure is the dangerous one because it reports a clean zero.
  //
  // Measured 2026-07-26: a headless Chromium run reported
  // `source: "long-animation-frame", count: 0` — an observer was constructed, so
  // the source check below passed — while six deliberately injected 180 ms
  // main-thread blocks went uncounted. LoAF never delivers entries in that
  // environment, so the counter cannot rise no matter what the page does.
  //
  // The cross-check is the FRAME TIMELINE, collected independently for M1. If the
  // recorded frames contain one longer than the threshold while the long-frame
  // observer reported none, the two instruments disagree and the observer is the
  // one that is wrong. If the longest recorded frame is short, zero long frames is
  // consistent and believed.
  //
  // Deliberately NOT inferred from frame-time variance. An earlier version of this
  // guard treated a near-zero MAD as proof of a synthetic clock, which fails the
  // moment the harness runs headed: an idle arm legitimately produces near-uniform
  // vsync-locked frames, so real passing runs were reported as blind. Disagreement
  // between two instruments is evidence; uniformity is not.
  const LONG_FRAME_THRESHOLD_MS = 50;
  const longestLoadedFrameMs = Number(m1?.loaded?.max ?? 0);
  const observerMissedALongFrame =
    m2.count === 0 && longestLoadedFrameMs > LONG_FRAME_THRESHOLD_MS;

  if (m2.source === 'none') {
    failures.push({
      metric: 'M2',
      reason: 'no long-frame observer available (neither long-animation-frame nor longtask); M2 is unmeasurable in this environment',
    });
  } else if (observerMissedALongFrame) {
    failures.push({
      metric: 'M2',
      reason:
        `reported 0 long frames from source "${m2.source}", but the frame timeline recorded a ` +
        `${longestLoadedFrameMs.toFixed(1)}ms frame (threshold ${LONG_FRAME_THRESHOLD_MS}ms). ` +
        'The two instruments disagree, so the observer is not firing and this zero is UNMEASURED, ' +
        'not achieved. Re-run in a headed browser.',
    });
  } else if (m2.count > budgets.M2_maxLongFrames) {
    failures.push({
      metric: 'M2',
      reason: `${m2.count} long frames (budget ${budgets.M2_maxLongFrames}), worst ${m2.worstMs.toFixed(1)}ms, source ${m2.source}`,
    });
  }

  // M3 gates on the highest percentile the sample can actually support, plus
  // an absolute ceiling.
  //
  // p99 needs n>=1000 (see percentileReliable). Interaction samples come from
  // Event Timing entries for a typing probe over ~24s of measured windows —
  // realistically low hundreds, never 1000. Gating on an unreliable p99 would
  // read a number the harness's own math says is meaningless, so the gate
  // steps down to p95 (n>=200) and reports which statistic it used. `max` is
  // always gated, because an under-sampled percentile cannot see a single
  // catastrophic interaction and that is precisely what M3 exists to catch.
  const m3 = report.metrics.m3_interactionLatency;
  if (!m3.n) {
    failures.push({ metric: 'M3', reason: 'no interactions recorded; probe is not exercising input' });
  } else {
    const usedStatistic = m3.tailReliable?.p99 ? 'p99' : m3.tailReliable?.p95 ? 'p95' : null;
    if (!usedStatistic) {
      failures.push({
        metric: 'M3',
        reason: `only ${m3.n} interactions; too few to support even p95 (need >=200). Lengthen the run or raise probe input rate.`,
      });
    } else if (m3[usedStatistic] > budgets.M3_interactionP99Ms) {
      failures.push({
        metric: 'M3',
        reason: `interaction ${usedStatistic} ${m3[usedStatistic].toFixed(1)}ms > ${budgets.M3_interactionP99Ms}ms (n=${m3.n})`,
      });
    }
    if (budgets.M3_interactionMaxMs != null && m3.max > budgets.M3_interactionMaxMs) {
      failures.push({
        metric: 'M3',
        reason: `worst interaction ${m3.max.toFixed(1)}ms > ${budgets.M3_interactionMaxMs}ms ceiling`,
      });
    }
  }

  // A truncated GC sample means >256 collections landed in one window and the
  // true worst pause may have been evicted. The reported max is then a lower
  // bound, not a maximum — it must never be allowed to pass.
  if (report.metrics.m7_sampleTruncated) {
    failures.push({
      metric: 'M7',
      reason: 'GC pause buffer overflowed within a window (>256 collections); reported max is a lower bound, not a maximum. Shorten windowMs.',
    });
  } else if (
    report.metrics.m7_maxGCPauseMs === 0 &&
    report.metrics.m7_collectionsObserved === 0 &&
    report.metrics.m7_probeLive === false
  ) {
    // A zero pause derived from zero collections BY A PROBE THAT SAW NOTHING is
    // not a measurement.
    //
    // The probe-live term is what makes this correct rather than merely cautious.
    // Zero collections is the SUCCESS condition for M7 — the metric exists to
    // bound "residency re-imports GC pressure onto the render thread", and a
    // render thread that never collects has none. Failing that would reject the
    // outcome the architecture is trying to produce. Only a probe that recorded no
    // commits and no render time is actually blind.
    //
    // BOTH conditions are required, and both are checked for an EXPLICIT zero
    // rather than for falsiness. A non-zero pause proves a collection happened, so
    // the count adds nothing there. And an ABSENT count (an older report, or a
    // hand-built fixture) means "unknown", which must not gate — treating
    // undefined as zero made this fire on a report carrying a perfectly good
    // 1.2ms pause.
    //
    // Checked against the COLLECTION COUNT rather than against the frame clock.
    // An earlier version inferred "unmeasured" from a synthetic frame clock, which
    // false-positived the moment the harness ran headed: an idle arm legitimately
    // produces near-uniform vsync-locked frames, so a valid run was reported as
    // blind. The collection count is direct evidence and needs no proxy.
    failures.push({
      metric: 'M7',
      reason:
        'no GC collections were observed in the loaded arm, so max GC pause is UNMEASURED rather than 0ms. ' +
        'A run that imports, re-indexes and decodes a dataset without collecting once is not a plausible pass — ' +
        'check that the Go probe is reporting MemStats.',
    });
  } else if (report.metrics.m7_maxGCPauseMs > budgets.M7_maxGCPauseMs) {
    failures.push({
      metric: 'M7',
      reason: `max GC pause ${report.metrics.m7_maxGCPauseMs.toFixed(2)}ms > ${budgets.M7_maxGCPauseMs}ms`,
    });
  }

  return { passed: failures.length === 0, failures };
}
