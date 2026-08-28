import assert from "node:assert/strict";
import test from "node:test";

import { hasDownloadableOutput, targetCapabilities, targetCapabilityHint, targetCapabilityTags } from "./target-capabilities.mjs";

test("WASI artifacts advertise browser execution", () => {
  const target = { name: "wasi/wasm32", output: "app.wasm", runnable: true };
  assert.deepEqual(targetCapabilities(target), { runsInBrowser: true, flashable: false });
  assert.deepEqual(targetCapabilityTags(target).map((tag) => tag.label), ["Runs in browser"]);
  assert.equal(targetCapabilityHint(target), "Run this target in the browser or download app.wasm.");
});

test("native artifacts do not claim an execution capability", () => {
  const target = { name: "linux/amd64", output: "app" };
  assert.deepEqual(targetCapabilities(target), { runsInBrowser: false, flashable: false });
  assert.deepEqual(targetCapabilityTags(target), []);
  assert.equal(targetCapabilityHint(target), "Build and download app for this target.");
});

test("board firmware advertises flashing without claiming browser execution", () => {
  const target = { name: "esp32/esp32c3", device: "esp32", output: "firmware.bin" };
  assert.deepEqual(targetCapabilities(target), { runsInBrowser: false, flashable: true });
  assert.deepEqual(targetCapabilityTags(target).map((tag) => tag.label), ["Flash over USB"]);
  assert.equal(targetCapabilityHint(target), "Flash this board over USB or download firmware.bin.");
});

test("RP2 firmware advertises monitor loading and UF2 download", () => {
  const target = { name: "pico/thumb", device: "rp2", output: "app.uf2" };
  assert.deepEqual(targetCapabilities(target), { runsInBrowser: false, flashable: true });
  assert.deepEqual(targetCapabilityTags(target).map((tag) => tag.label), ["Flash over USB"]);
  assert.equal(targetCapabilityHint(target), "Load through the resident USB monitor or download app.uf2.");
});

test("download availability follows the compiled output rather than a capability tag", () => {
  assert.equal(hasDownloadableOutput({ output: "app.exe" }), true);
  assert.equal(hasDownloadableOutput({}), false);
});
