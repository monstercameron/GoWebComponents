import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";

const source = await readFile(new URL("./desktop.js", import.meta.url), "utf8");
const desktop = await import(`data:text/javascript;base64,${Buffer.from(source).toString("base64")}`);
const { createDesktopTransport } = desktop;

function decode(parseReply) {
  return JSON.parse(parseReply);
}

test("non-RPC feature capabilities are bounded immutable snapshots, not invented methods", () => {
  const parseFeatures = ["native-menus"];
  const parseTransport = createDesktopTransport({ methods: {}, features: parseFeatures });
  parseFeatures.push("clipboard");
  assert.deepEqual(decode(parseTransport.capabilities()).data.features, ["native-menus"]);
  assert.equal(decode(parseTransport.start("desktop.menu.install", "[]")).code, "missing_export");
  for (const parseInvalid of [null, "native-menus", [null], [""], ["x".repeat(129)], Array(65).fill("native-menus")]) {
    assert.throws(() => createDesktopTransport({ methods: {}, features: parseInvalid }), /feature capabilities/);
  }
  parseTransport.close();
  assert.equal(decode(parseTransport.capabilities()).code, "disposed");
});

function makeEvents() {
  const parseListeners = new Map();
  let parseNextID = 0;
  return {
    On(parseTopic, parseHandler) {
      const parseID = ++parseNextID;
      parseListeners.set(parseID, { topic: parseTopic, handler: parseHandler });
      return () => parseListeners.delete(parseID);
    },
    emit(parseTopic, parseData) {
      for (const parseListener of parseListeners.values()) {
        if (parseListener.topic === parseTopic) parseListener.handler({ data: parseData });
      }
    },
    get size() { return parseListeners.size; },
  };
}

test("resolved, null, nested, bytes, time, and wide strings round-trip", async () => {
  const parseEvents = makeEvents();
  const parseTransport = createDesktopTransport({
    methods: {
      values: () => ({ nested: { nullValue: null, bytes: new Uint8Array([1, 2]), time: new Date("2026-09-08T00:00:00.000Z"), wide: "𐐷" } }),
      nothing: () => null,
    }, topics: [], events: parseEvents,
  });
  const parseID = decode(parseTransport.start("values", "[]")).data;
  await Promise.resolve();
  const parseValue = decode(parseTransport.poll(parseID)).data.data;
  assert.equal(parseValue.nested.nullValue, null);
  assert.deepEqual(parseValue.nested.bytes, { 0: 1, 1: 2 });
  assert.equal(parseValue.nested.time, "2026-09-08T00:00:00.000Z");
  assert.equal(parseValue.nested.wide, "𐐷");
  const parseNullID = decode(parseTransport.start("nothing", "[]")).data;
  await Promise.resolve();
  assert.equal(decode(parseTransport.poll(parseNullID)).data.data, null);
});

test("rejection, unknown names, and invalid arguments have stable error codes", async () => {
  const parseTransport = createDesktopTransport({
    methods: { fail: () => Promise.reject(new Error("bad native call")) }, events: makeEvents(),
  });
  assert.equal(decode(parseTransport.start("missing", "[]")).code, "missing_export");
  assert.equal(decode(parseTransport.start("fail", "{}")).code, "encode");
  const parseID = decode(parseTransport.start("fail", "[]")).data;
  await Promise.resolve();
  assert.equal(decode(parseTransport.poll(parseID)).data.code, "remote_error");
  assert.equal(decode(parseTransport.listen("missing-topic")).code, "missing_export");
});

test("pending cancellation calls the original promise cancel exactly once and ignores late completion", async () => {
  let parseResolve;
  let parseCancelCount = 0;
  const parsePromise = new Promise((parseDone) => { parseResolve = parseDone; });
  parsePromise.cancel = () => { parseCancelCount++; };
  const parseTransport = createDesktopTransport({ methods: { wait: () => parsePromise }, events: makeEvents() });
  const parseID = decode(parseTransport.start("wait", "[]")).data;
  assert.deepEqual(parseTransport.stats(), { requests: 1, subscriptions: 0 });
  assert.equal(decode(parseTransport.cancel(parseID)).data, null);
  assert.equal(decode(parseTransport.cancel(parseID)).data, null);
  assert.equal(parseCancelCount, 1);
  parseResolve("late");
  await Promise.resolve();
  assert.deepEqual(parseTransport.stats(), { requests: 0, subscriptions: 0 });
});

test("never-settling requests and listeners are bounded, reusable, and close safely", () => {
  let parseCancelCount = 0;
  const parseNever = new Promise(() => {});
  parseNever.cancel = () => { parseCancelCount++; };
  const parseEvents = makeEvents();
  const parseTransport = createDesktopTransport({ limit: 2, methods: { wait: () => parseNever }, topics: ["test.tick"], events: parseEvents });
  const parseFirst = decode(parseTransport.start("wait", "[]")).data;
  const parseSecond = decode(parseTransport.start("wait", "[]")).data;
  assert.equal(decode(parseTransport.start("wait", "[]")).code, "quota_exceeded");
  parseTransport.cancel(parseFirst);
  assert.equal(parseCancelCount, 1);
  assert.notEqual(decode(parseTransport.start("wait", "[]")).code, "quota_exceeded");
  const parseSub = decode(parseTransport.listen("test.tick")).data;
  const parseSub2 = decode(parseTransport.listen("test.tick")).data;
  assert.equal(decode(parseTransport.listen("test.tick")).code, "quota_exceeded");
  parseTransport.unlisten(parseSub);
  parseTransport.unlisten(parseSub);
  parseTransport.unlisten(parseSub2);
  assert.equal(parseEvents.size, 0);
  parseTransport.close();
  parseTransport.close();
  assert.deepEqual(parseTransport.stats(), { requests: 0, subscriptions: 0 });
  assert.equal(decode(parseTransport.poll(parseSecond)).code, "disposed");
});

test("event bursts coalesce to the latest value and unlisten is idempotent", () => {
  const parseEvents = makeEvents();
  const parseTransport = createDesktopTransport({ topics: ["test.tick"], events: parseEvents, methods: {} });
  const parseID = decode(parseTransport.listen("test.tick")).data;
  parseEvents.emit("test.tick", { sequence: 1 });
  parseEvents.emit("test.tick", { sequence: 2 });
  assert.deepEqual(decode(parseTransport.next(parseID)).data.data, { sequence: 2 });
  assert.deepEqual(decode(parseTransport.next(parseID)).data, { done: false });
  assert.equal(decode(parseTransport.unlisten(parseID)).data, null);
  assert.equal(decode(parseTransport.unlisten(parseID)).data, null);
});
