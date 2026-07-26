// metrics.js — the instrument.
//
// The 201 harness answers "how long did this scenario take". This one answers
// a different question: "what did the frame timeline look like WHILE the app
// was busy". Those need different instruments, which is why this is a separate
// collector rather than more fields on the existing one.
//
// Collected per measurement window:
//   frames       rAF interval distribution        → M1
//   longFrames   Long Animation Frames (LoAF)     → M2, with script attribution
//   interactions Event Timing entries             → M3 (this is INP)
//   go           phase totals + max GC pause      → M7 and phase attribution
//
// Everything is a DISTRIBUTION, never a single number. A mean frame time of
// 12ms with a p99 of 180ms is a broken app that reports as healthy.

/**
 * Detects the display's actual frame interval instead of assuming 16.7ms.
 * A 120Hz panel makes a hardcoded 60Hz threshold under-report dropped frames
 * by 2x, and this machine's external displays vary.
 */
export async function detectFrameInterval(sampleCount = 60) {
  const deltas = [];
  let previousTimestamp = await nextFrame();
  for (let i = 0; i < sampleCount; i++) {
    const timestamp = await nextFrame();
    deltas.push(timestamp - previousTimestamp);
    previousTimestamp = timestamp;
  }
  deltas.sort((a, b) => a - b);
  // Median, not mean: startup frames are irregular and would drag a mean up.
  const median = deltas[Math.floor(deltas.length / 2)];
  // Snap to a known refresh rate when close, so tiny jitter does not produce
  // an odd budget like 16.94ms in the report.
  const known = [1000 / 60, 1000 / 90, 1000 / 120, 1000 / 144];
  const snapped = known.find((candidate) => Math.abs(candidate - median) < 1.2);
  return snapped ?? median;
}

function nextFrame() {
  return new Promise((resolve) => requestAnimationFrame(resolve));
}

/**
 * Long Animation Frames is strictly better than the longtask API for this
 * job: it reports the whole animation frame (not just one task), attributes
 * blocking time to specific scripts, and separates style/layout cost. It is
 * Chrome 123+. The longtask fallback keeps the harness usable elsewhere, but
 * loses attribution — the report records which one was used so a run measured
 * with the weaker instrument is never silently compared against a stronger one.
 */
function createLongFrameObserver(sink, since = 0) {
  if (typeof PerformanceObserver === 'undefined') {
    return { source: 'none', disconnect() {} };
  }
  const supported = PerformanceObserver.supportedEntryTypes ?? [];

  if (supported.includes('long-animation-frame')) {
    const observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        // buffered:true replays everything recorded since PAGE LOAD into this
        // freshly created observer. Without this filter each window counted the
        // entire session's history again: the per-window long-frame counts came
        // out monotonically increasing (2, 15, 26, 30, 33, 35) with an identical
        // worst frame in every window, and M2 summed those cumulative snapshots
        // across twelve windows. buffered:true is still wanted — it catches
        // entries between the window opening and this observer existing — so the
        // fix is to drop the history, not the buffering.
        if (entry.startTime < since) continue;
        sink.push({
          startTime: entry.startTime,
          duration: entry.duration,
          blockingDuration: entry.blockingDuration ?? 0,
          styleAndLayoutDuration: entry.styleAndLayoutDuration ?? 0,
          renderStart: entry.renderStart ?? 0,
          scripts: (entry.scripts ?? []).map((script) => ({
            name: script.name,
            duration: script.duration,
            invoker: script.invoker,
            // "forced style/layout inside script" — the classic cause of a
            // long frame that phase totals alone will not explain.
            forcedStyleAndLayoutDuration: script.forcedStyleAndLayoutDuration ?? 0,
          })),
        });
      }
    });
    observer.observe({ type: 'long-animation-frame', buffered: true });
    return { source: 'long-animation-frame', disconnect: () => observer.disconnect() };
  }

  if (supported.includes('longtask')) {
    const observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.startTime < since) continue;
        sink.push({
          startTime: entry.startTime,
          duration: entry.duration,
          blockingDuration: Math.max(0, entry.duration - 50),
          styleAndLayoutDuration: 0,
          renderStart: 0,
          scripts: [],
        });
      }
    });
    observer.observe({ type: 'longtask', buffered: true });
    return { source: 'longtask', disconnect: () => observer.disconnect() };
  }

  return { source: 'none', disconnect() {} };
}

