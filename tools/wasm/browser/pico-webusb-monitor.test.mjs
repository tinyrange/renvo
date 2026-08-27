import assert from "node:assert/strict";
import test from "node:test";

import { PicoMonitorHotReloadSession, PicoWebUSBMonitor } from "./pico-webusb-monitor.mjs";

test("Renvo Pico monitor claims its vendor bulk interface", async () => {
  const device = new MockMonitor();
  const monitor = new PicoWebUSBMonitor(device, new MockUSB());
  await monitor.open();
  assert.deepEqual(device.claimed, [2]);
  assert.equal(device.out[0][4], 1);
  assert.equal(monitor.fastWrite, true);
});

test("Renvo Pico monitor falls back to acknowledged writes with an older monitor", async () => {
  const device = new MockMonitor(false);
  const monitor = new PicoWebUSBMonitor(device, new MockUSB());
  await monitor.open();
  await monitor.write(0x20010000, new Uint8Array(4));
  assert.equal(monitor.fastWrite, false);
  assert.equal(device.out.at(-1)[4], 3);
  assert.equal(device.inCount, 2);
});

test("Renvo Pico monitor chunks ELF patches and commits the Thumb entry", async () => {
  const monitor = new RecordingMonitor();
  const session = new PicoMonitorHotReloadSession(monitor);
  const image = testELF(0x20020101, 0x20020000, new Uint8Array(120).map((_, index) => index));
  const result = await session.update(image);
  assert.equal(result.bytesWritten, 120);
  assert.equal(result.patchCount, 3);
  assert.deepEqual(monitor.calls.map((call) => call[0]), ["open", 2, "write", "write", "write", 4]);
  assert.equal(monitor.calls.at(-1)[1], 0x20020101);
  assert.equal((await session.update(image)).unchanged, true);
});

class MockUSB {
  addEventListener() {}
  removeEventListener() {}
}

class MockMonitor {
  constructor(fastWrite = true) {
    this.fastWrite = fastWrite;
    this.inCount = 0;
    this.vendorId = 0xcafe; this.productId = 0x4021; this.opened = false; this.configuration = null; this.claimed = []; this.out = [];
    this.configurations = [{ configurationValue: 1, interfaces: [{ interfaceNumber: 2, alternates: [{
      alternateSetting: 0, interfaceClass: 0xff, interfaceSubclass: 0x52, interfaceProtocol: 1,
      endpoints: [{ endpointNumber: 1, direction: "out", type: "bulk", packetSize: 64 }, { endpointNumber: 1, direction: "in", type: "bulk", packetSize: 64 }],
    }] }] }];
  }
  async open() { this.opened = true; }
  async close() { this.opened = false; }
  async selectConfiguration(value) { this.configuration = this.configurations.find((item) => item.configurationValue === value); }
  async claimInterface(number) { this.claimed.push(number); }
  async transferOut(_endpoint, data) { this.out.push(data); return { status: "ok" }; }
  async transferIn() {
    this.inCount++;
    const response = new Uint8Array(64); response.set([0x52, 0x4e, 0x56, 0x32, this.out.at(-1)[4], 0]);
    new DataView(response.buffer).setUint32(20, this.fastWrite ? 1 : 0, true);
    return { status: "ok", data: new DataView(response.buffer) };
  }
}

class RecordingMonitor {
  constructor() { this.calls = []; }
  async open() { this.calls.push(["open"]); }
  async command(operation, address, data = new Uint8Array()) { this.calls.push([operation, address, data.length]); }
  async write(address, data) { this.calls.push(["write", address, data.length]); }
  async close() {}
}

function testELF(entry, address, contents) {
  const result = new Uint8Array(84 + contents.length);
  result.set([0x7f, 0x45, 0x4c, 0x46, 1, 1, 1]);
  const view = new DataView(result.buffer);
  view.setUint16(16, 2, true); view.setUint16(18, 40, true); view.setUint32(20, 1, true);
  view.setUint32(24, entry, true); view.setUint32(28, 52, true);
  view.setUint16(40, 52, true); view.setUint16(42, 32, true); view.setUint16(44, 1, true);
  view.setUint32(52, 1, true); view.setUint32(56, 84, true); view.setUint32(60, address, true);
  view.setUint32(64, address, true); view.setUint32(68, contents.length, true); view.setUint32(72, contents.length, true);
  view.setUint32(76, 5, true); view.setUint32(80, 4, true); result.set(contents, 84);
  return result;
}
