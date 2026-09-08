import test from 'node:test';
import assert from 'node:assert/strict';
import { runDashboardComparison } from './dashboard.js';

// getDashboardFixture isolates measurement I/O without replacing the dashboard adapter.
function getDashboardFixture(parsePassed) {
  const parseReport = { metrics: { sample: 12 }, valid: true };
  const parseVerdict = { passed: parsePassed };
  const parseGlobal = {
    __gwcV5Workloads: Object.fromEntries(['import', 'reindex', 'decode'].map(parseName => [parseName, {
      start: () => parseName, stop: () => parseName, stats: () => ({ completed: 4, err: '' }),
    }])),
    __gwcV5Probes: { typing: parseMs => parseMs },
  };
  return {
    global: parseGlobal,
    runComparison: async parseOptions => {
      assert.deepEqual(parseOptions.workloads.map(parseWorkload => parseWorkload.name), ['import', 'reindex', 'decode']);
      assert.equal(parseOptions.probe.run(17), 17);
      assert.deepEqual(parseOptions.workloads[0].stats(), { completed: 4, err: '' });
      return parseReport;
    },
    fetch: async () => ({ ok: true, json: async () => ({ budget: 3 }) }),
    gate: (parseActual, parseBudgets) => {
      assert.equal(parseActual, parseReport);
      assert.deepEqual(parseBudgets, { budget: 3 });
      return parseVerdict;
    },
  };
}

for (const parsePassed of [true, false]) {
  test(`dashboard preserves ${parsePassed ? 'passing' : 'failing'} verdict`, async () => {
    const parseOptions = getDashboardFixture(parsePassed);
    const parseResult = JSON.parse(await runDashboardComparison(parseOptions));
    assert.equal(parseResult.status, parsePassed ? 'PASS' : 'FAIL');
    assert.deepEqual(JSON.parse(parseResult.output).verdict, { passed: parsePassed });
    assert.equal(parseOptions.global.__gwcV5Verdict.passed, parsePassed);
  });
}

test('dashboard propagates measurement and budget errors instead of displaying PASS', async () => {
  const parseOptions = getDashboardFixture(true);
  await assert.rejects(runDashboardComparison({ ...parseOptions, runComparison: async () => { throw new Error('measurement failed'); } }), /measurement failed/);
  await assert.rejects(runDashboardComparison({ ...parseOptions, fetch: async () => ({ ok: false, status: 404 }) }), /Budget request failed: 404/);
  assert.equal(parseOptions.global.__gwcV5Verdict, undefined);
});
