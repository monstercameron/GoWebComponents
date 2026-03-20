self.postMessage({ phase: "ready", name: "bootstrap" });

self.onmessage = event => {
  const message = event.data || {};
  if (message.phase !== "request") {
    return;
  }

  const payload = message.payload || {};
  const requestID = String(message.id || "");
  const name = String(message.name || "validate-saved-view-import");
  const text = String(payload.text || "");

  const postProgress = (percent, stage) => {
    self.postMessage({
      id: requestID,
      phase: "progress",
      name,
      payload: { percent, stage }
    });
  };

  try {
    postProgress(15, "parsing payload");
    const parsed = JSON.parse(text || "{\"items\":[]}");
    const items = Array.isArray(parsed.items) ? parsed.items : null;
    if (!items) {
      throw new Error("Saved-view import payload must include an items array.");
    }

    postProgress(60, "validating entries");
    const scopes = new Set();
    const preview = [];
    const warnings = [];
    let invalidCount = 0;

    items.forEach((item, index) => {
      const current = item || {};
      const nameValue = String(current.name || "").trim();
      const scopeValue = String(current.scope || "").trim();
      const filtersJSON = String(current.filtersJSON || "").trim();

      if (nameValue !== "") {
        preview.push(nameValue);
      }
      if (scopeValue !== "") {
        scopes.add(scopeValue);
      } else {
        invalidCount += 1;
        warnings.push(`Item ${index + 1} is missing scope.`);
      }
      if (nameValue === "") {
        invalidCount += 1;
        warnings.push(`Item ${index + 1} is missing name.`);
      }
      if (filtersJSON !== "") {
        try {
          JSON.parse(filtersJSON);
        } catch (err) {
          invalidCount += 1;
          warnings.push(`Item ${index + 1} has invalid filtersJSON.`);
        }
      }
    });

    postProgress(100, "building summary");
    self.postMessage({
      id: requestID,
      phase: "result",
      name,
      payload: {
        valid: invalidCount === 0,
        itemCount: items.length,
        scopeCount: scopes.size,
        scopes: Array.from(scopes).sort(),
        preview: preview.slice(0, 5),
        invalidCount,
        warnings: warnings.slice(0, 8)
      }
    });
  } catch (error) {
    self.postMessage({
      id: requestID,
      phase: "error",
      name,
      error: error instanceof Error ? error.message : String(error)
    });
  }
};
