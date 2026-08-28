import assert from "node:assert/strict";
import test from "node:test";

import { PicoCMSISDAP, PicoDAPHotReloadSession, parsePicoDebugImage, planPicoPatches } from "./pico-cmsis-dap.mjs";

test("CMSIS-DAP claims the probe bulk interface", async () => {
  const device = new MockDevice();
  const usb = new MockUSB();
  const debug = new PicoCMSISDAP(device, usb, { initialize: async () => {} });
  await debug.open();
  assert.deepEqual(device.claimed, [1]);
  assert.match(debug.diagnostics(), /interface 1, endpoints OUT 2\/IN 1/);
  await debug.close();
  assert.equal(device.opened, false);
  assert.equal(usb.listeners.size, 0);
});

test("Pico debug ELF parsing accepts shared SRAM ARM images", () => {
  const image = parsePicoDebugImage(testELF(0x20020101, [{
    address: 0x20020000, data: Uint8Array.of(1, 2, 3, 4, 5), memorySize: 8,
  }]));
  assert.equal(image.entry, 0x20020101);
  assert.equal(image.stack, 0x20042000);
  assert.deepEqual(Array.from(image.segments[0].data), [1, 2, 3, 4, 5, 0, 0, 0]);
  assert.throws(() => parsePicoDebugImage(testELF(0x10000101, [{
    address: 0x10000000, data: Uint8Array.of(1, 2, 3, 4), memorySize: 4,
  }])), /shared RP2 SRAM/);
});

test("DPv3 initialization recovers a target left in dormant state", async () => {
  const debug = new PicoCMSISDAP(new MockDevice(), new MockUSB(), { initialize: async () => {} });
  debug.opened = true;
  debug.dap = { bulkOut: { packetSize: 64 }, bulkIn: { packetSize: 64 } };
  const commands = [];
  debug.command = async (source) => {
    commands.push(Array.from(source));
    const response = new Uint8Array(64);
    response[0] = source[0];
    response[1] = source[0] === 2 ? 1 : 0;
    return response;
  };
  let reads = 0;
  debug.dpRead = async () => {
    reads++;
    if (reads === 1) throw new Error("protocol error");
    if (reads === 2) return 0x4c013477;
    return 0xf0000040;
  };
  debug.dpWrite = async () => {};
  debug.apWrite = async () => {};
  await debug.initialize();
  assert.equal(debug.apNumber, 0x2000);
  assert.equal(debug.apRegisterBase, 0xd00);
  assert.ok(commands.some((command) => command[0] === 0x12 && command[1] === 224));
});

test("Pico hot reload writes only changed words and resets core state", async () => {
  const debug = new RecordingDebugger();
  const session = new PicoDAPHotReloadSession(debug);
  const first = testELF(0x20020101, [{ address: 0x20020000, data: Uint8Array.of(1, 2, 3, 4, 5, 6, 7, 8), memorySize: 8 }]);
  const initial = await session.update(first);
  assert.equal(initial.bytesWritten, 8);
  assert.deepEqual(debug.calls, ["open", "halt", "write", "fence", "sp", "xpsr", "pc", "resume"]);
  debug.calls.length = 0;
  assert.equal((await session.update(first)).unchanged, true);
  assert.deepEqual(debug.calls, []);
  const next = parsePicoDebugImage(testELF(0x20020101, [{ address: 0x20020000, data: Uint8Array.of(1, 2, 99, 4, 5, 6, 7, 8), memorySize: 8 }]));
  const old = parsePicoDebugImage(first);
  const patches = planPicoPatches(old, next);
  assert.equal(patches.length, 1);
  assert.equal(patches[0].address, 0x20020000);
  assert.deepEqual(Array.from(patches[0].data), [1, 2, 99, 4]);
});

class RecordingDebugger {
  constructor() { this.calls = []; }
  async open() { this.calls.push("open"); }
  async halt() { this.calls.push("halt"); }
  async writeMemory() { this.calls.push("write"); }
  async fenceI() { this.calls.push("fence"); }
  async setSP() { this.calls.push("sp"); }
  async setXPSR() { this.calls.push("xpsr"); }
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
    this.vendorId = 0x2e8a; this.productId = 0x000c; this.opened = false; this.configuration = null;
    this.configurations = [{ configurationValue: 1, interfaces: [{ interfaceNumber: 1, alternates: [{
      alternateSetting: 0, interfaceClass: 0xff, interfaceSubclass: 0, interfaceProtocol: 0,
      interfaceName: "CMSIS-DAP v2 Interface", endpoints: [
        { endpointNumber: 2, direction: "out", type: "bulk", packetSize: 64 },
        { endpointNumber: 1, direction: "in", type: "bulk", packetSize: 64 },
      ],
    }] }] }];
    this.claimed = [];
  }
  async open() { this.opened = true; }
  async close() { this.opened = false; }
  async selectConfiguration(value) { this.configuration = this.configurations.find((item) => item.configurationValue === value); }
  async claimInterface(number) { this.claimed.push(number); }
  async selectAlternateInterface() {}
}

function testELF(entry, segments) {
  const headerSize = 52, programSize = 32;
  let offset = headerSize + programSize * segments.length;
  const result = new Uint8Array(offset + segments.reduce((sum, segment) => sum + segment.data.length, 0));
  result.set([0x7f, 0x45, 0x4c, 0x46, 1, 1, 1]);
  const view = new DataView(result.buffer);
  view.setUint16(16, 2, true); view.setUint16(18, 40, true); view.setUint32(20, 1, true);
  view.setUint32(24, entry, true); view.setUint32(28, headerSize, true);
  view.setUint16(40, headerSize, true); view.setUint16(42, programSize, true); view.setUint16(44, segments.length, true);
  for (let index = 0; index < segments.length; index++) {
    const at = headerSize + index * programSize, segment = segments[index];
    view.setUint32(at, 1, true); view.setUint32(at + 4, offset, true);
    view.setUint32(at + 8, segment.address, true); view.setUint32(at + 12, segment.address, true);
    view.setUint32(at + 16, segment.data.length, true); view.setUint32(at + 20, segment.memorySize, true);
    view.setUint32(at + 24, 5, true); view.setUint32(at + 28, 4, true);
    result.set(segment.data, offset); offset += segment.data.length;
  }
  return result;
}
