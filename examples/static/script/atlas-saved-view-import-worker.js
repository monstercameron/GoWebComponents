// Atlas saved-view validation worker. It deliberately performs only shape
// validation; persistence remains owned by the server action.
self.onmessage = function (event) {
  var message = event && event.data ? event.data : {};
  if (message.phase !== "request") return;
  var payload = message.payload || {};
  var result = { valid: false, error: "Saved-view payload must be a JSON object with an items array." };
  self.postMessage({ id: message.id, phase: "progress", name: message.name, payload: { percent: 50, stage: "validate" } });
  try {
    var parsed = JSON.parse(String(payload.text || ""));
    if (parsed && Array.isArray(parsed.items)) {
      result = { valid: true, error: "", count: parsed.items.length };
    }
  } catch (error) {
    result.error = "Saved-view payload is not valid JSON.";
  }
  self.postMessage({ id: message.id, phase: "result", name: message.name, payload: result });
};
