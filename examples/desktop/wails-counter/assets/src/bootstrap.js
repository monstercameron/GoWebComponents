// Local external import helper avoids the Function-constructor fallback in interop.
import { createDesktopTransport } from "/desktop.js";
const getBindingURL = "/bindings/example.com/gwc-wails-counter/internal/services/index.js";
const getSmokeMode = new URLSearchParams(location.search).get("smoke") === "1";
const isObserver = getSmokeMode && new URLSearchParams(location.search).get("observer") === "1";
const getSmokeFault = getSmokeMode ? new URLSearchParams(location.search).get("fault") : "";
const isPersistenceEnabled = () => Boolean(getCapabilities?.methods?.includes("storage.load"));
let getBindings;
let getDesktopBindings;
let getCapabilities;
let getFileDialogHost;
let getRuntime;
let isFallback = false;
let getWasmMIME = "";
let getStorageEvents = 0;

// getNativeMethods routes only host-advertised desktop envelopes to the gated dispatcher.
const getNativeMethods = (parseMethods, parseExecute) => {
  const parseResult = {};
  for (const parseMethod of parseMethods) {
    if (parseMethod.startsWith("desktop.") && parseMethod !== "desktop.files.select") {
      if (typeof parseExecute !== "function") throw new Error("Missing NativeHost.Execute binding");
      parseResult[parseMethod] = parseExecute;
    }
  }
  return parseResult;
};

// setBootStatus lives outside the GWC mount so rendering cannot erase failures.
const setBootStatus = (parseMessage, isError = false) => {
  const parseElement = document.getElementById("boot-status");
  parseElement.textContent = parseMessage;
  parseElement.dataset.state = isError ? "error" : "ready";
};

// Native calls and subscriptions are owned by the reusable desktop adapter.
globalThis.__gwcWailsReady = (async () => {
  getRuntime = await import("/wails/runtime.js");
  getFileDialogHost = (await import("/bindings/github.com/monstercameron/GoWebComponents/v6/desktop/index.js")).FileDialogHost;
  getDesktopBindings = await import("/bindings/github.com/monstercameron/GoWebComponents/v6/desktop/index.js");
  if (typeof getFileDialogHost?.SelectPaths !== "function") throw new Error("Missing FileDialogHost.SelectPaths binding");
  getBindings = await import(getSmokeFault === "missing-binding" ? "/missing-bindings.js" : getBindingURL);
  for (const parseMethod of ["Increment", "OpenFile", "RunProgress", "Reject", "RunWork", "GetWorkState", "GetCapabilities"]) {
    if (typeof getBindings.CounterService?.[parseMethod] !== "function") {
      throw new Error(`Missing CounterService binding: ${parseMethod}`);
    }
  }
  if (typeof getRuntime.Events?.On !== "function") throw new Error("Wails event runtime unavailable");
  if (getSmokeMode) {
    const parseStopDiagnostics = getRuntime.Events.On("storage.changed", () => { getStorageEvents++; });
    window.addEventListener("pagehide", parseStopDiagnostics, { once: true });
  }
  getCapabilities = await getBindings.CounterService.GetCapabilities().cancelOn(AbortSignal.timeout(10000));
  if (getCapabilities.protocol !== 1 || getCapabilities.platform !== "windows" || getCapabilities.hostVersion !== "v3.0.0-beta.17") {
    throw new Error(`Unsupported native host contract: ${JSON.stringify(getCapabilities)}`);
  }
  const parseMethods = {
      ...(getCapabilities.methods.includes("desktop.files.select") ? { "desktop.files.select": getFileDialogHost.SelectPaths } : {}),
      "counter.increment": getBindings.CounterService.Increment,
      "counter.openFile": getBindings.CounterService.OpenFile,
      "counter.progress": getBindings.CounterService.RunProgress,
      "counter.reject": getBindings.CounterService.Reject,
      "counter.work": getBindings.CounterService.RunWork,
      "api.run": getBindings.APIService.Run,
      "api.report": getBindings.APIService.GetReport,
      "api.observe": getBindings.APIService.RecordObservation,
      ...(getCapabilities.methods.includes("storage.load") ? { "storage.load": getBindings.StorageService.Load } : {}),
      ...(getCapabilities.methods.includes("storage.save") ? { "storage.save": getBindings.StorageService.Save } : {}),
      ...(getCapabilities.methods.includes("storage.delete") ? { "storage.delete": getBindings.StorageService.Delete } : {}),
      ...(getCapabilities.methods.includes("storage.keys") ? { "storage.keys": getBindings.StorageService.Keys } : {}),
      ...(getCapabilities.methods.includes("api.export") ? { "api.export": getBindings.APIService.ExportReport } : {}),
      ...getNativeMethods(getCapabilities.methods, getDesktopBindings.NativeHost?.Execute),
    };
  globalThis.__gwcDesktop = createDesktopTransport({ methods: parseMethods,
    topics: getCapabilities.topics, features: getCapabilities.features || [], events: getRuntime.Events,
    platform: getCapabilities.platform, hostVersion: getCapabilities.hostVersion,
  });
  const parseInstalledCapabilities = JSON.parse(globalThis.__gwcDesktop.capabilities()).data;
  if (JSON.stringify([...getCapabilities.methods].sort()) !== JSON.stringify([...parseInstalledCapabilities.methods].sort())) {
    globalThis.__gwcDesktop.close();
    throw new Error(`Native method capabilities disagree with installed generated bindings: host=${JSON.stringify(getCapabilities.methods)} installed=${JSON.stringify(parseInstalledCapabilities.methods)}`);
  }
  if (JSON.stringify([...(getCapabilities.features || [])].sort()) !== JSON.stringify([...(parseInstalledCapabilities.features || [])].sort())) {
    globalThis.__gwcDesktop.close();
    throw new Error("Native feature capabilities disagree with the installed transport");
  }
  window.addEventListener("pagehide", () => globalThis.__gwcDesktop.close(), { once: true });
})();

