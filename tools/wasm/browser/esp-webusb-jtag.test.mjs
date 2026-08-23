import assert from "node:assert/strict";
import test from "node:test";

import {
  ESPJTAGHotReloadSession, ESPWebUSBJTAG,
  parseJTAGLoadImage, planJTAGPatches, supportsESPWebUSBJTAG,
} from "./esp-webusb-jtag.mjs";

test("WebUSB JTAG claims only the vendor interface", async () => {
  const device = new MockDevice();
  const usb = new MockUSB();
  const debug = new ESPWebUSBJTAG(device, usb, { initialize: async () => {} });
  await debug.open();
  assert.deepEqual(device.claimed, [2]);
  assert.equal(device.opened, true);
  assert.match(debug.diagnostics(), /interface 2, endpoints OUT 2\/IN 3/);
  await debug.close();
  assert.equal(device.opened, false);
  assert.equal(usb.listeners.size, 0);
});

test("JTAG halt acknowledges a board reset before waiting for the hart", async () => {
  const writes = [];
  const debug = new ESPWebUSBJTAG(new MockDevice(), new MockUSB(), { initialize: async () => {} });
  debug.dmiWrite = async (address, data) => writes.push({ address, data });
  debug.dmiRead = async () => 1 << 9;
  await debug.halt();
  assert.equal(writes[0].data, 0x90000001);
});

test("JTAG hot reload writes the first image and only changed words thereafter", async () => {
  const debug = new RecordingDebugger();
  const progress = [];
  const session = new ESPJTAGHotReloadSession(debug, { progress: (value) => progress.push(value) });
  const first = testELF(0x40824100, [
    { address: 0x40824000, data: Uint8Array.from([1, 2, 3, 4, 5, 6, 7, 8]), memorySize: 8, flags: 5 },
    { address: 0x40800000, data: Uint8Array.from([9, 10, 11, 12]), memorySize: 12, flags: 6 },
  ]);
  const initial = await session.update(first);
  assert.deepEqual(initial, { entry: 0x40824100, patchCount: 2, bytesWritten: 12, unchanged: false });
  assert.deepEqual(debug.calls, ["halt", "write", "write", "fence", "pc", "resume"]);
  assert.equal(progress.at(-1), 1);

  debug.calls.length = 0;
  const unchanged = await session.update(first);
  assert.equal(unchanged.unchanged, true);
  assert.deepEqual(debug.calls, []);

  const changed = testELF(0x40824100, [{
    address: 0x40824000, data: Uint8Array.from([1, 2, 99, 4, 5, 6, 7, 8]), memorySize: 8, flags: 5,
  }, {
    address: 0x40800000, data: Uint8Array.from([9, 10, 11, 12]), memorySize: 12, flags: 6,
  }]);
  const delta = await session.update(changed);
  assert.equal(delta.bytesWritten, 4);
  assert.equal(debug.writes.at(-1).address, 0x40824000);
  assert.deepEqual(Array.from(debug.writes.at(-1).data), [1, 2, 99, 4]);
});

test("JTAG image parsing rejects an ordinary flash-linked C6 ELF", () => {
  assert.equal(supportsESPWebUSBJTAG("esp32c6/riscv32"), true);
  assert.equal(supportsESPWebUSBJTAG("esp32s3/xtensa_lx7"), false);
  const flashELF = testELF(0x42000100, [{
    address: 0x42000000, data: Uint8Array.from([1, 2, 3, 4]), memorySize: 4, flags: 5,
  }]);
  assert.throws(() => parseJTAGLoadImage(flashELF), /esp32c6-jtag\/riscv32/);
});

test("patch planning pads writes to JTAG word alignment", () => {
  const image = parseJTAGLoadImage(testELF(0x40824100, [{
    address: 0x40824000, data: Uint8Array.from([1, 2, 3, 4, 5]), memorySize: 8, flags: 5,
  }]));
  const patches = planJTAGPatches(undefined, image);
  assert.equal(patches.length, 1);
  assert.deepEqual(Array.from(patches[0].data), [1, 2, 3, 4, 5, 0, 0, 0]);
  assert.deepEqual(planJTAGPatches(image, image), []);
});

