const RASPBERRY_PI_VENDOR_ID = 0x2e8a;
const DEBUG_PROBE_PRODUCT_ID = 0x000c;

const DAP_INFO = 0x00;
const DAP_CONNECT = 0x02;
const DAP_DISCONNECT = 0x03;
const DAP_TRANSFER_CONFIGURE = 0x04;
const DAP_TRANSFER = 0x05;
const DAP_SWJ_CLOCK = 0x11;
const DAP_SWJ_SEQUENCE = 0x12;
const DAP_SWD_CONFIGURE = 0x13;

const DP_ABORT = 0x00;
const DP_CTRL_STAT = 0x04;
const DP_SELECT = 0x08;
const DHCSR = 0xe000edf0;
const DCRSR = 0xe000edf4;
const DCRDR = 0xe000edf8;
const DHCSR_KEY = 0xa05f0000;
const RP2_SRAM_START = 0x20000000;
const RP2_SRAM_END = 0x20042000;

const delay = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

export async function requestPicoDebugProbe() {
  if (!globalThis.navigator?.usb) {
    throw new Error("WebUSB is unavailable. Open Renvo over HTTPS in a Chromium-based browser.");
  }
  const permitted = await navigator.usb.getDevices?.() || [];
  let device = permitted.find(isPicoDebugProbe);
  if (!device) {
    device = await navigator.usb.requestDevice({ filters: [{ vendorId: RASPBERRY_PI_VENDOR_ID, productId: DEBUG_PROBE_PRODUCT_ID }] });
  }
  return new PicoCMSISDAP(device, navigator.usb);
}

export class PicoCMSISDAP {
  constructor(device, usb = globalThis.navigator?.usb, options = {}) {
    this.transport = "webusb-cmsis-dap";
    this.device = device;
    this.usb = usb;
    this.initializeTarget = options.initialize || ((debug) => debug.initialize());
    this.opened = false;
    this.disconnected = false;
    this.select = undefined;
    this.handleDisconnect = (event) => {
      if (event.device !== this.device) return;
      this.disconnected = true;
      this.opened = false;
    };
    this.usb?.addEventListener?.("disconnect", this.handleDisconnect);
  }

  getInfo() { return { usbVendorId: this.device.vendorId, usbProductId: this.device.productId }; }
  canReopen() { return !this.disconnected; }

  diagnostics() {
    return `CMSIS-DAP interface ${this.dap?.iface?.interfaceNumber ?? "?"}, endpoints OUT ${this.dap?.bulkOut?.endpointNumber ?? "?"}/IN ${this.dap?.bulkIn?.endpointNumber ?? "?"}`;
  }

  async open() {
    if (this.opened) return;
    this.disconnected = false;
    try {
      if (!this.device.opened) await this.device.open();
      const selected = findCMSISDAPConfiguration(this.device.configurations);
      if (!selected) throw new Error("device does not expose a CMSIS-DAP v2 bulk interface");
      if (this.device.configuration?.configurationValue !== selected.configuration.configurationValue) {
        await this.device.selectConfiguration(selected.configuration.configurationValue);
      }
      this.dap = findCMSISDAPConfiguration([this.device.configuration]) || selected;
      await this.device.claimInterface(this.dap.iface.interfaceNumber);
      if (this.dap.alternate.alternateSetting !== 0) {
        await this.device.selectAlternateInterface(this.dap.iface.interfaceNumber, this.dap.alternate.alternateSetting);
      }
      this.opened = true;
      await this.initializeTarget(this);
    } catch (error) {
      await this.close();
      throw new Error(`could not open Pico Debug Probe: ${error.message || error}`);
    }
  }

  async close() {
    if (this.opened) {
      try { await this.command(Uint8Array.of(DAP_DISCONNECT)); } catch {}
    }
    this.opened = false;
    this.select = undefined;
    try { if (this.device.opened) await this.device.close(); } catch {}
    this.usb?.removeEventListener?.("disconnect", this.handleDisconnect);
  }

