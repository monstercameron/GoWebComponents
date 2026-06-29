// bridge.js is the pure, host-independent core of the GoWebComponents DevTools extension: it
// turns a `gwc.devtools.extension.v1` payload (emitted by devtools/extension_bridge.go and
// posted from the running wasm app) into a flat view model the panel renders. It does not
// touch the chrome.devtools API, so it is unit-testable under plain Node — the only
// non-trivial logic lives here; panel.js is thin DOM glue.

// flattenTree walks the component tree into a depth-tagged list (for an indented tree view).
// The Go Node struct has no json tags, so its keys are PascalCase (Name/Kind/Dirty/Children).
function flattenTree(parseNode, parseDepth, parseOut) {
  parseDepth = parseDepth || 0;
  parseOut = parseOut || [];
  if (!parseNode) return parseOut;
  parseOut.push({
    name: parseNode.Name || parseNode.Kind || "(node)",
    kind: parseNode.Kind || "",
    depth: parseDepth,
    dirty: !!parseNode.Dirty,
    fineGrained: !!parseNode.FineGrained,
    renderMs: ((parseNode.RenderDurationNs || 0) / 1e6),
  });
  const parseChildren = parseNode.Children || [];
  for (let parseI = 0; parseI < parseChildren.length; parseI++) {
    flattenTree(parseChildren[parseI], parseDepth + 1, parseOut);
  }
  return parseOut;
}

// summarizePayload extracts the headline view model: schema, current route, the flattened
// component tree, the fiber stats, and the profiling counters.
function summarizePayload(parsePayload) {
  if (!parsePayload) {
    return { schema: "", route: "", nodes: [], stats: {}, profiling: {} };
  }
  return {
    schema: parsePayload.schemaVersion || "",
    route: (parsePayload.route && parsePayload.route.Path) || "",
    nodes: flattenTree(parsePayload.tree),
    stats: parsePayload.stats || {},
    profiling: parsePayload.profiling || {},
  };
}

// isSupportedPayload reports whether a payload matches the schema this extension understands.
function isSupportedPayload(parsePayload) {
  return !!parsePayload && parsePayload.schemaVersion === "gwc.devtools.extension.v1";
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = { flattenTree, summarizePayload, isSupportedPayload };
}
