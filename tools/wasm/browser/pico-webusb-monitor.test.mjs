import assert from "node:assert/strict";
import test from "node:test";

import { formatPicoMonitorInfo, PicoMonitorHotReloadSession, PicoWebUSBMonitor } from "./pico-webusb-monitor.mjs";

test("Renvo Pico monitor claims its vendor bulk interface", async () => {
  const device = new MockMonitor();
  const monitor = new PicoWebUSBMonitor(device, new MockUSB());
  await monitor.open();
  assert.deepEqual(device.claimed, [2]);
  assert.equal(device.out[0][4], 1);
  assert.equal(device.out[0][6], 1);
  assert.equal(new DataView(device.out[0].buffer).getUint32(8, true), 0x00010000);
  assert.equal(monitor.fastWrite, true);
  assert.deepEqual(monitor.getInfo(), {
    usbVendorId: 0xcafe, usbProductId: 0x4021,
    protocolMajor: 1, protocolMinor: 0, generation: 7,
    reloadStart: 0x20010000, reloadEnd: 0x20040000,
    capabilities: 1, monitorVersion: 0x00010000, chip: 0x2040,
    clientVersion: 0x00010000,
  });
});

test("Renvo Pico monitor falls back when fast writes are unavailable", async () => {
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

test("Renvo Pico monitor rejects an unversioned monitor handshake", async () => {
  const device = new MockMonitor(true, 0);
  const monitor = new PicoWebUSBMonitor(device, new MockUSB());
  await assert.rejects(monitor.open(), /monitor protocol 0\.0.*install the current monitor/);
});

test("Renvo Pico monitor formats negotiated version data", () => {
  assert.equal(formatPicoMonitorInfo({ monitorVersion: 0x00010203, protocolMajor: 1, protocolMinor: 4, chip: 0x2350, generation: 9 }),
    "firmware 1.2.3 · protocol 1.4 · RP2350 · generation 9");
});

class MockUSB {
  addEventListener() {}
  removeEventListener() {}
}

class MockMonitor {
  constructor(fastWrite = true, protocolMajor = 1) {
    this.fastWrite = fastWrite;
    this.protocolMajor = protocolMajor;
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
    const response = new Uint8Array(64); response.set([0x52, 0x4e, 0x56, 0x32, this.out.at(-1)[4], 0, this.protocolMajor, 0]);
    const view = new DataView(response.buffer);
    view.setUint32(8, 7, true); view.setUint32(12, 0x20010000, true); view.setUint32(16, 0x20040000, true);
    view.setUint32(20, this.fastWrite ? 1 : 0, true); view.setUint32(24, 0x00010000, true);
    view.setUint32(28, 0x2040, true); view.setUint32(32, viewFor(this.out.at(-1)).getUint32(8, true), true);
    return { status: "ok", data: new DataView(response.buffer) };
  }
}

function viewFor(bytes) { return new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength); }

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