  async command(source) {
    if (!this.opened) throw new Error("CMSIS-DAP transport is not open");
    const packetSize = this.dap.bulkOut.packetSize || 64;
    if (source.length > packetSize) throw new Error("CMSIS-DAP command exceeds the USB packet size");
    const packet = new Uint8Array(packetSize);
    packet.set(source);
    const sent = await this.device.transferOut(this.dap.bulkOut.endpointNumber, packet);
    if (sent.status && sent.status !== "ok") throw new Error(`CMSIS-DAP USB write ${sent.status}`);
    const received = await this.device.transferIn(this.dap.bulkIn.endpointNumber, this.dap.bulkIn.packetSize || 64);
    if (received.status && received.status !== "ok") throw new Error(`CMSIS-DAP USB read ${received.status}`);
    const result = new Uint8Array(received.data.buffer, received.data.byteOffset, received.data.byteLength);
    if (result[0] !== source[0]) throw new Error(`CMSIS-DAP response mismatch for command 0x${source[0].toString(16)}`);
    return result;
  }

  async initialize() {
    const connected = await this.command(Uint8Array.of(DAP_CONNECT, 1));
    if (connected[1] !== 1) throw new Error("debug probe could not connect to the SWD target");
    await this.expectOK(Uint8Array.of(DAP_SWJ_CLOCK, 0x00, 0x09, 0x3d, 0x00), "set the SWD clock");
    await this.expectOK(Uint8Array.of(DAP_TRANSFER_CONFIGURE, 0, 64, 0, 0, 0), "configure SWD transfers");
    await this.expectOK(Uint8Array.of(DAP_SWD_CONFIGURE, 0), "configure SWD");
    await this.expectOK(Uint8Array.of(DAP_SWJ_SEQUENCE, 136,
      0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
      0x9e, 0xe7,
      0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
      0x00), "switch the target to SWD");
    let dpidr;
    try {
      dpidr = await this.dpRead(0x00);
    } catch {
      // DPv3 targets and probes previously closed by another debugger can be
      // left in dormant state. Use the ADIv6 selection-alert path as fallback.
      await this.expectOK(Uint8Array.of(DAP_SWJ_SEQUENCE, 40, 0xff, 0x75, 0x77, 0x77, 0x67), "enter SWD dormant state");
      await this.expectOK(Uint8Array.of(DAP_SWJ_SEQUENCE, 224,
        0xff, 0x92, 0xf3, 0x09, 0x62, 0x95, 0x2d, 0x85, 0x86,
        0xe9, 0xaf, 0xdd, 0xe3, 0xa2, 0x0e, 0xbc, 0x19,
        0xa0, 0xf1, 0xff,
        0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00), "activate SWD from dormant state");
      dpidr = await this.dpRead(0x00);
    }
    const version = (dpidr >>> 12) & 15;
    this.apNumber = version >= 3 ? 0x2000 : 0;
    this.apRegisterBase = version >= 3 ? 0xd00 : 0;
    await this.dpWrite(DP_ABORT, 0x1e);
    await this.dpWrite(DP_CTRL_STAT, 0x50000000);
    let status = 0;
    for (let attempt = 0; attempt < 100; attempt++) {
      status = await this.dpRead(DP_CTRL_STAT);
      if (((status & 0xa0000000) >>> 0) === 0xa0000000) break;
      await delay(1);
    }
    if (((status & 0xa0000000) >>> 0) !== 0xa0000000) throw new Error(`SWD debug power-up timed out (CTRL/STAT 0x${status.toString(16)})`);
    await this.setAPSelect(this.apRegisterBase);
    await this.apWrite(this.apRegisterBase, 0x23000052);
  }

  async expectOK(command, action) {
    const response = await this.command(command);
    if (response[1] !== 0) throw new Error(`CMSIS-DAP could not ${action}`);
  }

  async transfer(request, value) {
    const write = (request & 2) === 0;
    const packet = new Uint8Array(write ? 8 : 4);
    packet.set([DAP_TRANSFER, 0, 1, request]);
    if (write) new DataView(packet.buffer).setUint32(4, value >>> 0, true);
    const response = await this.command(packet);
    if (response[1] !== 1 || (response[2] & 7) !== 1) {
      throw new Error(`SWD transfer failed (count ${response[1] || 0}, status 0x${(response[2] || 0).toString(16)})`);
    }
    return write ? undefined : new DataView(response.buffer, response.byteOffset, response.byteLength).getUint32(3, true);
  }

