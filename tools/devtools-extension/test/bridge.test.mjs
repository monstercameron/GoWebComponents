import test from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const { flattenTree, summarizePayload, isSupportedPayload } = require("../bridge.js");

const payload = {
  schemaVersion: "gwc.devtools.extension.v1",
  route: { Path: "/dashboard" },
  tree: {
    Name: "App", Kind: "component", Children: [
      { Name: "Header", Kind: "component", Dirty: true, RenderDurationNs: 2_000_000 },
      { Name: "List", Kind: "component", FineGrained: true, Children: [
        { Name: "Item", Kind: "component" },
      ] },
    ],
  },
  stats: { TotalFibers: 4, DirtyFibers: 1 },
  profiling: { CommitCount: 7, RenderCalls: 12 },
};

test("isSupportedPayload gates on the schema version", () => {
  assert.ok(isSupportedPayload(payload));
  assert.ok(!isSupportedPayload({ schemaVersion: "other" }));
  assert.ok(!isSupportedPayload(null));
});

test("flattenTree produces a depth-tagged list with dirty/fine-grained flags", () => {
  const nodes = flattenTree(payload.tree);
  assert.equal(nodes.length, 4);
  assert.deepEqual(nodes.map((n) => n.depth), [0, 1, 1, 2]);
  assert.equal(nodes[0].name, "App");
  assert.equal(nodes[1].dirty, true);
  assert.equal(nodes[1].renderMs, 2);
  assert.equal(nodes[2].fineGrained, true);
});

test("summarizePayload extracts route, tree, stats, profiling", () => {
  const v = summarizePayload(payload);
  assert.equal(v.schema, "gwc.devtools.extension.v1");
  assert.equal(v.route, "/dashboard");
  assert.equal(v.nodes.length, 4);
  assert.equal(v.stats.TotalFibers, 4);
  assert.equal(v.profiling.CommitCount, 7);
});

test("summarizePayload tolerates an empty payload", () => {
  assert.deepEqual(summarizePayload(null).nodes, []);
});