globalThis.__gwcImportModule = async (parseSpecifier) => {
  await globalThis.__gwcWailsReady;
  return import(parseSpecifier);
};

// startWasm retries with buffered bytes if streaming instantiation is unavailable.
const startWasm = async () => {
  await globalThis.__gwcWailsReady;
  if (typeof globalThis.Go !== "function") throw new Error("wasm_exec.js did not provide Go");
  const parseGo = new globalThis.Go();
  const parseURL = getSmokeFault === "missing-wasm" ? "/missing-app.wasm" : "/app.wasm";
  const parseResponse = await fetch(parseURL);
  if (!parseResponse.ok) throw new Error(`Wasm fetch failed (${parseResponse.status})`);
  getWasmMIME = parseResponse.headers.get("Content-Type") || "";
  const parseBytes = parseResponse.clone();
  let parseResult;
  try {
    if (getSmokeFault === "streaming") throw new TypeError("smoke: streaming unavailable");
    parseResult = await WebAssembly.instantiateStreaming(parseResponse, parseGo.importObject);
  } catch (parseStreamingError) {
    isFallback = true;
    parseResult = await WebAssembly.instantiate(await parseBytes.arrayBuffer(), parseGo.importObject);
  }
  parseGo.run(parseResult.instance).catch((parseError) => {
    console.error("GoWebComponents Wasm runtime stopped", parseError);
    setBootStatus(`Runtime stopped: ${parseError.message || parseError}`, true);
  });
};

// waitFor observes rendered results, rather than treating dispatched input as proof.
const waitFor = async (parsePredicate, parseLabel, parseTimeout = 5000) => {
  const parseDeadline = Date.now() + parseTimeout;
  // Hidden WebViews can throttle timers; inspect the final state after waking
  // before declaring expiry, rather than rejecting a result already rendered.
  while (true) {
    if (parsePredicate()) return;
    if (Date.now() >= parseDeadline) throw new Error(`smoke wait timed out: ${parseLabel}`);
    await new Promise((parseResolve) => setTimeout(parseResolve, 25));
  }
};

const getText = (parseID) => document.getElementById(parseID)?.textContent;
const handleClick = (parseID) => {
  const parseElement = document.getElementById(parseID);
  if (!parseElement || parseElement.disabled) throw new Error(`Control unavailable: ${parseID}`);
  parseElement.click();
};