/**
 * Event Timing. `duration` here is input-to-next-paint — exactly M3, and the
 * same quantity Core Web Vitals calls INP.
 *
 * Caveat recorded in the report: Chrome rounds `duration` to 8ms for privacy,
 * so this metric has 8ms granularity. Adequate for a 50ms gate, useless for
 * chasing a 2ms improvement — use the Go phase totals for that.
 */
function createInteractionObserver(sink, since = 0) {
  if (typeof PerformanceObserver === 'undefined') return { supported: false, disconnect() {} };
  const supported = PerformanceObserver.supportedEntryTypes ?? [];
  if (!supported.includes('event')) return { supported: false, disconnect() {} };

  const observer = new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) {
      // interactionId === 0 means "not a discrete user interaction"
      // (e.g. a programmatic or continuous event). Those are excluded so
      // scroll noise cannot dilute the interaction percentile.
      if (!entry.interactionId) continue;
      // Same history replay as the long-frame observer, and the same
      // consequence: every window re-measured every interaction since page
      // load, so the M3 percentile described the whole session rather than the
      // window, and drifted upward as the run went on.
      if (entry.startTime < since) continue;
      sink.push({
        name: entry.name,
        startTime: entry.startTime,
        duration: entry.duration,
        inputDelay: entry.processingStart - entry.startTime,
        processingTime: entry.processingEnd - entry.processingStart,
        presentationDelay: entry.startTime + entry.duration - entry.processingEnd,
        interactionId: entry.interactionId,
      });
    }
  });
  observer.observe({ type: 'event', buffered: true, durationThreshold: 16 });
  return { supported: true, disconnect: () => observer.disconnect() };
}

/** Reads the Go-side probe installed by probe.go. */
function readGoProbe() {
  const probe = globalThis.__gwcV5Probe;
  if (typeof probe !== 'function') return null;
  try {
    return JSON.parse(probe());
  } catch {
    return null;
  }
}

// Exported for integration.test.mjs. This is the function the original
// pauseSampleTruncated-drop bug lived in; a test that fabricates its output
// instead of calling it cannot catch that bug class recurring.
export function diffGoProbe(before, after) {
  if (!before || !after) return null;
  return {
    renderNs: after.renderNs - before.renderNs,
    diffNs: after.diffNs - before.diffNs,
    commitNs: after.commitNs - before.commitNs,
    effectNs: after.effectNs - before.effectNs,
    cleanupNs: after.cleanupNs - before.cleanupNs,
    commitCount: after.commitCount - before.commitCount,
    workLoopPasses: after.workLoopPasses - before.workLoopPasses,
    processedUnits: after.processedUnits - before.processedUnits,
    numGC: after.numGC - before.numGC,
    // Window max, not cumulative total — a single 9ms pause is the thing that
    // drops a frame, and PauseTotalNs cannot show it. See probe.go.
    maxGCPauseNs: after.windowMaxPauseNs ?? 0,
    // MUST be propagated. probe.go sets this when more than 256 collections
    // occurred inside the window, meaning older pauses were evicted from the
    // circular buffer and the true maximum is unknowable. Dropping it here
    // would let an overflowed window report a clean low max and PASS M7 while
    // the real worst pause was silently discarded — the exact failure the Go
    // side was written to prevent. gate() treats it as a hard failure.
    pauseSampleTruncated: after.pauseSampleTruncated === true,
    heapAllocBytes: after.heapAllocBytes,
  };
}

/**
 * One measurement window.
 *
 * Usage:
 *   const session = startMeasurement({ frameIntervalMs });
 *   ... drive the app ...
 *   const result = await session.stop();
 */
