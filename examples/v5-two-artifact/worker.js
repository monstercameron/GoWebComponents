// Bootstrap for the domain worker (services.wasm).
//
// Loads Go's wasm shim and instantiates the worker binary. The worker's own
// main() installs onmessage and posts {ready:true}, so the app knows when the
// scope can actually receive commands — a message sent before that handler
// exists is dropped silently, which presents as a command that never returns
// rather than as an error anyone can find.
importScripts('/wasm_exec.js');

const go = new Go();

WebAssembly.instantiateStreaming(fetch('./services.wasm'), go.importObject)
  .then((result) => {
    // go.run resolves only when main() returns, and main() blocks forever on
    // purpose, so it is deliberately not awaited.
    go.run(result.instance);
  })
  .catch((err) => {
    // A failed instantiate has to be visible. Staying silent here is
    // indistinguishable from a worker that is merely slow to boot, and the app
    // would wait for a reply that can never come.
    postMessage({ id: 0, err: 'worker instantiate failed: ' + err });
  });