// runSmoke exercises Go UI handlers, interop, native calls and the renderer.
const runSmoke = async () => {
  if (!getSmokeMode) return;
  if (!(await getBindings.SmokeService.GetMode())) throw new Error("Smoke host unavailable");
  if (isObserver) {
    if (!isPersistenceEnabled()) {
      await getBindings.SmokeService.ReportObserver({ ready: true, value: 0, local: 0, error: "" });
      return;
    }
    try {
      await waitFor(() => document.getElementById("shared-increment")?.disabled === false && getText("local-count") === "Count: 0", "observer initial state", 15000);
      await getBindings.SmokeService.ReportObserver({ ready: true, value: 0, local: 0, error: "" });
      await waitFor(() => getText("shared-count") === "Saved count: 1", "second window native invalidation", 10000);
      if (getText("local-count") !== "Count: 0") throw new Error("Local UI state leaked across windows");
      handleClick("local-increment");
      await waitFor(() => getText("local-count") === "Count: 1", "observer local increment");
      await getBindings.SmokeService.ReportObserver({ ready: true, value: 1, local: 1, error: "" });
    } catch (parseError) { await getBindings.SmokeService.ReportObserver({ ready: false, value: 0, local: 0, error: String(parseError?.message || parseError) + `; native events ${getStorageEvents}; UI: ${getText("shared-count")}; error: ${getText("native-error")}; stats: ${JSON.stringify(globalThis.__gwcDesktop.stats())}` }); }
    return;
  }
  const parseChecks = [];
  try {
    await waitFor(() => getText("route-title") === "Wails Counter", "GWC mount");
    parseChecks.push("dom", "native-handshake");
    const parseUnattributed = await fetch("/wails/runtime", { referrerPolicy: "no-referrer" });
    if (parseUnattributed.status !== 403) throw new Error(`Native endpoint accepted missing initiator: ${parseUnattributed.status}`);
    parseChecks.push("origin-guard");
    await waitFor(() => document.getElementById("native-increment")?.disabled === false, "binding readiness");
    handleClick("local-increment");
    await waitFor(() => getText("local-count") === "Count: 1", "local counter");
    parseChecks.push("local-counter");
    handleClick("native-increment");
    await waitFor(() => getText("native-status") === "Native response: 1", "typed Go native response");
    parseChecks.push("native-call");
    handleClick("native-reject");
    await waitFor(() => getText("native-error")?.includes("counter expected rejection"), "Go native error");
    parseChecks.push("native-error");
    handleClick("native-progress");
    handleClick("local-increment");
    await waitFor(() => getText("progress-status") === "Native progress: 100%" && getText("local-count") === "Count: 2", "native progress in Go UI");
    parseChecks.push("native-event");
    const waitForObserver = async (parseValue) => {
      const parseDeadline = Date.now() + 15000;
      let parseLastObserver;
      while (Date.now() < parseDeadline) {
        const parseObserver = await getBindings.SmokeService.GetObserver();
        parseLastObserver = parseObserver;
        if (parseObserver.error) throw new Error(parseObserver.error);
        if (parseObserver.ready && parseObserver.value === parseValue && parseObserver.local === parseValue) return;
        await new Promise((parseResolve) => setTimeout(parseResolve, 25));
      }
      const parseStored = await getBindings.StorageService.Load("shared-counter");
      throw new Error(`Second window did not report value ${parseValue}: ${JSON.stringify(parseLastObserver)}; primary shared: ${getText("shared-count")}; stored: ${JSON.stringify(parseStored)}; transport: ${JSON.stringify(globalThis.__gwcDesktop.stats())}`);
    };
    if (isPersistenceEnabled()) {
      await waitForObserver(0);
      handleClick("shared-increment");
      await waitForObserver(1);
      if (getText("local-count") !== "Count: 2") throw new Error("Second window changed primary local state");
      parseChecks.push("durable-native-state", "two-window-invalidation", "window-local-state");
    } else {
      parseChecks.push("persistent-storage-disabled");
    }
    // Observe the backend's context, not merely cancellation of the UI wait.
    const waitForWork = async (parseActive, parseCancelled) => {
      const parseDeadline = Date.now() + 5000;
      while (Date.now() < parseDeadline) {
        const parseState = await getBindings.CounterService.GetWorkState();
        if (parseState.active === parseActive && parseState.cancelled === parseCancelled) return;
        await new Promise((parseResolve) => setTimeout(parseResolve, 25));
      }
      throw new Error("Native work cancellation was not observed");
    };
    handleClick("native-work");
    await waitForWork(1, 0);
    handleClick("native-cancel");
    await waitForWork(0, 1);
    parseChecks.push("backend-cancellation");
    handleClick("native-work");
    await waitForWork(1, 1);
    handleClick("about-link");
    await waitForWork(0, 2);
    await waitFor(() => globalThis.__gwcDesktop.stats().requests === 0 && globalThis.__gwcDesktop.stats().subscriptions === 0, "adapter unmount cleanup");
    handleClick("counter-link");
    await waitFor(() => document.getElementById("native-progress")?.disabled === false, "adapter remount");
    parseChecks.push("unmount-cancellation", "adapter-cleanup");
    for (let parseCycle = 0; parseCycle < 3; parseCycle++) {
      handleClick("about-link");
      await waitFor(() => getText("route-title") === "About this desktop example" && !document.getElementById("local-count"), "about route unmount");
      handleClick("counter-link");
      await waitFor(() => getText("route-title") === "Wails Counter" && getText("local-count") === "Count: 0", "counter remount");
      await waitFor(() => document.getElementById("native-progress")?.disabled === false, "remount bridge readiness");
      handleClick("native-progress");
      await waitFor(() => getText("progress-status") === "Native progress: 100%", "remount native event");
    }
    parseChecks.push("routing", "route-remount");
    handleClick("tester-link");
    await waitFor(() => document.getElementById("api-window-info") && document.getElementById("api-context-target"), "API lab mount");
    if (getComputedStyle(document.getElementById("api-context-target")).getPropertyValue("--custom-contextmenu").trim() !== "api-tester") {
      throw new Error("API lab is not wired to the registered native context menu");
    }
    parseChecks.push("api-tester-ui");
    if (getCapabilities.methods.includes("desktop.window.control")) {
      handleClick("api-window-info");
      const parseAPIDeadline = Date.now() + 10000;
      while (true) {
        const parseAPIReport = await getBindings.APIService.GetReport();
        // The button invokes the portable SDK once, not the legacy report-writing service.
        const parseInfo = getText("api-window-info-result");
        if (parseInfo.includes("completed") && parseInfo.includes("window ") && parseInfo.includes("size=") && parseAPIReport.fixtureDir && parseAPIReport.platform === "windows") break;
        if (Date.now() >= parseAPIDeadline) throw new Error("API UI/native window report missing");
        await new Promise((parseResolve) => setTimeout(parseResolve, 25));
      }
      parseChecks.push("api-window-info", "api-session-report");
    } else {
      const parseAPIReport = await getBindings.APIService.GetReport();
      if (!parseAPIReport.fixtureDir || parseAPIReport.platform !== "windows") throw new Error("API session report unavailable");
      parseChecks.push("native-window-disabled", "api-session-report");
    }
    // Exercise the actual generated SDK binding without opening any OS dialog.
    const parseFileCaps = await getBindings.CounterService.GetCapabilities();
    const isFilesEnabled = parseFileCaps.methods.includes("desktop.files.select");
    const parseFileReply = await getFileDialogHost.SelectPaths({version: 2, kind: "open-file", options: {}});
    if (parseFileReply.code !== (isFilesEnabled ? "invalid" : "unavailable")) {
      throw new Error(`File SDK host validation/policy failed: ${JSON.stringify(parseFileReply)}`);
    }
    if (!isFilesEnabled) {
      // A caller bypassing the SDK and sending a valid request still cannot open a picker.
      const parseDenied = await getFileDialogHost.SelectPaths({version: 1, kind: "open-file", options: {}});
      if (parseDenied.code !== "unavailable") throw new Error("Disabled file host allowed direct invocation");
      for (const parseLegacyCall of [() => getBindings.CounterService.OpenFile(), () => getBindings.APIService.Run("open-file")]) {
        let isDenied = false;
        try { await parseLegacyCall(); } catch (parseError) { isDenied = String(parseError?.message || parseError).includes("disabled or unavailable"); }
        if (!isDenied) throw new Error("Legacy picker route bypassed disabled file policy");
      }
    }
    parseChecks.push("desktop-file-contract");
    // Exercise the versioned NativeHost envelope without touching user clipboard or opening dialogs.
    const parseScreenReply = await getDesktopBindings.NativeHost.Execute({ version: 1, method: "desktop.screens.list", args: null });
    if (getCapabilities.methods.includes("desktop.screens.list")) {
      if (parseScreenReply.code || !Array.isArray(parseScreenReply.data)) throw new Error(`Native SDK screens contract failed: ${JSON.stringify(parseScreenReply)}`);
    } else if (parseScreenReply.code !== "unavailable") {
      throw new Error(`Disabled Native SDK screens call was not denied: ${JSON.stringify(parseScreenReply)}`);
    }
    const parseMessageReply = await getDesktopBindings.NativeHost.Execute({ version: 1, method: "desktop.message.show", args: { kind: "invalid", title: "", message: "" } });
    if (getCapabilities.methods.includes("desktop.message.show")) {
      if (parseMessageReply.code !== "invalid") throw new Error(`Native SDK message validation failed: ${JSON.stringify(parseMessageReply)}`);
    } else if (parseMessageReply.code !== "unavailable") {
      throw new Error(`Disabled Native SDK message call was not denied: ${JSON.stringify(parseMessageReply)}`);
    }
    for (const [parseMethod, parseArgs] of [
      ["desktop.clipboard.write", { text: "\u0000" }],
      ["desktop.window.control", { action: "unknown" }],
    ]) {
      const parseReply = await getDesktopBindings.NativeHost.Execute({ version: 1, method: parseMethod, args: parseArgs });
      const parseExpected = getCapabilities.methods.includes(parseMethod) ? "invalid" : "unavailable";
      if (parseReply.code !== parseExpected) throw new Error(`Native SDK guard failed for ${parseMethod}: ${JSON.stringify(parseReply)}`);
    }
    parseChecks.push("native-sdk-contract");
    if (getCapabilities.methods.includes("desktop.window.control")) {
      const parseControl = async (parseArgs) => {
        const parseReply = await getDesktopBindings.NativeHost.Execute({ version: 1, method: "desktop.window.control", args: parseArgs });
        if (parseReply.code || !parseReply.data?.id) throw new Error(`Window control failed: ${JSON.stringify(parseReply)}`);
        return parseReply.data;
      };
      const parseOriginal = await parseControl({ action: "info" });
      try {
        const parseSized = await parseControl({ action: "resize", width: 720, height: 520 });
        if (parseSized.width !== 720 || parseSized.height !== 520) throw new Error("Native size did not change");
        const parseMoved = await parseControl({ action: "set-position", x: 100, y: 100 });
        if (parseMoved.x !== 100 || parseMoved.y !== 100) throw new Error("Native position did not change");
        const parseZoomed = await parseControl({ action: "set-zoom", zoom: 1.25 });
        if (Math.abs(parseZoomed.zoom - 1.25) > 0.01) throw new Error("Native zoom did not change");
        const parseLowZoom = await getDesktopBindings.NativeHost.Execute({ version: 1, method: "desktop.window.control", args: { action: "set-zoom", zoom: 0.5 } });
        if (parseLowZoom.code !== "invalid") throw new Error("Windows low zoom was not explicitly rejected");
        const parseAfterLowZoom = await parseControl({ action: "info" });
        if (Math.abs(parseAfterLowZoom.zoom - 1.25) > 0.01) throw new Error("Rejected low zoom changed native state");
        await parseControl({ action: "zoom-reset" });
        const parseFixed = await parseControl({ action: "set-resizable", enabled: false });
        if (parseFixed.resizable !== false) throw new Error("Native resize policy did not change");
      } finally {
        // Attempt every restoration even when one native setter fails.
        const parseRestoreErrors = [];
        for (const parseRequest of [
          { action: "set-zoom", zoom: parseOriginal.zoom },
          { action: "set-resizable", enabled: parseOriginal.resizable },
          { action: "resize", width: parseOriginal.width, height: parseOriginal.height },
          { action: "set-position", x: parseOriginal.x, y: parseOriginal.y },
        ]) {
          try { await parseControl(parseRequest); }
          catch (parseError) { parseRestoreErrors.push(String(parseError)); }
        }
        if (parseRestoreErrors.length) throw new Error(`Window restoration failed: ${parseRestoreErrors.join("; ")}`);
      }
      parseChecks.push("native-window-roundtrip");
    }
    // Read-only platform probes use the real generated bindings and WebView2 caller.
    if (getCapabilities.methods.includes("desktop.system.environment")) {
      const parseEnvironment = await getDesktopBindings.NativeHost.Execute({ version: 1, method: "desktop.system.environment", args: {} });
      if (parseEnvironment.code || parseEnvironment.data?.os !== "windows" || !["light", "dark"].includes(parseEnvironment.data?.theme)) {
        throw new Error(`Windows environment probe failed: ${JSON.stringify(parseEnvironment)}`);
      }
      parseChecks.push("windows-environment");
    }
    if (getCapabilities.methods.includes("desktop.screens.geometry")) {
      const parseGeometry = await getDesktopBindings.NativeHost.Execute({ version: 1, method: "desktop.screens.geometry", args: { operation: "nearest-dip-point", point: { x: 0, y: 0 } } });
      if (parseGeometry.code || !parseGeometry.data?.screenId) {
        throw new Error(`Windows screen geometry probe failed: ${JSON.stringify(parseGeometry)}`);
      }
      parseChecks.push("windows-screen-geometry");
    }
    if (getCapabilities.methods.includes("desktop.url.open")) {
      const parseURLReply = await getDesktopBindings.NativeHost.Execute({ version: 1, method: "desktop.url.open", args: { url: "javascript:alert(1)" } });
      if (parseURLReply.code !== "invalid") throw new Error("Unsafe external URL was not rejected");
      parseChecks.push("external-url-denial");
    }
    // Native dialogs, clipboard and interactive menu selection are intentionally
    // excluded: unattended smoke cannot certify an operator's visual results.
    if (!getWasmMIME.startsWith("application/wasm")) throw new Error(`Unexpected Wasm MIME: ${getWasmMIME}`);
    parseChecks.push("wasm-mime");
    let isEvalBlocked = false;
    try { new Function("return 1")(); } catch (parseError) { isEvalBlocked = parseError instanceof EvalError; }
    if (!isEvalBlocked) throw new Error("CSP allowed the Function constructor");
    parseChecks.push("csp");
    if (getSmokeFault === "streaming") {
      if (!isFallback) throw new Error("buffered Wasm fallback was not used");
      parseChecks.push("wasm-fallback");
    }
    if (document.getElementById("boot-status").dataset.state !== "ready") throw new Error("boot diagnostics report failure");
    await getBindings.SmokeService.ReportResult({ ok: true, checks: parseChecks, error: "" });
  } catch (parseError) {
    await getBindings.SmokeService.ReportResult({ ok: false, checks: parseChecks, error: String(parseError?.message || parseError) + `; UI status: ${getText("native-status")}; UI error: ${getText("native-error")}; progress: ${getText("progress-status")}` });
  }
};

globalThis.__gwcWailsReady
  .then(startWasm)
  .then(() => { setBootStatus("Ready"); return runSmoke(); })
  .catch(async (parseError) => {
    console.error("GoWebComponents desktop boot failed", parseError);
    setBootStatus(`Startup failed: ${parseError.message || parseError}`, true);
    if (getSmokeMode) {
      // Reporting bypasses the deliberately failed adapter, not its UI readiness gate.
      const parseReporter = await import(getBindingURL);
      await parseReporter.SmokeService.ReportResult({
        ok: false,
        checks: document.getElementById("boot-status")?.dataset.state === "error" && !document.getElementById("native-increment") ? ["visible-boot-error", "native-controls-unavailable"] : [],
        error: String(parseError?.message || parseError)
      });
    }
  });
