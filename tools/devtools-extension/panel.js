// Thin DOM glue: read the gwc.devtools.extension.v1 payload from the inspected page and
// render the view model that bridge.js computes. The page exposes the latest payload as
// window.__GWC_DEVTOOLS__ (set by the app's agentbridge devtools emitter).
function refresh() {
  chrome.devtools.inspectedWindow.eval("window.__GWC_DEVTOOLS__ || null", function (payload) {
    if (!payload || !isSupportedPayload(payload)) {
      document.getElementById("route").textContent = "waiting for a gwc.devtools.extension.v1 payload…";
      return;
    }
    const view = summarizePayload(payload);
    document.getElementById("route").textContent = "route: " + view.route + "  (" + view.schema + ")";
    document.getElementById("tree").innerHTML = "";
    for (const n of view.nodes) {
      const div = document.createElement("div");
      div.className = "node" + (n.dirty ? " dirty" : "");
      div.textContent = "  ".repeat(n.depth) + (n.fineGrained ? "~" : "") + n.name + (n.renderMs ? "  (" + n.renderMs.toFixed(2) + "ms)" : "");
      document.getElementById("tree").appendChild(div);
    }
    const s = view.stats, p = view.profiling;
    document.getElementById("stats").textContent =
      "fibers: " + (s.TotalFibers || 0) + " (dirty " + (s.DirtyFibers || 0) + ")  |  commits: " + (p.CommitCount || 0) + "  renders: " + (p.RenderCalls || 0);
  });
}
refresh();
setInterval(refresh, 1000);
