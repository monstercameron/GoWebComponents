// Bootstrap for the harness's domain worker (services.wasm).
//
// Loads Go's wasm shim and instantiates the worker binary. The worker's own
// main() installs onmessage and posts {ready:true}, so the app knows when the
// scope can actually receive work — messages sent before that are dropped
// silently, which would present as a workload that never starts.
importScripts('/wasm_exec.js');

const go = new Go();

WebAssembly.instantiateStreaming(fetch('./v5services.wasm'), go.importObject)
  .then((result) => {
    // go.run resolves only when main() returns, and main() blocks forever on
    // purpose, so it is deliberately not awaited.
    go.run(result.instance);
  })
  .catch((err) => {
    // A failed instantiate must be visible. Silence here looks exactly like a
    // workload that produces no progress, and the harness would report a run
    // that measured nothing as though it measured everything.
    postMessage({ workload: '', err: 'worker instantiate failed: ' + err });
  });
