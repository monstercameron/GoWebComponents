self.postMessage({ phase: "ready", name: "bootstrap" });

self.onmessage = event => {
  const message = event.data || {};
  if (message.phase !== "request") {
    return;
  }

  const payload = message.payload || {};
  const text = String(payload.text || "");
  const query = String(payload.query || "").trim().toLowerCase();
  const requestID = String(message.id || "");
  const name = String(message.name || "build-index");

  self.postMessage({
    id: requestID,
    phase: "progress",
    name,
    payload: { percent: 15, stage: "normalizing text" }
  });

  const normalized = text
    .toLowerCase()
    .replace(/[^a-z0-9\s]+/g, " ")
    .split(/\s+/)
    .filter(Boolean);

  self.postMessage({
    id: requestID,
    phase: "progress",
    name,
    payload: { percent: 65, stage: "counting terms" }
  });

  const counts = new Map();
  let queryHits = 0;
  for (const word of normalized) {
    counts.set(word, (counts.get(word) || 0) + 1);
    if (query !== "" && word === query) {
      queryHits += 1;
    }
  }

  const topTerms = Array.from(counts.entries())
    .sort((left, right) => {
      if (right[1] !== left[1]) {
        return right[1] - left[1];
      }
      return left[0].localeCompare(right[0]);
    })
    .slice(0, 8)
    .map(([term, count]) => ({ term, count }));

  self.postMessage({
    id: requestID,
    phase: "result",
    name,
    payload: {
      totalWords: normalized.length,
      uniqueWords: counts.size,
      query,
      queryHits,
      topTerms
    }
  });
};
