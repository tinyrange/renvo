import assert from "node:assert/strict";
import test from "node:test";

import { fetchAsset } from "./asset-fetch.mjs";

const noPause = async () => {};

test("retries transient HTTP failures", async () => {
  const statuses = [503, 502, 200];
  const response = await fetchAsset("asset", {
    fetcher: async () => ({ status: statuses.shift(), ok: statuses.length === 0 }),
    pause: noPause,
  });
  assert.equal(response.status, 200);
  assert.equal(statuses.length, 0);
});

test("does not retry a permanent missing asset", async () => {
  let calls = 0;
  const response = await fetchAsset("missing", {
    fetcher: async () => { calls++; return { status: 404, ok: false }; },
    pause: noPause,
  });
  assert.equal(response.status, 404);
  assert.equal(calls, 1);
});

test("retries a temporary network failure", async () => {
  let calls = 0;
  const response = await fetchAsset("asset", {
    fetcher: async () => {
      calls++;
      if (calls === 1) throw new TypeError("network unavailable");
      return { status: 200, ok: true };
    },
    pause: noPause,
  });
  assert.equal(response.status, 200);
  assert.equal(calls, 2);
});