  dpRead(address) { return this.transfer(2 | (address & 12)); }
  dpWrite(address, value) { return this.transfer(address & 12, value); }

  async setAPSelect(register) {
    const select = (this.apNumber | (register & 0xff0)) >>> 0;
    if (this.select === select) return;
    await this.dpWrite(DP_SELECT, select);
    this.select = select;
  }

  async apRead(register) {
    await this.setAPSelect(register);
    return this.transfer(3 | (register & 12));
  }

  async apWrite(register, value) {
    await this.setAPSelect(register);
    await this.transfer(1 | (register & 12), value);
  }

  async writeMemory(address, source) {
    const data = bytes(source);
    if (!data.length) return;
    if ((address & 3) !== 0 || (data.length & 3) !== 0) throw new Error("SWD memory writes must be word aligned");
    const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
    const csw = this.apRegisterBase;
    const tar = csw + 4;
    const drw = csw + 12;
    await this.apWrite(csw, 0x23000052);
    for (let offset = 0; offset < data.length;) {
      const batchEnd = Math.min(data.length, offset + (0x400 - ((address + offset) & 0x3ff)));
      await this.apWrite(tar, (address + offset) >>> 0);
      while (offset < batchEnd) {
        await this.apWrite(drw, view.getUint32(offset, true));
        offset += 4;
      }
    }
  }

  async readWord(address) {
    const base = this.apRegisterBase;
    await this.apWrite(base, 0x23000052);
    await this.apWrite(base + 4, address >>> 0);
    return this.apRead(base + 12);
  }

  async halt() {
    await this.writeWord(DHCSR, DHCSR_KEY | 3);
    await this.waitDHCSR(1 << 17, "halt");
  }

  async resume() {
    await this.writeWord(DHCSR, DHCSR_KEY | 1);
    for (let attempt = 0; attempt < 100; attempt++) {
      if (((await this.readWord(DHCSR)) & (1 << 17)) === 0) return;
      await delay(1);
    }
    throw new Error("Cortex-M core did not resume");
  }

  async waitDHCSR(mask, action) {
    let status = 0;
    for (let attempt = 0; attempt < 100; attempt++) {
      status = await this.readWord(DHCSR);
      if ((status & mask) === mask) return status;
      await delay(1);
    }
    throw new Error(`Cortex-M ${action} timed out (DHCSR 0x${status.toString(16)})`);
  }

  async writeWord(address, value) {
    const data = new Uint8Array(4);
    new DataView(data.buffer).setUint32(0, value >>> 0, true);
    await this.writeMemory(address, data);
  }

  async writeCoreRegister(register, value) {
    await this.writeWord(DCRDR, value);
    await this.writeWord(DCRSR, 0x10000 | register);
    await this.waitDHCSR(1 << 16, "register write");
  }

  setPC(address) { return this.writeCoreRegister(15, address | 1); }
  setSP(address) { return this.writeCoreRegister(17, address); }
  setXPSR(value = 0x01000000) { return this.writeCoreRegister(16, value); }
  async fenceI() {}
}

export class PicoDAPHotReloadSession {
  constructor(debug, options = {}) {
    this.debug = debug;
    this.progress = options.progress || (() => {});
    this.previous = undefined;
  }

  async update(source) {
    const image = parsePicoDebugImage(source);
    const patches = planPicoPatches(this.previous, image);
    if (!patches.length && this.previous) {
      return { entry: image.entry, patchCount: 0, bytesWritten: 0, unchanged: true };
    }
    await this.debug.open();
    await this.debug.halt();
    let written = 0;
    const total = patches.reduce((sum, patch) => sum + patch.data.length, 0);
    for (const patch of patches) {
      await this.debug.writeMemory(patch.address, patch.data);
      written += patch.data.length;
      this.progress(total ? written / total : 1);
    }
    await this.debug.fenceI();
    await this.debug.setSP(image.stack);
    await this.debug.setXPSR();
    await this.debug.setPC(image.entry);
    await this.debug.resume();
    this.previous = image;
    this.progress(1);
    return { entry: image.entry, patchCount: patches.length, bytesWritten: written, unchanged: false };
  }

