import { runComparison, gate } from './harness.js';

// runDashboardComparison adapts measurements to data; only GWC renders the result.
export async function runDashboardComparison(parseOptions = {}) {
  const parseGlobal = parseOptions.global ?? globalThis;
  const parseRun = parseOptions.runComparison ?? runComparison;
  const parseGate = parseOptions.gate ?? gate;
  const parseFetch = parseOptions.fetch ?? fetch;
  const parseWorkloads = ['import', 'reindex', 'decode'].map((parseName) => {
    const parseWorkload = parseGlobal.__gwcV5Workloads[parseName];
    return {
      name: parseName,
      start: async () => parseWorkload.start(),
      stop: async () => parseWorkload.stop(),
      stats: () => {
        const parseStats = parseWorkload.stats();
        return { completed: parseStats.completed, err: parseStats.err };
      },
    };
  });
  const parseReport = await parseRun({
    probe: { name: 'typing', run: (parseMs) => parseGlobal.__gwcV5Probes.typing(parseMs) },
    workloads: parseWorkloads,
  });
  const parseResponse = await parseFetch('./budgets.json');
  if (!parseResponse.ok) throw new Error(`Budget request failed: ${parseResponse.status}`);
  const parseVerdict = parseGate(parseReport, await parseResponse.json());
  parseGlobal.__gwcV5Report = parseReport;
  parseGlobal.__gwcV5Verdict = parseVerdict;
  return JSON.stringify({
    status: parseVerdict.passed ? 'PASS' : 'FAIL',
    output: JSON.stringify({ verdict: parseVerdict, metrics: parseReport.metrics, valid: parseReport.valid }, null, 2),
  });
}