class RecordingDebugger {
  constructor() { this.calls = []; this.writes = []; }
  async open() {}
  async halt() { this.calls.push("halt"); }
  async writeMemory(address, data) {
    this.calls.push("write");
    this.writes.push({ address, data: data.slice() });
  }
  async fenceI() { this.calls.push("fence"); }
  async setPC() { this.calls.push("pc"); }
  async resume() { this.calls.push("resume"); }
  async close() {}
}

class MockUSB {
  constructor() { this.listeners = new Set(); }
  addEventListener(name, listener) { if (name === "disconnect") this.listeners.add(listener); }
  removeEventListener(name, listener) { if (name === "disconnect") this.listeners.delete(listener); }
}

class MockDevice {
  constructor() {
    this.vendorId = 0x303a;
    this.productId = 0x1001;
    this.opened = false;
    this.configuration = null;
    this.configurations = [configuration()];
    this.claimed = [];
  }
  async open() { this.opened = true; }
  async close() { this.opened = false; }
  async selectConfiguration(value) { this.configuration = this.configurations.find((item) => item.configurationValue === value); }
  async claimInterface(number) { this.claimed.push(number); }
  async selectAlternateInterface() {}
}

function configuration() {
  return {
    configurationValue: 1,
    interfaces: [
      { interfaceNumber: 0, alternates: [{ alternateSetting: 0, interfaceClass: 2, interfaceSubclass: 2, interfaceProtocol: 0, endpoints: [] }] },
      { interfaceNumber: 1, alternates: [{ alternateSetting: 0, interfaceClass: 10, interfaceSubclass: 2, interfaceProtocol: 0, endpoints: [
        { endpointNumber: 1, direction: "out", type: "bulk", packetSize: 64 },
        { endpointNumber: 1, direction: "in", type: "bulk", packetSize: 64 },
      ] }] },
      { interfaceNumber: 2, alternates: [{ alternateSetting: 0, interfaceClass: 0xff, interfaceSubclass: 0xff, interfaceProtocol: 1, endpoints: [
        { endpointNumber: 2, direction: "out", type: "bulk", packetSize: 64 },
        { endpointNumber: 3, direction: "in", type: "bulk", packetSize: 64 },
      ] }] },
    ],
  };
}

function testELF(entry, segments) {
  const headerSize = 52;
  const programSize = 32;
  let offset = headerSize + programSize * segments.length;
  const length = offset + segments.reduce((sum, segment) => sum + segment.data.length, 0);
  const result = new Uint8Array(length);
  result.set([0x7f, 0x45, 0x4c, 0x46, 1, 1, 1]);
  const view = new DataView(result.buffer);
  view.setUint16(16, 2, true);
  view.setUint16(18, 243, true);
  view.setUint32(20, 1, true);
  view.setUint32(24, entry, true);
  view.setUint32(28, headerSize, true);
  view.setUint16(40, headerSize, true);
  view.setUint16(42, programSize, true);
  view.setUint16(44, segments.length, true);
  for (let index = 0; index < segments.length; index++) {
    const at = headerSize + index * programSize;
    const segment = segments[index];
    view.setUint32(at, 1, true);
    view.setUint32(at + 4, offset, true);
    view.setUint32(at + 8, segment.address, true);
    view.setUint32(at + 12, segment.address, true);
    view.setUint32(at + 16, segment.data.length, true);
    view.setUint32(at + 20, segment.memorySize, true);
    view.setUint32(at + 24, segment.flags, true);
    view.setUint32(at + 28, 4, true);
    result.set(segment.data, offset);
    offset += segment.data.length;
  }
  return result;
}