  close() { return this.debug.close(); }
}

export function parsePicoDebugImage(source) {
  const data = bytes(source);
  if (data.length < 52 || data[0] !== 0x7f || data[1] !== 0x45 || data[2] !== 0x4c || data[3] !== 0x46) {
    throw new Error("Pico probe loading requires an ELF image");
  }
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
  if (data[4] !== 1 || data[5] !== 1 || view.getUint16(18, true) !== 40) {
    throw new Error("Pico probe loading requires a 32-bit little-endian ARM ELF");
  }
  const entry = view.getUint32(24, true);
  const headerAt = view.getUint32(28, true);
  const headerSize = view.getUint16(42, true);
  const count = view.getUint16(44, true);
  const segments = [];
  for (let index = 0; index < count; index++) {
    const at = headerAt + index * headerSize;
    if (at + 32 > data.length) throw new Error("Pico debug ELF has a truncated program header");
    if (view.getUint32(at, true) !== 1) continue;
    const fileAt = view.getUint32(at + 4, true);
    const address = view.getUint32(at + 8, true);
    const fileSize = view.getUint32(at + 16, true);
    const memorySize = view.getUint32(at + 20, true);
    if (fileAt + fileSize > data.length || memorySize < fileSize || address < RP2_SRAM_START || address + memorySize > RP2_SRAM_END) {
      throw new Error("Pico debug ELF contains a segment outside shared RP2 SRAM");
    }
    const contents = new Uint8Array((memorySize + 3) & ~3);
    contents.set(data.subarray(fileAt, fileAt + fileSize));
    segments.push({ address, data: contents });
  }
  if (!segments.length || entry < RP2_SRAM_START || entry >= RP2_SRAM_END) {
    throw new Error("Pico debug ELF is not linked for the rp2-debug/thumb SRAM target");
  }
  return { entry, stack: RP2_SRAM_END, segments };
}

export function planPicoPatches(previous, next) {
  if (!previous) return next.segments.map((segment) => ({ address: segment.address, data: segment.data.slice() }));
  const oldWords = imageWords(previous);
  const patches = [];
  for (const segment of next.segments) {
    let start = -1;
    for (let offset = 0; offset < segment.data.length; offset += 4) {
      const address = segment.address + offset;
      const changed = oldWords.get(address) !== wordAt(segment.data, offset);
      if (changed && start < 0) start = offset;
      if (!changed && start >= 0) {
        patches.push({ address: segment.address + start, data: segment.data.slice(start, offset) });
        start = -1;
      }
    }
    if (start >= 0) patches.push({ address: segment.address + start, data: segment.data.slice(start) });
  }
  return patches;
}

function imageWords(image) {
  const result = new Map();
  for (const segment of image.segments) {
    for (let offset = 0; offset < segment.data.length; offset += 4) result.set(segment.address + offset, wordAt(segment.data, offset));
  }
  return result;
}

function wordAt(data, offset) {
  return new DataView(data.buffer, data.byteOffset + offset, 4).getUint32(0, true);
}

function isPicoDebugProbe(device) {
  return device.vendorId === RASPBERRY_PI_VENDOR_ID && device.productId === DEBUG_PROBE_PRODUCT_ID;
}

function findCMSISDAPConfiguration(configurations = []) {
  for (const configuration of configurations) {
    for (const iface of configuration.interfaces || []) {
      for (const alternate of iface.alternates || []) {
        const bulkOut = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "out");
        const bulkIn = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "in");
        const named = /CMSIS-DAP/i.test(alternate.interfaceName || "");
        if (bulkOut && bulkIn && (named || alternate.interfaceClass === 0xff)) {
          return { configuration, iface, alternate, bulkOut, bulkIn };
        }
      }
    }
  }
  return undefined;
}

function bytes(source) {
  if (source instanceof Uint8Array) return source;
  if (source instanceof ArrayBuffer) return new Uint8Array(source);
  if (ArrayBuffer.isView(source)) return new Uint8Array(source.buffer, source.byteOffset, source.byteLength);
  throw new TypeError("expected firmware bytes");
}
