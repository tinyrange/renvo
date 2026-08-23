import assert from "node:assert/strict";
import test from "node:test";

import {
  ESPWebUSBPort, preferredESPTransport, selectAuthorizedESPUSBDevice,
  supportsESPWebUSB, supportsESPWebUSBPlatform,
} from "./esp-webusb.mjs";

test("ESP WebUSB discovers CDC interfaces and streams partial transfers", async () => {
  const usb = new MockUSB();
  const device = new MockDevice();
  const port = new ESPWebUSBPort(device, usb);

  await port.open({ baudRate: 230400 });
  assert.deepEqual(device.claimed, [0, 1]);
  assert.equal(device.controls[0].setup.request, 0x20);
  assert.deepEqual(Array.from(device.controls[0].data), [0x00, 0x84, 0x03, 0x00, 0, 0, 8]);

  await port.setSignals({ dataTerminalReady: true, requestToSend: false });
  await port.setSignals({ requestToSend: true });
  assert.deepEqual(device.controls.slice(1).map(({ setup }) => setup.value), [1, 3]);

  const writer = port.writable.getWriter();
  await writer.write(Uint8Array.from([1, 2, 3, 4, 5]));
  writer.releaseLock();
  assert.deepEqual(device.writes.map((part) => Array.from(part)), [[1, 2], [3, 4], [5]]);

  device.reads.push(Uint8Array.from([6, 7, 8]));
  const reader = port.readable.getReader();
  const result = await reader.read();
  assert.deepEqual(Array.from(result.value), [6, 7, 8]);
  await reader.cancel();
  reader.releaseLock();

  await port.close();
  assert.equal(device.opened, false);
  assert.equal(usb.listeners.size, 0);

  await port.open();
  assert.equal(device.opened, true);
  assert.deepEqual(device.claimed, [0, 1, 0, 1]);
  await port.close();
});

test("ESP WebUSB rejects a host-owned CDC control interface with desktop guidance", async () => {
  const device = new MockDevice();
  device.failControlClaim = true;
  const port = new ESPWebUSBPort(device, new MockUSB());
  await assert.rejects(() => port.open(), /use WebSerial on desktop/);
  assert.deepEqual(device.claimed, []);
  assert.equal(device.opened, false);
});

test("ESP WebUSB disconnect rejects a pending read and permits cleanup", async () => {
  const usb = new MockUSB();
  const device = new MockDevice();
  const port = new ESPWebUSBPort(device, usb);
  await port.open();
  device.pendingRead = new Promise(() => {});
  const reader = port.readable.getReader();
  const pending = reader.read();
  usb.disconnect(device);
  await assert.rejects(pending, /disconnected/);
  assert.equal(port.readable, null);
  await port.close();
});

test("ESP WebUSB target support is explicit", () => {
  assert.equal(supportsESPWebUSB("esp32c6/riscv32"), true);
  assert.equal(supportsESPWebUSB("esp32s3/xtensa_lx7"), true);
  assert.equal(supportsESPWebUSB("linux/amd64"), false);
});

test("flash transport preference follows platform, availability, and saved choice", () => {
  assert.equal(preferredESPTransport({ android: true, webSerial: true, webUSB: true }), "webusb");
  assert.equal(preferredESPTransport({ android: false, webSerial: true, webUSB: true }), "webserial");
  assert.equal(preferredESPTransport({ android: false, webSerial: false, webUSB: true }), "webusb");
  assert.equal(preferredESPTransport({ saved: "webusb", android: false, webSerial: true, webUSB: true }), "webusb");
  assert.equal(supportsESPWebUSBPlatform({ platform: "Android" }), true);
  assert.equal(supportsESPWebUSBPlatform({ platform: "Linux", maxTouchPoints: 10, coarsePointer: true }), true);
  assert.equal(supportsESPWebUSBPlatform({ platform: "MacIntel", userAgent: "Mozilla/5.0 (Macintosh)" }), false);
});

test("authorized WebUSB devices collapse duplicate identities but preserve real choices", () => {
  const board = (serialNumber, opened = false) => ({
    vendorId: 0x303a, productId: 0x1001, serialNumber, opened,
  });
  const oldIdentity = board("same-board");
  const currentIdentity = board("same-board", true);
  assert.equal(selectAuthorizedESPUSBDevice([oldIdentity, currentIdentity]), currentIdentity);
  assert.equal(selectAuthorizedESPUSBDevice([board("board-a"), board("board-b")]), undefined);
  assert.equal(selectAuthorizedESPUSBDevice([{ vendorId: 1, productId: 2 }]), undefined);
});

class MockUSB {
  constructor() { this.listeners = new Set(); }
  async getDevices() { return []; }
  addEventListener(name, listener) { if (name === "disconnect") this.listeners.add(listener); }
  removeEventListener(name, listener) { if (name === "disconnect") this.listeners.delete(listener); }
  disconnect(device) { for (const listener of this.listeners) listener({ device }); }
}

class MockDevice {
  constructor() {
    this.vendorId = 0x303a;
    this.productId = 0x1001;
    this.opened = false;
    this.configuration = null;
    this.configurations = [configuration()];
    this.claimed = [];
    this.released = [];
    this.controls = [];
    this.writes = [];
    this.reads = [];
  }

  async open() { this.opened = true; }
  async close() { this.opened = false; }
  async selectConfiguration(value) {
    this.configuration = this.configurations.find((item) => item.configurationValue === value);
  }
  async claimInterface(number) {
    if (number === 0 && this.failControlClaim) throw new DOMException("interface busy", "NetworkError");
    this.claimed.push(number);
  }
  async releaseInterface(number) { this.released.push(number); }
  async selectAlternateInterface() {}
  async controlTransferOut(setup, data = new Uint8Array()) {
    const copy = new Uint8Array(data.buffer, data.byteOffset, data.byteLength).slice();
    this.controls.push({ setup: { ...setup }, data: copy });
    return { status: "ok", bytesWritten: copy.byteLength };
  }
  async transferOut(_endpoint, data) {
    const count = Math.min(2, data.byteLength);
    this.writes.push(data.slice(0, count));
    return { status: "ok", bytesWritten: count };
  }
  async transferIn() {
    if (this.pendingRead) return this.pendingRead;
    const data = this.reads.shift() || new Uint8Array();
    return { status: "ok", data: new DataView(data.buffer, data.byteOffset, data.byteLength) };
  }
  async clearHalt() {}
}

function configuration() {
  return {
    configurationValue: 1,
    interfaces: [
      { interfaceNumber: 0, alternates: [{
        alternateSetting: 0, interfaceClass: 0x02, interfaceSubclass: 0x02,
        interfaceProtocol: 0, endpoints: [],
      }] },
      { interfaceNumber: 1, alternates: [{
        alternateSetting: 0, interfaceClass: 0x0a, interfaceSubclass: 0x02,
        interfaceProtocol: 0, endpoints: [
          { endpointNumber: 1, direction: "out", type: "bulk", packetSize: 64 },
          { endpointNumber: 1, direction: "in", type: "bulk", packetSize: 64 },
        ],
      }] },
      { interfaceNumber: 2, alternates: [{
        alternateSetting: 0, interfaceClass: 0xff, interfaceSubclass: 0xff,
        interfaceProtocol: 1, endpoints: [
          { endpointNumber: 2, direction: "out", type: "bulk", packetSize: 64 },
          { endpointNumber: 3, direction: "in", type: "bulk", packetSize: 64 },
        ],
      }] },
    ],
  };
}