export function startMeasurement(options = {}) {
  const { frameIntervalMs = 1000 / 60, label = 'window' } = options;

  const frameDeltas = [];
  const longFrames = [];
  const interactions = [];

  // Captured BEFORE the observers so nothing recorded during this window can
  // be filtered out by its own start time.
  const windowStartedAt = performance.now();
  const longFrameObserver = createLongFrameObserver(longFrames, windowStartedAt);
  const interactionObserver = createInteractionObserver(interactions, windowStartedAt);
  const goBefore = readGoProbe();

  let running = true;
  let previousTimestamp = performance.now();
  const startedAt = previousTimestamp;

  function onFrame(timestamp) {
    if (!running) return;
    frameDeltas.push(timestamp - previousTimestamp);
    previousTimestamp = timestamp;
    requestAnimationFrame(onFrame);
  }
  requestAnimationFrame(onFrame);

  return {
    label,
    async stop() {
      running = false;
      // One extra frame so the final rAF delta and any trailing observer
      // entries are delivered before disconnecting.
      await nextFrame();
      const goAfter = readGoProbe();
      longFrameObserver.disconnect();
      interactionObserver.disconnect();

      // The first delta is measured against startMeasurement's clock read
      // rather than a real frame boundary, so it is not a frame interval.
      const frames = frameDeltas.slice(1);

      return {
        label,
        durationMs: performance.now() - startedAt,
        frameIntervalMs,
        frames,
        droppedFrames: frames.filter((delta) => delta > frameIntervalMs * 1.5).length,
        longFrames,
        longFrameSource: longFrameObserver.source,
        interactions,
        interactionsSupported: interactionObserver.supported,
        go: diffGoProbe(goBefore, goAfter),
      };
    },
  };
}

/**
 * Collapses a raw window into the gated metrics. Kept separate from
 * collection so a stored raw run can be re-scored against new budgets without
 * re-running the benchmark.
 */
export function deriveMetrics(window_, stats) {
  const frameSummary = stats.summarize(window_.frames);
  const interactionDurations = window_.interactions.map((entry) => entry.duration);

  return {
    label: window_.label,
    durationMs: window_.durationMs,

    frame: frameSummary,
    droppedFrames: window_.droppedFrames,
    droppedFrameRatio: window_.frames.length ? window_.droppedFrames / window_.frames.length : 0,

    // M2: the count that must be zero, plus the total blocking time that
    // explains how bad it was when it is not.
    longFrameCount: window_.longFrames.filter((entry) => entry.duration > 50).length,
    totalBlockingMs: window_.longFrames.reduce((sum, entry) => sum + (entry.blockingDuration ?? 0), 0),
    worstLongFrameMs: window_.longFrames.reduce((worst, entry) => Math.max(worst, entry.duration), 0),
    // Composition of the long frames, not just their count.
    //
    // A count says the render thread missed a deadline; it cannot say whether it
    // was busy or absent. Long Animation Frame entries carry per-script
    // attribution, so the time that belongs to scripts can be separated from the
    // time that belongs to nothing the main thread executed — and a frame with
    // no scripts and no style/layout is a thread that did not run, which no
    // amount of framework work would shorten.
    longFrameScriptedMs: window_.longFrames.reduce(
      (sum, entry) => sum + (entry.scripts ?? []).reduce((s, script) => s + script.duration, 0), 0),
    longFrameStyleLayoutMs: window_.longFrames.reduce(
      (sum, entry) => sum + (entry.styleAndLayoutDuration ?? 0), 0),
    longFrameTopInvokers: (() => {
      const byInvoker = new Map();
      for (const entry of window_.longFrames) {
        for (const script of entry.scripts ?? []) {
          const key = `${script.invokerType ?? '?'} ${script.invoker ?? '?'}`;
          byInvoker.set(key, (byInvoker.get(key) ?? 0) + script.duration);
        }
      }
      return [...byInvoker.entries()].sort((a, b) => b[1] - a[1]).slice(0, 3)
        .map(([name, ms]) => ({ name, ms }));
    })(),
    longFrameSource: window_.longFrameSource,

    // M3
    interaction: interactionDurations.length
      ? stats.summarize(interactionDurations)
      : null,
    interactionsSupported: window_.interactionsSupported,

    // M7 + phase attribution
    go: window_.go,
    maxGCPauseMs: window_.go ? window_.go.maxGCPauseNs / 1e6 : null,
    gcSampleTruncated: window_.go ? window_.go.pauseSampleTruncated === true : false,

    // Surfaced, never smoothed: a bimodal frame distribution IS the bug.
    clusters: stats.splitClusters(window_.frames),
    drift: stats.detectDrift(window_.frames),
  };
}
