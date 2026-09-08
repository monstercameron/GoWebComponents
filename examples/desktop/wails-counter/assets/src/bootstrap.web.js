// Web builds never import the native runtime, generated bindings, or transport.
globalThis.__gwcImportModule = (parseSpecifier) => import(parseSpecifier);

// setWebStatus keeps startup failures visible outside the application mount.
const setWebStatus = (parseMessage, isError = false) => {
  const parseStatus = document.getElementById("boot-status");
  parseStatus.textContent = parseMessage;
  parseStatus.dataset.state = isError ? "error" : "ready";
};

// startWebApp loads only the matching Go Wasm runtime and portable UI artifact.
const startWebApp = async () => {
  if (typeof globalThis.Go !== "function") throw new Error("Matching wasm_exec.js is unavailable");
  const parseRuntime = new globalThis.Go();
  const parseResponse = await fetch("/app.wasm");
  if (!parseResponse.ok) throw new Error(`Wasm fetch failed (${parseResponse.status})`);
  const parseFallback = parseResponse.clone();
  let parseModule;
  try {
    parseModule = await WebAssembly.instantiateStreaming(parseResponse, parseRuntime.importObject);
  } catch {
    parseModule = await WebAssembly.instantiate(await parseFallback.arrayBuffer(), parseRuntime.importObject);
  }
  parseRuntime.run(parseModule.instance).catch((parseError) => {
    console.error("Web runtime stopped", parseError);
    setWebStatus(`Runtime stopped: ${parseError.message || parseError}`, true);
  });
  setWebStatus("Web mode · native APIs unavailable");
};

startWebApp().catch((parseError) => {
  console.error("Web startup failed", parseError);
  setWebStatus(`Startup failed: ${parseError.message || parseError}`, true);
});
