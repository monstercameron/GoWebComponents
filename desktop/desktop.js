// createDesktopTransport adapts explicitly registered generated Wails exports.
// No Go js.Func is captured by a promise; registries and event queues are bounded.
export function createDesktopTransport({ methods, topics = [], features = [], events, platform, hostVersion, limit = 256 }) {
  if (!Number.isInteger(limit) || limit < 1 || limit > 4096) throw new Error("Invalid desktop registry limit");
  const getMethods = new Map(Object.entries(methods));
  const getTopics = new Set(topics);
  if (!Array.isArray(features) || features.length > 64 || features.some(parseFeature => typeof parseFeature !== "string" || !parseFeature || parseFeature.length > 128)) throw new Error("Invalid desktop feature capabilities");
  const getFeatures = new Set(features);
  for (const [parseName, parseMethod] of getMethods) {
    if (!parseName || typeof parseMethod !== "function") throw new Error("Desktop methods require names and callable exports");
  }
  for (const parseTopic of getTopics) {
    if (typeof parseTopic !== "string" || !parseTopic.includes(".")) throw new Error("Desktop topics must be namespaced");
  }
  if (getTopics.size && typeof events?.On !== "function") throw new Error("Desktop event runtime unavailable");
  const getRequests = new Map();
  const getListeners = new Map();
  let getSequence = 0;
  let isClosed = false;
  const getOK = (parseData = null) => JSON.stringify({ data: parseData });
  // Host errors may contain throwing getters; error reporting must itself remain total.
  const getMessage = (parseError) => {
    try { return String(parseError?.message || parseError); }
    catch { return "Desktop host error could not be serialized"; }
  };
  const getError = (parseCode, parseMessage) => JSON.stringify({ code: parseCode, message: getMessage(parseMessage) });
  const getReply = (parseValue) => {
    try { return { done: true, data: JSON.parse(JSON.stringify(parseValue ?? null, (parseKey, parseItem) => {
      if (typeof parseItem === "number" && (!Number.isFinite(parseItem) || (Number.isInteger(parseItem) && !Number.isSafeInteger(parseItem)))) throw new Error("Numeric payload exceeds JavaScript safe range; use strings");
      return parseItem;
    })) }; }
    catch (parseError) { return { done: true, code: "decode", message: getMessage(parseError) }; }
  };
  const cancelPromise = (parsePromise) => {
    try { if (typeof parsePromise?.cancel === "function") Promise.resolve(parsePromise.cancel()).catch(() => {}); }
    catch { /* The owned registry is already released, even if host cancellation throws. */ }
  };
  // Cancellation errors are consumed, but cancellation does not roll back writes.
  const cancelRequest = (parseID) => {
    const parseEntry = getRequests.get(parseID);
    getRequests.delete(parseID);
    if (parseEntry && !parseEntry.reply.done) cancelPromise(parseEntry.promise);
  };
  const stopListener = (parseID) => {
    const parseEntry = getListeners.get(parseID);
    getListeners.delete(parseID);
    if (parseEntry) parseEntry.stop();
  };
  const getTransport = {
    capabilities() {
      return isClosed ? getError("disposed", "Desktop window closed") : getOK({ protocol: 1, platform, hostVersion, methods: [...getMethods.keys()], topics: [...getTopics], features: [...getFeatures] });
    },
    start(parseMethod, parseJSON) {
      if (isClosed) return getError("disposed", "Desktop window closed");
      if (!getMethods.has(parseMethod)) return getError("missing_export", "Method not registered");
      if (getRequests.size >= limit) return getError("quota_exceeded", "Desktop request limit reached");
      let parseArgs;
      try { parseArgs = JSON.parse(parseJSON); if (!Array.isArray(parseArgs)) throw new Error("Arguments must be an array"); }
      catch (parseError) { return getError("encode", parseError); }
      const parseID = String(++getSequence);
      try {
        const parsePromise = getMethods.get(parseMethod)(...parseArgs);
        if (isClosed) {
          Promise.resolve(parsePromise).catch(() => {});
          cancelPromise(parsePromise);
          return getError("disposed", "Desktop window closed during method invocation");
        }
        getRequests.set(parseID, { promise: parsePromise, reply: { done: false } });
        Promise.resolve(parsePromise).then(
          (parseValue) => { const parseEntry = getRequests.get(parseID); if (parseEntry) { parseEntry.reply = getReply(parseValue); parseEntry.promise = null; } },
          (parseError) => { const parseEntry = getRequests.get(parseID); if (parseEntry) { parseEntry.reply = { done: true, code: "remote_error", message: getMessage(parseError) }; parseEntry.promise = null; } }
        );
        return getOK(parseID);
      } catch (parseError) { cancelRequest(parseID); return getError("remote_error", parseError); }
    },
    poll(parseID) {
      if (isClosed) return getError("disposed", "Desktop window closed");
      const parseEntry = getRequests.get(parseID);
      return parseEntry ? getOK(parseEntry.reply) : getError("disposed", "Request released");
    },
    cancel(parseID) { cancelRequest(parseID); return getOK(); },
    listen(parseTopic) {
      if (isClosed) return getError("disposed", "Desktop window closed");
      if (!getTopics.has(parseTopic)) return getError("missing_export", "Topic not registered");
      if (getListeners.size >= limit) return getError("quota_exceeded", "Desktop subscription limit reached");
      const parseID = String(++getSequence);
      const parseEntry = { reply: { done: false }, stop: () => {} };
      getListeners.set(parseID, parseEntry);
      try {
        parseEntry.stop = events.On(parseTopic, (parseEvent) => {
          if (!getListeners.has(parseID)) return;
          try { parseEntry.reply = getReply(parseEvent.data); }
          catch (parseError) { parseEntry.reply = { done: true, code: "decode", message: getMessage(parseError) }; }
        });
        if (typeof parseEntry.stop !== "function") throw new Error("Desktop event runtime did not return unsubscribe");
        if (isClosed) {
          parseEntry.stop();
          return getError("disposed", "Desktop window closed during event registration");
        }
      }
      catch (parseError) { getListeners.delete(parseID); return getError("remote_error", parseError); }
      return getOK(parseID);
    },
    next(parseID) {
      if (isClosed) return getError("disposed", "Desktop window closed");
      const parseEntry = getListeners.get(parseID);
      if (!parseEntry) return getError("disposed", "Subscription released");
      const parseReply = parseEntry.reply;
      parseEntry.reply = { done: false };
      return getOK(parseReply);
    },
    unlisten(parseID) { try { stopListener(parseID); return getOK(); } catch (parseError) { return getError("remote_error", parseError); } },
    close() {
      if (isClosed) return;
      isClosed = true;
      for (const parseID of getRequests.keys()) cancelRequest(parseID);
      for (const parseID of getListeners.keys()) { try { stopListener(parseID); } catch (parseError) { console.error("Desktop unsubscribe failed", parseError); } }
    },
    // stats exposes counts for lifecycle tests without exposing request payloads.
    stats() { return { requests: getRequests.size, subscriptions: getListeners.size }; }
  };
  return getTransport;
}
